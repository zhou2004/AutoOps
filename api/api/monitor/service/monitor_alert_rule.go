package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"dodevops-api/api/monitor/dao"
	"dodevops-api/api/monitor/model"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type MonitorAlertRuleService interface {
	// Group
	CreateGroup(data *model.MonitorAlertGroupRule) error
	DeleteGroup(id uint) error
	UpdateGroup(data *model.MonitorAlertGroupRule) error
	GetGroupByID(id uint) (*model.MonitorAlertGroupRule, error)
	GetGroupList(page, pageSize int) ([]*model.MonitorAlertGroupRule, int64, error)

	// Rule
	CreateRule(data *model.MonitorAlertRule) error
	DeleteRule(id uint) error
	UpdateRule(data *model.MonitorAlertRule) error
	GetRuleByID(id uint) (*model.MonitorAlertRule, error)
	GetRuleListByGroup(groupId uint, page, pageSize int) ([]*model.MonitorAlertRule, int64, error)
	GetRuleList(req *model.MonitorAlertRuleQuery) ([]*model.MonitorAlertRule, int64, error)
}

type monitorAlertRuleService struct {
	groupRuleDao  dao.MonitorAlertGroupRuleDao
	ruleDao       dao.MonitorAlertRuleDao
	dataSourceDao dao.MonitorDataSourceDao
}

func NewMonitorAlertRuleService() MonitorAlertRuleService {
	s := &monitorAlertRuleService{
		groupRuleDao:  dao.NewMonitorAlertGroupRuleDao(),
		ruleDao:       dao.NewMonitorAlertRuleDao(),
		dataSourceDao: dao.NewMonitorDataSourceDao(),
	}

	go s.syncStatusLoop()
	return s
}

// ============== Kubernetes Engine ==============
type k8sDataSourceConfig struct {
	AuthType              string `json:"auth_type"`
	K8sApiUrl             string `json:"k8s_api_url"`
	Namespace             string `json:"namespace"`
	Token                 string `json:"token"`
	InsecureSkipTlsVerify bool   `json:"insecure_skip_tls_verify"`
}

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

func (s *monitorAlertRuleService) applyGroupToDataSources(group *model.MonitorAlertGroupRule, action string) error {
	ds, err := s.dataSourceDao.GetByID(group.DataSourceID)
	if err != nil {
		return err
	}
	if ds.Type == "Prometheus" && ds.DeployMethod == "Kubernetes" {
		return s.applyToKubernetes(ds, group.RuleContent, action)
	}
	return nil
}

// ============== Yaml AST Parsing Types ==============

type promYamlRoot struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   promMeta `yaml:"metadata"`
	Spec       promSpec `yaml:"spec"`
}

type promMeta struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

type promSpec struct {
	Groups []promGroup `yaml:"groups"`
}

type promGroup struct {
	Name  string      `yaml:"name"`
	Rules []promAlert `yaml:"rules"`
}

