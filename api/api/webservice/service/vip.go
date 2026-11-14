package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/webservice/model"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"time"
)

type VIPNode struct {
	InstallPackageVersion string        `json:"installPackageVersion,omitempty"`
	ClusterID             uint          `json:"clusterId"`
	InstanceID            uint          `json:"instanceId"`
	Config                model.VIPSpec `json:"config"`
	Address               string        `json:"address"`
	Peers                 []string      `json:"peers"`
	PeerMACs              []string      `json:"peerMACs"`
	Priority              int           `json:"priority"`
	ConfigDigest          string        `json:"configDigest"`
	PreviousOwner         uint          `json:"previousOwner"`
}
type VIPObservation struct {
	Healthy               bool     `json:"healthy"`
	InstallPackageVersion string   `json:"installPackageVersion,omitempty"`
	Bound                 bool     `json:"bound"`
	Active                bool     `json:"active"`
	Owned                 bool     `json:"owned"`
	Address               string   `json:"address"`
	MAC                   string   `json:"mac"`
	ConfigDigest          string   `json:"configDigest"`
	ARPOwners             []string `json:"arpOwners"`
	State                 string   `json:"state,omitempty"`
	Error                 string   `json:"error,omitempty"`
}
type VIPState struct {
	Message         string                  `json:"message,omitempty"`
	Config          *model.VIPSpec          `json:"config"`
	Status          string                  `json:"status"`
	OwnerInstanceID uint                    `json:"ownerInstanceId"`
	Nodes           map[uint]VIPObservation `json:"nodes"`
	CheckedAt       time.Time               `json:"checkedAt"`
}

func isVIPAction(a string) bool {
	return a == "configure-vip" || a == "verify-vip" || a == "switch-vip"
}

