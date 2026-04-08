package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"dodevops-api/api/monitor/dao"
	"dodevops-api/api/monitor/model"
	"dodevops-api/common"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"gopkg.in/yaml.v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	// "sigs.k8s.io/yaml"
)

type MonitorAlertRuleService interface {
	Create(data *model.MonitorAlertRule) error
	Delete(id uint) error
	Update(data *model.MonitorAlertRule) error
	GetByID(id uint) (*model.MonitorAlertRule, error)
	GetList(page, pageSize int) ([]*model.MonitorAlertRule, int64, error)
}

type monitorAlertRuleService struct {
	ruleDao       dao.MonitorAlertRuleDao
	dataSourceDao dao.MonitorDataSourceDao
}

func NewMonitorAlertRuleService() MonitorAlertRuleService {
	s := &monitorAlertRuleService{
		ruleDao:       dao.NewMonitorAlertRuleDao(),
		dataSourceDao: dao.NewMonitorDataSourceDao(),
	}

	// 启动后台轮询，从 Prometheus 直接同步状态到 DB
	go s.syncStatusLoop()
	return s
}

// syncStatusLoop 定时调用 Prometheus 原生 API 更新每个 Rule 的最新状态
func (s *monitorAlertRuleService) syncStatusLoop() {
	time.Sleep(5 * time.Second) // 等待服务启动
	for {
		s.syncStatusOnce()
		time.Sleep(30 * time.Second) // 每30s轮询一次
	}
}

type promRulesResponse struct {
	Status string `json:"status"`
	Data   struct {
		Groups []struct {
			Rules []struct {
				State string `json:"state"`
				Name  string `json:"name"`
				Type  string `json:"type"`
			} `json:"rules"`
		} `json:"groups"`
	} `json:"data"`
}

type localPromRuleDoc struct {
	Spec struct {
		Groups []struct {
			Rules []struct {
				Alert string `yaml:"alert"`
			} `yaml:"rules"`
		} `yaml:"groups"`
	} `yaml:"spec"`
}

func (s *monitorAlertRuleService) syncStatusOnce() {
	// 获取所有数据源和规则
	dsList, _, err := s.dataSourceDao.GetList(0, 0)
	if err != nil {
		log.Printf("[状态同步] 获取数据源失败: %v\n", err)
		return
	}
	ruleList, _, err := s.ruleDao.GetList(0, 0)
	if err != nil || len(ruleList) == 0 {
		return
	}

	// 映射 Data Source ID -> DS
	dsMap := make(map[uint]*model.MonitorDataSource)
	for _, ds := range dsList {
		// 忽略大小写匹配
		if strings.EqualFold(ds.Type, "Prometheus") {
			dsMap[ds.ID] = ds
		}
	}

	// 分组 Rules by DataSourceID
	rulesByDS := make(map[uint][]*model.MonitorAlertRule)
	for _, r := range ruleList {
		rulesByDS[r.DataSourceID] = append(rulesByDS[r.DataSourceID], r)
	}

	// 对每个 Prometheus 数据源更新
	for dsID, ds := range dsMap {
		rules := rulesByDS[dsID]
		if len(rules) == 0 {
			continue
		}

		// 调用 Prometheus /api/v1/rules
		url := fmt.Sprintf("%s/api/v1/rules?type=alert", strings.TrimRight(ds.ApiUrl, "/"))
		log.Printf("[同步检测] 数据源 ID=%d 准备请求 Prometheus API: %s\n", dsID, url)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("[状态同步] 无法连接 %s - %v\n", url, err)
			continue
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("[状态同步] 解析包体数据失败: %v\n", err)
			continue
		}

		var promResp promRulesResponse
		if err := json.Unmarshal(bodyBytes, &promResp); err != nil {
			snippet := string(bodyBytes)
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
			log.Printf("[状态同步] json解析出错: %v (响应前段: %s)\n", err, snippet)
			continue
		}

		if promResp.Status != "success" {
			continue
		}

		// 解析 Prometheus 返回的每个规则名 -> 状态的映射 (取最严重状态: firing > pending > inactive)
		alertStates := make(map[string]string)
		for _, g := range promResp.Data.Groups {
			for _, r := range g.Rules {
				if r.Type == "alerting" && r.Name != "" {
					existing := alertStates[r.Name]
					if existing == "" || r.State == "firing" || (r.State == "pending" && existing == "inactive") {
						alertStates[r.Name] = r.State
					}
				}
			}
		}

		// 根据 DB 里的 YAML 解出 Rule 里涉及的 alert 对象，结合最新的 Status 更新数据库
		for _, r := range rules {
			// **核心优化**: 使用 Redis 高速分布缓存存放当前规则包含的具体 Alert 名称集合，从而大幅度降低 CPU 算力损耗 (原 YAML 解析非常缓慢)
			var alertNames []string
			needParse := true // 是否需要进行厚重的 YAML 递归解析
			cacheKey := fmt.Sprintf("autoops:monitor:alert_rule:%d:alerts", r.ID)
			ctx := context.Background()
			redisCli := common.GetRedisClient()

			// 1. **缓存读取**: 验证 Redis 连接有效性，尝试命中缓存记录
			if redisCli != nil {
				cachedStr, err := redisCli.Get(ctx, cacheKey).Result()
				if err == nil && cachedStr != "" {
					// 缓存命中: 进行无损轻量级的反序列化
					if jsonErr := json.Unmarshal([]byte(cachedStr), &alertNames); jsonErr == nil {
						needParse = false
					}
				}
			}

			// 2. **缓存穿透回源**: 未命中缓存则立即启动 YAML 离线计算
			if needParse {
				var doc localPromRuleDoc
				if err := yaml.Unmarshal([]byte(r.RuleContent), &doc); err == nil {
					for _, g := range doc.Spec.Groups {
						for _, pr := range g.Rules {
							if pr.Alert != "" {
								alertNames = append(alertNames, pr.Alert)
							}
						}
					}
				}
				// 3. **缓存写入**: 将高净值计算结果持久化推送到 Redis 中（设定TTL防止脏数据存留太久，同时在 Update/Delete 时做主动清理）
				if redisCli != nil {
					if b, err := json.Marshal(alertNames); err == nil {
						redisCli.Set(ctx, cacheKey, string(b), 24*time.Hour)
					}
				}
			}

			finalStatus := "inactive"
			for _, alertName := range alertNames {
				state := alertStates[alertName]
				if state == "firing" {
					finalStatus = "firing"
				} else if state == "pending" && finalStatus != "firing" {
					finalStatus = "pending"
				}
			}

			// 只更新发生状态改变的规则数据
			if r.Status != finalStatus {
				log.Printf("[状态更新] 规则 [%s] (ID:%d) 状态由于发生变化正在更新: %s -> %s\n", r.Name, r.ID, r.Status, finalStatus)
				r.Status = finalStatus
				_ = s.ruleDao.Update(r)
			}
		}
	}
}