type promAlert struct {
	Alert       string            `yaml:"alert"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// yamlSyncGroupToRules (自上而下解析) 将 Group 的 RuleContent 拆解到子规则
func (s *monitorAlertRuleService) yamlSyncGroupToRules(group *model.MonitorAlertGroupRule) error {
	// 1. 删除旧规则
	s.ruleDao.DeleteByGroupID(group.ID)

	// 2. 解析群组YAML
	var doc promYamlRoot
	if err := yaml.Unmarshal([]byte(group.RuleContent), &doc); err != nil {
		return err
	}

	if len(doc.Spec.Groups) == 0 {
		return nil
	}

	pg := doc.Spec.Groups[0] // 仅保存第一个 Group

	// 构建子规则
	for _, pr := range pg.Rules {
		rLabels, _ := json.Marshal(pr.Labels)
		contentBytes, _ := yaml.Marshal(pr)

		rule := &model.MonitorAlertRule{
			GroupID:     group.ID,
			Alert:       pr.Alert,
			Expr:        pr.Expr,
			ForDuration: pr.For,
			Labels:      string(rLabels),
			Severity:    pr.Labels["severity"],
			Summary:     pr.Annotations["summary"],
			Description: pr.Annotations["description"],
			RuleContent: string(contentBytes),
			Status:      "inactive",
		}
		s.ruleDao.Create(rule)
	}

	return nil
}

// yamlSyncRulesToGroup (自下而上拼装) 将子规则拼装成 Group 的 RuleContent 并推向 K8s
func (s *monitorAlertRuleService) yamlSyncRulesToGroup(group *model.MonitorAlertGroupRule) error {
	rules, _ := s.ruleDao.GetByGroupID(group.ID)

	// 从群组读出 Labels 以便级联下发
	var groupLabels map[string]string
	_ = json.Unmarshal([]byte(group.Labels), &groupLabels)

	var doc promYamlRoot
	_ = yaml.Unmarshal([]byte(group.RuleContent), &doc) // 保持外壳
	if len(doc.Spec.Groups) == 0 {
		doc.Spec.Groups = []promGroup{{Name: group.GroupName}}
	}

	var newRules []promAlert
	for _, r := range rules {
		// 跳过未启用的规则
		if r.Enabled != nil && *r.Enabled == 0 {
			continue
		}
		var pr promAlert
		_ = yaml.Unmarshal([]byte(r.RuleContent), &pr)

		// 基础覆盖
		pr.Alert = r.Alert
		pr.Expr = r.Expr
		pr.For = r.ForDuration

		// Constraints处理：将非空的约束条件动态注入到 Expr 中
		if r.Constraints != "" && r.Constraints != "{}" {
			var constraintsMap map[string]string
			if err := json.Unmarshal([]byte(r.Constraints), &constraintsMap); err == nil {
				validConstraints := make(map[string]string)
				for k, v := range constraintsMap {
					if v != "" { // 过滤掉空值
						validConstraints[k] = v
					}
				}
				if len(validConstraints) > 0 {
					pr.Expr = modifyPromQL(pr.Expr, validConstraints)
				}
			}
		}

		// Labels合并与处理（Rule的Labels优先级大于Group的Labels）
		ruleL := make(map[string]string)
		_ = json.Unmarshal([]byte(r.Labels), &ruleL)

		finalLabels := make(map[string]string)
		for k, v := range groupLabels {
			finalLabels[k] = v // 来自Group
		}
		for k, v := range ruleL {
			finalLabels[k] = v // 覆盖Group
		}
		if r.Severity != "" {
			finalLabels["severity"] = r.Severity
		}
		pr.Labels = finalLabels

		// Annotations处理
		if pr.Annotations == nil {
			pr.Annotations = make(map[string]string)
		}
		if r.Summary != "" {
			pr.Annotations["summary"] = r.Summary
		}
		if r.Description != "" {
			pr.Annotations["description"] = r.Description
		}

		// 反向回写到子节点缓存 (确保单节点 YAML 正确)
		// b, _ := yaml.Marshal(pr)
		// newContent := string(b)

		// 优化: 如果最终生成的 YAML 与表里现存的完全一致，说明本条规则未被修改，则跳过不必要的全量子项 update 刷新
		// if r.RuleContent != newContent {
		// 	r.RuleContent = newContent
		// 	s.ruleDao.Update(r)
		// }

		newRules = append(newRules, pr)
	}

	doc.Spec.Groups[0].Rules = newRules

	// 写回群组
	outBytes, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	group.RuleContent = string(outBytes)
	s.groupRuleDao.Update(group)

	// 这里我们自动重新 apply K8s
	return s.applyGroupToDataSources(group, "apply")
}

// ============== CRUD for Group ==============

func (s *monitorAlertRuleService) CreateGroup(data *model.MonitorAlertGroupRule) error {
	if err := s.groupRuleDao.Create(data); err != nil {
		return err
	}
	s.yamlSyncGroupToRules(data)
	return s.applyGroupToDataSources(data, "apply")
}

func (s *monitorAlertRuleService) DeleteGroup(id uint) error {
	group, err := s.groupRuleDao.GetByID(id)
	if err != nil {
		return err
	}
	_ = s.applyGroupToDataSources(group, "delete")
	s.ruleDao.DeleteByGroupID(id)
	return s.groupRuleDao.Delete(id)
}

func (s *monitorAlertRuleService) UpdateGroup(data *model.MonitorAlertGroupRule) error {
	oldGroup, err := s.groupRuleDao.GetByID(data.ID)
	if err != nil {
		return err
	}

	// 判断是只改了外层基础字段还是直接传入了新的大 YAML 结构
	if data.RuleContent != oldGroup.RuleContent && data.RuleContent != "" {
		// 传入了新 YAML -> 冲刷子节点
		s.groupRuleDao.Update(data)
		s.yamlSyncGroupToRules(data)
	} else {
		// 只是改了基础字段 (如 GroupName 或 GroupLabels) - > 向下合并并且反推新 YAML
		if data.RuleContent == "" {
			data.RuleContent = oldGroup.RuleContent
		}
		s.groupRuleDao.Update(data)
		s.yamlSyncRulesToGroup(data)
	}

	return s.applyGroupToDataSources(data, "apply")
}

func (s *monitorAlertRuleService) GetGroupByID(id uint) (*model.MonitorAlertGroupRule, error) {
	return s.groupRuleDao.GetByID(id)
}

func (s *monitorAlertRuleService) GetGroupList(page, pageSize int) ([]*model.MonitorAlertGroupRule, int64, error) {
	return s.groupRuleDao.GetList(page, pageSize)
}

// ============== CRUD for Rule ==============

func (s *monitorAlertRuleService) CreateRule(data *model.MonitorAlertRule) error {
	data.Status = "inactive"
	if err := s.ruleDao.Create(data); err != nil {
		return err
	}
	group, _ := s.groupRuleDao.GetByID(data.GroupID)
	return s.yamlSyncRulesToGroup(group)
}

func (s *monitorAlertRuleService) DeleteRule(id uint) error {
	rule, _ := s.ruleDao.GetByID(id)
	if rule == nil {
		return nil
	}
	s.ruleDao.Delete(id)
	group, _ := s.groupRuleDao.GetByID(rule.GroupID)
	return s.yamlSyncRulesToGroup(group)
}

func (s *monitorAlertRuleService) UpdateRule(data *model.MonitorAlertRule) error {
	oldRule, err := s.ruleDao.GetByID(data.ID)
	if err != nil {
		return err
	}

	if data.RuleContent != oldRule.RuleContent && data.RuleContent != "" {
		// 修改了子规则的 YAML -> 解析抽出字段
		var pr promAlert
		if err := yaml.Unmarshal([]byte(data.RuleContent), &pr); err == nil {
			data.Alert = pr.Alert
			data.Expr = pr.Expr
			data.ForDuration = pr.For
			b, _ := json.Marshal(pr.Labels)
			data.Labels = string(b)
			data.Severity = pr.Labels["severity"]
			data.Summary = pr.Annotations["summary"]
			data.Description = pr.Annotations["description"]
		}
	}
	s.ruleDao.Update(data)
	group, _ := s.groupRuleDao.GetByID(oldRule.GroupID)
	return s.yamlSyncRulesToGroup(group)
}

func (s *monitorAlertRuleService) GetRuleByID(id uint) (*model.MonitorAlertRule, error) {
	return s.ruleDao.GetByID(id)
}

func (s *monitorAlertRuleService) GetRuleListByGroup(groupId uint, page, pageSize int) ([]*model.MonitorAlertRule, int64, error) {
	req := &model.MonitorAlertRuleQuery{
		GroupID:  groupId,
		Page:     page,
		PageSize: pageSize,
	}
	return s.ruleDao.GetListByQuery(req)
}

func (s *monitorAlertRuleService) GetRuleList(req *model.MonitorAlertRuleQuery) ([]*model.MonitorAlertRule, int64, error) {
	return s.ruleDao.GetListByQuery(req)
}

// ============== Sync Status Background ==============
func (s *monitorAlertRuleService) syncStatusLoop() {
	time.Sleep(5 * time.Second)
	for {
		s.syncStatusOnce()
		time.Sleep(30 * time.Second)
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

func (s *monitorAlertRuleService) syncStatusOnce() {
	dsList, _, _ := s.dataSourceDao.GetList(0, 0)
	groups, err := s.groupRuleDao.GetAll()
	if err != nil || len(groups) == 0 {
		return
	}

	dsMap := make(map[uint]*model.MonitorDataSource)
	for _, ds := range dsList {
		if strings.EqualFold(ds.Type, "Prometheus") {
			dsMap[ds.ID] = ds
		}
	}

	groupsByDS := make(map[uint][]*model.MonitorAlertGroupRule)
	for _, g := range groups {
		groupsByDS[g.DataSourceID] = append(groupsByDS[g.DataSourceID], g)
	}

	for dsID, ds := range dsMap {
		gs := groupsByDS[dsID]
		if len(gs) == 0 {
			continue
		}

		url := fmt.Sprintf("%s/api/v1/rules?type=alert", strings.TrimRight(ds.ApiUrl, "/"))
		resp, err := http.Get(url)
		if err != nil {
			continue
		}

		var promResp promRulesResponse
		if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if promResp.Status != "success" {
			continue
		}

		alertStates := make(map[string]string)
		for _, proMg := range promResp.Data.Groups {
			for _, r := range proMg.Rules {
				if r.Type == "alerting" && r.Name != "" {
					existing := alertStates[r.Name]
					if existing == "" || r.State == "firing" || (r.State == "pending" && existing == "inactive") {
						alertStates[r.Name] = r.State
					}
				}
			}
		}

		for _, g := range gs {
			rules, _ := s.ruleDao.GetByGroupID(g.ID)
			for _, r := range rules {
				if r.Alert == "" {
					continue
				}
				state := alertStates[r.Alert]
				if state == "" {
					state = "inactive"
				}
				if r.Status != state {
					log.Printf("[状态更新] 规则 [%s] (ID:%d) 状态由于发生变化正在更新: %s -> %s\n", r.Alert, r.ID, r.Status, state)
					s.ruleDao.UpdateStatus(r.ID, state)
				}
			}
		}
	}
}

// ProcessRuleYAML 接收 YAML 字符串以及 Labels 和 Constraints 的 JSON 格式字符串，将它们合并并注入到 expr 中，返回处理后的 YAML 字符串
func ProcessRuleYAML(yamlData string, labelsJSON string, constraintsJSON string) string {
	// 1. 合并解析 JSON (Labels 和 Constraints)
	newLabels := make(map[string]string)
	if labelsJSON != "" && labelsJSON != "{}" {
		if err := json.Unmarshal([]byte(labelsJSON), &newLabels); err != nil {
			log.Printf("解析 JSON 标签失败: %v", err)
		}
	}
	if constraintsJSON != "" && constraintsJSON != "{}" {
		var constraints map[string]string
		if err := json.Unmarshal([]byte(constraintsJSON), &constraints); err != nil {
			log.Printf("解析 JSON 约束失败: %v", err)
		}
		for k, v := range constraints {
			if v != "" { // 如果值为 ""，表示没有约束，忽略
				newLabels[k] = v
			}
		}
	}

	// 2. 解析 YAML (在这里将传入的 string 转换为 []byte)
	var root yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &root)
	if err != nil {
		log.Fatalf("解析 YAML 失败: %v", err)
	}

	// 3. 递归处理
	walkYAML(&root, newLabels)

	// 4. 重新生成 YAML
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	err = encoder.Encode(&root)
	if err != nil {
		log.Fatalf("生成 YAML 失败: %v", err)
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
	expr, err := parser.NewParser(parser.Options{}).ParseExpr(query)
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
