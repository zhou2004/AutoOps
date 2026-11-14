package controller

import (
	cmdbdao "dodevops-api/api/cmdb/dao"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/webservice/model"
	"dodevops-api/api/webservice/service"
	"dodevops-api/common"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type Controller struct{ S *service.Service }

func New(db *gorm.DB) *Controller { return &Controller{S: service.New(db)} }
func user(c *gin.Context) (uint, bool) {
	u, e := jwt.GetAdminId(c)
	if e != nil {
		result.Failed(c, 401, "请先登录")
		return 0, false
	}
	return u, true
}
func id(c *gin.Context) uint { n, _ := strconv.ParseUint(c.Param("id"), 10, 32); return uint(n) }
func queryID(c *gin.Context, k string) uint {
	n, _ := strconv.ParseUint(c.Query(k), 10, 32)
	return uint(n)
}
func fail(c *gin.Context, e error) {
	code := 400
	if errors.Is(e, service.ErrForbidden) {
		code = 403
	}
	if errors.Is(e, service.ErrConflict) {
		code = 409
	}
	if errors.Is(e, gorm.ErrRecordNotFound) {
		code = 404
	}
	result.Failed(c, code, e.Error())
}
func bind(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 6<<20)
	if e := c.ShouldBindJSON(v); e != nil {
		result.Failed(c, 400, "请求内容无效或超过大小限制")
		return false
	}
	return true
}
func (c *Controller) List(kind string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		u, ok := user(ctx)
		if !ok {
			return
		}
		page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
		size, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))
		rows, n, e := c.S.List(u, kind, service.Filter{Light: ctx.Query("view") == "options", Name: ctx.Query("name"), Engine: ctx.Query("engine"), Environment: ctx.Query("environment"), Status: ctx.Query("status"), GroupID: queryID(ctx, "groupId"), ClusterID: queryID(ctx, "clusterId"), ConfigGroupID: queryID(ctx, "configGroupId"), FileGroup: ctx.Query("fileGroup"), Page: page, Size: size})
		if e != nil {
			fail(ctx, e)
			return
		}
		for i := range rows {
			c.S.Redact(u, kind, &rows[i])
			if ctx.Query("view") == "options" {
				rows[i].Observed = nil
				if kind == "configurations" {
					rows[i].Spec = service.JSON(map[string]any{})
				}
			}
		}
		result.Success(ctx, gin.H{"list": rows, "total": n})
	}
}
func (c *Controller) Detail(kind string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		u, ok := user(ctx)
		if !ok {
			return
		}
		r, e := c.S.Get(kind, id(ctx))
		if e == nil {
			e = c.S.Authorize(u, kind, r, service.ReadAction(kind))
		}
		if e != nil {
			fail(ctx, e)
			return
		}
		c.S.Redact(u, kind, &r)
		result.Success(ctx, r)
	}
}
func (c *Controller) Save(kind string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		u, ok := user(ctx)
		if !ok {
			return
		}
		var body struct {
			model.Resource
			SecretInput   string                        `json:"secretInput"`
			MemberChanges *service.ClusterMemberChanges `json:"memberChanges"`
		}
		if !bind(ctx, &body) {
			return
		}
		if ctx.Request.Method != "POST" {
			body.ID = id(ctx)
		} else {
			body.ID = 0
		}
		var r model.Resource
		var e error
		if kind == "clusters" {
			r, e = c.S.SaveCluster(ctx.Request.Context(), u, body.Resource, body.MemberChanges)
		} else if body.MemberChanges != nil {
			e = errors.New("仅集群支持成员变更")
		} else {
			r, e = c.S.Save(ctx.Request.Context(), u, kind, body.Resource, body.SecretInput)
		}
		if e != nil {
			fail(ctx, e)
			return
		}
		c.S.Redact(u, kind, &r)
		result.Success(ctx, r)
	}
}
func (c *Controller) Delete(kind string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		u, ok := user(ctx)
		if !ok {
			return
		}
		if e := c.S.Delete(ctx.Request.Context(), u, kind, id(ctx)); e != nil {
			fail(ctx, e)
			return
		}
		result.Success(ctx, true)
	}
}
func (c *Controller) Discover(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	r, e := c.S.Discover(ctx.Request.Context(), u, id(ctx))
	if e != nil {
		fail(ctx, e)
		return
	}
	c.S.Redact(u, "sources", &r)
	result.Success(ctx, r)
}
func (c *Controller) Members(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	if ctx.Request.Method == "GET" {
		rows, e := c.S.Members(u, id(ctx))
		if e != nil {
			fail(ctx, e)
			return
		}
		for i := range rows {
			c.S.Redact(u, "instances", &rows[i])
		}
		result.Success(ctx, rows)
		return
	}
	var p struct {
		InstanceID uint `json:"instanceId"`
		Remove     bool `json:"remove"`
	}
	if !bind(ctx, &p) {
		return
	}
	if e := c.S.BindMember(ctx.Request.Context(), u, id(ctx), p.InstanceID, p.Remove); e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, true)
}
func (c *Controller) Capabilities(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	if e := c.S.Asset(u, "web_cluster", id(ctx), "web_view"); e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, c.S.Capabilities(u, id(ctx)))
}
func (c *Controller) Plan(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var p service.PlanInput
	if !bind(ctx, &p) {
		return
	}
	plan, e := c.S.Plan(ctx.Request.Context(), u, p)
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, plan)
}
func (c *Controller) Submit(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var p struct {
		PlanID         uint   `json:"planId"`
		IdempotencyKey string `json:"idempotencyKey"`
		Confirmed      bool   `json:"confirmed"`
	}
	if !bind(ctx, &p) {
		return
	}
	op, e := c.S.Submit(ctx.Request.Context(), u, p.PlanID, p.IdempotencyKey, p.Confirmed)
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, op)
}
func (c *Controller) Operation(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	op, steps, e := c.S.Operation(u, id(ctx))
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, gin.H{"operation": op, "steps": steps})
}
func (c *Controller) Cancel(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	op, _, e := c.S.Operation(u, id(ctx))
	if e != nil {
		fail(ctx, e)
		return
	}
	r, e := c.S.Get(op.TargetType, op.TargetID)
	if e == nil {
		e = c.S.Authorize(u, op.TargetType, r, "web_manage")
	}
	if e != nil {
		fail(ctx, e)
		return
	}
	e = c.S.DB.Model(&op).Where("status IN ?", []string{"queued", "preflight", "running"}).Update("cancel_requested", true).Error
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, true)
}
func (c *Controller) OperationEvents(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	if _, _, e := c.S.Operation(u, id(ctx)); e != nil {
		fail(ctx, e)
		return
	}
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("X-Accel-Buffering", "no")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var wakeups <-chan *redis.Message
	if client := common.GetRedisClient(); client != nil {
		subscription := client.Subscribe(ctx.Request.Context(), service.OperationChannel(id(ctx)))
		defer subscription.Close()
		wakeups = subscription.Channel()
	}

	for {
		if c.S.Module(ctx.Request.Context(), u, "read") != nil {
			return
		}
		op, steps, e := c.S.Operation(u, id(ctx))
		if e != nil {
			return
		}
		b, _ := json.Marshal(gin.H{"operation": op, "steps": steps})
		fmt.Fprintf(ctx.Writer, "data: %s\n\n", b)
		ctx.Writer.Flush()
		if op.Status == "succeeded" || op.Status == "failed" || op.Status == "rolled-back" || op.Status == "unknown" {
			return
		}
		select {
		case <-ctx.Request.Context().Done():
			return
		case <-ticker.C:
		case <-wakeups:
		}
	}
}
func (c *Controller) Audit(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))
	rows, total, e := c.S.Audit(u, service.Filter{Name: ctx.Query("name"), Engine: ctx.Query("engine"), Environment: ctx.Query("environment"), Status: ctx.Query("status"), GroupID: queryID(ctx, "groupId"), ClusterID: queryID(ctx, "clusterId"), Page: page, Size: size})
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, gin.H{"list": rows, "total": total})
}
func (c *Controller) Groups(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var groups []cmdb.CmdbGroup
	if e := c.S.DB.Find(&groups).Error; e != nil {
		fail(ctx, e)
		return
	}
	if !c.S.Admin(u) {
		keep := map[uint]bool{}
		grants, err := cmdbdao.NewCmdbAssetPermissionDao(c.S.DB).EffectivePermissions(u)
		if err != nil {
			fail(ctx, err)
			return
		}
		for _, grant := range grants {
			allowed := false
			for _, kind := range []string{"web_source", "web_cluster", "web_certificate"} {
				for _, action := range []string{"web_view", "web_manage", "web_certificate_manage"} {
					allowed = allowed || cmdbdao.RuleAllowsAction(grant, kind, action)
				}
			}
			if !allowed {
				continue
			}
			var ids []uint
			json.Unmarshal([]byte(grant.HostGroupIDs), &ids)
			for _, id := range ids {
				descendants, e := c.S.GroupIDs(id)
				if e != nil {
					fail(ctx, e)
					return
				}
				for _, child := range descendants {
					keep[child] = true
				}
			}
		}

		for _, kind := range []string{"sources", "clusters", "certificates"} {
			rows, _, _ := c.S.List(u, kind, service.Filter{Size: 100})
			for _, r := range rows {
				keep[r.GroupID] = true
			}
		}
		for changed := true; changed; {
			changed = false
			for _, g := range groups {
				if keep[g.ID] && g.ParentID > 0 && !keep[g.ParentID] {
					keep[g.ParentID] = true
					changed = true
				}
			}
		}
		out := []cmdb.CmdbGroup{}
		for _, g := range groups {
			if keep[g.ID] {
				out = append(out, g)
			}
		}
		groups = out
	}
	result.Success(ctx, groups)
}
func (c *Controller) Targets(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	if !c.S.Admin(u) {
		fail(ctx, service.ErrForbidden)
		return
	}
	var hosts []cmdb.CmdbHost
	var connections []cmdb.CmdbAssetConnection
	c.S.DB.Select("id,host_name,name,private_ip,group_id").Find(&hosts)
	if e := c.S.DB.Where("asset_type = ? AND protocol = ? AND enabled = ?", "host", "ssh", true).Find(&connections).Error; e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, gin.H{"hosts": hosts, "connections": connections})
}
func (c *Controller) ImportCertificate(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	u, ok := user(ctx)
	if !ok {
		return
	}
	var p struct {
		Certificate string `json:"certificate"`
		PrivateKey  string `json:"privateKey"`
	}
	if !bind(ctx, &p) {
		return
	}
	v, e := c.S.ImportCertificate(ctx.Request.Context(), u, id(ctx), p.Certificate, p.PrivateKey)
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, v)
}