// 模拟 k8s 配置结构
type k8sDataSourceConfig struct {
	AuthType              string `json:"auth_type"`
	K8sApiUrl             string `json:"k8s_api_url"`
	Namespace             string `json:"namespace"`
	Token                 string `json:"token"`
	InsecureSkipTlsVerify bool   `json:"insecure_skip_tls_verify"`
}

// applyToKubernetes 通过 K8s 动态客户端将规则应用到 K8s 集群
func (s *monitorAlertRuleService) applyToKubernetes(dataSource *model.MonitorDataSource, ruleContent string, action string) error {
	var k8sConfig k8sDataSourceConfig
	if err := json.Unmarshal([]byte(dataSource.Config), &k8sConfig); err != nil {
		return fmt.Errorf("解析数据源配置失败: %v", err)
	}

	config := &rest.Config{
		Host:        k8sConfig.K8sApiUrl,
		BearerToken: k8sConfig.Token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: k8sConfig.InsecureSkipTlsVerify,
		},
	}

	var un unstructured.Unstructured
	if err := yaml.Unmarshal([]byte(ruleContent), &un.Object); err != nil {
		return fmt.Errorf("解析YAML内容失败: %v", err)
	}

	if k8sConfig.Namespace != "" && un.GetNamespace() == "" {
		un.SetNamespace(k8sConfig.Namespace)
	}
	namespace := un.GetNamespace()
	if namespace == "" {
		namespace = "default"
	}
	name := un.GetName()
	if name == "" {
		return fmt.Errorf("YAML中未找到 metadata.name")
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("创建K8s动态客户端失败: %v", err)
	}

	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("创建K8s发现客户端失败: %v", err)
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	gvk := un.GroupVersionKind()
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("无法映射Kind到Resource: %v", err)
	}

	var resourceInterface dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		resourceInterface = dynClient.Resource(mapping.Resource).Namespace(namespace)
	} else {
		resourceInterface = dynClient.Resource(mapping.Resource)
	}

	ctx := context.TODO()

	if action == "apply" {
		existing, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				_, createErr := resourceInterface.Create(ctx, &un, metav1.CreateOptions{})
				if createErr != nil {
					return fmt.Errorf("创建资源失败: %v", createErr)
				}
			} else {
				return fmt.Errorf("查询资源状态失败: %v", err)
			}
		} else {
			un.SetResourceVersion(existing.GetResourceVersion())
			_, updateErr := resourceInterface.Update(ctx, &un, metav1.UpdateOptions{})
			if updateErr != nil {
				return fmt.Errorf("更新资源失败: %v", updateErr)
			}
		}
	} else if action == "delete" {
		err := resourceInterface.Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("删除资源失败: %v", err)
		}
	}

	return nil
}

// syncRuleToDataSource 同步规则到对应的数据源
func (s *monitorAlertRuleService) syncRuleToDataSource(rule *model.MonitorAlertRule, action string) error {
	dataSource, err := s.dataSourceDao.GetByID(rule.DataSourceID)
	if err != nil {
		return fmt.Errorf("获取数据源失败: %v", err)
	}

	// 根据数据源类型和部署方式执行对应逻辑
	if dataSource.Type == "Prometheus" && dataSource.DeployMethod == "Kubernetes" {
		return s.applyToKubernetes(dataSource, ProcessRuleYAML(rule.RuleContent, rule.Labels), action)
	}

	return nil
}

