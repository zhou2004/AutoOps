package controller

import (
	"net/http"
	"strconv"

	"dodevops-api/api/k8s/model"
	"dodevops-api/api/k8s/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type K8sRbacController struct {
	service service.IK8sRbacService
}

func NewK8sRbacController(db *gorm.DB) *K8sRbacController {
	return &K8sRbacController{
		service: service.NewK8sRbacService(db),
	}
}

// @Summary 创建K8s RBAC角色
// @Tags K8s RBAC
func (ctrl *K8sRbacController) CreateRole(c *gin.Context) {
	var req model.CreateRbacRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	ctrl.service.CreateRole(c, &req)
}

// @Summary 更新K8s RBAC角色
func (ctrl *K8sRbacController) UpdateRole(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	var req model.CreateRbacRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	ctrl.service.UpdateRole(c, uint(id), &req)
}

// @Summary 删除K8s RBAC角色
func (ctrl *K8sRbacController) DeleteRole(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	ctrl.service.DeleteRole(c, uint(id))
}

// @Summary 获取K8s RBAC角色列表
func (ctrl *K8sRbacController) GetRoleList(c *gin.Context) {
	clusterID, _ := strconv.Atoi(c.Query("clusterId"))
	namespace := c.Query("namespace")
	name := c.Query("name")
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	ctrl.service.GetRoleList(c, uint(clusterID), namespace, name, page, size)
}

// @Summary 创建K8s RBAC绑定
func (ctrl *K8sRbacController) CreateBinding(c *gin.Context) {
	var req model.CreateRbacBindingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	ctrl.service.CreateBinding(c, &req)
}

// @Summary 更新K8s RBAC绑定
func (ctrl *K8sRbacController) UpdateBinding(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	var req model.CreateRbacBindingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	ctrl.service.UpdateBinding(c, uint(id), &req)
}

// @Summary 删除K8s RBAC绑定
func (ctrl *K8sRbacController) DeleteBinding(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	ctrl.service.DeleteBinding(c, uint(id))
}

// @Summary 获取K8s RBAC绑定列表
func (ctrl *K8sRbacController) GetBindingList(c *gin.Context) {
	clusterID, _ := strconv.Atoi(c.Query("clusterId"))
	namespace := c.Query("namespace")
	subjectType := c.Query("subjectType")
	subjectName := c.Query("subjectName")
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	ctrl.service.GetBindingList(c, uint(clusterID), namespace, subjectType, subjectName, page, size)
}

// @Summary 获取我的K8s RBAC权限
func (ctrl *K8sRbacController) GetMyPermissions(c *gin.Context) {
	ctrl.service.GetMyPermissions(c)
}
