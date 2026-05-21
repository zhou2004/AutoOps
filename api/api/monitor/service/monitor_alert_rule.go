package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/monitor/dao"
	"dodevops-api/api/monitor/model"
	"dodevops-api/common"
	"dodevops-api/common/config"

	"github.com/go-redis/redis/v8"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"

	"gopkg.in/yaml.v3"
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
	CheckRuleExpr(dataSourceId uint, expr string) (interface{}, error)
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

	go s.evaluateRulesLoop()
	return s
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

// yamlSyncRulesToGroup (自下而上拼装) 将子规则拼装成 Group 的 RuleContent
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

		newRules = append(newRules, pr)
	}

	doc.Spec.Groups[0].Rules = newRules

	// 写回群组
	outBytes, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	group.RuleContent = string(outBytes)
	return s.groupRuleDao.Update(group)
}

// ============== CRUD for Group ==============

func (s *monitorAlertRuleService) CreateGroup(data *model.MonitorAlertGroupRule) error {
	if err := s.groupRuleDao.Create(data); err != nil {
		return err
	}
	return s.yamlSyncGroupToRules(data)
}

func (s *monitorAlertRuleService) DeleteGroup(id uint) error {
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
		return s.yamlSyncGroupToRules(data)
	} else {
		// 只是改了基础字段 (如 GroupName 或 GroupLabels) - > 向下合并并且反推新 YAML
		if data.RuleContent == "" {
			data.RuleContent = oldGroup.RuleContent
		}
		s.groupRuleDao.Update(data)
		return s.yamlSyncRulesToGroup(data)
	}
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

// ============== Rule Processing ============== //
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

// ============== Evaluation Engine ============== //
func (s *monitorAlertRuleService) evaluateRulesLoop() {
	time.Sleep(5 * time.Second)
	for {
		s.evaluateRulesOnce()
		time.Sleep(30 * time.Second)
	}
}

type PromQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type AlertEvalState struct {
	ActiveAt int64  `json:"active_at"`
	State    string `json:"state"` // "pending" | "firing"
}

func (s *monitorAlertRuleService) evaluateRulesOnce() {
	ctx := context.Background()
	rdb := common.GetRedisClient()
	if rdb == nil {
		log.Println("[Warning] Redis client is nil, skip alert evaluation")
		return
	}

	// 0. Evaluate internal monitor rules (domain_cert / api_endpoint)
	s.evaluateInternalRules(ctx, rdb)

	// 1. Get all groups and datasources
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

	// 2. Get active rules
	enabledVal := 1
	activeRules, _, _ := s.ruleDao.GetListByQuery(&model.MonitorAlertRuleQuery{
		Enabled: &enabledVal,
	})
	ruleMap := make(map[uint]*model.MonitorAlertRule)
	rulesByGroup := make(map[uint][]*model.MonitorAlertRule)
	for _, r := range activeRules {
		ruleMap[r.ID] = r
		rulesByGroup[r.GroupID] = append(rulesByGroup[r.GroupID], r)
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}

	for dsID, ds := range dsMap {
		gs := groupsByDS[dsID]
		for _, g := range gs {
			rules := rulesByGroup[g.ID]
			for _, r := range rules {
				if r.Expr == "" || r.Alert == "" {
					continue
				}
				// Skip internal-style rules, processed by evaluateInternalRules
				if r.Style == "domain_cert" || r.Style == "api_endpoint" {
					continue
				}

				evalExpr := r.Expr
				if r.Constraints != "" && r.Constraints != "{}" {
					var constraintsMap map[string]string
					if err := json.Unmarshal([]byte(r.Constraints), &constraintsMap); err == nil {
						validConstraints := make(map[string]string)
						for k, v := range constraintsMap {
							if v != "" {
								validConstraints[k] = v
							}
						}
						if len(validConstraints) > 0 {
							evalExpr = modifyPromQL(evalExpr, validConstraints)
						}
					}
				}

				// 3. Build query URL directly to data source
				u, _ := url.Parse(fmt.Sprintf("%s/api/v1/query", strings.TrimRight(ds.ApiUrl, "/")))
				q := u.Query()
				q.Set("query", evalExpr)
				u.RawQuery = q.Encode()

				resp, err := httpClient.Get(u.String())
				if err != nil {
					continue
				}

				var promResp PromQueryResponse
				if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
					resp.Body.Close()
					continue
				}
				resp.Body.Close()

				if promResp.Status != "success" {
					continue
				}

				// 4. Process result to update State and Fire Webhooks
				s.processRuleEvaluation(ctx, rdb, r, promResp.Data.Result)
			}
		}
	}
}