func (s *monitorAlertRuleService) Create(data *model.MonitorAlertRule) error {
	// 1. 同步到数据源（如 Kubernetes）
	if err := s.syncRuleToDataSource(data, "apply"); err != nil {
		return err
	}

	// 2. 状态默认为 inactive，创建入库
	if data.Status == "" {
		data.Status = "inactive"
	}
	return s.ruleDao.Create(data)
}

func (s *monitorAlertRuleService) Delete(id uint) error {
	// 联动清理该规则在 Redis 里面的 YAML 解析缓存 (防内存溢出与脏存)
	if redisCli := common.GetRedisClient(); redisCli != nil {
		cacheKey := fmt.Sprintf("autoops:monitor:alert_rule:%d:alerts", id)
		redisCli.Del(context.Background(), cacheKey)
	}

	// 1. 获取规则
	rule, err := s.ruleDao.GetByID(id)
	if err != nil {
		return err
	}

	// 2. 从数据源（如 Kubernetes）中删除规则
	// 忽略删除失败的错误或记录日志，以防止 K8s 中资源不存在时无法在 DB 中删除
	_ = s.syncRuleToDataSource(rule, "delete")

	// 3. 数据库中删除
	return s.ruleDao.Delete(id)
}

func (s *monitorAlertRuleService) Update(data *model.MonitorAlertRule) error {
	// 每次更新规则也重新主动删掉对应的 Redis 缓存键，促使下一循环读取最新配置文件进行热计算并重新记录
	if redisCli := common.GetRedisClient(); redisCli != nil {
		cacheKey := fmt.Sprintf("autoops:monitor:alert_rule:%d:alerts", data.ID)
		redisCli.Del(context.Background(), cacheKey)
	}

	// 1. 验证存在与否
	oldRule, err := s.ruleDao.GetByID(data.ID)
	if err != nil {
		return err
	}

	// 覆盖 DataSourceID 以防止被错误更新（或者直接取传递的）
	if data.DataSourceID == 0 {
		data.DataSourceID = oldRule.DataSourceID
	}

	// 2. 同步到数据源
	if err := s.syncRuleToDataSource(data, "apply"); err != nil {
		return err
	}

	// 3. 更新入库
	return s.ruleDao.Update(data)
}

func (s *monitorAlertRuleService) GetByID(id uint) (*model.MonitorAlertRule, error) {
	return s.ruleDao.GetByID(id)
}

func (s *monitorAlertRuleService) GetList(page, pageSize int) ([]*model.MonitorAlertRule, int64, error) {
	return s.ruleDao.GetList(page, pageSize)
}

// ProcessRuleYAML 接收 YAML 字符串和 JSON 标签字符串，返回处理后的 YAML 字符串
func ProcessRuleYAML(yamlData string, labelsJSON string) string {
	// 1. 解析 JSON
	var newLabels map[string]string
	err := json.Unmarshal([]byte(labelsJSON), &newLabels)
	if err != nil {
		log.Printf("解析 JSON 标签失败: %v", err)
		return yamlData
	}

	// 2. 解析 YAML (在这里将传入的 string 转换为 []byte)
	var root yaml.Node
	err = yaml.Unmarshal([]byte(yamlData), &root)
	if err != nil {
		log.Printf("解析 YAML 失败: %v", err)
		return yamlData
	}

	// 3. 递归处理
	walkYAML(&root, newLabels)

	// 4. 重新生成 YAML
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	err = encoder.Encode(&root)
	if err != nil {
		log.Printf("生成 YAML 失败: %v", err)
		return yamlData
	}

	// 5. 直接将 Buffer 转换为 string 返回
	return buf.String()
}

func walkYAML(node *yaml.Node, newLabels map[string]string) {
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]

			if keyNode.Value == "expr" && valNode.Kind == yaml.ScalarNode {
				valNode.Value = modifyPromQL(valNode.Value, newLabels)
			} else {
				walkYAML(valNode, newLabels)
			}
		}
	} else if node.Kind == yaml.SequenceNode || node.Kind == yaml.DocumentNode {
		for _, child := range node.Content {
			walkYAML(child, newLabels)
		}
	}
}

// modifyPromQL 使用 AST 引擎进行纯粹的新增与覆盖
func modifyPromQL(query string, newLabels map[string]string) string {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		log.Printf("警告: 无法解析 PromQL (%s): %v\n", query, err)
		return query
	}

	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		if vs, ok := node.(*parser.VectorSelector); ok {
			var newMatchers []*labels.Matcher

			for _, m := range vs.LabelMatchers {
				if m.Name == "__name__" {
					newMatchers = append(newMatchers, m)
					continue
				}

				if _, exists := newLabels[m.Name]; !exists {
					newMatchers = append(newMatchers, m)
				}
			}

			for k, v := range newLabels {
				newMatcher, err := labels.NewMatcher(labels.MatchEqual, k, v)
				if err == nil {
					newMatchers = append(newMatchers, newMatcher)
				}
			}

			vs.LabelMatchers = newMatchers
		}
		return nil
	})

	return expr.String()
}
