package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"dodevops-api/api/task/service/kubeasz"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type KubeaszController struct {
	service *kubeasz.Service
	db      *gorm.DB
}

func NewKubeaszController(db *gorm.DB) *KubeaszController {
	return &KubeaszController{service: kubeasz.NewService(db), db: db}
}

// System operations can provision machines outside an existing K8s cluster.
// They require the system administrator role, not a namespace-level K8s grant.
func (k *KubeaszController) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, e := jwt.GetAdmin(c)
		if e != nil {
			result.Failed(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}
		var count int64
		e = k.db.Table("sys_admin_role").Joins("JOIN sys_role ON sys_role.id = sys_admin_role.role_id").Where("sys_admin_role.admin_id = ? AND sys_role.role_key = ? AND sys_role.status = ?", admin.ID, "admin", 1).Count(&count).Error
		if e != nil || count == 0 {
			result.Failed(c, http.StatusForbidden, "kubeasz 系统运维需要系统管理员角色")
			c.Abort()
			return
		}
		c.Next()
	}
}
func (k *KubeaszController) Catalog(c *gin.Context) { result.Success(c, k.service.Catalog()) }
func kubeaszBody(c *gin.Context) (kubeasz.Request, error) {
	var req kubeasz.Request
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if e := decoder.Decode(&req); e != nil {
		return req, errors.New("请求 JSON 格式错误、包含未知字段或超过 64 KiB")
	}
	if e := decoder.Decode(new(interface{})); e != io.EOF {
		return req, errors.New("仅接受一个 JSON 对象")
	}
	return req, nil
}
func (k *KubeaszController) Preview(c *gin.Context) {
	req, e := kubeaszBody(c)
	if e != nil {
		result.Failed(c, 400, e.Error())
		return
	}
	plan, e := k.service.Preview(req)
	if e != nil {
		result.Failed(c, 400, e.Error())
		return
	}
	result.Success(c, plan)
}
func (k *KubeaszController) Start(c *gin.Context) {
	req, e := kubeaszBody(c)
	if e != nil {
		result.Failed(c, 400, e.Error())
		return
	}
	admin, e := jwt.GetAdmin(c)
	if e != nil {
		result.Failed(c, 401, "未登录")
		return
	}
	op, e := k.service.Start(req, admin.ID, admin.Username)
	if e != nil {
		result.Failed(c, 409, e.Error())
		return
	}
	result.Success(c, op)
}
func (k *KubeaszController) History(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	if size < 1 || size > 100 {
		size = 20
	}
	ops, total, e := k.service.History(c.Query("cluster"), page, size)
	if e != nil {
		result.Failed(c, 400, e.Error())
		return
	}
	result.SuccessWithPage(c, ops, total, page, size)
}
func kubeaszID(c *gin.Context) (uint, bool) {
	id, e := strconv.ParseUint(c.Param("id"), 10, 32)
	if e != nil || id == 0 {
		result.Failed(c, 400, "无效操作 ID")
		return 0, false
	}
	return uint(id), true
}
func (k *KubeaszController) Detail(c *gin.Context) {
	id, ok := kubeaszID(c)
	if !ok {
		return
	}
	op, log, e := k.service.Detail(id)
	if errors.Is(e, gorm.ErrRecordNotFound) {
		result.Failed(c, 404, "操作记录不存在")
		return
	}
	if e != nil {
		result.Failed(c, 500, "读取操作记录失败")
		return
	}
	result.Success(c, gin.H{"operation": op, "log": log})
}
func (k *KubeaszController) Cancel(c *gin.Context) {
	id, ok := kubeaszID(c)
	if !ok {
		return
	}
	if e := k.service.Cancel(id); e != nil {
		result.Failed(c, 409, e.Error())
		return
	}
	result.Success(c, gin.H{"message": "已请求终止；远端已完成的修改不会回滚"})
}

func (k *KubeaszController) Reconcile(c *gin.Context) {
	id, ok := kubeaszID(c)
	if !ok {
		return
	}
	var req struct {
		Confirmation string `json:"confirmation"`
		Note         string `json:"note"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		result.Failed(c, 400, "核查请求格式错误")
		return
	}
	admin, err := jwt.GetAdmin(c)
	if err != nil {
		result.Failed(c, 401, "未登录")
		return
	}
	if err = k.service.Reconcile(id, req.Confirmation, req.Note, admin.Username); err != nil {
		result.Failed(c, 409, err.Error())
		return
	}
	result.Success(c, gin.H{"message": "核查记录已保存"})
}