// evaluateInternalRules 评估 domain_cert 和 api_endpoint 风格的规则
func (s *monitorAlertRuleService) evaluateInternalRules(ctx context.Context, rdb *redis.Client) {
	enabledVal := 1
	activeRules, _, _ := s.ruleDao.GetListByQuery(&model.MonitorAlertRuleQuery{
		Enabled: &enabledVal,
	})
	if len(activeRules) == 0 {
		return
	}

	// 获取所有域名证书和API端点数据
	domainCertSvc := NewDomainCertService().(*DomainCertServiceImpl)
	apiEndpointSvc := NewAPIEndpointService().(*APIEndpointServiceImpl)

	domainCerts, _ := domainCertSvc.GetAllForEval()
	apiEndpoints, _ := apiEndpointSvc.GetAllForEval()

	for _, r := range activeRules {
		if r.Expr == "" || r.Alert == "" || r.Enabled == nil || *r.Enabled == 0 {
			continue
		}
		if r.Style != "domain_cert" && r.Style != "api_endpoint" {
			continue
		}

		if r.Style == "domain_cert" {
			s.evalDomainCertRule(ctx, rdb, r, domainCerts)
		} else if r.Style == "api_endpoint" {
			s.evalAPIEndpointRule(ctx, rdb, r, apiEndpoints)
		}
	}
}

// evalDomainCertRule 评估域名证书规则
func (s *monitorAlertRuleService) evalDomainCertRule(ctx context.Context, rdb *redis.Client, r *model.MonitorAlertRule, certs []model.DomainCert) {
	dur, _ := time.ParseDuration(r.ForDuration)
	now := time.Now()

	var results []struct {
		Metric map[string]string `json:"metric"`
		Value  []interface{}     `json:"value"`
	}

	for _, cert := range certs {
		matched := s.matchDomainCertRule(r, cert)
		metric := map[string]string{
			"domain":         cert.Domain,
			"port":           strconv.Itoa(cert.Port),
			"status":         strconv.Itoa(cert.Status),
			"remaining_days": strconv.Itoa(cert.RemainingDays),
		}

		value := []interface{}{float64(now.Unix()), "0"}
		if matched {
			value = []interface{}{float64(now.Unix()), "1"}
		}
		results = append(results, struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		}{Metric: metric, Value: value})
	}

	s.processInternalEval(ctx, rdb, r, results, dur)
}

// evalAPIEndpointRule 评估API端点规则
func (s *monitorAlertRuleService) evalAPIEndpointRule(ctx context.Context, rdb *redis.Client, r *model.MonitorAlertRule, endpoints []model.MonitorAPIEndpoint) {
	dur, _ := time.ParseDuration(r.ForDuration)
	now := time.Now()

	var results []struct {
		Metric map[string]string `json:"metric"`
		Value  []interface{}     `json:"value"`
	}

	for _, ep := range endpoints {
		matched := s.matchAPIEndpointRule(r, ep)
		metric := map[string]string{
			"name":              ep.Name,
			"url":               ep.URL,
			"method":            ep.Method,
			"status":            strconv.Itoa(ep.Status),
			"last_status_code":  strconv.Itoa(ep.LastStatusCode),
			"last_response_ms":  strconv.FormatInt(ep.LastResponseTime, 10),
		}

		value := []interface{}{float64(now.Unix()), "0"}
		if matched {
			value = []interface{}{float64(now.Unix()), "1"}
		}
		results = append(results, struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		}{Metric: metric, Value: value})
	}

	s.processInternalEval(ctx, rdb, r, results, dur)
}

