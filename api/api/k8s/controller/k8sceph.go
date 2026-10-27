package controller

import (
	"context"
	"strconv"
	"time"

	"dodevops-api/api/k8s/model"
	"dodevops-api/api/k8s/service"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type K8sCephController struct{ service *service.K8sCephService }

func NewK8sCephController(db *gorm.DB) *K8sCephController {
	return &K8sCephController{service: service.NewK8sCephService(db)}
}

// Handle serves the explicitly registered maintenance routes, never arbitrary commands.
func (ctrl *K8sCephController) Handle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 40*time.Second)
	defer cancel()
	actor := ""
	if admin, e := jwt.GetAdmin(c); e == nil {
		actor = admin.Username
	}
	var value any
	action := c.GetString("cephAction")
	switch action {
	case "capabilities":
		value = map[string]any{"operations": service.CephOperations(), "canWrite": c.GetBool("cephCanWrite"), "namespace": "rook-ceph"}
	case "inspect":
		p := map[string]string{}
		for k, v := range c.Request.URL.Query() {
			if k != "kind" && len(v) > 0 {
				p[k] = v[0]
			}
		}
		value, err = ctrl.service.Inspect(ctx, uint(id), c.Query("kind"), p)
	case "tasks":
		value, err = ctrl.service.Tasks(ctx, uint(id))
	case "create-task":
		var req model.CephTaskRequest
		if c.ShouldBindJSON(&req) != nil {
			result.Failed(c, 400, "任务参数无效")
			return
		}
		value, err = ctrl.service.CreateTask(ctx, uint(id), actor, req)
	case "logs":
		value, err = ctrl.service.TaskLogs(ctx, uint(id), c.Param("name"))
	case "cancel":
		value, err = ctrl.service.CancelTask(ctx, uint(id), c.Param("name"))
	case "schedules":
		value, err = ctrl.service.Schedules(ctx, uint(id))
	case "create-schedule":
		var req model.CephScheduleRequest
		if c.ShouldBindJSON(&req) != nil {
			result.Failed(c, 400, "定时计划参数无效")
			return
		}
		value, err = ctrl.service.CreateSchedule(ctx, uint(id), actor, req)
	case "update-schedule":
		var req struct {
			Suspended *bool `json:"suspended"`
		}
		if c.ShouldBindJSON(&req) != nil || req.Suspended == nil {
			result.Failed(c, 400, "必须指定 suspended")
			return
		}
		value, err = ctrl.service.ChangeSchedule(ctx, uint(id), c.Param("name"), *req.Suspended, false)
	case "delete-schedule":
		value, err = ctrl.service.ChangeSchedule(ctx, uint(id), c.Param("name"), false, true)
	case "repositories":
		value, err = ctrl.service.Repositories(ctx, uint(id))
	case "create-repository":
		var req model.CephRepositoryRequest
		if c.ShouldBindJSON(&req) != nil {
			result.Failed(c, 400, "仓库参数无效")
			return
		}
		value, err = ctrl.service.CreateRepository(ctx, uint(id), req)
	case "backups":
		value, err = ctrl.service.Backups(ctx, uint(id), c.Query("repository"))
	default:
		result.Failed(c, 404, "未知操作")
		return
	}
	if err != nil {
		code := 400
		if apierrors.IsForbidden(err) {
			code = 403
		}
		if apierrors.IsNotFound(err) {
			code = 404
		}
		if apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err) {
			code = 409
		}
		if ctx.Err() != nil {
			code = 504
		}
		result.Failed(c, code, err.Error())
		return
	}
	result.Success(c, value)
}

func (ctrl *K8sCephController) Action(action string) gin.HandlerFunc {
	return func(c *gin.Context) { c.Set("cephAction", action); ctrl.Handle(c) }
}
