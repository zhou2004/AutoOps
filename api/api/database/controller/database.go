package controller

import (
	cmdbmodel "dodevops-api/api/cmdb/model"
	"errors"
	"strconv"

	"dodevops-api/api/database/model"
	"dodevops-api/api/database/policy"
	"dodevops-api/api/database/service"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (ctrl *Controller) Groups(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	var groups []cmdbmodel.CmdbGroup
	if e := ctrl.s.DB.Select("id,parent_id,name").Find(&groups).Error; e != nil {
		fail(c, e)
		return
	}
	if !ctrl.s.IsAdmin(u) {
		scope, e := ctrl.s.Scope(u, "db_cluster_view")
		if e != nil {
			fail(c, e)
			return
		}
		ids := []uint{}
		for id := range scope {
			ids = append(ids, id)
		}
		var groupIDs []uint
		ctrl.s.DB.Model(&model.Cluster{}).Where("id IN ?", ids).Pluck("group_id", &groupIDs)
		keep := map[uint]bool{}
		for _, id := range groupIDs {
			keep[id] = true
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
		filtered := []cmdbmodel.CmdbGroup{}
		for _, g := range groups {
			if keep[g.ID] {
				filtered = append(filtered, g)
			}
		}
		groups = filtered
	}
	result.Success(c, cmdbmodel.BuildCmdbGroupTree(groups))
}
func (ctrl *Controller) Repositories(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	rows := []model.Repository{}
	q := ctrl.s.DB.Model(&model.Repository{})
	if !ctrl.s.IsAdmin(u) {
		scope, e := ctrl.s.Scope(u, "db_cluster_view")
		if e != nil {
			fail(c, e)
			return
		}
		ids := []uint{}
		for id := range scope {
			ids = append(ids, id)
		}
		q = q.Where("group_id IN (?)", ctrl.s.DB.Model(&model.Cluster{}).Select("group_id").Where("id IN ?", ids))
	}
	f := filter(c)
	if f.GroupID > 0 {
		ids, e := ctrl.s.GroupIDs(f.GroupID)
		if e != nil {
			fail(c, e)
			return
		}
		q = q.Where("group_id IN ?", ids)
	}
	if f.Name != "" {
		q = q.Where("name LIKE ?", "%"+f.Name+"%")
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.ClusterID > 0 || f.Engine != "" || f.Environment != "" {
		cq := ctrl.s.DB.Model(&model.Cluster{}).Select("group_id")
		if f.ClusterID > 0 {
			cq = cq.Where("id = ?", f.ClusterID)
		}
		if f.Engine != "" {
			cq = cq.Where("engine = ?", f.Engine)
		}
		if f.Environment != "" {
			cq = cq.Where("environment = ?", f.Environment)
		}
		q = q.Where("group_id IN (?)", cq)
	}
	var total int64
	q.Count(&total)
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 || f.Size > 100 {
		f.Size = 20
	}
	if e := q.Order("id DESC").Offset((f.Page - 1) * f.Size).Limit(f.Size).Find(&rows).Error; e != nil {
		fail(c, e)
		return
	}
	result.Success(c, gin.H{"list": rows, "total": total})
}
func (ctrl *Controller) SaveRepository(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	var r model.Repository
	if c.ShouldBindJSON(&r) != nil {
		result.Failed(c, 400, "仓库参数无效")
		return
	}
	saved, e := ctrl.s.SaveRepository(c.Request.Context(), u, r)
	if e != nil {
		fail(c, e)
		return
	}
	result.Success(c, saved)
}
func (ctrl *Controller) CheckRepository(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	r, e := ctrl.s.CheckRepository(c.Request.Context(), u, id(c))
	if e != nil {
		fail(c, e)
		return
	}
	result.Success(c, r)
}

type Controller struct{ s *service.Service }

func New(db *gorm.DB) *Controller { return &Controller{s: service.New(db)} }
func user(c *gin.Context) (uint, bool) {
	id, e := jwt.GetAdminId(c)
	if e != nil {
		result.Failed(c, 401, "请先登录")
		return 0, false
	}
	return id, true
}
func id(c *gin.Context) uint { v, _ := strconv.ParseUint(c.Param("id"), 10, 32); return uint(v) }
func queryID(c *gin.Context, key string) uint {
	v, _ := strconv.ParseUint(c.Query(key), 10, 32)
	return uint(v)
}
func filter(c *gin.Context) service.Filter {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	return service.Filter{Name: c.Query("name"), Engine: c.Query("engine"), Environment: c.Query("environment"), Status: c.Query("status"), GroupID: queryID(c, "groupId"), ClusterID: queryID(c, "clusterId"), Page: page, Size: size}
}
func fail(c *gin.Context, e error) {
	status := 400
	if errors.Is(e, service.ErrForbidden) {
		status = 403
	}
	if errors.Is(e, service.ErrConflict) {
		status = 409
	}
	result.Failed(c, status, e.Error())
}
func (ctrl *Controller) List(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	rows, total, e := ctrl.s.List(u, filter(c))
	if e != nil {
		fail(c, e)
		return
	}
	result.Success(c, gin.H{"list": rows, "total": total})
}
func (ctrl *Controller) Create(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	var req service.ClusterRequest
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, 400, "参数无效")
		return
	}
	row, e := ctrl.s.Create(c.Request.Context(), req, u)
	if e != nil {
		fail(c, e)
		return
	}
	result.Success(c, row)
}
func (ctrl *Controller) Discover(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	var req service.ClusterRequest
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, 400, "参数无效")
		return
	}
	row, e := ctrl.s.Discover(c.Request.Context(), req, u)
	if e != nil {
		fail(c, e)
		return
	}
	result.Success(c, row)
}
func (ctrl *Controller) cluster(c *gin.Context, action string) (*model.Cluster, uint, bool) {
	u, ok := user(c)
	if !ok {
		return nil, 0, false
	}
	if e := ctrl.s.Authorize(u, id(c), action); e != nil {
		fail(c, e)
		return nil, 0, false
	}
	row, e := ctrl.s.Cluster(id(c))
	if e != nil {
		fail(c, e)
		return nil, 0, false
	}
	return row, u, true
}
func (ctrl *Controller) Detail(c *gin.Context) {
	row, u, ok := ctrl.cluster(c, "db_cluster_view")
	if !ok {
		return
	}
	service.MarkStale(row)
	result.Success(c, gin.H{"cluster": row, "capabilities": ctrl.s.Capabilities(u, row), "slots": service.SlotSummary(row)})
}
func (ctrl *Controller) Members(c *gin.Context) {
	row, _, ok := ctrl.cluster(c, "db_cluster_view")
	if !ok {
		return
	}
	result.Success(c, gin.H{"list": row.Members})
}
func (ctrl *Controller) Capabilities(c *gin.Context) {
	row, u, ok := ctrl.cluster(c, "db_cluster_view")
	if !ok {
		return
	}
	result.Success(c, ctrl.s.Capabilities(u, row))
}
func (ctrl *Controller) Plan(c *gin.Context) {
	row, u, ok := ctrl.cluster(c, "db_cluster_view")
	if !ok {
		return
	}
	var req struct {
		Action     string             `json:"action"`
		Parameters service.Parameters `json:"parameters"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, 400, "参数无效")
		return
	}
	plan, e := ctrl.s.Plan(c.Request.Context(), row.ID, u, req.Action, req.Parameters)
	if e != nil {
		fail(c, e)
		return
	}
	result.Success(c, plan)
}
func (ctrl *Controller) Submit(c *gin.Context) {
	row, u, ok := ctrl.cluster(c, "db_cluster_view")
	if !ok {
		return
	}
	var req struct {
		PlanID         uint   `json:"planId"`
		IdempotencyKey string `json:"idempotencyKey"`
		ConfirmName    string `json:"confirmName"` // compatibility with older clients
		Confirmed      bool   `json:"confirmed"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, 400, "参数无效")
		return
	}
	var operation *model.Operation
	var e error
	if req.Confirmed || req.ConfirmName == "" {
		operation, e = ctrl.s.SubmitConfirmed(c.Request.Context(), row.ID, u, req.PlanID, req.IdempotencyKey, req.Confirmed)
	} else {
		operation, e = ctrl.s.Submit(c.Request.Context(), row.ID, u, req.PlanID, req.IdempotencyKey, req.ConfirmName)
	}
	if e != nil {
		fail(c, e)
		return
	}
	result.Success(c, operation)
}
func (ctrl *Controller) Backup(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	backup, err := ctrl.s.BackupRecord(c.Request.Context(), u, id(c))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		result.Failed(c, 404, "备份不存在")
		return
	}
	if err != nil {
		fail(c, err)
		return
	}
	result.Success(c, ctrl.recordLabels([]model.Backup{*backup})[0])
}