// matchDomainCertRule 检查单条域名证书记录是否匹配规则条件
// expr支持: status != 1, remaining_days <= 30, status > 1 等形式
// constraints支持: {"domain": "xxx.test.com", "port": "443"} 做精准匹配
func (s *monitorAlertRuleService) matchDomainCertRule(r *model.MonitorAlertRule, cert model.DomainCert) bool {
	// 先检查约束条件 (Constraints) 是否匹配
	if r.Constraints != "" && r.Constraints != "{}" {
		var constraints map[string]string
		if err := json.Unmarshal([]byte(r.Constraints), &constraints); err == nil {
			// 约束中的字段必须全部匹配
			for k, v := range constraints {
				if v == "" {
					continue
				}
				switch k {
				case "domain", "domain_name":
					if cert.Domain != v {
						return false
					}
				case "port":
					portStr := strconv.Itoa(cert.Port)
					if portStr != v {
						return false
					}
				case "status":
					statusStr := strconv.Itoa(cert.Status)
					if statusStr != v {
						return false
					}
				}
			}
		}
	}

	expr := strings.TrimSpace(r.Expr)
	if expr == "" {
		return false
	}

	// 简单条件解析: remaining_days <= 30, status != 1, status > 1 等
	parts := strings.SplitN(expr, " ", 3)
	if len(parts) < 3 {
		return false
	}

	field := strings.TrimSpace(parts[0])
	op := strings.TrimSpace(parts[1])
	valStr := strings.TrimSpace(parts[2])

	var fieldVal int
	switch field {
	case "remaining_days":
		fieldVal = cert.RemainingDays
	case "status":
		fieldVal = cert.Status
	default:
		return false
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return false
	}

	switch op {
	case "<=":
		return fieldVal <= val
	case ">=":
		return fieldVal >= val
	case "<":
		return fieldVal < val
	case ">":
		return fieldVal > val
	case "==", "=":
		return fieldVal == val
	case "!=":
		return fieldVal != val
	}
	return false
}

// matchAPIEndpointRule 检查单条API端点记录是否匹配规则条件
// constraints支持: {"name": "生产环境API", "url": "https://api.example.com/health"}
func (s *monitorAlertRuleService) matchAPIEndpointRule(r *model.MonitorAlertRule, ep model.MonitorAPIEndpoint) bool {
	// 先检查约束条件 (Constraints) 是否匹配
	if r.Constraints != "" && r.Constraints != "{}" {
		var constraints map[string]string
		if err := json.Unmarshal([]byte(r.Constraints), &constraints); err == nil {
			for k, v := range constraints {
				if v == "" {
					continue
				}
				switch k {
				case "name":
					if ep.Name != v {
						return false
					}
				case "url":
					if ep.URL != v {
						return false
					}
				case "method":
					if ep.Method != v {
						return false
					}
				case "status":
					statusStr := strconv.Itoa(ep.Status)
					if statusStr != v {
						return false
					}
				}
			}
		}
	}

	expr := strings.TrimSpace(r.Expr)
	if expr == "" {
		return false
	}

	parts := strings.SplitN(expr, " ", 3)
	if len(parts) < 3 {
		return false
	}

	field := strings.TrimSpace(parts[0])
	op := strings.TrimSpace(parts[1])
	valStr := strings.TrimSpace(parts[2])

	var fieldVal int
	switch field {
	case "status":
		fieldVal = ep.Status
	case "last_status_code":
		fieldVal = ep.LastStatusCode
	case "last_response_ms":
		fieldVal = int(ep.LastResponseTime)
	default:
		return false
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return false
	}

	switch op {
	case "<=":
		return fieldVal <= val
	case ">=":
		return fieldVal >= val
	case "<":
		return fieldVal < val
	case ">":
		return fieldVal > val
	case "==", "=":
		return fieldVal == val
	case "!=":
		return fieldVal != val
	}
	return false
}

