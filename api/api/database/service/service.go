package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"

	cmdbdao "dodevops-api/api/cmdb/dao"
	cmdb "dodevops-api/api/cmdb/model"
	cmdbservice "dodevops-api/api/cmdb/service"
	accounts "dodevops-api/api/configcenter/model"
	"dodevops-api/api/database/model"
	"dodevops-api/common/util"
	"dodevops-api/pkg/apikey"
	"gorm.io/gorm"
)

type Service struct{ DB *gorm.DB }

func New(db *gorm.DB) *Service   { return &Service{DB: db} }
func JSON(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
func object(raw json.RawMessage) map[string]any {
	v := map[string]any{}
	_ = json.Unmarshal(raw, &v)
	if v == nil {
		v = map[string]any{}
	}
	return v
}
func token() string      { b := make([]byte, 24); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func nowPtr() *time.Time { t := time.Now().UTC(); return &t }
func collectedFresh(at *time.Time, window time.Duration) bool {
	if at == nil {
		return false
	}
	age := time.Since(*at)
	return age >= -5*time.Second && age < window
}

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var unitPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.@-]{0,99}$`)
var ErrForbidden = errors.New("没有当前集群的操作权限")
var ErrConflict = errors.New("集群正在执行其他操作，或计划已过期，请刷新并重新预检")

func (s *Service) keyExecutionAllowed(ctx context.Context, user uint) error {
	identity, e := apikey.EffectiveIdentity(ctx, s.DB, user)
	if e != nil || !identity.Allows("database:execute") {
		return ErrForbidden
	}
	return nil
}

func (s *Service) IsAdmin(user uint) bool {
	ok, e := cmdbdao.NewCmdbAssetPermissionDao(s.DB).IsAdministrator(user)
	return e == nil && ok
}
func (s *Service) Scope(user uint, action string) (map[uint]bool, error) {
	d := cmdbdao.NewCmdbAssetPermissionDao(s.DB)
	admin, e := d.IsAdministrator(user)
	if e != nil {
		return nil, e
	}
	if admin {
		return nil, nil
	}
	grants, e := d.EffectivePermissions(user)
	if e != nil {
		return nil, e
	}
	return d.AllowedAssetIDs(grants, "database_cluster", action)
}
func (s *Service) Authorize(user, id uint, action string) error {
	scope, e := s.Scope(user, action)
	if e != nil {
		return e
	}
	if scope != nil && !scope[id] {
		return ErrForbidden
	}
	return nil
}
func (s *Service) Cluster(id uint) (*model.Cluster, error) {
	var c model.Cluster
	if id == 0 {
		return nil, errors.New("集群 ID 无效")
	}
	if e := s.DB.Preload("Members", func(q *gorm.DB) *gorm.DB { return q.Order("id") }).Preload("Endpoints").First(&c, id).Error; e != nil {
		return nil, errors.New("集群不存在")
	}
	linkPatroniMembers(&c)
	c.CompatibilityReason = compatibilityReason(&c)
	c.PlacementWarnings = placementWarnings(&c)
	c.Replication = ReplicationLinks(&c)
	var vip model.VirtualIP
	if s.DB.First(&vip, "cluster_id = ?", id).Error == nil {
		if !collectedFresh(vip.CollectedAt, 90*time.Second) {
			vip.Status = "unknown"
		}
		c.VIP = &vip
	}
	return &c, nil
}

type Filter struct {
	Name, Engine, Environment, Status string
	GroupID, ClusterID                uint
	Page, Size                        int
}

func (s *Service) GroupIDs(id uint) ([]uint, error) {
	var all []cmdb.CmdbGroup
	if e := s.DB.Select("id,parent_id").Find(&all).Error; e != nil {
		return nil, e
	}
	seen := map[uint]bool{id: true}
	changed := true
	for changed {
		changed = false
		for _, g := range all {
			if seen[g.ParentID] && !seen[g.ID] {
				seen[g.ID] = true
				changed = true
			}
		}
	}
	ids := []uint{}
	for id := range seen {
		ids = append(ids, id)
	}
	return ids, nil
}
func (s *Service) List(user uint, f Filter) ([]model.Cluster, int64, error) {
	scope, e := s.Scope(user, "db_cluster_view")
	if e != nil {
		return nil, 0, e
	}
	q := cmdbdao.ApplyAssetIDScope(s.DB.Model(&model.Cluster{}), scope)
	if f.ClusterID > 0 {
		q = q.Where("id = ?", f.ClusterID)
	}
	if f.Name != "" {
		q = q.Where("name LIKE ?", "%"+strings.TrimSpace(f.Name)+"%")
	}
	if f.Engine != "" {
		q = q.Where("engine = ?", f.Engine)
	}
	if f.Environment != "" {
		q = q.Where("environment = ?", f.Environment)
	}
	if f.Status == "stale" {
		q = q.Where("collected_at IS NULL OR collected_at < ?", time.Now().Add(-90*time.Second))
	} else if f.Status == "planned" {
		q = q.Where("status = ?", "planned")
	} else if f.Status != "" {
		q = q.Where("status = ? AND collected_at >= ?", f.Status, time.Now().Add(-90*time.Second))
	}
	if f.GroupID != 0 {
		ids, e := s.GroupIDs(f.GroupID)
		if e != nil {
			return nil, 0, e
		}
		q = q.Where("group_id IN ?", ids)
	}
	var total int64
	if e = q.Count(&total).Error; e != nil {
		return nil, 0, e
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 || f.Size > 100 {
		f.Size = 20
	}
	rows := []model.Cluster{}
	e = q.Preload("Members").Preload("Endpoints").Order("id DESC").Offset((f.Page - 1) * f.Size).Limit(f.Size).Find(&rows).Error
	if e != nil {
		return nil, 0, e
	}
	if len(rows) > 0 {
		ids := make([]uint, len(rows))
		for i := range rows {
			ids[i] = rows[i].ID
		}
		var backups []model.BackupSummary
		latest := s.DB.Model(&model.Backup{}).Select("MAX(id)").Where("cluster_id IN ?", ids).Group("cluster_id")
		if e = s.DB.Model(&model.Backup{}).Select("id,cluster_id,status,created_at,verified_at").Where("id IN (?)", latest).Scan(&backups).Error; e != nil {
			return nil, 0, e
		}
		var operations []model.OperationSummary
		if e = s.DB.Model(&model.Operation{}).Select("id,cluster_id,action,status").Where("cluster_id IN ? AND status IN ?", ids, []string{"queued", "preflight", "running", "verifying", "unknown"}).Order("id DESC").Scan(&operations).Error; e != nil {
			return nil, 0, e
		}
		for i := range rows {
			for j := range backups {
				if backups[j].ClusterID == rows[i].ID {
					rows[i].LatestBackup = &backups[j]
					break
				}
			}
			for j := range operations {
				if operations[j].ClusterID == rows[i].ID {
					rows[i].CurrentOperation = &operations[j]
					break
				}
			}
		}
	}
	for i := range rows {
		rows[i].CompatibilityReason = compatibilityReason(&rows[i])
		rows[i].PlacementWarnings = placementWarnings(&rows[i])
		MarkStale(&rows[i])
	}
	return rows, total, e
}
func MarkStale(c *model.Cluster) {
	if c.Status == "planned" {
		return
	}
	if !collectedFresh(c.CollectedAt, 90*time.Second) {
		c.Status = "stale"
		for i := range c.Replication {
			c.Replication[i].State = "unknown"
			c.Replication[i].Message = "采集过期，复制状态未知"
		}
		h := object(c.Health)
		for _, key := range []string{"database", "replication", "control", "endpoint", "backup"} {
			h[key] = "unknown"
		}
		c.Health = JSON(h)
		for i := range c.Members {
			c.Members[i].State = "unknown"
		}
		for i := range c.Endpoints {
			c.Endpoints[i].Status = "unknown"
		}
	}
}

type MemberInput struct {
	DataMemberID uint   `json:"dataMemberId"`
	TLSMode      string `json:"tlsMode"`
	HostID       uint   `json:"hostId"`
	SourceID     uint   `json:"sourceId"`
	AccountID    uint   `json:"accountId"`
	Address      string `json:"address"`
	Port         int    `json:"port"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Role         string `json:"role"`
	Shard        string `json:"shard"`
	DataDir      string `json:"dataDir"`
	Service      string `json:"service"`
}
type ClusterRequest struct {
	Name        string           `json:"name"`
	Engine      string           `json:"engine"`
	Topology    string           `json:"topology"`
	GroupID     uint             `json:"groupId"`
	Environment string           `json:"environment"`
	Provision   bool             `json:"provision"`
	Password    string           `json:"password"`
	Members     []MemberInput    `json:"members"`
	Endpoints   []model.Endpoint `json:"endpoints"`
	Policy      json.RawMessage  `json:"policy"`
}
type Secrets struct {
	Password    string `json:"password"`
	Replication string `json:"replication"`
	Control     string `json:"control"`
	Sentinel    string `json:"sentinel"`
	Topology    string `json:"topology"`
	Fencing     string `json:"fencing"`
}