func (ctrl *Controller) Operation(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	var op model.Operation
	if ctrl.s.DB.Where("audit_deleted_at IS NULL").Preload("Steps").First(&op, id(c)).Error != nil {
		result.Failed(c, 404, "任务不存在")
		return
	}
	if e := ctrl.s.Authorize(u, op.ClusterID, "db_cluster_audit"); e != nil {
		fail(c, e)
		return
	}
	result.Success(c, op)
}
func (ctrl *Controller) Storage(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	rows, e := ctrl.s.Storage(u, filter(c))
	if e != nil {
		fail(c, e)
		return
	}
	filtered := []model.StorageSnapshot{}
	for _, row := range rows {
		if c.Query("kind") == "" || c.Query("kind") == "all" || row.Kind == c.Query("kind") {
			filtered = append(filtered, row)
		}
	}
	total := len(filtered)
	f := filter(c)
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 || f.Size > 100 {
		f.Size = 20
	}
	start := (f.Page - 1) * f.Size
	if start > total {
		start = total
	}
	end := start + f.Size
	if end > total {
		end = total
	}
	result.Success(c, gin.H{"list": ctrl.recordLabels(filtered[start:end]), "total": total})
}
func (ctrl *Controller) Configurations(c *gin.Context) {
	row, _, ok := ctrl.cluster(c, "db_cluster_view")
	if !ok {
		return
	}
	out, e := ctrl.s.Configurations(c.Request.Context(), row)
	if e != nil {
		fail(c, e)
		return
	}
	result.Success(c, out)
}
func (ctrl *Controller) SavePolicy(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	var p model.BackupPolicy
	if c.Request.Method == "PATCH" {
		if ctrl.s.DB.First(&p, id(c)).Error != nil {
			fail(c, errors.New("策略不存在"))
			return
		}
		if e := ctrl.s.Authorize(u, p.ClusterID, "db_cluster_backup"); e != nil {
			fail(c, e)
			return
		}
	}
	if c.ShouldBindJSON(&p) != nil {
		result.Failed(c, 400, "参数无效")
		return
	}
	p.ID = id(c)
	saved, e := ctrl.s.SaveBackupPolicy(c.Request.Context(), u, p)
	if e != nil {
		fail(c, e)
		return
	}
	result.Success(c, saved)
}

