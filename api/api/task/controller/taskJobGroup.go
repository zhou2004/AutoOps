package controller

import (
	"strconv"

	"dodevops-api/api/task/service"
	"dodevops-api/common"
	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
)

func ListTaskJobGroups(c *gin.Context) {
	groups, err := service.NewTaskJobGroupService(common.GetDB()).List()
	if err != nil {
		result.Failed(c, 500, "获取分组失败: "+err.Error())
		return
	}
	result.Success(c, groups)
}

func SaveTaskJobGroup(c *gin.Context) {
	var id uint
	if c.Request.Method == "PUT" {
		var ok bool
		id, ok = parsePathUint(c, "group_id")
		if !ok {
			return
		}
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "请输入分组名称")
		return
	}
	group, err := service.NewTaskJobGroupService(common.GetDB()).Save(id, req.Name)
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	result.Success(c, group)
}

func DeleteTaskJobGroup(c *gin.Context) {
	id, ok := parsePathUint(c, "group_id")
	if !ok {
		return
	}
	if err := service.NewTaskJobGroupService(common.GetDB()).Delete(id); err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	result.Success(c, nil)
}

// Missing group_id means all jobs; 0 means ungrouped jobs.
func parseTaskGroupFilter(c *gin.Context) (*uint, bool) {
	value := c.Query("group_id")
	if value == "" {
		return nil, true
	}
	id, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的任务分组")
		return nil, false
	}
	groupID := uint(id)
	return &groupID, true
}
