package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	cmdbmodel "dodevops-api/api/cmdb/model"
	"dodevops-api/api/task/dao"
	"dodevops-api/api/task/model"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// TaskJobService 任务服务接口
type TaskJobService interface {
	CreateTask(task *model.Task) error
	GetTask(id uint) (*model.Task, error)
	UpdateTask(id uint, task *model.Task) error
	DeleteTask(id uint) error
	ListTasks(page, pageSize int, name string, status int) ([]model.Task, int64, error)
	ListTasksWithDetails(page, pageSize int, name string, status int) ([]model.TaskWithDetails, int64, error)
	TaskNameExists(name string) (bool, error)
	GetTasksByName(name string) ([]model.Task, error)
	GetTasksByType(taskType int) ([]model.Task, error)
	GetTasksByStatus(status int) ([]model.Task, error)
	GetNextExecutionTime(cronExpr string) (time.Time, error)
	GetTaskTemplatesWithStatus(taskId uint) ([]map[string]interface{}, error)
	IncrementExecuteCount(taskId uint) error
	UpdateNextRunTime(taskId uint, nextTime time.Time) error
	UpdateTaskAfterExecution(taskId uint, cronExpr string) error
}

type taskJobServiceImpl struct {
	db      *gorm.DB
	taskDao dao.TaskDao
}

func NewTaskService(db *gorm.DB) TaskJobService {
	return &taskJobServiceImpl{
		db:      db,
		taskDao: dao.NewTaskDao(db),
	}
}

func (s *taskJobServiceImpl) CreateTask(task *model.Task) error {
	if err := s.normalizeAndValidateTask(task); err != nil {
		return err
	}

	// 创建主任务记录（DAO层会自动创建task_work记录）
	err := s.taskDao.Create(task)
	if err != nil {
		return errors.New("创建任务失败: " + err.Error())
	}

	return nil
}

func (s *taskJobServiceImpl) GetTask(id uint) (*model.Task, error) {
	return s.taskDao.GetById(id)
}

func (s *taskJobServiceImpl) UpdateTask(id uint, task *model.Task) error {
	if err := s.normalizeAndValidateTask(task); err != nil {
		return err
	}

	existingTask, err := s.taskDao.GetById(id)
	if err != nil {
		return err
	}

	active, err := s.hasActiveExecution(id)
	if err != nil {
		return err
	}
	if existingTask.Status == model.TaskStatusRunning || active {
		return errors.New("运行中或已启用调度的任务不能修改，请先停止或暂停")
	}

	existingTask.Name = task.Name
	existingTask.Type = task.Type
	existingTask.Shell = task.Shell
	existingTask.ExecutionMode = task.ExecutionMode
	existingTask.Command = task.Command
	existingTask.CommandType = task.CommandType
	existingTask.HostIDs = task.HostIDs
	existingTask.CronExpr = task.CronExpr
	existingTask.Remark = task.Remark
	existingTask.MaxHistoryKeep = task.MaxHistoryKeep
	existingTask.TaskCount = task.TaskCount

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(existingTask).Error; err != nil {
			return err
		}
		return dao.NewTaskDao(tx).ReplaceTaskWorks(tx, existingTask)
	})
}

