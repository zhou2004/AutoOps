package service

import (
	"net/http"

	"dodevops-api/api/task/dao"
	"dodevops-api/api/task/model"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ITaskAnsibleViewService interface {
	Create(c *gin.Context, req *CreateViewRequest)
	Update(c *gin.Context, id uint, req *UpdateViewRequest)
	Delete(c *gin.Context, id uint)
	GetAll(c *gin.Context)
	List(c *gin.Context, page, size int)
}

type TaskAnsibleViewServiceImpl struct {
	dao *dao.TaskAnsibleViewDao
}

func NewTaskAnsibleViewService(db *gorm.DB) ITaskAnsibleViewService {
	return &TaskAnsibleViewServiceImpl{
		dao: dao.NewTaskAnsibleViewDao(db),
	}
}

type CreateViewRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateViewRequest struct {
	Name string `json:"name" binding:"required"`
}

type ViewListResponse struct {
	List  []model.TaskAnsibleView `json:"list"`
	Total int64                   `json:"total"`
}

// Create 创建视图
func (s *TaskAnsibleViewServiceImpl) Create(c *gin.Context, req *CreateViewRequest) {
	// 检查名称是否已存在
	existing, err := s.dao.GetByName(req.Name)
	if err == nil && existing != nil && existing.ID > 0 {
		result.Failed(c, http.StatusBadRequest, "视图名称已存在")
		return
	}

	view := &model.TaskAnsibleView{
		Name: req.Name,
	}

	if err := s.dao.Create(view); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建视图失败: "+err.Error())
		return
	}

	result.Success(c, view)
}

// Update 更新视图
func (s *TaskAnsibleViewServiceImpl) Update(c *gin.Context, id uint, req *UpdateViewRequest) {
	view, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "视图不存在")
		return
	}

	// 检查新名称是否与其他视图冲突
	existing, err := s.dao.GetByName(req.Name)
	if err == nil && existing != nil && existing.ID != id {
		result.Failed(c, http.StatusBadRequest, "视图名称已存在")
		return
	}

	view.Name = req.Name
	if err := s.dao.Update(view); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新视图失败: "+err.Error())
		return
	}

	result.Success(c, view)
}

// Delete 删除视图
func (s *TaskAnsibleViewServiceImpl) Delete(c *gin.Context, id uint) {
	if err := s.dao.Delete(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除视图失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

// GetAll 获取所有视图
func (s *TaskAnsibleViewServiceImpl) GetAll(c *gin.Context) {
	views, err := s.dao.GetAll()
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取视图列表失败: "+err.Error())
		return
	}

	if views == nil {
		views = []model.TaskAnsibleView{}
	}

	result.Success(c, gin.H{
		"list":  views,
		"total": len(views),
	})
}

// List 分页获取视图列表
func (s *TaskAnsibleViewServiceImpl) List(c *gin.Context, page, size int) {
	views, err := s.dao.GetAll()
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取视图列表失败: "+err.Error())
		return
	}

	total := len(views)

	// 简单分页
	start := (page - 1) * size
	if start >= total {
		views = []model.TaskAnsibleView{}
	} else {
		end := start + size
		if end > total {
			end = total
		}
		views = views[start:end]
	}

	result.Success(c, gin.H{
		"list":  views,
		"total": total,
	})
}
