package controller

import (
	"net/http"
	"strconv"

	"dodevops-api/api/task/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
)

type TaskAnsibleViewController struct {
	service service.ITaskAnsibleViewService
}

func NewTaskAnsibleViewController(service service.ITaskAnsibleViewService) *TaskAnsibleViewController {
	return &TaskAnsibleViewController{service: service}
}

// Create 创建视图
// @Summary 创建Ansible视图
// @Description 创建Ansible任务分组视图
// @Tags 任务视图
// @Accept json
// @Produce json
// @Param request body service.CreateViewRequest true "创建视图请求"
// @Success 200 {object} result.Result{data=model.TaskAnsibleView}
// @Router /api/v1/task/ansible/view [post]
// @Security ApiKeyAuth
func (c *TaskAnsibleViewController) Create(ctx *gin.Context) {
	var req service.CreateViewRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.Failed(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	c.service.Create(ctx, &req)
}

// Update 更新视图
// @Summary 更新Ansible视图
// @Description 更新Ansible任务分组视图
// @Tags 任务视图
// @Accept json
// @Produce json
// @Param id path int true "视图ID"
// @Param request body service.UpdateViewRequest true "更新视图请求"
// @Success 200 {object} result.Result{data=model.TaskAnsibleView}
// @Router /api/v1/task/ansible/view/{id} [put]
// @Security ApiKeyAuth
func (c *TaskAnsibleViewController) Update(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(ctx, http.StatusBadRequest, "无效的ID")
		return
	}
	var req service.UpdateViewRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.Failed(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	c.service.Update(ctx, uint(id), &req)
}

// Delete 删除视图
// @Summary 删除Ansible视图
// @Description 删除Ansible任务分组视图（关联的任务 view_id 将被置空）
// @Tags 任务视图
// @Accept json
// @Produce json
// @Param id path int true "视图ID"
// @Success 200 {object} result.Result
// @Router /api/v1/task/ansible/view/{id} [delete]
// @Security ApiKeyAuth
func (c *TaskAnsibleViewController) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(ctx, http.StatusBadRequest, "无效的ID")
		return
	}
	c.service.Delete(ctx, uint(id))
}

// GetAll 获取所有视图列表
// @Summary 获取所有Ansible视图
// @Description 获取所有Ansible任务分组视图
// @Tags 任务视图
// @Accept json
// @Produce json
// @Success 200 {object} result.Result{data=object{list=[]model.TaskAnsibleView,total=int}}
// @Router /api/v1/task/ansible/view/all [get]
// @Security ApiKeyAuth
func (c *TaskAnsibleViewController) GetAll(ctx *gin.Context) {
	c.service.GetAll(ctx)
}

// List 分页获取视图列表
// @Summary 分页获取Ansible视图
// @Description 分页获取Ansible任务分组视图
// @Tags 任务视图
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} result.Result{data=object{list=[]model.TaskAnsibleView,total=int}}
// @Router /api/v1/task/ansible/view [get]
// @Security ApiKeyAuth
func (c *TaskAnsibleViewController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))
	c.service.List(ctx, page, size)
}