func (s *taskJobServiceImpl) normalizeAndValidateTask(task *model.Task) error {
	task.Name = strings.TrimSpace(task.Name)
	task.HostIDs = strings.TrimSpace(task.HostIDs)
	task.CronExpr = strings.TrimSpace(task.CronExpr)
	if task.Name == "" {
		return errors.New("任务名称不能为空")
	}
	if task.Type != model.TaskTypeImmediate && task.Type != model.TaskTypeScheduled {
		return errors.New("任务类型仅支持立即执行或定时任务")
	}
	if task.Type == model.TaskTypeScheduled {
		if task.CronExpr == "" {
			return errors.New("定时任务必须包含定时表达式")
		}
		if _, err := cron.ParseStandard(task.CronExpr); err != nil {
			return fmt.Errorf("无效的Cron表达式: %w", err)
		}
	} else {
		task.CronExpr = ""
		task.NextRunTime = nil
	}
	hostIDs, err := parseUintList(task.HostIDs)
	if err != nil {
		return fmt.Errorf("主机配置无效: %w", err)
	}
	if len(hostIDs) == 0 {
		return errors.New("请至少选择一台执行主机")
	}
	var hostCount int64
	if err := s.db.Model(&cmdbmodel.CmdbHost{}).Where("id IN ?", hostIDs).Count(&hostCount).Error; err != nil {
		return err
	}
	if int(hostCount) != len(hostIDs) {
		return errors.New("部分执行主机不存在")
	}
	if task.ExecutionMode == 0 {
		task.ExecutionMode = 1
	}
	if task.ExecutionMode != 1 && task.ExecutionMode != 2 {
		return errors.New("执行内容来源无效")
	}
	if task.ExecutionMode == 2 {
		task.Command = strings.TrimSpace(task.Command)
		if task.Command == "" {
			return errors.New("直接命令不能为空")
		}
		if task.CommandType == 0 {
			task.CommandType = model.TemplateTypeCommand
		}
		if !model.IsSupportedTemplateType(task.CommandType) {
			return errors.New("不支持的命令语言")
		}
		task.Shell = ""
		task.TaskCount = 1
	} else {
		ids, err := parseUintList(task.Shell)
		if err != nil || len(ids) == 0 {
			return errors.New("请至少选择一个有效任务模板")
		}
		var count int64
		if err := s.db.Model(&model.TaskTemplate{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
			return err
		}
		if int(count) != len(ids) {
			return errors.New("部分任务模板不存在")
		}
		task.TaskCount = len(ids)
		task.Command = ""
		task.CommandType = 0
	}
	if task.MaxHistoryKeep <= 0 {
		task.MaxHistoryKeep = 20
	}
	if task.MaxHistoryKeep > 200 {
		return errors.New("最大日志记录数不能超过200")
	}
	return nil
}

func (s *taskJobServiceImpl) DeleteTask(id uint) error {
	task, err := s.taskDao.GetById(id)
	if err != nil {
		return err
	}
	active, err := s.hasActiveExecution(id)
	if err != nil {
		return err
	}
	if task.Status == model.TaskStatusRunning || active {
		return errors.New("运行中或已启用调度的任务不能删除，请先停止或暂停")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var historyIDs []uint
		if err := tx.Model(&model.TaskExecutionHistory{}).Where("task_id = ?", id).Pluck("id", &historyIDs).Error; err != nil {
			return err
		}
		if len(historyIDs) > 0 {
			if err := tx.Where("history_id IN ?", historyIDs).Delete(&model.TaskExecutionLog{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("task_id = ?", id).Delete(&model.TaskExecutionHistory{}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", id).Delete(&model.TaskWork{}).Error; err != nil {
			return errors.New("删除子任务失败: " + err.Error())
		}
		return tx.Where("id = ?", id).Delete(&model.Task{}).Error
	})
}

func (s *taskJobServiceImpl) hasActiveExecution(taskID uint) (bool, error) {
	var count int64
	err := s.db.Model(&model.TaskWork{}).
		Where("task_id = ? AND history_id > 0 AND status IN ?", taskID, []int{model.TaskStatusRunning, model.TaskStatusPaused}).
		Count(&count).Error
	return count > 0, err
}

func (s *taskJobServiceImpl) ListTasks(page, pageSize int, name string, status int) ([]model.Task, int64, error) {
	offset := (page - 1) * pageSize
	return s.taskDao.List(offset, pageSize, name, status)
}

func (s *taskJobServiceImpl) TaskNameExists(name string) (bool, error) {
	return s.taskDao.ExistsByName(name)
}

func (s *taskJobServiceImpl) GetTasksByName(name string) ([]model.Task, error) {
	return s.taskDao.GetTasksByName(name)
}

func (s *taskJobServiceImpl) GetTasksByType(taskType int) ([]model.Task, error) {
	return s.taskDao.GetTasksByType(taskType)
}

func (s *taskJobServiceImpl) GetTasksByStatus(status int) ([]model.Task, error) {
	return s.taskDao.GetTasksByStatus(status)
}

func (s *taskJobServiceImpl) GetNextExecutionTime(cronExpr string) (time.Time, error) {
	schedule, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return time.Time{}, err
	}

	return schedule.Next(time.Now()), nil
}

func (s *taskJobServiceImpl) GetTaskTemplatesWithStatus(taskId uint) ([]map[string]interface{}, error) {
	return s.taskDao.GetTaskTemplatesWithStatus(taskId)
}

// IncrementExecuteCount 增加任务执行次数
func (s *taskJobServiceImpl) IncrementExecuteCount(taskId uint) error {
	return s.db.Model(&model.Task{}).Where("id = ?", taskId).
		Update("execute_count", gorm.Expr("execute_count + ?", 1)).Error
}

// UpdateNextRunTime 更新下次执行时间
func (s *taskJobServiceImpl) UpdateNextRunTime(taskId uint, nextTime time.Time) error {
	return s.db.Model(&model.Task{}).Where("id = ?", taskId).
		Update("next_run_time", nextTime).Error
}

// UpdateTaskAfterExecution 任务执行完成后更新execute_count和next_run_time
func (s *taskJobServiceImpl) UpdateTaskAfterExecution(taskId uint, cronExpr string) error {
	// 增加执行次数
	if err := s.IncrementExecuteCount(taskId); err != nil {
		return err
	}

	// 如果是定时任务，更新下次执行时间
	if cronExpr != "" {
		nextTime, err := s.GetNextExecutionTime(cronExpr)
		if err != nil {
			return err
		}
		if err := s.UpdateNextRunTime(taskId, nextTime); err != nil {
			return err
		}
	}

	return nil
}

func (s *taskJobServiceImpl) ListTasksWithDetails(page, pageSize int, name string, status int) ([]model.TaskWithDetails, int64, error) {
	offset := (page - 1) * pageSize

	// 获取基本任务列表
	tasks, total, err := s.taskDao.List(offset, pageSize, name, status)
	if err != nil {
		return nil, 0, err
	}

	// 转换为TaskWithDetails格式
	var tasksWithDetails []model.TaskWithDetails
	for _, task := range tasks {
		taskDetail := model.TaskWithDetails{
			ID:             task.ID,
			Name:           task.Name,
			Type:           task.Type,
			Shell:          task.Shell,
			ExecutionMode:  task.ExecutionMode,
			Command:        task.Command,
			CommandType:    task.CommandType,
			HostIDs:        task.HostIDs,
			CronExpr:       task.CronExpr,
			Tasklog:        task.Tasklog,
			Status:         task.Status,
			Duration:       task.Duration,
			TaskCount:      task.TaskCount,
			ExecuteCount:   task.ExecuteCount,
			MaxHistoryKeep: task.MaxHistoryKeep,
			NextRunTime:    task.NextRunTime,
			Remark:         task.Remark,
			StartTime:      task.StartTime,
			EndTime:        task.EndTime,
			CreatedAt:      task.CreatedAt,
		}

		var activeWorks []model.TaskWork
		if err := s.db.Where("task_id = ? AND history_id > 0 AND status IN ?", task.ID, []int{model.TaskStatusRunning, model.TaskStatusPaused}).
			Order("history_id DESC, id ASC").Find(&activeWorks).Error; err != nil {
			return nil, 0, err
		}
		for _, work := range activeWorks {
			if work.HistoryID > taskDetail.CurrentHistoryID {
				taskDetail.CurrentHistoryID = work.HistoryID
			}
			if work.Status == model.TaskStatusRunning {
				taskDetail.ExecutionStatus = model.TaskStatusRunning
			} else if taskDetail.ExecutionStatus == 0 {
				taskDetail.ExecutionStatus = model.TaskStatusPaused
			}
		}

		// 获取关联的模板信息
		if task.Shell != "" {
			templates, err := s.getTemplatesByIds(task.Shell)
			if err == nil {
				taskDetail.Templates = templates
			}
		}

		// 获取关联的主机信息
		if task.HostIDs != "" {
			hosts, err := s.getHostsByIds(task.HostIDs)
			if err == nil {
				taskDetail.Hosts = hosts
			}
		}

		tasksWithDetails = append(tasksWithDetails, taskDetail)
	}

	return tasksWithDetails, total, nil
}

// getTemplatesByIds 根据模板ID字符串获取模板信息
func (s *taskJobServiceImpl) getTemplatesByIds(shellIds string) ([]model.TaskTemplateInfo, error) {
	if shellIds == "" {
		return []model.TaskTemplateInfo{}, nil
	}

	// 解析逗号分隔的ID字符串
	idStrs := strings.Split(shellIds, ",")
	var ids []uint
	for _, idStr := range idStrs {
		if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); err == nil {
			ids = append(ids, uint(id))
		}
	}

	if len(ids) == 0 {
		return []model.TaskTemplateInfo{}, nil
	}

	// 查询模板信息
	var templates []model.TaskTemplate
	if err := s.db.Where("id IN ?", ids).Find(&templates).Error; err != nil {
		return nil, err
	}

	// 转换为TaskTemplateInfo格式
	var templateInfos []model.TaskTemplateInfo
	for _, template := range templates {
		templateInfos = append(templateInfos, model.TaskTemplateInfo{
			ID:      template.ID,
			Name:    template.Name,
			Type:    template.Type,
			Content: template.Content,
		})
	}

	return templateInfos, nil
}

// getHostsByIds 根据主机ID字符串获取主机信息
func (s *taskJobServiceImpl) getHostsByIds(hostIds string) ([]model.HostInfo, error) {
	if hostIds == "" {
		return []model.HostInfo{}, nil
	}

	// 解析逗号分隔的ID字符串
	idStrs := strings.Split(hostIds, ",")
	var ids []uint
	for _, idStr := range idStrs {
		if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); err == nil {
			ids = append(ids, uint(id))
		}
	}

	if len(ids) == 0 {
		return []model.HostInfo{}, nil
	}

	// 查询主机信息
	var hosts []cmdbmodel.CmdbHost
	if err := s.db.Where("id IN ?", ids).Find(&hosts).Error; err != nil {
		return nil, err
	}

	// 转换为HostInfo格式
	var hostInfos []model.HostInfo
	for _, host := range hosts {
		hostInfos = append(hostInfos, model.HostInfo{
			ID:        host.ID,
			HostName:  host.HostName,
			PrivateIP: host.PrivateIP,
			PublicIP:  host.PublicIP,
			SSHIP:     host.SSHIP,
			Status:    host.Status,
		})
	}

	return hostInfos, nil
}