// processInternalEval 处理内部评估结果，使用 Redis 状态管理
func (s *monitorAlertRuleService) processInternalEval(ctx context.Context, rdb *redis.Client, r *model.MonitorAlertRule, results []struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}, dur time.Duration) {
	hashKey := fmt.Sprintf("alert:eval:rule:%d", r.ID)
	activeFps := make(map[string]bool)
	currentRuleStatus := "inactive"

	for _, res := range results {
		fp := metricFingerprint(res.Metric)
		activeFps[fp] = true

		var state AlertEvalState
		valData, err := rdb.HGet(ctx, hashKey, fp).Result()
		if err == redis.Nil {
			state = AlertEvalState{ActiveAt: time.Now().Unix(), State: "pending"}
		} else if err == nil {
			json.Unmarshal([]byte(valData), &state)
		} else {
			continue
		}

		// 检查是否匹配（Value[1] 为 "1" 表示匹配）
		isFiring := false
		if len(res.Value) >= 2 {
			if valStr, ok := res.Value[1].(string); ok && valStr == "1" {
				isFiring = true
			}
		}

		oldState := state.State

		if isFiring {
			// 匹配中: pending -> firing (如果持续时间到了)
			if state.State == "pending" {
				if time.Now().Unix()-state.ActiveAt >= int64(dur.Seconds()) {
					state.State = "firing"
				}
			}
			// firing 保持 firing, inactive 转为 pending
			if state.State == "inactive" {
				state.State = "pending"
				state.ActiveAt = time.Now().Unix()
			}
		} else {
			// 不再匹配: 从 firing 转为 resolved
			if state.State == "firing" {
				go sendWebhook(r, res.Metric, "resolved", time.Unix(state.ActiveAt, 0))
				rdb.HDel(ctx, hashKey, fp)
				// 从 activeFps 移除, 避免被下面的清理重复处理
				delete(activeFps, fp)
				continue
			}
			// pending 或不活跃 -> 删除状态, 下次重新开始
			rdb.HDel(ctx, hashKey, fp)
			delete(activeFps, fp)
			continue
		}

		if state.State == "firing" {
			currentRuleStatus = "firing"
		} else if state.State == "pending" && currentRuleStatus != "firing" {
			currentRuleStatus = "pending"
		}

		if state.State == "firing" && oldState != "firing" {
			go sendWebhook(r, res.Metric, "firing", time.Unix(state.ActiveAt, 0))
		}

		stateBytes, _ := json.Marshal(state)
		rdb.HSet(ctx, hashKey, fp, string(stateBytes))
	}

	// 清理已恢复的 (metric 从数据源完全消失)
	allFps, _ := rdb.HGetAll(ctx, hashKey).Result()
	for fp, valStr := range allFps {
		if !activeFps[fp] {
			var state AlertEvalState
			json.Unmarshal([]byte(valStr), &state)
			if state.State == "firing" {
				go sendWebhook(r, nil, "resolved", time.Unix(state.ActiveAt, 0))
			}
			rdb.HDel(ctx, hashKey, fp)
		}
	}

	if r.Status != currentRuleStatus {
		log.Printf("[状态更新] 内部规则 [%s] (ID:%d) 状态变化: %s -> %s\n", r.Alert, r.ID, r.Status, currentRuleStatus)
		s.ruleDao.UpdateStatus(r.ID, currentRuleStatus)
	}
}

func metricFingerprint(metric map[string]string) string {
	bytes, _ := json.Marshal(metric)
	return string(bytes)
}

