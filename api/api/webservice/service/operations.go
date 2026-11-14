package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"dodevops-api/pkg/apikey"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Parameters struct {
	VIP             *model.VIPSpec `json:"vip,omitempty"`
	ConfigurationID uint           `json:"configurationId"`
	RevisionID      uint           `json:"revisionId"`
	BindingID       uint           `json:"bindingId"`
	OperationID     uint           `json:"operationId"`
	CertificatePEM  string         `json:"-"`
	KeyPEM          string         `json:"-"`
}
type PlanInput struct {
	TargetType string     `json:"targetType"`
	TargetID   uint       `json:"targetId"`
	Action     string     `json:"action"`
	Parameters Parameters `json:"parameters"`
}
type Check struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}
type Target struct {
	VIP                  *VIPNode                 `json:"vip,omitempty"`
	HealthChecks         []model.Entry            `json:"healthChecks,omitempty"`
	PreviousCertificates map[uint]json.RawMessage `json:"previousCertificates,omitempty"`
	TLS                  []TLSProbe               `json:"tls,omitempty"`
	ResultDigest         string                   `json:"resultDigest,omitempty"`
	Instance             model.Resource           `json:"instance"`
	SourceRevision       uint64                   `json:"sourceRevision"`
	Spec                 model.SourceSpec         `json:"spec"`
	Observation          Observation              `json:"observation"`
	Files                []model.File             `json:"files"`
}
type Capability struct {
	Action  string `json:"action"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

var actions = map[string]string{"adopt": "接管服务", "publish": "发布配置", "rollback": "回滚配置", "deploy-certificate": "部署证书", "reload": "重载", "start": "启动", "stop": "优雅停止", "restart": "重启", "provision": "部署实例", "issue": "签发证书", "renew": "续期证书", "validate": "校验配置", "reconcile": "核验中断任务", "configure-vip": "配置 VIP", "verify-vip": "核验 VIP", "switch-vip": "切换 VIP"}

func actionPermission(a string) string {
	switch a {
	case "publish", "rollback", "deploy-certificate":
		return "web_publish"
	case "issue", "renew":
		return "web_certificate_manage"
	case "validate":
		return "web_config_view"
	default:
		return "web_manage"
	}
}
func (s *Service) Capabilities(u, cid uint) []Capability {
	c, e := s.Get("clusters", cid)
	if e != nil {
		return nil
	}
	out := []Capability{}
	for _, a := range []string{"adopt", "publish", "rollback", "deploy-certificate", "reload", "start", "stop", "restart", "validate", "reconcile", "configure-vip", "verify-vip", "switch-vip"} {
		v := Capability{Action: a, Label: actions[a], Enabled: true}
		if s.Authorize(u, "clusters", c, actionPermission(a)) != nil {
			v.Enabled = false
			v.Reason = "没有操作授权"
		}
		if a != "adopt" && a != "reconcile" && a != "validate" && c.Management != "operate" {
			v.Enabled = false
			v.Reason = "请先完成运维接管"
		}
		if isVIPAction(a) {
			spec, _ := decode[model.ClusterSpec](c.Spec)
			if !s.Admin(u) {
				v.Enabled = false
				v.Reason = "VIP 操作需要管理员权限"
			}
			if a != "configure-vip" && spec.VIP == nil {
				v.Enabled = false
				v.Reason = "请先配置 VIP"
			}
		}
		out = append(out, v)
	}
	return out
}
func (s *Service) Plan(ctx context.Context, u uint, in PlanInput) (model.Plan, error) {
	p := model.Plan{UserID: u, APIKeyID: apikey.ID(ctx), TargetType: in.TargetType, TargetID: in.TargetID, Action: in.Action, Parameters: JSON(in.Parameters), ExpiresAt: time.Now().UTC().Add(5 * time.Minute)}
	if e := s.Module(ctx, u, "read"); e != nil {
		return p, e
	}
	if _, ok := actions[in.Action]; !ok {
		return p, errors.New("未知操作")
	}
	r, e := s.Get(in.TargetType, in.TargetID)
	if e != nil {
		return p, e
	}
	if e = s.Authorize(u, in.TargetType, r, actionPermission(in.Action)); e != nil {
		return p, e
	}
	if isVIPAction(in.Action) && in.Parameters.VIP == nil {
		spec, _ := decode[model.ClusterSpec](r.Spec)
		in.Parameters.VIP = spec.VIP
	}
	p.Revision = r.Revision
	valid := (in.TargetType == "clusters" && in.Action != "issue" && in.Action != "renew" && in.Action != "provision") || (in.TargetType == "certificates" && (in.Action == "issue" || in.Action == "renew")) || (in.TargetType == "sources" && (in.Action == "provision" || in.Action == "reconcile"))
	if !valid {
		return p, errors.New("操作与目标类型不匹配")
	}
	var configuration *model.Resource
	if in.TargetType == "clusters" && (in.Action == "publish" || in.Action == "validate" || in.Action == "rollback") && in.Parameters.RevisionID == 0 && in.Parameters.ConfigurationID > 0 {
		draft, err := s.Get("configurations", in.Parameters.ConfigurationID)
		if err != nil {
			return p, err
		}
		if draft.ClusterID != r.ID {
			return p, ErrForbidden
		}
		configuration = &draft
	}
	checks := []Check{}
	add := func(name string, e error) {
		c := Check{Name: name, Passed: e == nil}
		if e != nil {
			c.Message = e.Error()
		}
		checks = append(checks, c)
	}
	var targets []Target
	var issuanceDigest string
	if in.TargetType == "clusters" {
		rows, e := s.Members(u, r.ID)
		if e != nil {
			return p, e
		}
		rows, e = s.operationMembers(r.ID, in, rows)
		if e != nil {
			return p, e
		}
		if len(rows) == 0 {
			add("集群成员", errors.New("请先添加实例"))
		}
		if in.Action != "adopt" && in.Action != "validate" && in.Action != "reconcile" && r.Management != "operate" {
			add("运维接管", errors.New("请先接管集群"))
		}
		for _, i := range rows {
			src, e := s.Get("sources", i.SourceID)
			if e != nil {
				return p, e
			}
			spec, e := decode[model.SourceSpec](src.Spec)
			if e != nil {
				return p, e
			}
			ob, e := (Nginx{}).Discover(ctx, s.Remote, spec)
			if e != nil && in.Action == "start" && spec.Runtime == "docker" {
				raw, err := s.Remote.Run(ctx, RemoteRequest{Action: "inspect", Spec: spec})
				identity, _ := decode[Observation](raw)
				previous, _ := decode[Observation](i.Observed)
				if err == nil && identity.State == "stopped" && identity.Identity == previous.Identity {
					ob = previous
					e = nil
				}
			}
			if e != nil && in.Action == "reconcile" {
				raw, err := s.Remote.Run(ctx, RemoteRequest{Action: "reconcile", Spec: spec, OperationID: in.Parameters.OperationID})
				v, _ := decode[Observation](raw)
				if err == nil && v.State == "recoverable" {
					ob, _ = decode[Observation](i.Observed)
					e = nil
				}
			}
			add("发现 "+i.Name, e)
			if in.Action != "adopt" && in.Action != "reconcile" && in.Action != "validate" {
				previous, _ := decode[Observation](i.Observed)
				if previous.ExpectedDigest != "" && previous.ExpectedDigest != ob.Digest {
					add("基线 "+i.Name, errors.New("检测到平台外配置变更，请检查漂移后重新接管"))
				}
			}
			add("目标权限 "+i.Name, s.Authorize(u, "instances", i, actionPermission(in.Action)))
			t := Target{Instance: i, SourceRevision: src.Revision, Spec: spec, Observation: ob}
			if in.Action == "adopt" {
				if len(spec.ManagedPaths) == 0 || spec.HealthURL == "" {
					add("接管范围 "+i.Name, errors.New("请设置受管目录及健康检查地址"))
				} else {
					_, e = s.Remote.Run(ctx, RemoteRequest{Action: "health", Spec: spec})
					add("健康检查 "+i.Name, e)
				}
			}
			if in.Action == "publish" || in.Action == "rollback" || in.Action == "validate" || in.Action == "deploy-certificate" {
				t.Files, e = s.operationFiles(u, r.ID, in, t)
				if e == nil && in.Action != "validate" {
					var versions map[uint]uint
					if in.Parameters.RevisionID > 0 {
						var rev model.Revision
						if err := s.DB.First(&rev, in.Parameters.RevisionID).Error; err != nil {
							return p, err
						}
						cs, _ := decode[model.ConfigSpec](rev.Content)
						versions = cs.Certificates[t.Instance.ID]
					}
					t.Files, t.TLS, e = s.prepareCertificates(u, r.ID, t.Files, in.Parameters.BindingID, versions)
					t.PreviousCertificates = map[uint]json.RawMessage{}
					for _, probe := range t.TLS {
						binding, err := s.Get("certificate-bindings", probe.BindingID)
						if err != nil {
							return p, err
						}
						var state struct {
							Nodes map[uint]json.RawMessage `json:"nodes"`
						}
						json.Unmarshal(binding.Observed, &state)
						t.PreviousCertificates[binding.ID] = state.Nodes[t.Instance.ID]
					}
				}
				add("配置版本 "+i.Name, e)
				if e == nil {
					_, e = s.Remote.Run(ctx, RemoteRequest{Action: "validate", Spec: spec, Files: t.Files, Baseline: ob.Digest, Identity: ob.Identity})
					add("候选配置校验 "+i.Name, e)
				}
			}
			var sites []model.Resource
			if e := s.DB.Table(model.Tables["sites"]).Where("cluster_id = ?", r.ID).Find(&sites).Error; e != nil {
				return p, e
			}
			for _, site := range sites {
				v, _ := decode[model.SiteSpec](site.Spec)
				if v.HealthURL == "" {
					continue
				}
				for _, f := range t.Files {
					if f.Path == v.Path && !f.Deleted {
						t.HealthChecks = append(t.HealthChecks, model.Entry{Name: site.Name, URL: v.HealthURL, Host: v.HealthHost, ExpectedStatus: v.ExpectedStatus})
					}
				}
			}
			targets = append(targets, t)
		}
	} else if in.TargetType == "sources" {
		spec, e := decode[model.SourceSpec](r.Spec)
		if e != nil {
			return p, e
		}
		if !s.Admin(u) {
			return p, ErrForbidden
		}
		if spec.Runtime == "host" && spec.PackageVersion == "" {
			add("固定版本", errors.New("请指定 Nginx 包版本"))
		}
		if spec.Runtime == "docker" && !imageDigestPattern.MatchString(spec.Image) {
			add("固定镜像", errors.New("Docker 镜像必须固定 sha256 摘要"))
		}
		if in.Action == "provision" {
			_, err := s.Remote.Run(ctx, RemoteRequest{Action: "provision-check", Spec: spec})
			add("部署目标预检", err)
		}
		targets = []Target{{Instance: r, Spec: spec, SourceRevision: r.Revision}}
	} else {
		issuanceDigest, e = s.preflightIssuance(ctx, u, r)
		add("证书账户与域名", e)
	}
	if in.Action == "reconcile" {
		var op model.Operation
		if e = s.DB.First(&op, in.Parameters.OperationID).Error; e != nil || op.TargetType != in.TargetType || op.TargetID != r.ID || op.Status != "unknown" {
			add("待核验任务", errors.New("请选择当前集群的待核验任务"))
		} else {
			add("原任务操作授权", s.Authorize(u, in.TargetType, r, actionPermission(op.Action)))
		}
	}
	if isVIPAction(in.Action) {
		add("Keepalived / VIP 预检", s.prepareVIP(ctx, u, r, in, &targets))
		p.Parameters = JSON(in.Parameters)
	}
	p.Executable = true
	for _, c := range checks {
		if !c.Passed {
			p.Executable = false
		}
	}
	p.Checks = JSON(checks)
	sealed, e := seal(string(JSON(targets)))
	if e != nil {
		return p, e
	}
	snapshot := map[string]string{"sealed": sealed}
	if issuanceDigest != "" {
		snapshot["issuanceDigest"] = issuanceDigest
	}
	if configuration != nil {
		current, err := s.Get("configurations", configuration.ID)
		if err != nil {
			return p, err
		}
		if current.Revision != configuration.Revision || digest(current.Spec) != digest(configuration.Spec) {
			return p, ErrConflict
		}
		snapshot["configurationRevision"] = strconv.FormatUint(configuration.Revision, 10)
		snapshot["configurationDigest"] = digest(configuration.Spec)
	}
	p.Snapshot = JSON(snapshot)
	if e = s.DB.Create(&p).Error; e != nil {
		return p, e
	}
	p.IdempotencyKey = fmt.Sprintf("web-plan-%d", p.ID)
	return p, nil
}

// A cluster remains the locking and authorization boundary, but a raw file
// draft can target specific members. Freeze that selection before discovery;
// neither execution nor recovery may expand it to other cluster members.
func (s *Service) operationMembers(cid uint, in PlanInput, members []model.Resource) ([]model.Resource, error) {
	var ids []uint
	scoped := false
	switch in.Action {
	case "configure-vip", "verify-vip", "switch-vip":
		if in.Parameters.VIP == nil {
			return nil, errors.New("请填写 VIP、网卡和目标实例")
		}
		ids = in.Parameters.VIP.InstanceIDs
		scoped = true
	case "publish", "validate", "rollback":
		if in.Parameters.RevisionID > 0 {
			var revision model.Revision
			if err := s.DB.First(&revision, in.Parameters.RevisionID).Error; err != nil {
				return nil, err
			}
			if revision.ClusterID != cid || revision.Kind != "configuration" {
				return nil, ErrForbidden
			}
			spec, err := decode[model.ConfigSpec](revision.Content)
			if err != nil {
				return nil, err
			}
			for id := range spec.Nodes {
				ids = append(ids, id)
			}
			scoped = true
		} else if in.Parameters.ConfigurationID > 0 {
			configuration, err := s.Get("configurations", in.Parameters.ConfigurationID)
			if err != nil {
				return nil, err
			}
			if configuration.ClusterID != cid {
				return nil, ErrForbidden
			}
			spec, err := decode[model.ConfigSpec](configuration.Spec)
			if err != nil {
				return nil, err
			}
			ids = spec.InstanceIDs
			scoped = len(ids) > 0
		}
	case "reconcile":
		var original model.Operation
		if err := s.DB.First(&original, in.Parameters.OperationID).Error; err != nil {
			return nil, err
		}
		if original.TargetType != "clusters" || original.TargetID != cid || original.Status != "unknown" {
			return nil, ErrConflict
		}
		var plan model.Plan
		if err := s.DB.First(&plan, original.PlanID).Error; err != nil {
			return nil, err
		}
		sealed, err := decode[map[string]string](plan.Snapshot)
		if err != nil {
			return nil, err
		}
		plaintext, err := unseal(sealed["sealed"])
		if err != nil {
			return nil, err
		}
		targets, err := decode[[]Target]([]byte(plaintext))
		if err != nil {
			return nil, err
		}
		for _, target := range targets {
			ids = append(ids, target.Instance.ID)
		}
		scoped = true
	}
	if !scoped {
		return members, nil
	}
	if len(ids) == 0 {
		return nil, errors.New("配置版本或任务没有目标实例")
	}
	wanted := make(map[uint]bool, len(ids))
	for _, id := range ids {
		if id == 0 || wanted[id] {
			return nil, errors.New("配置目标实例无效或重复")
		}
		wanted[id] = true
	}
	selected := make([]model.Resource, 0, len(ids))
	for _, member := range members {
		if wanted[member.ID] {
			selected = append(selected, member)
		}
	}
	if len(selected) != len(ids) {
		return nil, errors.New("配置目标实例不在当前集群中，请重新选择")
	}
	return selected, nil
}

func (s *Service) operationFiles(u, cid uint, in PlanInput, t Target) ([]model.File, error) {
	if in.Action == "deploy-certificate" {
		return s.certificateFiles(u, cid, in.Parameters.BindingID)
	}
	if in.Parameters.RevisionID > 0 {
		var rev model.Revision
		if e := s.DB.First(&rev, in.Parameters.RevisionID).Error; e != nil {
			return nil, e
		}
		if rev.ClusterID != cid || rev.Kind != "configuration" {
			return nil, ErrForbidden
		}
		p, e := decode[model.ConfigSpec](rev.Content)
		if e != nil {
			return nil, e
		}
		files, ok := p.Nodes[t.Instance.ID]
		if !ok {
			return nil, errors.New("历史版本不包含当前节点，请选择正确的集群版本")
		}
		return restoreFiles(t.Observation.Files, files), nil
	}
	c, e := s.Get("configurations", in.Parameters.ConfigurationID)
	if e != nil {
		return nil, e
	}
	if c.ClusterID != cid {
		return nil, ErrForbidden
	}
	p, e := decode[model.ConfigSpec](c.Spec)
	if e != nil {
		return nil, e
	}
	if baseline := p.Baselines[t.Instance.ID]; baseline != "" && baseline != t.Observation.Digest {
		return nil, ErrConflict
	}
	files := append([]model.File(nil), p.Files...)
	for i := range files {
		for k, v := range p.Variables[t.Instance.ID] {
			files[i].Content = stringsReplace(files[i].Content, "${"+k+"}", v)
		}
	}
	return files, nil
}
func (s *Service) Submit(ctx context.Context, u, planID uint, key string, confirmed bool) (model.Operation, error) {
	op := model.Operation{}
	if !confirmed || key != fmt.Sprintf("web-plan-%d", planID) {
		return op, errors.New("请确认有效的操作计划")
	}
	if e := s.Module(ctx, u, "execute"); e != nil {
		return op, e
	}
	e := s.DB.Transaction(func(tx *gorm.DB) error {
		local := *s
		local.DB = tx
		var p model.Plan
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, planID).Error; e != nil {
			return e
		}
		if p.UserID != u || p.APIKeyID != apikey.ID(ctx) {
			return ErrForbidden
		}
		old := tx.Where("plan_id = ?", p.ID).First(&op)
		if old.Error == nil {
			return nil
		}
		if !errors.Is(old.Error, gorm.ErrRecordNotFound) {
			return old.Error
		}
		if !p.Executable || time.Now().After(p.ExpiresAt) {
			return ErrConflict
		}
		// Serialize task submission with membership edits and cluster saves.
		// Checking revision before acquiring this row would admit a stale target set.
		var r model.Resource
		var e error
		if p.TargetType == "clusters" {
			e = tx.Table(model.Tables["clusters"]).Clauses(clause.Locking{Strength: "UPDATE"}).First(&r, p.TargetID).Error
		} else {
			r, e = local.Get(p.TargetType, p.TargetID)
		}
		if e != nil {
			return e
		}
		if r.Revision != p.Revision {
			return ErrConflict
		}
		if e = local.Authorize(u, p.TargetType, r, actionPermission(p.Action)); e != nil {
			return e
		}
		if e = local.checkIssuanceDependencies(r, p); e != nil {
			return e
		}
		op = model.Operation{PlanID: p.ID, UserID: u, APIKeyID: p.APIKeyID, TargetType: p.TargetType, TargetID: p.TargetID, Action: p.Action, Status: "queued", IdempotencyKey: key}
		if p.TargetType == "clusters" {
			op.ClusterID = p.TargetID
		}
		if e = tx.Create(&op).Error; e != nil {
			return e
		}
		lockKey := p.TargetType + ":" + fmt.Sprint(p.TargetID)
		if p.TargetType == "clusters" {
			lockKey = fmt.Sprintf("cluster:%d", p.TargetID)
		}
		if p.Action == "reconcile" {
			var params Parameters
			_ = json.Unmarshal(p.Parameters, &params)
			var lock model.Lock
			if e = tx.Where("`key` = ? AND operation_id = ?", lockKey, params.OperationID).First(&lock).Error; e != nil {
				return ErrConflict
			}
			if e = tx.Create(&model.Lock{Key: fmt.Sprintf("reconcile:%d", params.OperationID), OperationID: op.ID}).Error; e != nil {
				return ErrConflict
			}
			return nil
		}
		if isVIPAction(p.Action) {
			params, _ := decode[Parameters](p.Parameters)
			if params.VIP == nil {
				return ErrConflict
			}
			ip, _, err := net.ParseCIDR(params.VIP.CIDR)
			if err != nil {
				return err
			}
			for _, key := range []string{"web-vip-address:" + ip.String(), fmt.Sprintf("web-vip-vrid:%d", params.VIP.RouterID)} {
				if err := tx.Create(&model.Lock{Key: key, OperationID: op.ID}).Error; err != nil {
					return ErrConflict
				}
			}
		}
		if e = tx.Create(&model.Lock{Key: lockKey, OperationID: op.ID}).Error; e != nil {
			return ErrConflict
		}
		return nil
	})
	return op, e
}
func stringsReplace(s, old, value string) string { return strings.ReplaceAll(s, old, value) }
func (s *Service) Operation(u, id uint) (model.Operation, []model.Step, error) {
	var op model.Operation
	if e := s.DB.First(&op, id).Error; e != nil {
		return op, nil, e
	}
	r, e := s.Get(op.TargetType, op.TargetID)
	if e != nil {
		return op, nil, e
	}
	if e = s.Authorize(u, op.TargetType, r, "web_view"); e != nil {
		return op, nil, e
	}
	if e = s.Authorize(u, op.TargetType, r, "web_audit"); e != nil && op.UserID != u {
		return op, nil, e
	}
	var steps []model.Step
	e = s.DB.Where("operation_id = ?", id).Order("id").Find(&steps).Error
	if s.Authorize(u, op.TargetType, r, "web_config_view") != nil {
		for i := range steps {
			steps[i].Evidence = nil
		}
	}
	return op, steps, e
}
func (s *Service) step(op model.Operation, t Target, name, status, message string, evidence any) error {
	e := s.DB.Create(&model.Step{OperationID: op.ID, InstanceID: t.Instance.ID, Name: name, Status: status, Message: message, Evidence: JSON(evidence)}).Error
	if e == nil {
		s.notifyOperation(op.ID)
	}
	return e
}
func (s *Service) finish(op model.Operation, status, message string) {
	defer s.notifyOperation(op.ID)
	now := time.Now().UTC()
	_ = s.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&model.Operation{}).Where("id = ?", op.ID).Updates(map[string]any{"status": status, "message": message, "finished_at": now}).Error; e != nil {
			return e
		}
		if status != "unknown" {
			return tx.Where("operation_id = ?", op.ID).Delete(&model.Lock{}).Error
		}
		return nil
	})
}
func (s *Service) RunWorker(ctx context.Context, id string) {
	go s.RunCollector(ctx)
	go s.workerHeartbeat(ctx, id)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().UTC().Add(-2 * time.Minute)
			s.DB.Model(&model.Operation{}).Where("status IN ? AND heartbeat_at < ?", []string{"preflight", "running", "verifying"}, cutoff).Updates(map[string]any{"status": "unknown", "message": "Worker 心跳丢失，请核验远端状态"})
			var op model.Operation
			if s.DB.Where("status = ?", "queued").Order("id").First(&op).Error == nil {
				now := time.Now().UTC()
				result := s.DB.Model(&model.Operation{}).Where("id = ? AND status = ?", op.ID, "queued").Updates(map[string]any{"status": "preflight", "worker_id": id, "started_at": now, "heartbeat_at": now})
				if result.Error == nil && result.RowsAffected == 1 {
					s.execute(ctx, op)
				}
			}
			s.runRenewals(ctx)
		}
	}
}
func (s *Service) execute(ctx context.Context, op model.Operation) {
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-child.Done():
				return
			case <-ticker.C:
				result := s.DB.Model(&model.Operation{}).Where("id = ? AND status IN ?", op.ID, []string{"preflight", "running", "verifying"}).Update("heartbeat_at", time.Now().UTC())
				if result.Error != nil || result.RowsAffected != 1 {
					cancel()
					return
				}
			}
		}
	}()
	if op.APIKeyID > 0 {
		child = apikey.WithKey(child, op.APIKeyID)
	}
	if e := s.Module(child, op.UserID, "execute"); e != nil {
		s.finish(op, "failed", e.Error())
		return
	}
	var p model.Plan
	if e := s.DB.First(&p, op.PlanID).Error; e != nil {
		s.finish(op, "failed", e.Error())
		return
	}
	r, e := s.Get(p.TargetType, p.TargetID)
	if e != nil || r.Revision != p.Revision {
		s.finish(op, "failed", ErrConflict.Error())
		return
	}
	if e = s.Authorize(op.UserID, p.TargetType, r, actionPermission(p.Action)); e != nil {
		s.finish(op, "failed", e.Error())
		return
	}
	if op.CancelRequested {
		s.finish(op, "failed", "任务已取消，尚未执行远端动作")
		return
	}
	if p.TargetType == "certificates" {
		if e = s.checkIssuanceDependencies(r, p); e != nil {
			s.finish(op, "failed", e.Error())
			return
		}
		s.DB.Model(&op).Update("status", "running")
		if e = s.issue(child, op, r); e != nil {
			s.finish(op, "failed", e.Error())
		} else {
			s.finish(op, "succeeded", "证书签发完成，部署结果单独记录")
		}
		return
	}
	snapshot, e := decode[map[string]string](p.Snapshot)
	if e != nil {
		s.finish(op, "failed", e.Error())
		return
	}
	plaintext, e := unseal(snapshot["sealed"])
	if e != nil {
		s.finish(op, "failed", e.Error())
		return
	}
	targets, e := decode[[]Target]([]byte(plaintext))
	if e != nil {
		s.finish(op, "failed", e.Error())
		return
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Instance.ID < targets[j].Instance.ID })
	if op.Action == "reconcile" {
		s.reconcile(child, op, p, targets)
		return
	}
	if op.Action == "provision" {
		s.DB.Model(&op).Update("status", "running")
		_, e = s.Remote.Run(child, RemoteRequest{Action: "provision", Spec: targets[0].Spec, OperationID: op.ID})
		if e != nil {
			s.finish(op, "unknown", e.Error())
			return
		}
		_, e = s.Discover(child, op.UserID, r.ID)
		if e != nil {
			s.finish(op, "unknown", e.Error())
			return
		}
		s.finish(op, "succeeded", "部署与实例发现完成")
		return
	}
	for _, t := range targets {
		src, e := s.Get("sources", t.Instance.SourceID)
		if e != nil || src.Revision != t.SourceRevision {
			s.finish(op, "failed", ErrConflict.Error())
			return
		}
		current, e := (Nginx{}).Discover(child, s.Remote, t.Spec)
		if e != nil && op.Action == "start" && t.Spec.Runtime == "docker" {
			raw, err := s.Remote.Run(child, RemoteRequest{Action: "inspect", Spec: t.Spec})
			identity, _ := decode[Observation](raw)
			if err == nil && identity.State == "stopped" && identity.Identity == t.Observation.Identity {
				current = t.Observation
				e = nil
			}
		}
		if e != nil || current.Digest != t.Observation.Digest || current.Identity != t.Observation.Identity {
			s.finish(op, "failed", "实例身份或基线变化，请重新预检")
			return
		}
	}
	if isVIPAction(op.Action) {
		s.executeVIP(child, op, p, r, targets)
		return
	}
	s.DB.Model(&model.Operation{}).Where("id = ?", op.ID).Update("status", "running")
	if op.Action == "adopt" || op.Action == "publish" || op.Action == "rollback" || op.Action == "deploy-certificate" {
		if _, e = s.saveConfigurationRevision(op, targets, "before"); e != nil {
			s.finish(op, "failed", e.Error())
			return
		}
	}
	applied := []Target{}
	for _, t := range targets {
		var fresh model.Operation
		if s.DB.First(&fresh, op.ID).Error != nil {
			s.finish(op, "unknown", "无法读取任务状态，暂停远端动作")
			return
		}
		r, err := s.Get(p.TargetType, p.TargetID)
		if fresh.CancelRequested || err != nil || s.Module(child, op.UserID, "execute") != nil || s.Authorize(op.UserID, p.TargetType, r, actionPermission(op.Action)) != nil || s.Authorize(op.UserID, "instances", t.Instance, actionPermission(op.Action)) != nil {
			if len(applied) == 0 {
				s.finish(op, "failed", "任务取消或授权失效")
			} else if s.restoreApplied(child, op, applied) == nil {
				s.finish(op, "rolled-back", "后续节点停止，已变更节点完成回退")
			} else {
				s.finish(op, "unknown", "回退结果待核验")
			}
			return
		}
		for _, probe := range t.TLS {
			binding, err := s.Get("certificate-bindings", probe.BindingID)
			if err != nil {
				s.finish(op, "unknown", "证书绑定失效")
				return
			}
			bs, _ := decode[model.BindingSpec](binding.Spec)
			if s.Asset(op.UserID, "web_certificate", bs.CertificateID, "web_certificate_use") != nil {
				s.finish(op, "unknown", "证书使用授权失效")
				return
			}
		}
		if op.Action == "adopt" {
			t.Observation.ExpectedDigest = t.Observation.Digest
			if e = s.DB.Table(model.Tables["instances"]).Where("id = ?", t.Instance.ID).Updates(map[string]any{"management": "operate", "observed": JSON(t.Observation)}).Error; e != nil {
				s.finish(op, "failed", e.Error())
				return
			}
			continue
		}
		action := op.Action
		if action == "publish" || action == "rollback" || action == "deploy-certificate" {
			action = "apply"
		}
		if e = s.step(op, t, action, "running", "", t.Observation); e != nil {
			s.finish(op, "unknown", e.Error())
			return
		}
		raw, err := s.Remote.Run(child, RemoteRequest{Action: action, Spec: t.Spec, OperationID: op.ID, Baseline: t.Observation.Digest, Identity: t.Observation.Identity, Files: t.Files, TLS: t.TLS, HealthChecks: t.HealthChecks})
		if err != nil {
			s.step(op, t, action, "unknown", err.Error(), nil)
			s.finish(op, "unknown", err.Error())
			return
		}
		out, err := decode[Observation](raw)
		if err != nil {
			s.finish(op, "unknown", "远端证据格式无效")
			return
		}
		if out.State == "rolled-back" {
			s.step(op, t, action, "rolled-back", out.Message, out.Observed)
			if e = s.restoreApplied(child, op, applied); e != nil {
				s.finish(op, "unknown", "部分节点回退待核验: "+e.Error())
			} else {
				s.finish(op, "rolled-back", out.Message)
			}
			return
		}
		if action != "validate" && out.State != "succeeded" {
			s.finish(op, "unknown", "远端未确认操作成功")
			return
		}
		if out.Observed != nil {
			t.ResultDigest = out.Observed.Digest
		}
		applied = append(applied, t)
		if e = s.step(op, t, action, "succeeded", "", out.Observed); e != nil {
			s.finish(op, "unknown", "远端已执行，证据保存失败")
			return
		}
		if out.Observed != nil {
			out.Observed.ExpectedDigest = out.Observed.Digest
			if e = s.DB.Table(model.Tables["instances"]).Where("id = ?", t.Instance.ID).Updates(map[string]any{"observed": JSON(out.Observed), "collected_at": time.Now().UTC(), "status": "healthy"}).Error; e != nil {
				s.finish(op, "unknown", e.Error())
				return
			}
			if action == "apply" {
				if e = s.recordCertificateDeployment(t, *out.Observed); e != nil {
					s.finish(op, "unknown", e.Error())
					return
				}
			}
		}
	}
	if op.Action == "adopt" {
		s.DB.Table(model.Tables["clusters"]).Where("id = ?", r.ID).Updates(map[string]any{"management": "operate", "revision": gorm.Expr("revision + 1")})
	}
	if op.Action == "publish" || op.Action == "rollback" || op.Action == "deploy-certificate" {
		if _, e = s.saveConfigurationRevision(op, targets, "after"); e != nil {
			s.finish(op, "unknown", e.Error())
			return
		}
	}
	if op.Action == "publish" {
		if e = s.markPublishedConfiguration(p); e != nil {
			s.finish(op, "unknown", "远端已完成发布，草稿状态保存失败")
			return
		}
	}
	s.finish(op, "succeeded", "所有目标节点已完成操作与核验")
}

// Publication refers to the frozen draft, not whatever the editor has saved
// since the plan was generated. Legacy plans lack this evidence and leave the
// current draft status unchanged.
func (s *Service) markPublishedConfiguration(p model.Plan) error {
	params, err := decode[Parameters](p.Parameters)
	if err != nil || params.ConfigurationID == 0 || params.RevisionID > 0 {
		return err
	}
	snapshot, err := decode[map[string]string](p.Snapshot)
	if err != nil {
		return err
	}
	revision, err := strconv.ParseUint(snapshot["configurationRevision"], 10, 64)
	if err != nil || snapshot["configurationDigest"] == "" {
		return nil
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var draft model.Resource
		err := tx.Table(model.Tables["configurations"]).Clauses(clause.Locking{Strength: "UPDATE"}).First(&draft, params.ConfigurationID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if draft.Revision != revision || digest(draft.Spec) != snapshot["configurationDigest"] {
			return nil
		}
		return tx.Table(model.Tables["configurations"]).Where("id = ? AND revision = ?", draft.ID, revision).Update("status", "published").Error
	})
}

// Restore only the files belonging to the frozen candidate; unrelated drift is a conflict.
func restoreFiles(current, desired []model.File) []model.File {
	out := append([]model.File(nil), desired...)
	keep := map[string]bool{}
	for _, f := range desired {
		keep[f.Path] = true
	}
	for _, f := range current {
		if !keep[f.Path] {
			out = append(out, model.File{Path: f.Path, Deleted: true})
		}
	}
	return out
}
func (s *Service) restoreApplied(ctx context.Context, op model.Operation, targets []Target) error {
	for j := len(targets) - 1; j >= 0; j-- {
		t := targets[j]
		if t.ResultDigest == "" {
			return errors.New("缺少发布后摘要，禁止覆盖")
		}
		// Recovery uses the successful candidate's digest, never a newly accepted live baseline.
		// Remote journal retains the exact pre-change certificate and file versions.
		raw, e := s.Remote.Run(ctx, RemoteRequest{Action: "restore", Spec: t.Spec, OperationID: op.ID, Identity: t.Observation.Identity, Baseline: t.ResultDigest})
		if e != nil {
			return e
		}
		out, e := decode[Observation](raw)
		if e != nil {
			return e
		}
		if out.State != "rolled-back" || out.Observed == nil {
			return errors.New("回退未确认成功")
		}
		if e = s.step(op, t, "restore", "succeeded", "", out.Observed); e != nil {
			return e
		}
		if e = s.recordRestoredState(t, out.Observed); e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) recordRestoredState(t Target, observed *Observation) error {
	if observed == nil {
		return errors.New("回退缺少远端观测结果")
	}
	for bid, previous := range t.PreviousCertificates {
		b, e := s.Get("certificate-bindings", bid)
		if e != nil {
			return e
		}
		state, _ := decode[map[string]any](b.Observed)
		if state == nil {
			state = map[string]any{}
		}
		nodes, _ := state["nodes"].(map[string]any)
		if nodes == nil {
			nodes = map[string]any{}
		}
		if len(previous) == 0 {
			delete(nodes, fmt.Sprint(t.Instance.ID))
		} else {
			var v any
			json.Unmarshal(previous, &v)
			nodes[fmt.Sprint(t.Instance.ID)] = v
		}
		state["nodes"] = nodes
		if e = s.DB.Table(model.Tables["certificate-bindings"]).Where("id = ?", bid).Updates(map[string]any{"observed": JSON(state), "status": "partial"}).Error; e != nil {
			return e
		}
	}
	observed.ExpectedDigest = observed.Digest
	if e := s.DB.Table(model.Tables["instances"]).Where("id = ?", t.Instance.ID).Updates(map[string]any{"observed": JSON(observed), "status": "healthy", "collected_at": time.Now().UTC()}).Error; e != nil {
		return e
	}
	return nil
}
func (s *Service) saveConfigurationRevision(op model.Operation, targets []Target, phase string) (model.Revision, error) {
	config := model.ConfigSpec{Nodes: map[uint][]model.File{}, Certificates: map[uint]map[uint]uint{}}
	var bindings []model.Resource
	if e := s.DB.Table(model.Tables["certificate-bindings"]).Where("cluster_id = ?", op.ClusterID).Find(&bindings).Error; e != nil {
		return model.Revision{}, e
	}
	for _, t := range targets {
		config.InstanceIDs = append(config.InstanceIDs, t.Instance.ID)
		obs := t.Observation
		if phase != "before" {
			i, e := s.Get("instances", t.Instance.ID)
			if e != nil {
				return model.Revision{}, e
			}
			obs, e = decode[Observation](i.Observed)
			if e != nil {
				return model.Revision{}, e
			}
		}
		config.Nodes[t.Instance.ID] = obs.Files
		config.Certificates[t.Instance.ID] = map[uint]uint{}
		for _, b := range bindings {
			var state struct {
				Nodes map[uint]struct {
					VersionID uint `json:"versionId"`
				} `json:"nodes"`
			}
			json.Unmarshal(b.Observed, &state)
			if version := state.Nodes[t.Instance.ID].VersionID; version > 0 {
				config.Certificates[t.Instance.ID][b.ID] = version
			}
		}
	}
	rev := model.Revision{ClusterID: op.ClusterID, Kind: "configuration", Name: fmt.Sprintf("%s #%d %s", op.Action, op.ID, phase), Content: JSON(config), CreatedBy: op.UserID}
	rev.Digest = digest(rev.Content)
	return rev, s.DB.Create(&rev).Error
}
func (s *Service) recordCertificateDeployment(t Target, ob Observation) error {
	for _, probe := range t.TLS {
		b, e := s.Get("certificate-bindings", probe.BindingID)
		if e != nil {
			return e
		}
		state, _ := decode[map[string]any](b.Observed)
		if state == nil {
			state = map[string]any{}
		}
		nodes, _ := state["nodes"].(map[string]any)
		if nodes == nil {
			nodes = map[string]any{}
		}
		nodes[fmt.Sprint(t.Instance.ID)] = map[string]any{"versionId": probe.VersionID, "fingerprint": probe.Fingerprint, "verifiedAt": time.Now().UTC()}
		state["nodes"] = nodes
		state["desiredVersionId"] = probe.VersionID
		if e = s.DB.Table(model.Tables["certificate-bindings"]).Where("id = ?", b.ID).Updates(map[string]any{"observed": JSON(state), "status": s.bindingStatus(b.ClusterID, nodes, probe.VersionID)}).Error; e != nil {
			return e
		}
	}
	return nil
}

func (s *Service) reconcile(ctx context.Context, op model.Operation, p model.Plan, _ []Target) {
	params, _ := decode[Parameters](p.Parameters)
	var old model.Operation
	var original model.Plan
	if s.DB.First(&old, params.OperationID).Error != nil || old.Status != "unknown" || s.DB.First(&original, old.PlanID).Error != nil {
		s.finish(op, "failed", "原任务不存在或状态已变化")
		return
	}
	wrapped, e := decode[map[string]string](original.Snapshot)
	if e != nil {
		s.finish(op, "failed", e.Error())
		return
	}
	plain, e := unseal(wrapped["sealed"])
	if e != nil {
		s.finish(op, "failed", e.Error())
		return
	}
	targets, e := decode[[]Target]([]byte(plain))
	if e != nil {
		s.finish(op, "failed", e.Error())
		return
	}
	target, err := s.Get(old.TargetType, old.TargetID)
	if err != nil || s.Authorize(op.UserID, old.TargetType, target, actionPermission(old.Action)) != nil {
		s.finish(op, "failed", "没有原任务的恢复授权")
		return
	}
	if isVIPAction(old.Action) {
		s.reconcileVIP(ctx, op, old, original, target, targets)
		return
	}
	complete := true
	allSucceeded := true
	anyRolledBack := false
	changed := []Target{}
	for _, t := range targets {
		sourceID := t.Instance.SourceID
		if old.Action == "provision" {
			sourceID = t.Instance.ID
		}
		src, e := s.Get("sources", sourceID)
		if e != nil || src.Revision != t.SourceRevision {
			complete = false
			continue
		}
		if old.Action == "provision" {
			obs, err := (Nginx{}).Discover(ctx, s.Remote, t.Spec)
			if err == nil {
				_, err = s.Remote.Run(ctx, RemoteRequest{Action: "health", Spec: t.Spec})
			}
			if err != nil {
				complete = false
				continue
			}
			s.step(op, t, "reconcile", "succeeded", "", obs)
			_, err = s.Discover(ctx, op.UserID, sourceID)
			if err != nil {
				complete = false
			}
			continue
		}
		raw, e := s.Remote.Run(ctx, RemoteRequest{Action: "reconcile", Spec: t.Spec, OperationID: old.ID, TLS: t.TLS, Identity: t.Observation.Identity})
		if e != nil {
			complete = false
			continue
		}
		out, e := decode[Observation](raw)
		if e != nil {
			complete = false
			continue
		}
		if out.State == "recoverable" {
			if s.Authorize(op.UserID, "instances", t.Instance, actionPermission(old.Action)) != nil {
				complete = false
				continue
			}
			raw, e = s.Remote.Run(ctx, RemoteRequest{Action: "recover", Spec: t.Spec, OperationID: old.ID, Identity: t.Observation.Identity})
			if e != nil {
				complete = false
				s.step(op, t, "recover", "unknown", e.Error(), nil)
				continue
			}
			out, e = decode[Observation](raw)
			if e != nil {
				complete = false
				continue
			}
		}
		s.step(op, t, "reconcile", out.State, "", out.Observed)
		switch out.State {
		case "succeeded":
			if out.Observed == nil {
				complete = false
				continue
			}
			t.ResultDigest = out.Observed.Digest
			changed = append(changed, t)
			out.Observed.ExpectedDigest = out.Observed.Digest
			s.DB.Table(model.Tables["instances"]).Where("id = ?", t.Instance.ID).Update("observed", JSON(out.Observed))
			s.recordCertificateDeployment(t, *out.Observed)
		case "rolled-back":
			if e = s.recordRestoredState(t, out.Observed); e != nil {
				complete = false
				continue
			}
			anyRolledBack = true
			allSucceeded = false
		case "not-started":
			allSucceeded = false
			if out.Observed == nil || out.Observed.Digest != t.Observation.Digest {
				complete = false
			}
		default:
			complete = false
		}
	}
	if !complete {
		s.finish(op, "failed", "原任务仍有结果不明或冲突的节点，锁已保留")
		return
	}
	status := "succeeded"
	if !allSucceeded {
		if len(changed) > 0 && (old.Action == "publish" || old.Action == "rollback" || old.Action == "deploy-certificate") {
			if e = s.restoreApplied(ctx, old, changed); e != nil {
				s.finish(op, "failed", "已核验节点回退失败，原锁已保留")
				return
			}
			status = "rolled-back"
		} else {
			status = "failed"
			if anyRolledBack {
				status = "rolled-back"
			}
		}
	} else if old.Action == "publish" || old.Action == "rollback" || old.Action == "deploy-certificate" {
		if _, e = s.saveConfigurationRevision(old, targets, "reconciled"); e != nil {
			s.finish(op, "failed", e.Error())
			return
		}
		if old.Action == "publish" {
			if e = s.markPublishedConfiguration(original); e != nil {
				s.finish(op, "failed", "远端已完成发布，草稿状态保存失败，原锁已保留")
				return
			}
		}
	}
	s.finish(old, status, "远端状态已核验，详见核验任务")
	s.finish(op, "succeeded", "核验完成")
}

func (s *Service) workerHeartbeat(ctx context.Context, id string) {
	key := "worker:" + id
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		s.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&model.Lease{Key: key, Owner: id, ExpiresAt: time.Now().UTC().Add(45 * time.Second)})
		select {
		case <-ctx.Done():
			s.DB.Where("`key` = ?", key).Delete(&model.Lease{})
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) bindingStatus(cid uint, nodes map[string]any, version uint) string {
	if version == 0 {
		return "partial"
	}
	var members []model.Resource
	if s.DB.Table(model.Tables["instances"]).Where("cluster_id = ?", cid).Find(&members).Error != nil {
		return "partial"
	}
	for _, i := range members {
		b, _ := json.Marshal(nodes[fmt.Sprint(i.ID)])
		var v struct {
			VersionID uint `json:"versionId"`
		}
		if json.Unmarshal(b, &v) != nil || v.VersionID != version {
			return "partial"
		}
	}
	if len(members) == 0 {
		return "partial"
	}
	return "deployed"
}

// Preview strips certificate private material even for administrators.
func (s *Service) PlanPreview(p model.Plan) ([]map[string]any, error) {
	wrapped, e := decode[map[string]string](p.Snapshot)
	if e != nil {
		return nil, e
	}
	plain, e := unseal(wrapped["sealed"])
	if e != nil {
		return nil, e
	}
	targets, e := decode[[]Target]([]byte(plain))
	if e != nil {
		return nil, e
	}
	views := []map[string]any{}
	for _, t := range targets {
		files := append([]model.File(nil), t.Files...)
		for i := range files {
			if strings.Contains(files[i].Content, "PRIVATE KEY-----") {
				files[i].Content = "[private key omitted]"
			}
		}
		views = append(views, map[string]any{"instanceId": t.Instance.ID, "name": t.Instance.Name, "files": files, "baseline": t.Observation.Digest, "tls": t.TLS})
	}
	return views, nil
}
