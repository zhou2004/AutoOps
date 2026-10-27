package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/cmdb/model"
)

type orchestratorClient struct {
	endpoints      []string
	user, password string
	http           *http.Client
}

func newOrchestratorClient(endpoints []string, user, password string) *orchestratorClient {
	return &orchestratorClient{endpoints: endpoints, user: user, password: password, http: &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (s *DatabaseAutomationService) orchestrator(d *model.DatabaseDeployment, sec *databaseSecrets) *orchestratorClient {
	endpoints := []string{}
	for _, id := range decodeIDs(d.Managers) {
		p, e := s.hostConnection(id)
		if e == nil {
			endpoints = append(endpoints, "http://"+net.JoinHostPort(p.Address, "13000"))
		}
	}
	return newOrchestratorClient(endpoints, "autoops", sec.HTTP)
}
func (o *orchestratorClient) request(ctx context.Context, endpoint, path string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(endpoint, "/")+"/api/"+path, nil)
	if err != nil {
		return nil, errors.New("orchestrator 地址无效")
	}
	req.SetBasicAuth(o.user, o.password)
	req.Header.Set("Accept", "application/json")
	res, err := o.http.Do(req)
	if err != nil {
		return nil, errors.New("orchestrator 请求未完成，请核对审计和实时拓扑，勿盲目重试切换")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("orchestrator 返回 HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, (4<<20)+1))
	if err != nil || len(body) > 4<<20 || !json.Valid(body) {
		return nil, errors.New("orchestrator 响应无效或超出限制")
	}
	var envelope struct {
		Code    string
		Message string
	}
	_ = json.Unmarshal(body, &envelope)
	if envelope.Code != "" && envelope.Code != "OK" {
		return nil, errors.New("orchestrator 拒绝执行；请查看其恢复审计与拓扑状态")
	}
	return body, nil
}
func (o *orchestratorClient) call(ctx context.Context, path string, mutation bool) (json.RawMessage, error) {
	// Select a leader first. NEVER retry a mutation on another node after an
	// ambiguous timeout: the first request may already have taken effect.
	if len(o.endpoints) == 0 {
		return nil, errors.New("尚未配置 orchestrator 管理节点")
	}
	for _, endpoint := range o.endpoints {
		probe, cancel := context.WithTimeout(ctx, 3*time.Second)
		_, err := o.request(probe, endpoint, "leader-check")
		cancel()
		if err == nil {
			return o.request(ctx, endpoint, path)
		}
	}
	return nil, errors.New("orchestrator 没有可达的 Leader，请检查服务与 Raft 多数派")
}

type DatabaseHARequest struct {
	Action      string `json:"action"`
	HostID      uint   `json:"hostId"`
	CandidateID uint   `json:"candidateId"`
	ConfirmName string `json:"confirmName"`
	Reason      string `json:"reason"`
}

func (s *DatabaseAutomationService) managedInstance(d *model.DatabaseDeployment, id uint) (string, error) {
	for _, node := range decodeIDs(d.Nodes) {
		if node == id {
			p, err := s.hostConnection(id)
			if err != nil {
				return "", err
			}
			return databaseInstancePath(p.Address, d.Port)
		}
	}
	return "", errors.New("实例不属于当前部署")
}
func (s *DatabaseAutomationService) HAView(ctx context.Context, id uint, view string) (any, error) {
	d, err := s.Deployment(id)
	if err != nil {
		return nil, err
	}
	if d.Engine != "mysql" || len(decodeIDs(d.Managers)) == 0 {
		return nil, errors.New("该部署未启用 MySQL orchestrator")
	}
	sec, err := s.secrets(d)
	if err != nil {
		return nil, err
	}
	client := s.orchestrator(d, sec)
	if view == "topology" {
		// A demoted or unreachable member may temporarily have a different
		// cluster name. Read every managed member so the UI cannot silently
		// omit the new primary or an orphaned replica.
		members := []json.RawMessage{}
		for _, node := range decodeIDs(d.Nodes) {
			instance, e := s.managedInstance(d, node)
			if e != nil {
				return nil, e
			}
			body, e := client.call(ctx, "instance/"+instance, false)
			if e != nil {
				return nil, e
			}
			var record struct{ Key struct{ Hostname string } }
			if json.Unmarshal(body, &record) != nil || record.Key.Hostname == "" {
				return nil, fmt.Errorf("主机 %d 尚未被 Orchestrator 发现，请先发现该实例", node)
			}
			members = append(members, body)
		}
		return members, nil
	}
	paths := map[string]string{"analysis": "replication-analysis", "recoveries": "audit-recovery/0", "health": "health", "recovery-state": "check-global-recoveries", "maintenance": "maintenance", "problems": "problems"}
	path, ok := paths[view]
	if !ok {
		return nil, errors.New("不支持的高可用视图")
	}
	return client.call(ctx, path, false)
}
func haActionPath(r DatabaseHARequest, instance, candidate string, user uint) (string, error) {
	switch r.Action {
	case "discover", "refresh", "start-replica", "stop-replica", "restart-replica", "set-read-only", "set-writeable", "end-maintenance", "end-downtime":
		return r.Action + "/" + instance, nil
	case "graceful-master-takeover", "relocate", "recover":
		if candidate == "" || instance == candidate {
			return "", errors.New("请选择不同的候选实例")
		}
		action := r.Action
		if action == "graceful-master-takeover" {
			// The ordinary upstream endpoint leaves the demoted master's
			// replication stopped. The auto variant also starts it.
			action = "graceful-master-takeover-auto"
		}
		return action + "/" + instance + "/" + candidate, nil
	case "begin-maintenance", "begin-downtime":
		if strings.TrimSpace(r.Reason) == "" || len(r.Reason) > 200 {
			return "", errors.New("请输入不超过 200 字节的操作原因")
		}
		path := r.Action + "/" + instance + "/autoops-" + strconv.FormatUint(uint64(user), 10) + "/" + url.PathEscape(r.Reason)
		if r.Action == "begin-downtime" {
			path += "/1h"
		}
		return path, nil
	case "enable-global-recoveries", "disable-global-recoveries":
		return r.Action, nil
	}
	return "", errors.New("不支持的 orchestrator 操作")
}
func (s *DatabaseAutomationService) HAAction(ctx context.Context, id, user uint, r DatabaseHARequest) (any, error) {
	d, err := s.Deployment(id)
	if err != nil {
		return nil, err
	}
	if d.Engine != "mysql" || len(decodeIDs(d.Managers)) == 0 || d.Status != "ready" {
		return nil, errors.New("该部署尚未就绪或未启用 MySQL orchestrator")
	}
	if r.ConfirmName != d.Name {
		return nil, errors.New("请输入完整部署名称确认操作")
	}
	instance, err := s.managedInstance(d, r.HostID)
	if err != nil {
		return nil, err
	}
	candidate := ""
	if r.CandidateID != 0 {
		candidate, err = s.managedInstance(d, r.CandidateID)
		if err != nil {
			return nil, err
		}
	}
	path, err := haActionPath(r, instance, candidate, user)
	if err != nil {
		return nil, err
	}
	sec, err := s.secrets(d)
	if err != nil {
		return nil, err
	}
	run, err := s.acquire(id, user, r.Action)
	if err != nil {
		return nil, err
	}
	body, err := s.orchestrator(d, sec).call(ctx, path, true)
	status, message := "success", r.Action+" "+instance
	if err != nil {
		status = "unknown"
		message = err.Error()
	}
	s.finish(*run, status, message, "")
	return body, err
}

func (s *DatabaseAutomationService) OrchestratorAccess(id, user uint) (any, error) {
	d, err := s.Deployment(id)
	if err != nil {
		return nil, err
	}
	if len(decodeIDs(d.Managers)) == 0 {
		return nil, errors.New("尚未配置 orchestrator")
	}
	sec, err := s.secrets(d)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	audit := model.DatabaseAutomationRun{DeploymentID: id, Action: "view-orchestrator-credentials", Status: "success", UserID: user, Log: "管理员查看 Orchestrator 访问凭据", FinishedAt: &now}
	if err := s.db.Create(&audit).Error; err != nil {
		return nil, errors.New("凭据访问审计写入失败")
	}
	return map[string]any{"username": "autoops", "password": sec.HTTP, "endpoints": s.orchestrator(d, sec).endpoints}, nil
}
