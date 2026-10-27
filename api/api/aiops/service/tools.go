package service

import (
	"context"
	model "dodevops-api/api/aiops/model"
	cmdbdao "dodevops-api/api/cmdb/dao"
	cmdbmodel "dodevops-api/api/cmdb/model"
	cmdb "dodevops-api/api/cmdb/service"
	"dodevops-api/pkg/apikey"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Service) toolConnection(ctx context.Context, user, id uint, permission string, write bool) (cmdbmodel.CmdbAssetConnection, error) {
	var p cmdbmodel.CmdbAssetConnection
	if id == 0 || s.DB.WithContext(ctx).Where("id = ? AND protocol = ? AND enabled = ? AND asset_type IN ?", id, "ssh", true, []string{"host", "physical"}).First(&p).Error != nil {
		return p, errors.New("资产连接不可用")
	}
	if write && p.ReadOnly {
		return p, errors.New("连接为只读，无法执行操作")
	}
	if cmdb.AssetExists(s.DB.WithContext(ctx), p.AssetType, p.AssetID) != nil {
		return p, errors.New("资产不存在")
	}
	d := cmdbdao.NewCmdbAssetPermissionDao(s.DB.WithContext(ctx))
	allowed, err := d.CanUserAccess(user, p.AssetType, p.AssetID, permission)
	if err != nil || !allowed {
		return p, errors.New("没有该资产的 Docker 权限")
	}
	return p, nil
}
func (s *Service) ToolTargets(ctx context.Context, user uint) ([]map[string]any, error) {
	if err := s.Active(ctx, user); err != nil {
		return nil, err
	}
	var rows []cmdbmodel.CmdbAssetConnection
	if err := s.DB.WithContext(ctx).Where("protocol = ? AND enabled = ? AND asset_type IN ?", "ssh", true, []string{"host", "physical"}).Order("id DESC").Limit(1000).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, p := range rows {
		if _, err := s.toolConnection(ctx, user, p.ID, "docker_view", false); err != nil {
			continue
		}
		_, writeErr := s.toolConnection(ctx, user, p.ID, "docker_operate", true)
		out = append(out, map[string]any{"id": p.ID, "name": p.Name, "assetType": p.AssetType, "assetId": p.AssetID, "canOperate": writeErr == nil})
	}
	return out, nil
}
func ToolWrites(tool string) bool {
	return tool == "docker_start" || tool == "docker_stop" || tool == "docker_restart"
}

// Each invocation rechecks the current account, session grant, asset and credential permissions.
func (s *Service) ExecuteTool(ctx context.Context, user, session uint, canOperate bool, r model.ToolRequest) (out model.ToolCall, err error) {
	if err = s.Active(ctx, user); err != nil {
		return out, err
	}
	row, err := s.Session(ctx, user, session)
	if err != nil {
		return out, err
	}
	if len(r.CallID) < 8 || len(r.CallID) > 80 || len(r.Container) > 128 {
		return out, errors.New("工具请求参数无效")
	}
	permission := "docker_view"
	write := ToolWrites(r.Tool)
	identity, accessErr := apikey.EffectiveIdentity(ctx, s.DB, user)
	if accessErr != nil || !identity.Allows("aiops:read") || !identity.Allows("docker:read") || (write && (!identity.Allows("aiops:operate") || !identity.Allows("docker:execute"))) {
		return out, errors.New("当前用户的 AI 或 Docker 模块权限已撤销或不足")
	}

	switch r.Tool {
	case "docker_info", "docker_list":
	case "docker_logs":
		permission = "docker_logs"
	case "docker_start", "docker_stop", "docker_restart":
		permission = "docker_operate"
	default:
		return out, errors.New("该工具未开放")
	}
	var targets []uint
	_ = json.Unmarshal([]byte(row.Targets), &targets)
	found := false
	for _, id := range targets {
		if id == r.ConnectionID {
			found = true
		}
	}
	if !found {
		return out, errors.New("该资产未加入会话授权范围")
	}
	if write {
		allowed := false
		for _, a := range strings.Split(row.Actions, ",") {
			if a == r.Tool {
				allowed = true
			}
		}
		if !canOperate || !allowed || row.GrantUntil == nil || !row.GrantUntil.After(time.Now()) {
			return out, errors.New("操作未获授权或授权已过期，请在工作台权限面板授权")
		}
	}
	p, err := s.toolConnection(ctx, user, r.ConnectionID, permission, write)
	if err != nil {
		return out, err
	}
	args, _ := json.Marshal(r)
	var existing model.ToolCall
	if s.DB.WithContext(ctx).Where("session_id = ? AND call_id = ?", session, r.CallID).First(&existing).Error == nil {
		if existing.Arguments != string(args) {
			return out, errors.New("请求 ID 已被不同操作使用")
		}
		return existing, nil
	}
	out = model.ToolCall{SessionID: session, CallID: r.CallID, Tool: r.Tool, Arguments: string(args), Status: "running"}
	if s.DB.WithContext(ctx).Create(&out).Error != nil {
		return out, errors.New("无法记录工具请求，请使用原请求 ID 查询结果")
	}
	defer func() {
		finished := time.Now().UTC()
		out.FinishedAt = &finished
		out.Status = "succeeded"
		if err != nil {
			out.Status = "failed"
			out.Result = err.Error()
		}
		persist, cancel := persistContext()
		defer cancel()
		if e := s.DB.WithContext(persist).Model(&model.ToolCall{}).Where("id = ?", out.ID).Updates(map[string]any{"result": out.Result, "status": out.Status, "finished_at": finished}).Error; e != nil {
			err = errors.New("工具结果保存失败，请先核实实际状态，不要重复执行")
		}
	}()
	d := cmdb.NewDockerOperationService(s.DB)
	var data any
	switch r.Tool {
	case "docker_info":
		data, err = d.Info(ctx, p.AssetType, p.AssetID, p.ID)
	case "docker_list":
		data, err = d.Containers(ctx, p.AssetType, p.AssetID, p.ID)
	case "docker_logs":
		data, err = d.Logs(ctx, p.AssetType, p.AssetID, p.ID, r.Container, 100)
	default:
		if r.Container == "" {
			err = errors.New("请输入完整容器名称")
			return
		}
		data, err = d.Action(ctx, p.AssetType, p.AssetID, p.ID, r.Container, strings.TrimPrefix(r.Tool, "docker_"), r.Container)
	}
	if err != nil {
		return
	}
	raw, e := json.Marshal(data)
	if e != nil {
		err = e
		return
	}
	if len(raw) > 128<<10 {
		out.Result = fmt.Sprintf("输出过长（%d 字节），请缩小查询范围", len(raw))
	} else {
		out.Result = string(raw)
	}
	return
}