func (s *monitorAlertRuleService) processRuleEvaluation(ctx context.Context, rdb *redis.Client, r *model.MonitorAlertRule, results []struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}) {
	hashKey := fmt.Sprintf("alert:eval:rule:%d", r.ID)
	activeFps := make(map[string]bool)

	currentRuleStatus := "inactive"

	dur, err := time.ParseDuration(r.ForDuration)
	if err != nil {
		dur = 0
	}

	// Deal with current triggered results
	for _, res := range results {
		fp := metricFingerprint(res.Metric)
		activeFps[fp] = true

		var state AlertEvalState
		valData, err := rdb.HGet(ctx, hashKey, fp).Result()
		if err == redis.Nil {
			// First time seen
			state = AlertEvalState{ActiveAt: time.Now().Unix(), State: "pending"}
		} else if err == nil {
			json.Unmarshal([]byte(valData), &state)
		} else {
			continue
		}

		oldState := state.State
		if state.State == "pending" {
			// Check if duration has passed
			if time.Now().Unix()-state.ActiveAt >= int64(dur.Seconds()) {
				state.State = "firing"
			}
		}

		if state.State == "firing" {
			currentRuleStatus = "firing"
		} else if state.State == "pending" && currentRuleStatus != "firing" {
			currentRuleStatus = "pending"
		}

		// Transition: pending -> firing
		if state.State == "firing" && oldState != "firing" {
			go sendWebhook(r, res.Metric, "firing", time.Unix(state.ActiveAt, 0))
		}

		stateBytes, _ := json.Marshal(state)
		rdb.HSet(ctx, hashKey, fp, string(stateBytes))
	}

	// Clean up resolved (metric fingerprints that were firing but no longer exist)
	allFps, _ := rdb.HGetAll(ctx, hashKey).Result()
	for fp, valStr := range allFps {
		if !activeFps[fp] {
			var state AlertEvalState
			json.Unmarshal([]byte(valStr), &state)
			if state.State == "firing" {
				var metricMap map[string]string
				json.Unmarshal([]byte(fp), &metricMap)
				go sendWebhook(r, metricMap, "resolved", time.Unix(state.ActiveAt, 0))
			}
			rdb.HDel(ctx, hashKey, fp)
		}
	}

	// Update Rule Status in MySQL if needed
	if r.Status != currentRuleStatus {
		log.Printf("[状态更新] 自建引擎检查规则 [%s] (ID:%d) 状态变化: %s -> %s\n", r.Alert, r.ID, r.Status, currentRuleStatus)
		s.ruleDao.UpdateStatus(r.ID, currentRuleStatus)
	}
}

type AlertWebhookPayload struct {
	Status string `json:"status"` // firing or resolved
	Alerts []struct {
		Status      string            `json:"status"`
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
		StartsAt    string            `json:"startsAt"`
		EndsAt      string            `json:"endsAt"`
	} `json:"alerts"`
}