// Scope and filters apply before pagination to every cross-cluster list.
func (ctrl *Controller) Records(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := user(c)
		if !ok {
			return
		}
		action := "db_cluster_view"
		if kind == "operations" || kind == "events" {
			action = "db_cluster_audit"
		}
		scope, e := ctrl.s.Scope(u, action)
		if e != nil {
			fail(c, e)
			return
		}
		clusterQuery := ctrl.s.DB.Model(&model.Cluster{}).Select("id")
		if scope != nil {
			ids := []uint{}
			for id := range scope {
				ids = append(ids, id)
			}
			clusterQuery = clusterQuery.Where("id IN ?", ids)
		}
		f := filter(c)
		if v := queryID(c, "clusterId"); v > 0 {
			clusterQuery = clusterQuery.Where("id = ?", v)
		}
		if f.Name != "" {
			clusterQuery = clusterQuery.Where("name LIKE ?", "%"+f.Name+"%")
		}
		if f.Engine != "" {
			clusterQuery = clusterQuery.Where("engine = ?", f.Engine)
		}
		if f.Environment != "" {
			clusterQuery = clusterQuery.Where("environment = ?", f.Environment)
		}
		if f.GroupID > 0 {
			ids, e := ctrl.s.GroupIDs(f.GroupID)
			if e != nil {
				fail(c, e)
				return
			}
			clusterQuery = clusterQuery.Where("group_id IN ?", ids)
		}
		var dest any
		switch kind {
		case "operations":
			dest = &[]model.Operation{}
		case "events":
			dest = &[]model.Event{}
		case "backups":
			dest = &[]model.Backup{}
		case "backup-policies":
			dest = &[]model.BackupPolicy{}
		case "config-revisions":
			dest = &[]model.ConfigRevision{}
		default:
			result.Failed(c, 404, "记录类型不存在")
			return
		}
		q := ctrl.s.DB.Model(dest).Where("cluster_id IN (?)", clusterQuery)
		if kind == "operations" || kind == "events" {
			q = q.Where("audit_deleted_at IS NULL")
		}
		if f.Status != "" && (kind == "operations" || kind == "backups" || kind == "config-revisions") {
			q = q.Where("status = ?", f.Status)
		}
		var total int64
		q.Count(&total)
		if f.Page < 1 {
			f.Page = 1
		}
		if f.Size < 1 || f.Size > 100 {
			f.Size = 20
		}
		if e = q.Order("id DESC").Limit(f.Size).Offset((f.Page - 1) * f.Size).Find(dest).Error; e != nil {
			fail(c, e)
			return
		}
		result.Success(c, gin.H{"list": ctrl.recordLabels(dest), "total": total})
	}
}

func (ctrl *Controller) StorageThreshold(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	if !ctrl.s.IsAdmin(u) {
		fail(c, service.ErrForbidden)
		return
	}
	group, _ := strconv.ParseUint(c.Query("groupId"), 10, 64)
	result.Success(c, policy.StorageThresholds(ctrl.s.DB, uint(group)))
}
func (ctrl *Controller) SaveStorageThreshold(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	if !ctrl.s.IsAdmin(u) {
		fail(c, service.ErrForbidden)
		return
	}
	var r model.StorageThreshold
	if c.ShouldBindJSON(&r) != nil || r.GroupID == 0 || r.WarningPercent < 1 || r.WarningPercent >= r.CriticalPercent || r.CriticalPercent > 100 {
		fail(c, errors.New("阈值必须满足 1 ≤ 提醒 < 严重 ≤ 100"))
		return
	}
	var group cmdbmodel.CmdbGroup
	if ctrl.s.DB.First(&group, r.GroupID).Error != nil {
		fail(c, errors.New("业务分组不存在"))
		return
	}
	r.UpdatedBy = u
	if e := ctrl.s.DB.Save(&r).Error; e != nil {
		fail(c, e)
		return
	}
	result.Success(c, r)
}
