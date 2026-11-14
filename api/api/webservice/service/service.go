package service

import (
	"context"
	"crypto/sha256"
	cmdbdao "dodevops-api/api/cmdb/dao"
	cmdb "dodevops-api/api/cmdb/model"
	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/webservice/model"
	"dodevops-api/pkg/apikey"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrForbidden = errors.New("没有目标 Web 资产的操作权限")
var ErrConflict = errors.New("资源已变化或存在活动任务，请刷新并重新预检")

type Service struct {
	DB             *gorm.DB
	Remote         RuntimeConnector
	CloudflareHTTP *http.Client
}

func New(db *gorm.DB) *Service                   { s := &Service{DB: db}; s.Remote = &SSHRuntime{DB: db}; return s }
func JSON(v any) json.RawMessage                 { b, _ := json.Marshal(v); return b }
func decode[T any](v json.RawMessage) (T, error) { var t T; e := json.Unmarshal(v, &t); return t, e }
func digest(v []byte) string                     { h := sha256.Sum256(v); return hex.EncodeToString(h[:]) }
func (s *Service) Admin(u uint) bool {
	b, e := cmdbdao.NewCmdbAssetPermissionDao(s.DB).IsAdministrator(u)
	return e == nil && b
}
func (s *Service) Module(ctx context.Context, u uint, action string) error {
	i, e := apikey.EffectiveIdentity(ctx, s.DB, u)
	if e != nil || !i.Allows("web:"+action) {
		return ErrForbidden
	}
	return nil
}
func (s *Service) Scope(u uint, kind, action string) (map[uint]bool, error) {
	d := cmdbdao.NewCmdbAssetPermissionDao(s.DB)
	if s.Admin(u) {
		return nil, nil
	}
	p, e := d.EffectivePermissions(u)
	if e != nil {
		return nil, e
	}
	return d.AllowedAssetIDs(p, kind, action)
}
func (s *Service) Asset(u uint, kind string, id uint, action string) error {
	ids, e := s.Scope(u, kind, action)
	if e != nil {
		return e
	}
	if ids != nil && !ids[id] {
		return ErrForbidden
	}
	return nil
}
func (s *Service) Get(kind string, id uint) (model.Resource, error) {
	r := model.Resource{}
	t, ok := model.Tables[kind]
	if !ok || id == 0 {
		return r, errors.New("资源类型或 ID 无效")
	}
	e := s.DB.Table(t).First(&r, id).Error
	return r, e
}
func asset(kind string, r model.Resource) (string, uint) {
	switch kind {
	case "sources":
		return "web_source", r.ID
	case "instances":
		return "web_source", r.SourceID
	case "certificates":
		return "web_certificate", r.ID
	case "clusters":
		return "web_cluster", r.ID
	default:
		return "web_cluster", r.ClusterID
	}
}
func (s *Service) Authorize(u uint, kind string, r model.Resource, action string) error {
	if kind == "dns-accounts" || kind == "acme-accounts" {
		if !s.Admin(u) {
			return ErrForbidden
		}
		return nil
	}
	if kind == "renewal-policies" {
		spec, e := decode[model.RenewalSpec](r.Spec)
		if e != nil {
			return e
		}
		return s.Asset(u, "web_certificate", spec.CertificateID, action)
	}
	typ, id := asset(kind, r)
	return s.Asset(u, typ, id, action)
}
func readAction(kind string) string {
	switch kind {
	case "configurations", "sites", "upstreams", "config-groups":
		return "web_config_view"
	default:
		return "web_view"
	}
}
func writeAction(kind string) string {
	switch kind {
	case "configurations", "sites", "upstreams", "config-groups":
		return "web_config_edit"
	case "certificates", "renewal-policies":
		return "web_certificate_manage"
	case "certificate-bindings":
		return "web_publish"
	default:
		return "web_manage"
	}
}

type Filter struct {
	Light                                        bool
	Name, Engine, Environment, Status, FileGroup string
	GroupID, ClusterID, ConfigGroupID            uint
	Page, Size                                   int
}

func (s *Service) GroupIDs(id uint) ([]uint, error) {
	var rows []cmdb.CmdbGroup
	if e := s.DB.Select("id,parent_id").Find(&rows).Error; e != nil {
		return nil, e
	}
	keep := map[uint]bool{id: true}
	for changed := true; changed; {
		changed = false
		for _, g := range rows {
			if keep[g.ParentID] && !keep[g.ID] {
				keep[g.ID] = true
				changed = true
			}
		}
	}
	out := []uint{}
	for i := range keep {
		out = append(out, i)
	}
	return out, nil
}
func (s *Service) List(u uint, kind string, f Filter) ([]model.Resource, int, error) {
	table, ok := model.Tables[kind]
	if !ok {
		return nil, 0, errors.New("未知资源")
	}
	var all []model.Resource
	q := s.DB.Table(table)
	if f.Light {
		q = q.Omit("observed", "secret")
	}
	if f.ClusterID > 0 {
		if kind == "sources" {
			q = q.Where("id IN (?)", s.DB.Table(model.Tables["instances"]).Select("source_id").Where("cluster_id = ?", f.ClusterID))
		} else if kind == "certificates" {
			var bindings []model.Resource
			if e := s.DB.Table(model.Tables["certificate-bindings"]).Where("cluster_id = ?", f.ClusterID).Find(&bindings).Error; e != nil {
				return nil, 0, e
			}
			ids := []uint{0}
			for _, b := range bindings {
				p, _ := decode[model.BindingSpec](b.Spec)
				ids = append(ids, p.CertificateID)
			}
			q = q.Where("id IN ?", ids)
		} else if kind != "acme-accounts" && kind != "dns-accounts" && kind != "renewal-policies" {
			q = q.Where("cluster_id = ?", f.ClusterID)
		}
	}
	if f.GroupID > 0 {
		ids, e := s.GroupIDs(f.GroupID)
		if e != nil {
			return nil, 0, e
		}
		q = q.Where("group_id IN ?", ids)
	}
	if f.Name != "" {
		q = q.Where("name LIKE ?", "%"+f.Name+"%")
	}
	if f.Engine != "" {
		q = q.Where("engine = ?", f.Engine)
	}
	if f.Environment != "" {
		q = q.Where("environment = ?", f.Environment)
	}
	if e := q.Order("id DESC").Find(&all).Error; e != nil {
		return nil, 0, e
	}
	out := []model.Resource{}
	admin := s.Admin(u)
	for _, r := range all {
		if !admin && s.Authorize(u, kind, r, readAction(kind)) != nil {
			continue
		}
		if (kind == "sources" || kind == "instances" || kind == "clusters") && (r.CollectedAt == nil || time.Since(*r.CollectedAt) > 90*time.Second) {
			r.Status = "unknown"
		}
		if kind == "configurations" && (f.ConfigGroupID > 0 || f.FileGroup != "") {
			spec, err := decode[model.ConfigSpec](r.Spec)
			if err != nil {
				continue
			}
			if f.ConfigGroupID > 0 && spec.ConfigGroupID != f.ConfigGroupID {
				continue
			}
			if f.FileGroup != "" && !configHasFileGroup(spec, f.FileGroup) {
				continue
			}
		}
		if f.Status != "" && f.Status != r.Status {
			continue
		}
		out = append(out, r)
	}
	n := len(out)
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 || f.Size > 100 {
		f.Size = 20
	}
	start := (f.Page - 1) * f.Size
	if start > n {
		start = n
	}
	end := start + f.Size
	if end > n {
		end = n
	}
	return out[start:end], n, nil
}

// configHasFileGroup reports whether a draft still contains files of a group.
func configHasFileGroup(spec model.ConfigSpec, group string) bool {
	for _, f := range spec.Files {
		if !f.Deleted && f.Group == group {
			return true
		}
	}
	return false
}

// certificateExpiry exposes the current version dates on the list response so
// the page can warn before a certificate expires, without opening each record.
func (s *Service) certificateExpiry(r model.Resource) json.RawMessage {
	spec, e := decode[model.CertificateSpec](r.Spec)
	if e != nil || spec.CurrentVersionID == 0 {
		return JSON(map[string]any{})
	}
	var rev model.Revision
	if e = s.DB.Where("id = ? AND kind = ? AND resource_id = ?", spec.CurrentVersionID, "certificate", r.ID).First(&rev).Error; e != nil {
		return JSON(map[string]any{})
	}
	meta, e := decode[certContent](rev.Content)
	if e != nil {
		return JSON(map[string]any{})
	}
	return JSON(map[string]any{"notBefore": meta.NotBefore, "notAfter": meta.NotAfter, "renewAt": meta.RenewAt,
		"fingerprint": meta.Fingerprint, "domains": meta.Domains,
		"days": certificateDaysLeft(meta.NotAfter, time.Now()), "versionId": rev.ID})
}

func (s *Service) createAllowed(u uint, kind string, r model.Resource) error {
	if s.Admin(u) {
		return nil
	}
	if r.ClusterID > 0 {
		return s.Asset(u, "web_cluster", r.ClusterID, writeAction(kind))
	}
	typ := "web_cluster"
	if kind == "certificates" {
		typ = "web_certificate"
	}
	d := cmdbdao.NewCmdbAssetPermissionDao(s.DB)
	grants, e := d.EffectivePermissions(u)
	if e != nil {
		return e
	}
	for _, p := range grants {
		if !cmdbdao.RuleAllowsAction(p, typ, writeAction(kind)) {
			continue
		}
		var groups []uint
		_ = json.Unmarshal([]byte(p.HostGroupIDs), &groups)
		for _, g := range groups {
			ids, e := s.GroupIDs(g)
			if e != nil {
				return e
			}
			for _, id := range ids {
				if id == r.GroupID {
					return nil
				}
			}
		}
	}
	return ErrForbidden
}
func validPath(p string) bool {
	return strings.HasPrefix(p, "/") && path.Clean(p) == p && !strings.ContainsAny(p, "\x00\r\n") && len(p) < 1024
}

var objectName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.@-]{0,127}$`)

func (s *Service) validate(u uint, kind string, r *model.Resource) error {
	if strings.TrimSpace(r.Name) == "" || len([]rune(r.Name)) > 100 {
		return errors.New("名称不能为空且最多 100 字符")
	}
	if r.Engine == "" {
		r.Engine = "nginx"
	}
	if r.Engine != "nginx" {
		return errors.New("当前仅启用 Nginx 引擎")
	}
	var count int64
	if e := s.DB.Model(&cmdb.CmdbGroup{}).Where("id = ?", r.GroupID).Count(&count).Error; e != nil {
		return e
	}
	if count != 1 {
		return errors.New("请选择有效业务分组")
	}
	if r.ClusterID > 0 {
		c, e := s.Get("clusters", r.ClusterID)
		if e != nil {
			return e
		}
		if c.Engine != r.Engine || c.GroupID != r.GroupID {
			return errors.New("资源与集群的引擎和业务分组必须一致")
		}
	}
	switch kind {
	case "sources":
		if !s.Admin(u) {
			return ErrForbidden
		}
		p, e := decode[model.SourceSpec](r.Spec)
		if e != nil {
			return e
		}
		if p.Runtime != "host" && p.Runtime != "docker" {
			return errors.New("运行环境必须为 host 或 docker")
		}
		if p.Runtime == "docker" && !objectName.MatchString(p.Container) {
			return errors.New("容器名称无效")
		}
		if p.Binary == "" {
			p.Binary = "/usr/sbin/nginx"
		}
		if p.ConfigPath == "" {
			p.ConfigPath = "/etc/nginx/nginx.conf"
		}
		if p.Prefix == "" {
			p.Prefix = "/etc/nginx"
		}
		if !validPath(p.Binary) || !validPath(p.ConfigPath) || !validPath(p.Prefix) {
			return errors.New("配置、程序和前缀必须为规范绝对路径")
		}
		if p.Service != "" && !objectName.MatchString(p.Service) {
			return errors.New("服务名无效")
		}
		for _, v := range p.ManagedPaths {
			if !validPath(v) {
				return errors.New("受管路径无效")
			}
		}
		if p.HealthURL != "" {
			v, e := url.Parse(p.HealthURL)
			if e != nil || v.Hostname() == "" || (v.Scheme != "http" && v.Scheme != "https") || v.User != nil {
				return errors.New("健康检查地址无效")
			}
		}
		if p.ExpectedStatus == 0 {
			p.ExpectedStatus = 200
		}
		pfn, e := cmdbservice.GetAssetConnection(s.DB, "host", p.HostID, p.ConnectionID)
		if e != nil || pfn.Protocol != "ssh" || !pfn.Enabled {
			return errors.New("请选择已启用的主机 SSH 连接")
		}
		hash := digest([]byte(fmt.Sprintf("%d:%s:%s:%s", p.HostID, p.Runtime, p.Container, p.ConfigPath)))
		r.LocatorHash = &hash
		r.Spec = JSON(p)
	case "clusters":
		p, e := decode[model.ClusterSpec](r.Spec)
		if e != nil {
			return e
		}
		var previous model.ClusterSpec
		if r.ID > 0 {
			old, err := s.Get("clusters", r.ID)
			if err != nil {
				return err
			}
			previous, _ = decode[model.ClusterSpec](old.Spec)
		}
		if string(JSON(p.VIP)) != string(JSON(previous.VIP)) {
			return errors.New("VIP 配置只能通过受校验的操作计划变更")
		}

		for _, entry := range p.Entries {
			u, err := url.Parse(entry.URL)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || entry.ExpectedStatus < 100 || entry.ExpectedStatus > 599 {
				return errors.New("入口探测地址或状态码无效")
			}
		}
		r.Spec = JSON(p)
	case "sites":
		p, e := decode[model.SiteSpec](r.Spec)
		if e != nil {
			return e
		}

		if p.ExpectedStatus == 0 {
			p.ExpectedStatus = 200
		}
		if p.HealthURL != "" {
			v, err := url.Parse(p.HealthURL)
			if err != nil || (v.Scheme != "http" && v.Scheme != "https") || v.Host == "" || p.ExpectedStatus < 100 || p.ExpectedStatus > 599 {
				return errors.New("站点健康检查地址或状态码无效")
			}
		}
		if p.TLS && p.Mode != "raw" {
			var bindings []model.Resource
			if e = s.DB.Table(model.Tables["certificate-bindings"]).Where("cluster_id = ?", r.ClusterID).Find(&bindings).Error; e != nil {
				return e
			}
			found := false
			for _, b := range bindings {
				bs, err := decode[model.BindingSpec](b.Spec)
				if err == nil && bs.CertificateID == p.CertificateID {
					p.CertificatePath, p.KeyPath = bs.CertificatePath, bs.KeyPath
					found = true
					break
				}
			}
			if !found {
				return errors.New("请先为集群创建证书部署绑定")
			}
		}
		if _, e = RenderSite(p); e != nil {
			return e
		}
		if p.CertificateID > 0 {
			if e = s.Asset(u, "web_certificate", p.CertificateID, "web_certificate_use"); e != nil {
				return e
			}
		}
		r.Spec = JSON(p)
	case "upstreams":
		p, e := decode[UpstreamSpec](r.Spec)
		if e != nil {
			return e
		}
		if _, e = RenderUpstream(p); e != nil {
			return e
		}
		r.Spec = JSON(p)
	case "configurations":
		p, e := decode[model.ConfigSpec](r.Spec)
		if e != nil {
			return e
		}
		if len(p.Files) == 0 || len(p.Files) > 200 {
			return errors.New("配置文件数量必须在 1–200 之间")
		}
		seen := map[string]bool{}
		size := 0
		for _, f := range p.Files {
			if strings.Contains(f.Content, "PRIVATE KEY-----") {
				return errors.New("私钥请通过证书管理导入")
			}
			if !validPath(f.Path) || seen[f.Path] || strings.ContainsRune(f.Content, 0) {
				return errors.New("配置路径重复或无效")
			}
			if len([]rune(f.Group)) > 64 {
				return errors.New("配置分组名称最多 64 字符")
			}
			seen[f.Path] = true
			size += len(f.Content)
		}
		if size > 4<<20 {
			return errors.New("配置内容不能超过 4 MiB")
		}
		if p.ConfigGroupID > 0 {
			g, err := s.Get("config-groups", p.ConfigGroupID)
			if err != nil {
				return errors.New("请选择有效的配置分组")
			}
			if g.ClusterID != r.ClusterID {
				return errors.New("配置分组必须属于同一集群")
			}
			if e = s.Authorize(u, "config-groups", g, "web_config_edit"); e != nil {
				return e
			}
		}
		// Node snapshots and certificate version maps belong to immutable
		// publication revisions; clients cannot inject them into a draft.
		p.Nodes, p.Certificates = nil, nil
		r.Spec = JSON(p)
	case "config-groups":
		p, e := decode[model.ConfigGroupSpec](r.Spec)
		if e != nil {
			return e
		}
		if len([]rune(p.Description)) > 500 {
			return errors.New("配置分组说明最多 500 字符")
		}
		if p.Color != "" && !regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`).MatchString(p.Color) {
			return errors.New("配置分组颜色必须为 #RRGGBB")
		}
		var duplicate int64
		q := s.DB.Table(model.Tables["config-groups"]).Where("cluster_id = ? AND name = ?", r.ClusterID, r.Name)
		if r.ID > 0 {
			q = q.Where("id <> ?", r.ID)
		}
		if e = q.Count(&duplicate).Error; e != nil {
			return e
		}
		if duplicate > 0 {
			return errors.New("同一集群内配置分组名称重复")
		}
		if e = s.validateConfigGroupPaths(u, *r, p); e != nil {
			return e
		}
		r.Spec = JSON(p)
	case "certificates":
		p, e := decode[model.CertificateSpec](r.Spec)
		if e != nil {
			return e
		}
		if p.Owner == "" {
			p.Owner = "platform"
		}
		if p.Owner != "platform" && p.Owner != "external" {
			return errors.New("证书管理归属无效")
		}
		if len(p.Domains) == 0 {
			return errors.New("请输入证书域名")
		}
		for _, d := range p.Domains {
			if !domainPattern.MatchString(d) && net.ParseIP(d) == nil {
				return errors.New("证书域名无效")
			}
			if strings.HasPrefix(d, "*.") && p.Challenge == "http-01" {
				return errors.New("泛域名需要 DNS 验证")
			}
		}
		// Binding a signing identity is an administrator operation, independent of certificate use.
		if !s.Admin(u) {
			var previous model.CertificateSpec
			if r.ID > 0 {
				old, err := s.Get(kind, r.ID)
				if err != nil {
					return err
				}
				previous, _ = decode[model.CertificateSpec](old.Spec)
			}
			if p.ACMEAccountID != previous.ACMEAccountID || p.DNSAccountID != previous.DNSAccountID {
				return errors.New("签发账户绑定需要管理员配置")
			}
			if p.ACMEAccountID != 0 && string(JSON(p.Domains)) != string(JSON(previous.Domains)) {
				return errors.New("签发域名范围变更需要管理员配置")
			}
		}
		if r.ID > 0 {
			old, e := s.Get(kind, r.ID)
			if e != nil {
				return e
			}
			v, _ := decode[model.CertificateSpec](old.Spec)
			p.CurrentVersionID = v.CurrentVersionID
		} else {
			p.CurrentVersionID = 0
		}
		r.Spec = JSON(p)
	case "dns-accounts", "acme-accounts":
		if !s.Admin(u) {
			return ErrForbidden
		}
		p, e := decode[model.AccountSpec](r.Spec)
		if e != nil {
			return e
		}
		if kind == "dns-accounts" && p.Provider != "alidns" && p.Provider != "tencentcloud" && p.Provider != "cloudflare" {
			return errors.New("DNS 服务商无效")
		}
		if kind == "acme-accounts" {
			v, e := url.Parse(p.DirectoryURL)
			if e != nil || v.Scheme != "https" || v.Host == "" || p.Email == "" {
				return errors.New("请填写 HTTPS ACME 地址和邮箱")
			}
			p.Registration = nil
			if r.ID > 0 {
				old, e := s.Get(kind, r.ID)
				if e != nil {
					return e
				}
				a, _ := decode[model.AccountSpec](old.Spec)
				p.Registration = a.Registration
				if p.DirectoryURL != a.DirectoryURL {
					return errors.New("已有 ACME 账户不能更换 CA，请新建账户")
				}
			}
		}
		r.Spec = JSON(p)
	case "certificate-bindings":
		p, e := decode[model.BindingSpec](r.Spec)
		if e != nil {
			return e
		}
		if !validPath(p.CertificatePath) || !validPath(p.KeyPath) || p.KeyPath == p.CertificatePath {
			return errors.New("证书部署路径无效")
		}
		if e = s.Asset(u, "web_certificate", p.CertificateID, "web_certificate_use"); e != nil {
			return e
		}
		if p.VerifyHost == "" {
			p.VerifyHost = "127.0.0.1"
		}
		if p.VerifyPort == 0 {
			p.VerifyPort = 443
		}
		if p.VerifyPort < 1 || p.VerifyPort > 65535 || strings.ContainsAny(p.VerifyHost, " /\\\t\r\n") || !domainPattern.MatchString(p.ServerName) || strings.HasPrefix(p.ServerName, "*.") {
			return errors.New("请配置节点 TLS 核验地址、端口及具体域名")
		}
		r.Spec = JSON(p)
	case "renewal-policies":
		p, e := decode[model.RenewalSpec](r.Spec)
		if e != nil {
			return e
		}
		if e = s.Asset(u, "web_certificate", p.CertificateID, "web_certificate_manage"); e != nil {
			return e
		}
		if p.AutoDeploy && len(p.BindingIDs) == 0 {
			return errors.New("自动部署必须明确选择证书部署绑定")
		}
		for _, bid := range p.BindingIDs {
			b, err := s.Get("certificate-bindings", bid)
			if err != nil {
				return err
			}
			bs, err := decode[model.BindingSpec](b.Spec)
			if err != nil {
				return err
			}
			if bs.CertificateID != p.CertificateID || bs.VersionID != 0 {
				return errors.New("自动部署绑定必须关联同一证书并跟随最新版本")
			}
			if err = s.Authorize(u, "certificate-bindings", b, "web_publish"); err != nil {
				return err
			}
		}
		if p.NextRunAt.IsZero() {
			p.NextRunAt = time.Now().UTC()
		}
		r.Spec = JSON(p)
	default:
		return errors.New("该资源不支持编辑")
	}
	return nil
}
func (s *Service) Save(ctx context.Context, u uint, kind string, r model.Resource, secret string) (model.Resource, error) {
	if e := s.Module(ctx, u, "write"); e != nil {
		return r, e
	}
	if kind == "renewal-policies" {
		if e := s.Module(ctx, u, "execute"); e != nil {
			return r, e
		}
	}
	old := model.Resource{}
	var e error
	if r.ID > 0 {
		old, e = s.Get(kind, r.ID)
		if e != nil {
			return r, e
		}
		if e = s.Authorize(u, kind, old, writeAction(kind)); e != nil {
			return r, e
		}
		if r.GroupID != old.GroupID || r.ClusterID != old.ClusterID {
			return r, errors.New("已有资源不能直接更换分组或集群")
		}
		if r.Revision != old.Revision {
			return r, ErrConflict
		}
	} else if e = s.createAllowed(u, kind, r); e != nil {
		return r, e
	}
	if e = s.validate(u, kind, &r); e != nil {
		return r, e
	}
	if kind == "configurations" {
		candidate, _ := decode[model.ConfigSpec](r.Spec)
		checks, err := s.ValidateClusterConfig(ctx, u, r.ClusterID, candidate.Files, candidate)
		if err != nil {
			return r, err
		}
		for _, check := range checks {
			if !check.Valid {
				return r, fmt.Errorf("%s: %s", check.Name, check.Message)
			}
			if candidate.Baselines == nil {
				candidate.Baselines = make(map[uint]string)
			}
			candidate.Baselines[check.InstanceID] = check.Digest
		}
		r.Spec = JSON(candidate)
	}
	if kind == "renewal-policies" {
		p, _ := decode[model.RenewalSpec](r.Spec)
		previous, _ := decode[model.RenewalSpec](old.Spec)
		p.APIKeyID = apikey.ID(ctx)
		p.PendingOperationID = previous.PendingOperationID
		p.DeployedVersionID = previous.DeployedVersionID
		p.DeployOperationIDs = previous.DeployOperationIDs
		r.Spec = JSON(p)
	}
	r.Secret = old.Secret
	dnsChanged := false
	if kind == "dns-accounts" {
		dnsChanged, e = s.prepareDNSAccount(&r, old, secret)
		if e != nil {
			return r, e
		}
	} else if secret != "" {
		return r, ErrForbidden
	}
	r.Status = old.Status
	if kind == "sites" || kind == "upstreams" || kind == "configurations" {
		r.Status = "draft"
	}
	r.Management = old.Management
	r.Observed = old.Observed
	r.CollectedAt = old.CollectedAt
	if dnsChanged {
		r.Status = "configured"
		r.Observed = nil
		r.CollectedAt = nil
	}
	r.CreatedBy = old.CreatedBy
	if kind == "renewal-policies" {
		r.CreatedBy = u
	}
	r.SourceID = old.SourceID
	if r.ID == 0 {
		r.Status = "unknown"
		if kind == "sites" || kind == "upstreams" || kind == "configurations" {
			r.Status = "draft"
		}
		if kind == "acme-accounts" || kind == "dns-accounts" || kind == "renewal-policies" {
			r.Status = "configured"
		}
		r.CreatedAt = time.Time{}
		r.Management = "observe"
		r.Revision = 1
		r.CreatedBy = u
	} else {
		r.Revision++
	}
	e = s.DB.Transaction(func(tx *gorm.DB) error {
		// Group membership uses a shared cluster row lock so two concurrent
		// saves cannot assign the same native path to different groups.
		if kind == "config-groups" || kind == "configurations" {
			var cluster model.Resource
			if err := tx.Table(model.Tables["clusters"]).Clauses(clause.Locking{Strength: "UPDATE"}).First(&cluster, r.ClusterID).Error; err != nil {
				return err
			}
			local := *s
			local.DB = tx
			if kind == "config-groups" {
				if err := local.validate(u, kind, &r); err != nil {
					return err
				}
			} else {
				candidate, _ := decode[model.ConfigSpec](r.Spec)
				if candidate.ConfigGroupID > 0 {
					group, err := local.Get("config-groups", candidate.ConfigGroupID)
					if err != nil {
						return err
					}
					if err = local.Authorize(u, "config-groups", group, "web_config_edit"); err != nil {
						return err
					}
					groupSpec, err := decode[model.ConfigGroupSpec](group.Spec)
					if err != nil {
						return err
					}
					seen := map[string]bool{}
					for _, p := range groupSpec.Paths {
						seen[p] = true
					}
					for _, file := range candidate.Files {
						if !file.Deleted && !seen[file.Path] {
							groupSpec.Paths = append(groupSpec.Paths, file.Path)
							seen[file.Path] = true
						}
					}
					if err = local.validateConfigGroupPaths(u, group, groupSpec); err != nil {
						return err
					}
					updated := tx.Table(model.Tables["config-groups"]).Where("id = ? AND revision = ?", group.ID, group.Revision).Updates(map[string]any{"spec": JSON(groupSpec), "revision": gorm.Expr("revision + 1")})
					if updated.Error != nil {
						return updated.Error
					}
					if updated.RowsAffected != 1 {
						return ErrConflict
					}
				}
			}
		}
		if r.ID > 0 {
			var active int64
			key := fmt.Sprintf("%s:%d", kind, r.ID)
			if kind == "clusters" {
				key = fmt.Sprintf("cluster:%d", r.ID)
			}
			if err := tx.Model(&model.Lock{}).Where("`key` = ?", key).Count(&active).Error; err != nil {
				return err
			}
			if active > 0 {
				return ErrConflict
			}
		}
		if kind == "sources" && r.ID > 0 {
			// Discovery takes the same source lock. An unregistered source may be
			// corrected, but a discovered instance must keep its historical identity.
			var source model.Resource
			if err := tx.Table(model.Tables["sources"]).Clauses(clause.Locking{Strength: "UPDATE"}).First(&source, r.ID).Error; err != nil {
				return err
			}
			var member model.Resource
			if err := tx.Table(model.Tables["instances"]).Where("source_id = ?", r.ID).First(&member).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if member.ID > 0 && old.LocatorHash != nil && r.LocatorHash != nil && *old.LocatorHash != *r.LocatorHash {
				return errors.New("已登记实例的定位不能修改，请登记新的数据源")
			}
			var active int64
			if err := tx.Model(&model.Lock{}).Where("`key` IN ?", []string{fmt.Sprintf("sources:%d", r.ID), fmt.Sprintf("cluster:%d", member.ClusterID)}).Count(&active).Error; err != nil {
				return err
			}
			if active > 0 {
				return ErrConflict
			}
		}
		if r.ClusterID > 0 || kind == "clusters" {
			id := r.ClusterID
			if kind == "clusters" {
				id = r.ID
			}
			var locks int64
			if e := tx.Model(&model.Lock{}).Where("`key` = ?", fmt.Sprintf("cluster:%d", id)).Count(&locks).Error; e != nil {
				return e
			}
			if locks > 0 {
				return ErrConflict
			}
		}
		if kind == "configurations" {
			config, _ := decode[model.ConfigSpec](r.Spec)
			var sites []model.Resource
			if e := tx.Table(model.Tables["sites"]).Where("cluster_id = ?", r.ClusterID).Find(&sites).Error; e != nil {
				return e
			}
			for _, site := range sites {
				spec, err := decode[model.SiteSpec](site.Spec)
				if err != nil {
					return err
				}
				if spec.Mode == "raw" {
					continue
				}
				rendered, err := RenderSite(spec)
				if err != nil {
					return err
				}
				for _, file := range config.Files {
					if file.Path != spec.Path || (!file.Deleted && file.Content == rendered) {
						continue
					}
					spec.Mode = "raw"
					spec.Raw = file.Content
					if file.Deleted {
						spec.Raw = "# Removed by configuration publication\n"
					}
					res := tx.Table(model.Tables["sites"]).Where("id = ? AND revision = ?", site.ID, site.Revision).Updates(map[string]any{"spec": JSON(spec), "revision": gorm.Expr("revision + 1")})
					if res.Error != nil {
						return res.Error
					}
					if res.RowsAffected != 1 {
						return ErrConflict
					}
				}
			}
		}
		if r.ID == 0 {
			return tx.Table(model.Tables[kind]).Create(&r).Error
		}
		columns := []string{"name", "spec", "secret", "revision", "updated_at", "created_by", "environment", "status", "locator_hash"}
		if kind == "dns-accounts" {
			columns = append(columns, "observed", "collected_at")
		}
		result := tx.Table(model.Tables[kind]).Where("id = ? AND revision = ?", r.ID, old.Revision).Select(columns).Updates(&r)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrConflict
		}
		return nil
	})
	return r, e
}
func (s *Service) Delete(ctx context.Context, u uint, kind string, id uint) error {
	if e := s.Module(ctx, u, "write"); e != nil {
		return e
	}
	r, e := s.Get(kind, id)
	if e != nil {
		return e
	}
	if e = s.Authorize(u, kind, r, writeAction(kind)); e != nil {
		return e
	}
	if kind == "instances" {
		return errors.New("请通过集群成员操作移出实例")
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var n int64
		cid := r.ClusterID
		if kind == "clusters" {
			cid = r.ID
		}
		if e := tx.Model(&model.Operation{}).Where("status IN ? AND ((target_type = ? AND target_id = ?) OR (cluster_id = ? AND cluster_id <> 0))", []string{"queued", "preflight", "running", "verifying", "unknown"}, kind, id, cid).Count(&n).Error; e != nil {
			return e
		}
		if n > 0 {
			return ErrConflict
		}
		if kind == "clusters" || kind == "sources" {
			column := "cluster_id"
			if kind == "sources" {
				column = "source_id"
			}
			if e := tx.Table(model.Tables["instances"]).Where(column+" = ?", id).Count(&n).Error; e != nil {
				return e
			}
			if n > 0 {
				return errors.New("请先移出关联实例")
			}
		}
		if kind == "certificates" || kind == "dns-accounts" || kind == "acme-accounts" {
			return errors.New("证书及签发账户保留历史引用，请停用相关续期策略")
		}
		if kind == "config-groups" {
			var drafts []model.Resource
			if e := tx.Table(model.Tables["configurations"]).Where("cluster_id = ?", r.ClusterID).Find(&drafts).Error; e != nil {
				return e
			}
			for _, draft := range drafts {
				spec, err := decode[model.ConfigSpec](draft.Spec)
				if err != nil {
					return err
				}
				if spec.ConfigGroupID == id {
					return errors.New("仍有配置草稿引用该分组，请先调整草稿分组")
				}
			}
		}
		return tx.Table(model.Tables[kind]).Where("id = ? AND revision = ?", id, r.Revision).Delete(&model.Resource{}).Error
	})
}
func (s *Service) Members(u, cid uint) ([]model.Resource, error) {
	if e := s.Asset(u, "web_cluster", cid, "web_view"); e != nil {
		return nil, e
	}
	var rows []model.Resource
	e := s.DB.Table(model.Tables["instances"]).Where("cluster_id = ?", cid).Order("id").Find(&rows).Error
	return rows, e
}
func ReadAction(kind string) string { return readAction(kind) }
func (s *Service) Redact(u uint, kind string, r *model.Resource) {
	r.Connection = nil
	if kind == "certificates" {
		r.Observed = s.certificateExpiry(*r)
	}
	if (kind == "sources" || kind == "instances" || kind == "clusters") && (r.CollectedAt == nil || time.Since(*r.CollectedAt) > 90*time.Second) {
		r.Status = "unknown"
	}
	if kind == "sources" || kind == "instances" {
		var spec model.SourceSpec
		if json.Unmarshal(r.Spec, &spec) == nil && spec.ConnectionID > 0 {
			var view model.ConnectionView
			q := s.DB.Table("cmdb_asset_connection c").Select("c.asset_id AS host_id, h.host_name, c.address, c.port, c.username, c.name AS alias").Joins("JOIN cmdb_host h ON h.id = c.asset_id").Where("c.id = ? AND c.asset_type = ? AND c.asset_id = ?", spec.ConnectionID, "host", spec.HostID).Scan(&view)
			if q.Error == nil && q.RowsAffected == 1 {
				r.Connection = &view
			}
		}
		var ob Observation
		if json.Unmarshal(r.Observed, &ob) == nil {
			if s.Authorize(u, kind, *r, "web_config_view") != nil {
				ob.Files = nil
			}
			if s.Authorize(u, kind, *r, "web_logs") != nil {
				for _, metric := range []string{"response_bytes_per_second", "request_bytes_per_second", "error_rate", "average_latency_ms"} {
					delete(ob.Metrics, metric)
				}
			}
			r.Observed = JSON(ob)
		}
	}
}