func (c *Controller) CertificateVersions(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	if e := c.S.Module(ctx.Request.Context(), u, "read"); e != nil {
		fail(ctx, e)
		return
	}
	rows, current, e := c.S.CertificateVersions(u, id(ctx))
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, gin.H{"list": rows, "currentVersionId": current})
}

func (c *Controller) ExportCertificate(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
	u, ok := user(ctx)
	if !ok {
		return
	}
	version, err := strconv.ParseUint(ctx.Param("versionId"), 10, 32)
	if err != nil {
		result.Failed(ctx, 400, "证书版本标识无效")
		return
	}
	includeKey := ctx.Query("key") == "true" || ctx.Query("key") == "1"
	data, e := c.S.ExportCertificate(ctx.Request.Context(), u, id(ctx), uint(version), includeKey)
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, data)
}

func (c *Controller) InstanceConfig(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	value, e := c.S.InstanceConfigContext(ctx.Request.Context(), u, id(ctx))
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, value)
}

// ValidateInstanceConfig runs `nginx -t` on the candidate files synchronously so
// the page can show the exact syntax error before anything is saved or published.
func (c *Controller) ValidateInstanceConfig(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var p struct {
		Files []model.File `json:"files"`
	}
	if !bind(ctx, &p) {
		return
	}
	v, e := c.S.ValidateInstanceConfig(ctx.Request.Context(), u, id(ctx), p.Files)
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, v)
}

