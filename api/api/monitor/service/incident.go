package service

import (
	"strconv"

	"dodevops-api/api/monitor/dao"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
)

type IncidentServiceInterface interface {
	GetList(c *gin.Context)
	GetStats(c *gin.Context)
	Resolve(c *gin.Context)
	Delete(c *gin.Context)
}

type IncidentServiceImpl struct {
	dao *dao.IncidentDao
}

func NewIncidentService() IncidentServiceInterface {
	return &IncidentServiceImpl{
		dao: dao.NewIncidentDao(),
	}
}

func (s *IncidentServiceImpl) GetList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status := c.Query("status")
	level := c.Query("level")
	source := c.Query("source")

	list, total, err := s.dao.GetList(page, pageSize, status, level, source)
	if err != nil {
		result.Failed(c, 500, "查询失败: "+err.Error())
		return
	}
	result.Success(c, map[string]interface{}{"list": list, "total": total})
}

func (s *IncidentServiceImpl) GetStats(c *gin.Context) {
	stats, err := s.dao.GetStats()
	if err != nil {
		result.Failed(c, 500, "统计失败: "+err.Error())
		return
	}
	result.Success(c, stats)
}

func (s *IncidentServiceImpl) Resolve(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的ID")
		return
	}
	if err := s.dao.Resolve(uint(id)); err != nil {
		result.Failed(c, 500, "解决失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *IncidentServiceImpl) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的ID")
		return
	}
	if err := s.dao.Delete(uint(id)); err != nil {
		result.Failed(c, 500, "删除失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}