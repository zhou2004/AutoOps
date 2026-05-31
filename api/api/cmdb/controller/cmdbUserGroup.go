package controller

import (
	"net/http"
	"strconv"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
)

// ======================= CMDB User Group =======================

type CmdbUserGroupController struct {
	svc service.ICmdbUserGroupService
}

func NewCmdbUserGroupController() *CmdbUserGroupController {
	return &CmdbUserGroupController{svc: service.NewCmdbUserGroupService()}
}

func (ctrl *CmdbUserGroupController) Create(c *gin.Context) {
	var req model.CmdbCreateUserGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Create(c, &req)
}

func (ctrl *CmdbUserGroupController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	var req model.CmdbUpdateUserGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Update(c, uint(id), &req)
}

func (ctrl *CmdbUserGroupController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.Delete(c, uint(id))
}

func (ctrl *CmdbUserGroupController) GetList(c *gin.Context) {
	var query model.CmdbUserGroupQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误")
		return
	}
	ctrl.svc.GetList(c, query)
}

func (ctrl *CmdbUserGroupController) GetMembers(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.GetMembers(c, uint(id))
}

func (ctrl *CmdbUserGroupController) AddMembers(c *gin.Context) {
	var req model.CmdbAddGroupMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.AddMembers(c, &req)
}

func (ctrl *CmdbUserGroupController) RemoveMember(c *gin.Context) {
	var req model.CmdbRemoveGroupMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.RemoveMember(c, &req)
}

func (ctrl *CmdbUserGroupController) GetAll(c *gin.Context) {
	ctrl.svc.GetAll(c)
}

// ======================= Credential Permission =======================

type CmdbCredentialPermissionController struct {
	svc service.ICmdbCredentialPermissionService
}

func NewCmdbCredentialPermissionController() *CmdbCredentialPermissionController {
	return &CmdbCredentialPermissionController{svc: service.NewCmdbCredentialPermissionService()}
}

func (ctrl *CmdbCredentialPermissionController) Create(c *gin.Context) {
	var req model.CmdbCredentialPermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Create(c, &req)
}

func (ctrl *CmdbCredentialPermissionController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	var req model.CmdbCredentialPermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Update(c, uint(id), &req)
}

func (ctrl *CmdbCredentialPermissionController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.Delete(c, uint(id))
}

func (ctrl *CmdbCredentialPermissionController) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.GetByID(c, uint(id))
}

func (ctrl *CmdbCredentialPermissionController) GetList(c *gin.Context) {
	var query model.CmdbCredentialPermissionQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误")
		return
	}
	ctrl.svc.GetList(c, query)
}

func (ctrl *CmdbCredentialPermissionController) GetMyCredentials(c *gin.Context) {
	ctrl.svc.GetMyCredentials(c)
}