func (c *Controller) ValidateClusterConfig(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var p model.ConfigSpec
	if !bind(ctx, &p) {
		return
	}
	rows, e := c.S.ValidateClusterConfig(ctx.Request.Context(), u, id(ctx), p.Files, p)
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, rows)
}
func (c *Controller) Revisions(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	cid := queryID(ctx, "clusterId")
	cert := queryID(ctx, "certificateId")
	q := c.S.DB.Model(&model.Revision{})
	if cert > 0 {
		if e := c.S.Asset(u, "web_certificate", cert, "web_view"); e != nil {
			fail(ctx, e)
			return
		}
		q = q.Where("kind = ? AND resource_id = ?", "certificate", cert)
	} else {
		if e := c.S.Asset(u, "web_cluster", cid, "web_config_view"); e != nil {
			fail(ctx, e)
			return
		}
		q = q.Where("kind = ? AND cluster_id = ?", "configuration", cid)
	}
	var rows []model.Revision
	if e := q.Order("id DESC").Limit(100).Find(&rows).Error; e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, rows)
}
func (c *Controller) Render(kind string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		u, ok := user(ctx)
		if !ok {
			return
		}
		r, e := c.S.Get(kind, id(ctx))
		if e == nil {
			e = c.S.Authorize(u, kind, r, "web_config_edit")
		}
		if e != nil {
			fail(ctx, e)
			return
		}
		var content, path string
		if kind == "sites" {
			var p model.SiteSpec
			e = json.Unmarshal(r.Spec, &p)
			if e == nil {
				content, e = service.RenderSite(p)
			}
			path = p.Path
		} else {
			var p service.UpstreamSpec
			e = json.Unmarshal(r.Spec, &p)
			if e == nil {
				content, e = service.RenderUpstream(p)
			}
			path = p.Path
		}
		if e != nil {
			fail(ctx, e)
			return
		}
		result.Success(ctx, model.ConfigSpec{Files: []model.File{{Path: path, Content: content, Mode: 0644}}})
	}
}
func (c *Controller) Logs(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	i, e := c.S.Get("instances", id(ctx))
	if e == nil {
		e = c.S.Authorize(u, "instances", i, "web_logs")
	}
	if e != nil {
		fail(ctx, e)
		return
	}
	if ctx.Query("logId") == "" {
		var ob service.Observation
		json.Unmarshal(i.Observed, &ob)
		result.Success(ctx, ob.Logs)
		return
	}
	src, e := c.S.Get("sources", i.SourceID)
	if e != nil {
		fail(ctx, e)
		return
	}
	var spec model.SourceSpec
	json.Unmarshal(src.Spec, &spec)
	lines, _ := strconv.Atoi(ctx.DefaultQuery("lines", "500"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	var observed service.Observation
	json.Unmarshal(i.Observed, &observed)
	data, e := c.S.Remote.Run(ctx.Request.Context(), service.RemoteRequest{Action: "logs", Spec: spec, Logs: observed.Logs, Offset: offset, From: ctx.Query("from"), To: ctx.Query("to"), LogID: ctx.Query("logId"), Keyword: ctx.Query("keyword"), Rotation: ctx.Query("rotation"), Lines: lines, Status: ctx.Query("status"), IP: ctx.Query("ip"), URI: ctx.Query("uri")})
	if e != nil {
		fail(ctx, e)
		return
	}
	c.S.DB.Create(&model.Event{UserID: u, ClusterID: i.ClusterID, SourceID: i.SourceID, Kind: "log-query", Message: fmt.Sprintf("instance=%d log=%s rotation=%s lines=%d download=%s", i.ID, ctx.Query("logId"), ctx.Query("rotation"), lines, ctx.Query("download"))})
	if ctx.Query("download") == "true" || ctx.Query("download") == "json" {
		var d struct {
			Text string `json:"text"`
		}
		json.Unmarshal(data, &d)
		if ctx.Query("download") == "json" {
			result.Success(ctx, data)
			return
		}
		ctx.Header("Content-Disposition", "attachment; filename=web-log.txt")
		ctx.Data(200, "text/plain; charset=utf-8", []byte(d.Text))
		return
	}
	result.Success(ctx, data)
}