var vipInterface = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,14}$`)

func validateVIP(p *model.VIPSpec) error {
	if p == nil {
		return errors.New("请填写 VIP、网卡和目标实例")
	}
	ip, network, err := net.ParseCIDR(p.CIDR)
	if err != nil || ip.To4() == nil || !ip.IsGlobalUnicast() || !vipInterface.MatchString(p.Interface) || p.RouterID < 1 || p.RouterID > 255 || len(p.InstanceIDs) < 2 || len(p.InstanceIDs) > 10 {
		return errors.New("VIP 需要有效 IPv4/CIDR、网卡、VRID（1–255）和 2–10 个实例")
	}
	bits, _ := network.Mask.Size()
	broadcast := append(net.IP(nil), network.IP.To4()...)
	for i := range broadcast {
		broadcast[i] |= ^network.Mask[i]
	}
	if bits < 8 || bits > 30 || ip.Equal(network.IP) || ip.Equal(broadcast) {
		return errors.New("VIP 必须为前缀 8–30 的可用主机地址")
	}
	seen := map[uint]bool{}
	for _, id := range p.InstanceIDs {
		if id == 0 || seen[id] {
			return errors.New("VIP 实例不能重复")
		}
		seen[id] = true
	}
	if p.PreferredInstanceID > 0 && !seen[p.PreferredInstanceID] {
		return errors.New("优先实例必须属于 VIP 成员")
	}
	return nil
}
func (s *Service) vipConflicts(cid uint, p *model.VIPSpec) error {
	ip, _, _ := net.ParseCIDR(p.CIDR)
	var count int64
	if err := s.DB.Model(&cmdb.CmdbHost{}).Where("private_ip = ? OR public_ip = ? OR ssh_ip = ?", ip.String(), ip.String(), ip.String()).Count(&count).Error; err != nil {
		return errors.New("CMDB 地址冲突检查失败")
	}
	if count > 0 {
		return errors.New("VIP 与 CMDB 主机地址冲突")
	}
	var clusters []model.Resource
	if err := s.DB.Table(model.Tables["clusters"]).Find(&clusters).Error; err != nil {
		return err
	}
	for _, c := range clusters {
		spec, _ := decode[model.ClusterSpec](c.Spec)
		if spec.VIP == nil {
			continue
		}
		if c.ID == cid {
			if !reflect.DeepEqual(spec.VIP, p) {
				return errors.New("已有 VIP 配置不可直接覆盖，请先退役原配置")
			}
			continue
		}
		old, _, _ := net.ParseCIDR(spec.VIP.CIDR)
		if ip.Equal(old) || p.RouterID == spec.VIP.RouterID {
			return errors.New("VIP 或 VRID 已被其他 Web 集群使用")
		}
	}
	// Also reject IDs already registered by the database platform.
	if s.DB.Migrator().HasTable("database_virtual_ip") {
		var registered []struct {
			CIDR     string
			RouterID int
		}
		if err := s.DB.Table("database_virtual_ip").Find(&registered).Error; err != nil {
			return err
		}
		for _, v := range registered {
			old, _, _ := net.ParseCIDR(v.CIDR)
			if ip.Equal(old) || p.RouterID == v.RouterID {
				return errors.New("VIP 或 VRID 已被数据库集群使用")
			}
		}
	}
	return nil
}
func (s *Service) prepareVIP(ctx context.Context, u uint, c model.Resource, in PlanInput, targets *[]Target) error {
	if !s.Admin(u) {
		return ErrForbidden
	}
	p := in.Parameters.VIP
	if err := validateVIP(p); err != nil {
		return err
	}
	if err := s.vipConflicts(c.ID, p); err != nil {
		return err
	}
	if len(*targets) != len(p.InstanceIDs) {
		return errors.New("VIP 成员与集群不匹配")
	}
	spec, _ := decode[model.ClusterSpec](c.Spec)
	if in.Action != "configure-vip" && spec.VIP == nil {
		return errors.New("请先部署 VIP")
	}
	hosts := map[uint]bool{}
	addresses := map[uint]string{}
	observations := map[uint]VIPObservation{}
	allowedMAC := map[string]bool{}
	healthPort, healthScheme, healthPath := "", "", ""
	for index := range *targets {
		t := &(*targets)[index]
		if t.Spec.Runtime != "host" || t.Spec.HostID == 0 || hosts[t.Spec.HostID] || t.Instance.Management != "operate" {
			return errors.New("VIP 需要不同主机上的已接管 Nginx 实例")
		}
		hosts[t.Spec.HostID] = true
		health, err := url.Parse(t.Spec.HealthURL)
		if err != nil || health.Hostname() == "" {
			return errors.New("每个 VIP 实例必须设置业务健康检查地址")
		}
		port := health.Port()
		if port == "" {
			if health.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}
		if healthPort != "" && (healthPort != port || healthScheme != health.Scheme || healthPath != health.RequestURI()) {
			return errors.New("VIP 实例的健康检查协议、端口和路径必须一致")
		}
		healthPort, healthScheme, healthPath = port, health.Scheme, health.RequestURI()
		if err = s.Authorize(u, "instances", t.Instance, "web_manage"); err != nil {
			return err
		}
		if err := s.requireHostAccess(ctx, u, t.Spec.HostID, "command"); err != nil {
			return err
		}
		connection, err := cmdbservice.GetAssetConnection(s.DB, "host", t.Spec.HostID, t.Spec.ConnectionID)
		if err != nil || !connection.Enabled || connection.Protocol != "ssh" || connection.ReadOnly {
			return errors.New("VIP 需要启用且可写的主机 SSH 连接")
		}
		node := VIPNode{ClusterID: c.ID, InstanceID: t.Instance.ID, Config: *p}
		_, network, _ := net.ParseCIDR(p.CIDR)
		if ip := net.ParseIP(connection.Address); ip != nil && network.Contains(ip) {
			node.Address = ip.String()
		}
		raw, err := s.Remote.Run(ctx, RemoteRequest{Action: "vip-check", Spec: t.Spec, VIP: &node})
		if err != nil {
			return fmt.Errorf("%s: %w", t.Instance.Name, err)
		}
		observation, err := decode[VIPObservation](raw)
		if err != nil || net.ParseIP(observation.Address) == nil {
			return errors.New("VIP 网卡检查结果无效")
		}
		addresses[t.Instance.ID] = observation.Address
		observations[t.Instance.ID] = observation
		if observation.Bound {
			if spec.VIP == nil || !observation.Owned || !observation.Active {
				return errors.New("VIP 已被未纳管服务占用")
			}
			allowedMAC[observation.MAC] = true
		}
		if spec.VIP != nil && (!observation.Owned || !observation.Active) {
			return errors.New("已部署 VIP 服务缺失或未启动，请先核验修复")
		}
		if _, err = s.Remote.Run(ctx, RemoteRequest{Action: "health", Spec: t.Spec}); err != nil {
			return fmt.Errorf("%s 健康检查失败: %w", t.Instance.Name, err)
		}
	}
	owner := uint(0)
	owners := 0
	for id, o := range observations {
		if o.Bound {
			owner = id
			owners++
		}
		for _, mac := range o.ARPOwners {
			if !allowedMAC[mac] {
				return errors.New("ARP 检查发现 VIP 被其他设备占用")
			}
		}
	}
	if owners > 1 || (in.Action != "configure-vip" && owners != 1) {
		return errors.New("VIP 必须恰好由一个健康节点持有")
	}
	order := append([]uint(nil), p.InstanceIDs...)
	if p.PreferredInstanceID > 0 {
		sort.SliceStable(order, func(i, j int) bool { return order[i] == p.PreferredInstanceID && order[j] != p.PreferredInstanceID })
	}
	for index := range *targets {
		t := &(*targets)[index]
		peers := []string{}
		peerMACs := []string{}
		priority := 200
		for i, id := range order {
			if id == t.Instance.ID {
				priority = 200 - i*10
			} else {
				peers = append(peers, addresses[id])
				peerMACs = append(peerMACs, observations[id].MAC)
			}
		}
		t.VIP = &VIPNode{ClusterID: c.ID, InstanceID: t.Instance.ID, Config: *p, Address: addresses[t.Instance.ID], Peers: peers, PeerMACs: peerMACs, Priority: priority, ConfigDigest: observations[t.Instance.ID].ConfigDigest, PreviousOwner: owner, InstallPackageVersion: observations[t.Instance.ID].InstallPackageVersion}
	}
	return nil
}
func (s *Service) inspectVIPState(ctx context.Context, p *model.VIPSpec, targets []Target) (VIPState, error) {
	state := VIPState{Config: p, Status: "healthy", Nodes: map[uint]VIPObservation{}, CheckedAt: time.Now().UTC()}
	owners := 0
	for _, t := range targets {
		raw, err := s.Remote.Run(ctx, RemoteRequest{Action: "vip-inspect", Spec: t.Spec, VIP: t.VIP})
		o := VIPObservation{}
		if err == nil {
			o, err = decode[VIPObservation](raw)
		}
		if err != nil {
			o.Error = err.Error()
			state.Status = "unknown"
		} else {
			if o.Bound {
				owners++
				state.OwnerInstanceID = t.Instance.ID
			}
			if !o.Active || !o.Owned || !o.Healthy {
				if state.Status != "unknown" {
					state.Status = "degraded"
				}
			}
		}
		state.Nodes[t.Instance.ID] = o
	}
	if owners > 1 {
		state.Status = "conflict"
		state.OwnerInstanceID = 0
	} else if owners == 0 {
		state.OwnerInstanceID = 0
		if state.Status != "unknown" {
			state.Status = "unavailable"
		}
	}
	if state.Status != "healthy" {
		return state, errors.New("VIP 服务或唯一持有者核验未通过")
	}
	return state, nil
}
func (s *Service) verifyVIPHTTP(ctx context.Context, p *model.VIPSpec, targets []Target) error {
	if len(targets) == 0 {
		return errors.New("VIP 成员为空")
	}
	ip, _, _ := net.ParseCIDR(p.CIDR)
	for _, t := range targets {
		spec := t.Spec
		endpoint, err := url.Parse(spec.HealthURL)
		if err != nil {
			return err
		}
		port := endpoint.Port()
		endpoint.Host = ip.String()
		if port != "" {
			endpoint.Host = net.JoinHostPort(ip.String(), port)
		}
		spec.HealthURL = endpoint.String()
		if _, err = s.Remote.Run(ctx, RemoteRequest{Action: "health", Spec: spec}); err != nil {
			return fmt.Errorf("%s 经 VIP 的业务核验失败: %w", t.Instance.Name, err)
		}
	}
	return nil
}
func (s *Service) waitVIP(ctx context.Context, p *model.VIPSpec, targets []Target, previous uint) (VIPState, error) {
	deadline := time.Now().Add(35 * time.Second)
	var state VIPState
	var err error
	success := 0
	for {
		state, err = s.inspectVIPState(ctx, p, targets)
		if state.Status == "conflict" {
			return state, errors.New("VIP 出现多个持有者，停止后续部署")
		}
		if err == nil && previous > 0 && state.OwnerInstanceID == previous {
			err = errors.New("VIP 尚未切换至其他节点")
		}
		if err == nil {
			err = s.verifyVIPHTTP(ctx, p, targets)
		}
		if err == nil {
			success++
		} else {
			success = 0
		}
		if success >= 2 {
			return state, nil
		}
		if time.Now().After(deadline) {
			return state, err
		}
		select {
		case <-ctx.Done():
			return state, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
func (s *Service) persistVIP(c model.Resource, p *model.VIPSpec, state VIPState) error {
	spec := model.ClusterSpec{}
	if len(c.Spec) > 0 {
		var err error
		spec, err = decode[model.ClusterSpec](c.Spec)
		if err != nil {
			return err
		}
	}
	spec.VIP = p
	observed := map[string]any{}
	_ = json.Unmarshal(c.Observed, &observed)
	observed["vip"] = state
	result := s.DB.Table(model.Tables["clusters"]).Where("id = ? AND revision = ?", c.ID, c.Revision).Updates(map[string]any{"spec": JSON(spec), "observed": JSON(observed), "revision": gorm.Expr("revision + 1")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}
func (s *Service) executeVIP(ctx context.Context, op model.Operation, plan model.Plan, c model.Resource, targets []Target) {
	params, _ := decode[Parameters](plan.Parameters)
	original := append([]Target(nil), targets...)
	if err := s.prepareVIP(ctx, op.UserID, c, PlanInput{Action: op.Action, Parameters: params}, &targets); err != nil {
		s.finish(op, "failed", err.Error())
		return
	}
	for i := range targets {
		if original[i].VIP == nil || !reflect.DeepEqual(original[i].VIP, targets[i].VIP) {
			s.finish(op, "failed", "VIP 基线或节点地址变化，请重新预检")
			return
		}
	}
	s.DB.Model(&op).Update("status", "running")
	currentSpec, _ := decode[model.ClusterSpec](c.Spec)
	changed := false
	run := func(t Target, action string) error {
		var fresh model.Operation
		if s.DB.First(&fresh, op.ID).Error != nil || fresh.CancelRequested || s.Module(ctx, op.UserID, "execute") != nil || !s.Admin(op.UserID) || s.Authorize(op.UserID, "instances", t.Instance, "web_manage") != nil {
			return errors.New("任务取消或授权变化")
		}
		if err := s.step(op, t, action, "running", "", t.VIP); err != nil {
			return err
		}
		changed = true
		raw, err := s.Remote.Run(ctx, RemoteRequest{Action: action, Spec: t.Spec, VIP: t.VIP, OperationID: op.ID, Identity: t.Observation.Identity, Baseline: t.Observation.Digest})
		if err != nil {
			s.step(op, t, action, "unknown", err.Error(), nil)
			return err
		}
		return s.step(op, t, action, "succeeded", "", raw)
	}
	if op.Action == "configure-vip" && currentSpec.VIP == nil {
		sort.SliceStable(targets, func(i, j int) bool { return targets[i].VIP.Priority > targets[j].VIP.Priority })
		applied := []Target{}
		for _, t := range targets {
			if err := run(t, "vip-configure"); err != nil {
				if changed {
					s.finish(op, "unknown", err.Error())
				} else {
					s.finish(op, "failed", err.Error())
				}
				return
			}
			applied = append(applied, t)
			state, err := s.waitVIP(ctx, params.VIP, applied, 0)
			if err != nil {
				if state.Status == "conflict" && s.rollbackVIP(ctx, op, applied, op.UserID) == nil {
					s.finish(op, "rolled-back", "VIP 冲突，已回退本次独立服务；原有 Nginx 和数据库 VIP 未变更")
				} else {
					s.finish(op, "unknown", err.Error())
				}
				return
			}
		}
	}
	previous := uint(0)
	var owner *Target
	if op.Action == "switch-vip" {
		previous = targets[0].VIP.PreviousOwner
		for i := range targets {
			if targets[i].Instance.ID == previous {
				owner = &targets[i]
				break
			}
		}
		if owner == nil {
			s.finish(op, "failed", "VIP 当前持有者不存在")
			return
		}
		if err := run(*owner, "vip-switch"); err != nil {
			s.finish(op, "unknown", err.Error())
			return
		}
	}
	s.DB.Model(&op).Update("status", "verifying")
	// During the brief switch the old VIP service is intentionally stopped.
	if owner != nil {
		if err := s.waitVIPMoved(ctx, params.VIP, targets, previous); err != nil {
			s.finish(op, "unknown", err.Error())
			return
		}
		if err := run(*owner, "vip-resume"); err != nil {
			s.finish(op, "unknown", err.Error())
			return
		}
	}
	state, err := s.waitVIP(ctx, params.VIP, targets, previous)
	if err != nil {
		if changed {
			s.finish(op, "unknown", err.Error())
		} else {
			s.finish(op, "failed", err.Error())
		}
		return
	}
	if err = s.persistVIP(c, params.VIP, state); err != nil {
		s.finish(op, "unknown", err.Error())
		return
	}
	s.finish(op, "succeeded", "VIP 唯一绑定、节点服务及业务入口已核验")
}
func (s *Service) waitVIPMoved(ctx context.Context, p *model.VIPSpec, targets []Target, previous uint) error {
	deadline := time.Now().Add(35 * time.Second)
	success := 0
	for {
		state, _ := s.inspectVIPState(ctx, p, targets)
		ok := state.OwnerInstanceID > 0 && state.OwnerInstanceID != previous && state.Status == "degraded"
		for id, o := range state.Nodes {
			if o.Error != "" || !o.Owned || !o.Healthy || (id != previous && !o.Active) {
				ok = false
			}
		}
		if ok && s.verifyVIPHTTP(ctx, p, targets) == nil {
			success++
		} else {
			success = 0
		}
		if success >= 2 {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("VIP 切换未确认，保留锁；原服务已设置独立救援定时器")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
func (s *Service) reconcileVIP(ctx context.Context, op, old model.Operation, plan model.Plan, c model.Resource, targets []Target) {
	params, _ := decode[Parameters](plan.Parameters)
	if !s.Admin(op.UserID) || params.VIP == nil {
		s.finish(op, "failed", ErrForbidden.Error())
		return
	}
	for _, t := range targets {
		source, err := s.Get("sources", t.Instance.SourceID)
		if t.VIP == nil || err != nil || source.Revision != t.SourceRevision || s.requireHostAccess(ctx, op.UserID, t.Spec.HostID, "command") != nil || s.Authorize(op.UserID, "instances", t.Instance, "web_manage") != nil {
			s.finish(op, "failed", "VIP 恢复的源版本或授权变化，原锁保留")
			return
		}
		current, err := (Nginx{}).Discover(ctx, s.Remote, t.Spec)
		if err != nil || current.Identity != t.Observation.Identity || current.Digest != t.Observation.Digest {
			s.finish(op, "failed", "VIP 恢复的实例身份或配置基线发生变化，原锁保留")
			return
		}
	}
	incomplete := false
	for _, t := range targets {
		needsJournal := (old.Action == "configure-vip" && t.VIP.ConfigDigest == "") || (old.Action == "switch-vip" && t.Instance.ID == t.VIP.PreviousOwner)
		action := "vip-inspect"
		if needsJournal {
			action = "vip-reconcile"
		}
		raw, err := s.Remote.Run(ctx, RemoteRequest{Action: action, Spec: t.Spec, VIP: t.VIP, OperationID: old.ID, Identity: t.Observation.Identity, Baseline: t.Observation.Digest})
		if err != nil {
			s.finish(op, "failed", err.Error())
			return
		}
		evidence, err := decode[VIPObservation](raw)
		if err != nil {
			s.finish(op, "failed", "VIP 恢复证据无效")
			return
		}
		if !needsJournal && (!evidence.Owned || evidence.ConfigDigest != t.VIP.ConfigDigest) {
			s.finish(op, "failed", "VIP 未变更节点出现配置漂移，原锁保留")
			return
		}
		if old.Action == "configure-vip" && needsJournal && evidence.State != "succeeded" {
			incomplete = true
		}
	}
	if incomplete {
		if err := s.rollbackVIP(ctx, old, targets, op.UserID); err != nil {
			s.finish(op, "failed", err.Error())
			return
		}
		s.finish(old, "rolled-back", "VIP 未完成部署已按原始清单回退，已安装依赖包保留")
		s.finish(op, "succeeded", "VIP 回退核验完成")
		return
	}
	if old.Action == "switch-vip" {
		for _, t := range targets {
			if t.Instance.ID != t.VIP.PreviousOwner {
				continue
			}
			if _, err := s.Remote.Run(ctx, RemoteRequest{Action: "vip-resume", Spec: t.Spec, VIP: t.VIP, OperationID: old.ID, Identity: t.Observation.Identity, Baseline: t.Observation.Digest}); err != nil {
				s.finish(op, "failed", err.Error())
				return
			}
		}
	}
	state, err := s.waitVIP(ctx, params.VIP, targets, 0)
	if err != nil {
		s.finish(op, "failed", "VIP 远端状态未收敛，原锁保留；请检查独立服务和执行证据后重新核验")
		return
	}
	if err = s.persistVIP(c, params.VIP, state); err != nil {
		s.finish(op, "failed", err.Error())
		return
	}
	status := "succeeded"
	if old.Action == "switch-vip" && len(targets) > 0 && state.OwnerInstanceID == targets[0].VIP.PreviousOwner {
		status = "rolled-back"
	}
	s.finish(old, status, "VIP 已核验，未重放原配置或切换动作")
	s.finish(op, "succeeded", "VIP 核验完成")
}

func (s *Service) rollbackVIP(ctx context.Context, op model.Operation, targets []Target, actor uint) error {
	for i := len(targets) - 1; i >= 0; i-- {
		t := targets[i]
		if s.Module(ctx, actor, "execute") != nil || s.requireHostAccess(ctx, actor, t.Spec.HostID, "command") != nil {
			return ErrForbidden
		}
		raw, err := s.Remote.Run(ctx, RemoteRequest{Action: "vip-rollback", Spec: t.Spec, VIP: t.VIP, OperationID: op.ID, Identity: t.Observation.Identity, Baseline: t.Observation.Digest})
		if err != nil {
			return err
		}
		evidence, err := decode[VIPObservation](raw)
		if err != nil || (evidence.State != "rolled-back" && evidence.State != "not-started") || evidence.Bound || evidence.Active {
			return errors.New("VIP 回退结果未确认")
		}
		if err = s.step(op, t, "vip-rollback", "succeeded", "", evidence); err != nil {
			return err
		}
	}
	return nil
}
