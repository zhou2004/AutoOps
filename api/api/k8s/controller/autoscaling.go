package controller

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"dodevops-api/api/k8s/model"
	"dodevops-api/api/k8s/service"
	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type K8sAutoscalingController struct {
	service *service.K8sAutoscalingService
}

func NewK8sAutoscalingController(db *gorm.DB) *K8sAutoscalingController {
	return &K8sAutoscalingController{service: service.NewK8sAutoscalingService(db)}
}
func (ctrl *K8sAutoscalingController) Handle(c *gin.Context) {
	id, e := strconv.ParseUint(c.Param("id"), 10, 32)
	if e != nil || id == 0 {
		result.Failed(c, 400, "无效集群 ID")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	ns, kind, name := c.Param("namespaceName"), c.Param("kind"), c.Param("name")
	var value interface{}
	var err error
	if value := c.Query("dryRun"); value != "" && value != "true" && value != "false" {
		result.Failed(c, 400, "dryRun 必须为 true 或 false")
		return
	}
	dryRun := c.Query("dryRun") == "true"
	if kind == "" {
		var capabilities map[string]interface{}
		capabilities, err = ctrl.service.Capabilities(ctx, uint(id))
		if err == nil {
			permissions, _ := c.Get("autoscalingPermissions")
			value = map[string]interface{}{"capabilities": capabilities, "permissions": permissions}
		}
	} else {
		switch c.Request.Method {
		case http.MethodGet:
			if name == "" {
				value, err = ctrl.service.List(ctx, uint(id), ns, kind)
			} else {
				value, err = ctrl.service.Get(ctx, uint(id), ns, kind, name)
			}
		case http.MethodPost, http.MethodPut:
			var req model.AutoscalingRequest
			decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
			decoder.DisallowUnknownFields()
			if e := decoder.Decode(&req); e != nil {
				result.Failed(c, 400, "配置无效: "+e.Error())
				return
			}
			if e := decoder.Decode(new(interface{})); e != io.EOF {
				result.Failed(c, 400, "请求必须仅包含一个 JSON 对象")
				return
			}
			value, err = ctrl.service.Save(ctx, uint(id), ns, kind, name, req, c.Request.Method == http.MethodPost, dryRun)
		case http.MethodDelete:
			value, err = ctrl.service.Delete(ctx, uint(id), ns, kind, name, c.Query("resourceVersion"), dryRun)
		}
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
