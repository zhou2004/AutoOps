package controller

import (
	"net/http"
	"strconv"

	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/api/task/model"
	"dodevops-api/api/task/service"
	"dodevops-api/common"
	"dodevops-api/common/constant"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
)

func RunTaskNow(c *gin.Context) {
	taskID, ok := parsePathUint(c, "id")
	if !ok {
		return
	}
	history, err := service.NewTaskWorkService().RunNow(taskID, model.TaskTriggerManual, currentOperator(c))
	if err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	result.Success(c, gin.H{"message": "任务已提交执行", "history_id": history.ID})
}

func ScheduleTask(c *gin.Context) {
	taskID, ok := parsePathUint(c, "id")
	if !ok {
		return
	}
	if err := service.NewTaskWorkService().ScheduleTask(taskID); err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	result.Success(c, gin.H{"message": "定时调度已启用"})
}

func PauseTaskExecution(c *gin.Context) {
	taskID, ok := parsePathUint(c, "id")
	if !ok {
		return
	}
	if err := service.NewTaskWorkService().PauseExecution(taskID); err != nil {
		result.Failed(c, http.StatusBadRequest, "暂停执行失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"message": "任务执行已暂停"})
}

func ResumeTaskExecution(c *gin.Context) {
	taskID, ok := parsePathUint(c, "id")
	if !ok {
		return
	}
	if err := service.NewTaskWorkService().ResumeExecution(taskID); err != nil {
		result.Failed(c, http.StatusBadRequest, "恢复执行失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"message": "任务执行已恢复"})
}

func StopTaskExecution(c *gin.Context) {
	taskID, ok := parsePathUint(c, "id")
	if !ok {
		return
	}
	if err := service.NewTaskWorkService().StopExecution(taskID, currentOperator(c)); err != nil {
		result.Failed(c, http.StatusBadRequest, "停止执行失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"message": "任务执行已停止"})
}

func ListTaskHistory(c *gin.Context) {
	taskID, ok := parsePathUint(c, "id")
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	status, _ := strconv.Atoi(c.Query("status"))
	data, err := service.NewTaskHistoryService(common.GetDB()).List(taskID, page, pageSize, status)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取执行历史失败: "+err.Error())
		return
	}
	result.SuccessWithPage(c, data.List, data.Total, page, pageSize)
}

func GetTaskHistoryDetail(c *gin.Context) {
	taskID, ok := parsePathUint(c, "id")
	if !ok {
		return
	}
	historyID, ok := parsePathUint(c, "history_id")
	if !ok {
		return
	}
	data, err := service.NewTaskHistoryService(common.GetDB()).Detail(taskID, historyID)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "执行历史不存在")
		return
	}
	result.Success(c, data)
}

func RollbackTaskHistory(c *gin.Context) {
	taskID, ok := parsePathUint(c, "id")
	if !ok {
		return
	}
	historyID, ok := parsePathUint(c, "history_id")
	if !ok {
		return
	}
	if err := service.NewTaskHistoryService(common.GetDB()).Rollback(taskID, historyID, currentOperator(c)); err != nil {
		result.Failed(c, http.StatusBadRequest, "回滚失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"message": "已恢复该次执行配置，未自动重新执行"})
}

func parsePathUint(c *gin.Context, name string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil || value == 0 {
		result.Failed(c, http.StatusBadRequest, "无效的ID参数")
		return 0, false
	}
	return uint(value), true
}

func currentOperator(c *gin.Context) string {
	value, exists := c.Get(constant.ContextKeyUserObj)
	if !exists {
		return "System"
	}
	admin, ok := value.(*systemmodel.JwtAdmin)
	if !ok || admin.Username == "" {
		return "System"
	}
	return admin.Username
}