func (s *Service) secrets(c *model.Cluster) (Secrets, error) {
	var sec Secrets
	if c.Credentials == "" {
		if c.DeploymentID != nil {
			v, e := cmdbservice.NewDatabaseAutomationService(s.DB).DeploymentCredentials(*c.DeploymentID)
			if e != nil {
				return sec, e
			}
			sec = Secrets{Password: v["password"], Replication: v["replication"], Control: v["control"], Topology: v["topology"]}
		}
		return sec, nil
	}
	v, e := util.AESDecrypt(c.Credentials)
	if e != nil {
		return sec, errors.New("无法读取受管集群凭据")
	}
	e = json.Unmarshal([]byte(v), &sec)
	return sec, e
}
func (s *Service) memberAccount(c *model.Cluster, m model.Member) (string, string, error) {
	if m.AccountID != 0 {
		var a accounts.AccountAuth
		if s.DB.First(&a, m.AccountID).Error != nil {
			return "", "", errors.New("关联凭据不存在")
		}
		p, e := a.DecryptPassword()
		return a.Name, p, e
	}
	sec, e := s.secrets(c)
	if e != nil {
		return "", "", e
	}
	if m.Kind != "data" {
		return "autoops", sec.Control, nil
	}
	return map[string]string{"mysql": "root", "postgresql": "postgres", "redis": "default"}[c.Engine], sec.Password, nil
}
func validAddress(v string) bool {
	if ip := net.ParseIP(v); ip != nil {
		return !ip.IsUnspecified() && !ip.IsMulticast()
	}
	return len(v) <= 253 && regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9.-]*[a-zA-Z0-9])?$`).MatchString(v)
}

func (s *Service) prepare(r ClusterRequest) (*model.Cluster, error) {
	if !namePattern.MatchString(r.Name) || r.GroupID == 0 {
		return nil, errors.New("集群名称须为 1–64 位字母数字、下划线或连字符，并选择业务分组")
	}
	allowed := map[string][]string{"mysql": {"standalone", "mysql-gtid"}, "postgresql": {"standalone", "postgresql-patroni"}, "redis": {"standalone", "redis-sentinel", "redis-cluster"}}
	ok := false
	for _, t := range allowed[r.Engine] {
		if t == r.Topology {
			ok = true
		}
	}
	if !ok {
		return nil, errors.New("数据库引擎与拓扑类型不匹配")
	}
	var group cmdb.CmdbGroup
	if s.DB.First(&group, r.GroupID).Error != nil {
		return nil, errors.New("业务分组不存在")
	}
	if len(r.Members) == 0 || len(r.Members) > 64 {
		return nil, errors.New("需要 1–64 个成员")
	}
	if r.Provision {
		r.Members = withControlMembers(r)
	}
	c := &model.Cluster{Name: r.Name, Engine: r.Engine, Topology: r.Topology, GroupID: r.GroupID, Environment: r.Environment, Management: "observe", Status: "unknown", Revision: 1, Policy: JSON(map[string]any{"priority": "data-protection"}), Health: JSON(map[string]any{})}
	if len(r.Environment) > 32 {
		return nil, errors.New("环境名称过长")
	}
	policy := object(r.Policy)
	// Policy is declarative configuration, never a source of validation attestations.
	for key := range policy {
		if !map[string]bool{"masterName": true, "memoryMB": true, "warnPercent": true, "criticalPercent": true, "replicaLagSeconds": true, "proxyPort": true, "readPort": true, "vip": true, "interface": true, "vipCIDR": true, "fenceAccountId": true, "fenceAddress": true, "fencePort": true}[key] {
			return nil, fmt.Errorf("不支持的策略参数 %s", key)
		}
	}
	if v, ok := policy["masterName"]; ok && !namePattern.MatchString(text(v)) {
		return nil, errors.New("Sentinel masterName 须为 1–64 位字母数字、下划线或连字符")
	}
	for key, limits := range map[string][2]float64{"memoryMB": {32, 1048576}, "replicaLagSeconds": {1, 3600}, "proxyPort": {1024, 65534}, "readPort": {1024, 65535}, "warnPercent": {1, 99}, "criticalPercent": {2, 100}} {
		if v, ok := policy[key]; ok {
			n := number(v)
			if math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) || n < limits[0] || n > limits[1] {
				return nil, fmt.Errorf("策略参数 %s 超出范围", key)
			}
		}
	}
	if policy["warnPercent"] != nil && policy["criticalPercent"] != nil && number(policy["warnPercent"]) >= number(policy["criticalPercent"]) {
		return nil, errors.New("严重阈值必须高于提醒阈值")
	}
	policy["priority"] = "data-protection"
	c.Policy = JSON(policy)
	seen := map[string]bool{}
	dataCount := 0
	hosts := map[uint]bool{}
	for i, in := range r.Members {
		m := model.Member{TLSMode: in.TLSMode, DataMemberID: in.DataMemberID, HostID: in.HostID, SourceID: in.SourceID, AccountID: in.AccountID, Name: in.Name, Address: in.Address, Port: in.Port, Kind: in.Kind, Role: in.Role, Shard: in.Shard, DataDir: in.DataDir, Service: in.Service, State: "unknown", Metrics: JSON(map[string]any{})}
		if m.Kind == "" {
			m.Kind = "data"
		}
		if !map[string]bool{"data": true, "orchestrator": true, "sentinel": true, "patroni": true, "etcd": true, "proxy": true}[m.Kind] {
			return nil, errors.New("成员类型无效")
		}
		if m.SourceID > 0 {
			var src cmdb.CmdbSQL
			var a accounts.AccountAuth
			if s.DB.First(&src, m.SourceID).Error != nil || s.DB.First(&a, src.AccountID).Error != nil {
				return nil, errors.New("数据源或关联账号不存在")
			}
			if map[int]string{1: "mysql", 2: "postgresql", 3: "redis"}[src.Type] != r.Engine || m.Kind != "data" {
				return nil, errors.New("数据源类型不匹配")
			}
			m.HostID, m.AccountID, m.Address, m.Port = src.HostID, src.AccountID, a.Host, a.Port
			m.TLSMode = src.TLSMode
		}
		if r.Provision {
			if m.HostID == 0 {
				return nil, errors.New("新建集群必须选择 CMDB 主机")
			}
			p, e := cmdbservice.NewDatabaseAutomationService(s.DB).HostProfile(m.HostID)
			if e != nil {
				return nil, e
			}
			m.Address = p.Address
			m.Managed = true
			m.Service = fmt.Sprintf("autoops-db-%s-%d", r.Name, i+1)
			m.DataDir = "/var/lib/autoops-database/" + r.Name + fmt.Sprintf("/%d", i+1)
		}
		if m.Kind == "data" || m.Kind == "sentinel" {
			if e := validateTLS(c.Engine, m.TLSMode); e != nil {
				return nil, e
			}
		}
		if !validAddress(m.Address) || m.Port < 1024 || m.Port > 65535 {
			return nil, errors.New("成员地址或端口无效（端口范围 1024–65535）")
		}
		if r.Provision && r.Engine == "mysql" && m.Kind == "data" && m.Port > 45535 {
			return nil, errors.New("MySQL 角色探针需要预留数据端口 +20000")
		}
		key := net.JoinHostPort(m.Address, fmt.Sprint(m.Port))
		if seen[key] {
			return nil, errors.New("成员地址和端口重复")
		}
		seen[key] = true
		if m.Service != "" && !unitPattern.MatchString(m.Service) {
			return nil, errors.New("服务名无效")
		}
		if m.DataDir != "" && (!strings.HasPrefix(m.DataDir, "/") || strings.Contains(m.DataDir, "..") || strings.ContainsAny(m.DataDir, "\n\r\x00")) {
			return nil, errors.New("数据目录无效")
		}
		if m.Name == "" {
			m.Name = fmt.Sprintf("%s-%d", m.Kind, i+1)
		}
		if len(m.Name) > 100 || len(m.Shard) > 64 {
			return nil, errors.New("成员名称过长")
		}
		if m.Kind == "data" {
			dataCount++
			hosts[m.HostID] = true
		}
		if !r.Provision && m.AccountID == 0 && m.Kind != "etcd" && m.Kind != "proxy" {
			return nil, errors.New("接管成员需要配置中心凭据引用")
		}
		c.Members = append(c.Members, m)
	}
	if dataCount == 0 {
		return nil, errors.New("至少需要一个数据节点")
	}
	if r.Provision {
		if r.Topology == "standalone" && dataCount != 1 {
			return nil, errors.New("单机只能有一个数据节点")
		}
		if r.Topology != "standalone" && (dataCount < 3 || len(hosts) < 3) {
			return nil, errors.New("高可用新建至少需要三个不同主机上的数据节点")
		}
		if r.Topology == "redis-cluster" && dataCount != 9 {
			return nil, errors.New("新建 Redis Cluster 使用三主六从共九个数据实例")
		}
		n := 0
		shardHosts := map[string]map[uint]bool{}
		for i := range c.Members {
			m := &c.Members[i]
			if m.Kind != "data" {
				continue
			}
			if m.Role == "" {
				m.Role = "replica"
				if n == 0 || r.Topology == "redis-cluster" && n < 3 {
					m.Role = "primary"
				}
			}
			if r.Topology == "redis-cluster" {
				if m.Port > 55535 {
					return nil, errors.New("Redis Cluster 端口须预留 +10000 的总线端口")
				}
				if m.Shard == "" {
					m.Shard = fmt.Sprintf("shard-%d", n%3+1)
				}
				if shardHosts[m.Shard] == nil {
					shardHosts[m.Shard] = map[uint]bool{}
				}
				if shardHosts[m.Shard][m.HostID] {
					return nil, errors.New("同一 Redis 分片的三个实例必须分布到三个不同主机")
				}
				shardHosts[m.Shard][m.HostID] = true
			}
			n++
		}
		if r.Password == "" {
			r.Password = token()
		}
		if !regexp.MustCompile(`^[A-Za-z0-9_!@%+=.,:-]{12,128}$`).MatchString(r.Password) {
			return nil, errors.New("密码需 12–128 位，可用字母数字与 _!@%+=.,:-")
		}
		secret := Secrets{Password: r.Password, Replication: token(), Control: token(), Sentinel: token()}
		encrypted, e := util.AESEncrypt(string(JSON(secret)))
		if e != nil {
			return nil, e
		}
		c.Credentials = encrypted
		c.Management = "operate"
		c.Status = "planned"
	}
	for _, endpoint := range r.Endpoints {
		if e := validateTLS(c.Engine, endpoint.TLSMode); e != nil {
			return nil, e
		}
		endpoint.ID = 0
		endpoint.ClusterID = 0
		endpoint.Status = "unknown"
		endpoint.VerifiedAt = nil
		if !map[string]bool{"read-write": true, "read-only": true, "sentinel": true, "cluster": true}[endpoint.Mode] {
			return nil, errors.New("访问入口类型无效")
		}
		var addresses []string
		if json.Unmarshal(endpoint.Addresses, &addresses) != nil || len(addresses) == 0 {
			return nil, errors.New("访问入口地址无效")
		}
		for _, a := range addresses {
			h, p, e := net.SplitHostPort(a)
			if e != nil || !validAddress(h) || p == "" {
				return nil, errors.New("入口须使用 host:port")
			}
		}
		c.Endpoints = append(c.Endpoints, endpoint)
	}
	return c, nil
}
func withControlMembers(r ClusterRequest) []MemberInput {
	out := append([]MemberInput(nil), r.Members...)
	hosts := []uint{}
	seen := map[uint]bool{}
	has := map[string]bool{}
	for _, m := range out {
		has[fmt.Sprintf("%d/%s", m.HostID, m.Kind)] = true
		if (m.Kind == "" || m.Kind == "data") && !seen[m.HostID] {
			hosts = append(hosts, m.HostID)
			seen[m.HostID] = true
		}
	}
	for i, h := range hosts {
		components := map[string]int{}
		switch r.Topology {
		case "redis-sentinel":
			components["sentinel"] = 26380
		case "postgresql-patroni":
			components["patroni"] = 18008
			components["etcd"] = 12379
		case "mysql-gtid":
			components["orchestrator"] = 13001
		}
		if (r.Topology == "postgresql-patroni" || r.Topology == "mysql-gtid") && i < 2 {
			port := int(number(object(r.Policy)["proxyPort"]))
			if port == 0 {
				port = 23306
				if r.Engine == "postgresql" {
					port = 25432
				}
			}
			components["proxy"] = port
		}
		kinds := []string{"etcd", "patroni", "orchestrator", "sentinel", "proxy"}
		for _, kind := range kinds {
			port, ok := components[kind]
			if ok && !has[fmt.Sprintf("%d/%s", h, kind)] {
				out = append(out, MemberInput{HostID: h, Kind: kind, Port: port, Name: fmt.Sprintf("%s-%d", kind, i+1)})
			}
		}
	}
	return out
}
func (s *Service) Create(ctx context.Context, r ClusterRequest, user uint) (*model.Cluster, error) {
	if !s.IsAdmin(user) {
		return nil, ErrForbidden
	}
	c, e := s.prepare(r)
	if e != nil {
		return nil, e
	}
	if !r.Provision {
		s.discoverMembers(ctx, c)
		c.Health = health(c)
		c.Status = healthStatus(c)
		c.CollectedAt = nowPtr()
	}
	c.CreatedBy = user
	e = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(c).Error; e != nil {
			return errors.New("集群名称或成员地址已被登记，请检查重复接管")
		}
		linkPatroniMembers(c)
		for _, m := range c.Members {
			if m.DataMemberID > 0 {
				if e := tx.Model(&m).Update("data_member_id", m.DataMemberID).Error; e != nil {
					return e
				}
			}
		}

		for _, m := range c.Members {
			if m.SourceID == 0 {
				continue
			}
			r := tx.Model(&cmdb.CmdbSQL{}).Where("id = ? AND (cluster_id IS NULL OR cluster_id = 0 OR cluster_id = ?)", m.SourceID, c.ID).Update("cluster_id", c.ID)
			if r.Error != nil || r.RowsAffected != 1 {
				return errors.New("数据源已关联其他集群，不能重复接管")
			}
		}
		return tx.Create(&model.Event{ClusterID: c.ID, UserID: user, Kind: "registered", Severity: "info", Message: "集群已登记；发现、安装与验收分别记录，未修改现有服务"}).Error
	})
	return c, e
}
func (s *Service) Discover(ctx context.Context, r ClusterRequest, user uint) (*model.Cluster, error) {
	if !s.IsAdmin(user) {
		return nil, ErrForbidden
	}
	r.Provision = false
	c, e := s.prepare(r)
	if e != nil {
		return nil, e
	}
	s.discoverMembers(ctx, c)
	c.Health = health(c)
	c.Status = healthStatus(c)
	return c, nil
}
func member(c *model.Cluster, id uint) (*model.Member, error) {
	for i := range c.Members {
		if c.Members[i].ID == id {
			return &c.Members[i], nil
		}
	}
	return nil, errors.New("成员不属于当前集群")
}
func (s *Service) event(cluster, user, operation uint, kind, severity, message string) {
	s.DB.Create(&model.Event{ClusterID: cluster, UserID: user, OperationID: operation, Kind: kind, Severity: severity, Message: message})
}
func sortedKeys(m map[uint]bool) []uint {
	out := []uint{}
	for id := range m {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