func sendWebhook(r *model.MonitorAlertRule, metric map[string]string, status string, startsAt time.Time) {
	payload := AlertWebhookPayload{
		Status: status,
	}

	labels := make(map[string]string)
	for k, v := range metric {
		labels[k] = v
	}
	labels["alertname"] = r.Alert
	labels["severity"] = r.Severity

	// Add custom labels
	var customLabels map[string]string
	if r.Labels != "" && r.Labels != "{}" {
		json.Unmarshal([]byte(r.Labels), &customLabels)
		for k, v := range customLabels {
			labels[k] = v
		}
	}

	annotations := map[string]string{
		"summary":     r.Summary,
		"description": r.Description,
	}

	endsAt := ""
	if status == "resolved" {
		endsAt = time.Now().Format(time.RFC3339)
	}

	payload.Alerts = append(payload.Alerts, struct {
		Status      string            `json:"status"`
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
		StartsAt    string            `json:"startsAt"`
		EndsAt      string            `json:"endsAt"`
	}{
		Status:      status,
		Labels:      labels,
		Annotations: annotations,
		StartsAt:    startsAt.Format(time.RFC3339),
		EndsAt:      endsAt,
	})

	payloadBytes, _ := json.Marshal(payload)

	// Webhook target URL
	port := "8000"
	if config.Config != nil && config.Config.Server.Address != "" {
		parts := strings.Split(config.Config.Server.Address, ":")
		if len(parts) > 0 {
			port = parts[len(parts)-1]
		}
	}
	webhookUrl := fmt.Sprintf("http://127.0.0.1:%s/api/v1/monitor/alert/webhook/prometheus", port)

	req, _ := http.NewRequest("POST", webhookUrl, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Webhook 发送失败] URL: %s, 错误: %v\n", webhookUrl, err)
	} else {
		defer resp.Body.Close()
		log.Printf("[Webhook 发送成功] URL: %s, 状态码: %d\n", webhookUrl, resp.StatusCode)
	}
}

func invertPromQLExpr(query string) string {
	exprNode, err := parser.NewParser(parser.Options{}).ParseExpr(query)
	if err != nil {
		return ""
	}
	if binExpr, ok := exprNode.(*parser.BinaryExpr); ok && binExpr.Op.IsComparisonOperator() {
		switch binExpr.Op {
		case parser.EQL:
			binExpr.Op = parser.NEQ
		case parser.NEQ:
			binExpr.Op = parser.EQL
		case parser.GTR:
			binExpr.Op = parser.LTE
		case parser.LSS:
			binExpr.Op = parser.GTE
		case parser.GTE:
			binExpr.Op = parser.LSS
		case parser.LTE:
			binExpr.Op = parser.GTR
		}
		return binExpr.String()
	}
	return ""
}

func (s *monitorAlertRuleService) CheckRuleExpr(dataSourceId uint, expr string) (interface{}, error) {
	ds, err := s.dataSourceDao.GetByID(dataSourceId)
	if err != nil {
		return nil, fmt.Errorf("获取数据源失败: %v", err)
	}
	if !strings.EqualFold(ds.Type, "Prometheus") {
		return nil, fmt.Errorf("仅支持 Prometheus 数据源检查")
	}

	apiUrl := strings.TrimRight(ds.ApiUrl, "/")

	// 定义内部查询函数
	doQuery := func(queryExpr string) (interface{}, error) {
		u, err := url.Parse(fmt.Sprintf("%s/api/v1/query", apiUrl))
		if err != nil {
			return nil, fmt.Errorf("构建API地址错误: %v", err)
		}

		q := u.Query()
		q.Set("query", queryExpr)
		u.RawQuery = q.Encode()

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(u.String())
		if err != nil {
			return nil, fmt.Errorf("请求数据源异常: %v", err)
		}
		defer resp.Body.Close()

		var promResp PromQueryResponse
		if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
			return nil, fmt.Errorf("解析数据源响应失败: %v", err)
		}

		if promResp.Status != "success" {
			return nil, fmt.Errorf("数据源返回异常状态: %s", promResp.Status)
		}

		if len(promResp.Data.Result) == 0 {
			return nil, fmt.Errorf("查询成功, 但结果集为空")
		}

		return promResp.Data.Result, nil
	}

	// 第一步：执行原始 expr
	res, err := doQuery(expr)
	if err == nil {
		return res, nil // 第一次执行就成功，直接返回
	}

	// 第二步：尝试执行反向 expr
	invertedExpr := invertPromQLExpr(expr)
	if invertedExpr != "" {
		resInverted, errInverted := doQuery(invertedExpr)
		if errInverted == nil {
			return resInverted, nil // 反向表达式执行成功，也算有效
		}
	}

	return nil, fmt.Errorf("两次执行均未取得结果，原始执行错误: %v", err)
}
