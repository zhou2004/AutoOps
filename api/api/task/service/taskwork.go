package service

import (
	cmdbmodel "dodevops-api/api/cmdb/model"
	configcentermodel "dodevops-api/api/configcenter/model"
	"dodevops-api/api/task/dao"
	"dodevops-api/api/task/model"
	"dodevops-api/common"
	"dodevops-api/common/util"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type ITaskWorkService interface {
	StartJob(taskID uint) error // 启动任务
	RunNow(taskID uint, trigger int, operator string) (*model.TaskExecutionHistory, error)
	ScheduleTask(taskID uint) error
	PauseExecution(taskID uint) error
	ResumeExecution(taskID uint) error
	StopExecution(taskID uint, operator string) error
	StopJob(taskID, templateID uint) error                               // 停止任务
	GetJobLog(taskID, templateID uint) (string, error)                   // 获取任务日志
	GetJobStatus(taskID, templateID uint) (*model.TaskWorkStatus, error) // 获取任务状态
	ScheduleJob(job *model.TaskWork) error                               // 添加任务
}

type TaskWorkServiceImpl struct {
	dao            dao.TaskWorkDaoInterface
	taskJobService TaskJobService
}

func (s *TaskWorkServiceImpl) StartJob(taskID uint) error {
	_, err := s.RunNow(taskID, model.TaskTriggerManual, "System")
	return err
}

func (s *TaskWorkServiceImpl) RunNow(taskID uint, trigger int, operator string) (*model.TaskExecutionHistory, error) {
	taskQueue := GetTaskQueue()
	if taskQueue == nil || !taskQueue.IsRunning() {
		return nil, fmt.Errorf("任务队列服务未运行")
	}
	historyService := NewTaskHistoryService(common.GetDB())
	history, jobs, err := historyService.BeginExecution(taskID, trigger, operator)
	if err != nil {
		return nil, err
	}
	submitted := 0
	for i := range jobs {
		job := jobs[i]
		if err := taskQueue.Enqueue(&job, s.getTaskPriority(&job)); err != nil {
			now := time.Now()
			message := fmt.Sprintf("提交任务到队列失败: %v", err)
			_ = historyService.CompleteLog(&job, model.TaskStatusFailed, message, "", now, now)
			continue
		}
		submitted++
	}
	if submitted == 0 {
		_ = historyService.FinalizeExecution(history.ID)
		return history, fmt.Errorf("没有工作项成功进入执行队列")
	}
	return history, nil
}

func (s *TaskWorkServiceImpl) ScheduleTask(taskID uint) error {
	task, err := s.dao.GetTaskJobByID(taskID)
	if err != nil {
		return fmt.Errorf("获取任务信息失败: %v", err)
	}
	if task.Type != model.TaskTypeScheduled {
		return errors.New("只有定时任务可以启用调度")
	}
	if task.CronExpr == "" {
		return errors.New("定时任务必须包含Cron表达式")
	}
	scheduler := GetGlobalScheduler()
	if scheduler == nil || !scheduler.IsRunning() {
		return errors.New("全局调度器未运行")
	}
	if err := scheduler.AddScheduledTask(taskID, task.CronExpr, createScheduledTaskHandler(taskID)); err != nil {
		return err
	}
	nextTime, err := cron.ParseStandard(task.CronExpr)
	if err != nil {
		return err
	}
	return common.GetDB().Model(&model.Task{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status": model.TaskStatusRunning, "next_run_time": nextTime.Next(time.Now()),
	}).Error
}

// PauseExecution 暂停任务当前正在执行的所有远端进程；不影响 Cron 调度开关。
func (s *TaskWorkServiceImpl) PauseExecution(taskID uint) error {
	task, works, err := s.loadControllableWorks(taskID, []int{model.TaskStatusRunning})
	if err != nil {
		return err
	}
	historyIDs := make(map[uint]struct{})
	var controlErrors []error
	paused := 0
	for i := range works {
		work := &works[i]
		processFound, signalErr := s.signalRemoteProcess(work, "pause")
		if signalErr != nil {
			controlErrors = append(controlErrors, fmt.Errorf("工作项 %d 暂停失败: %w", work.ID, signalErr))
			continue
		}
		if !processFound {
			executionLog, logErr := NewTaskHistoryService(common.GetDB()).GetExecutionLog(work.HistoryID, work.ID)
			if logErr != nil {
				controlErrors = append(controlErrors, fmt.Errorf("工作项 %d 状态读取失败: %w", work.ID, logErr))
				continue
			}
			if executionLog.StartedAt != nil {
				controlErrors = append(controlErrors, fmt.Errorf("工作项 %d 的远端进程已结束，正在汇总结果", work.ID))
				continue
			}
		}
		if err := s.setExecutionWorkStatus(work, model.TaskStatusRunning, model.TaskStatusPaused); err != nil {
			if processFound {
				_, _ = s.signalRemoteProcess(work, "resume")
			}
			controlErrors = append(controlErrors, fmt.Errorf("工作项 %d 状态更新失败: %w", work.ID, err))
			continue
		}
		historyIDs[work.HistoryID] = struct{}{}
		paused++
	}
	for historyID := range historyIDs {
		if err := s.refreshExecutionAggregate(task, historyID); err != nil {
			controlErrors = append(controlErrors, fmt.Errorf("执行记录 %d 状态汇总失败: %w", historyID, err))
		}
	}
	if paused == 0 {
		if len(controlErrors) > 0 {
			return errors.Join(controlErrors...)
		}
		return errors.New("任务没有可暂停的执行")
	}
	if len(controlErrors) > 0 {
		return fmt.Errorf("已暂停 %d 个工作项，但部分工作项操作失败: %w", paused, errors.Join(controlErrors...))
	}
	return nil
}

// ResumeExecution 恢复任务当前已暂停的所有远端进程；不影响 Cron 调度开关。
func (s *TaskWorkServiceImpl) ResumeExecution(taskID uint) error {
	task, works, err := s.loadControllableWorks(taskID, []int{model.TaskStatusPaused})
	if err != nil {
		return err
	}
	historyIDs := make(map[uint]struct{})
	var controlErrors []error
	resumed := 0
	for i := range works {
		work := &works[i]
		processFound, signalErr := s.signalRemoteProcess(work, "resume")
		if signalErr != nil {
			controlErrors = append(controlErrors, fmt.Errorf("工作项 %d 恢复失败: %w", work.ID, signalErr))
			continue
		}
		if err := s.setExecutionWorkStatus(work, model.TaskStatusPaused, model.TaskStatusRunning); err != nil {
			if processFound {
				_, _ = s.signalRemoteProcess(work, "pause")
			}
			controlErrors = append(controlErrors, fmt.Errorf("工作项 %d 状态更新失败: %w", work.ID, err))
			continue
		}
		historyIDs[work.HistoryID] = struct{}{}
		resumed++
	}
	for historyID := range historyIDs {
		if err := s.refreshExecutionAggregate(task, historyID); err != nil {
			controlErrors = append(controlErrors, fmt.Errorf("执行记录 %d 状态汇总失败: %w", historyID, err))
		}
	}
	if resumed == 0 {
		if len(controlErrors) > 0 {
			return errors.Join(controlErrors...)
		}
		return errors.New("任务没有可恢复的执行")
	}
	if len(controlErrors) > 0 {
		return fmt.Errorf("已恢复 %d 个工作项，但部分工作项操作失败: %w", resumed, errors.Join(controlErrors...))
	}
	return nil
}

// StopExecution 停止任务当前执行。定时任务只停止本次执行，后续 Cron 调度保持不变。
func (s *TaskWorkServiceImpl) StopExecution(taskID uint, operator string) error {
	_, works, err := s.loadControllableWorks(taskID, []int{model.TaskStatusRunning, model.TaskStatusPaused})
	if err != nil {
		return err
	}
	historyService := NewTaskHistoryService(common.GetDB())
	historyIDs := make(map[uint]struct{})
	var controlErrors []error
	stopped := 0
	for i := range works {
		work := &works[i]
		if _, signalErr := s.signalRemoteProcess(work, "stop"); signalErr != nil {
			controlErrors = append(controlErrors, fmt.Errorf("工作项 %d 停止失败: %w", work.ID, signalErr))
			continue
		}
		now := time.Now()
		startedAt := now
		content := ""
		if executionLog, logErr := historyService.GetExecutionLog(work.HistoryID, work.ID); logErr == nil {
			content = strings.TrimSpace(executionLog.Log)
			if executionLog.StartedAt != nil {
				startedAt = *executionLog.StartedAt
			}
		}
		message := fmt.Sprintf("任务由 %s 于 %s 手动停止", operator, now.Format("2006-01-02 15:04:05"))
		if content != "" {
			content += "\n"
		}
		content += message
		if err := historyService.CompleteLog(work, model.TaskStatusStopped, content, "", startedAt, now); err != nil {
			controlErrors = append(controlErrors, fmt.Errorf("工作项 %d 停止状态保存失败: %w", work.ID, err))
			continue
		}
		historyIDs[work.HistoryID] = struct{}{}
		stopped++
	}
	for historyID := range historyIDs {
		if err := historyService.FinalizeExecution(historyID); err != nil {
			controlErrors = append(controlErrors, fmt.Errorf("执行记录 %d 汇总失败: %w", historyID, err))
		}
	}
	if stopped == 0 {
		if len(controlErrors) > 0 {
			return errors.Join(controlErrors...)
		}
		return errors.New("任务没有可停止的执行")
	}
	if len(controlErrors) > 0 {
		return fmt.Errorf("已停止 %d 个工作项，但部分工作项操作失败: %w", stopped, errors.Join(controlErrors...))
	}
	return nil
}

func (s *TaskWorkServiceImpl) loadControllableWorks(taskID uint, statuses []int) (*model.Task, []model.TaskWork, error) {
	task, err := s.dao.GetTaskJobByID(taskID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取任务失败: %w", err)
	}
	var works []model.TaskWork
	if err := common.GetDB().Where("task_id = ? AND history_id > 0 AND status IN ?", taskID, statuses).
		Order("id ASC").Find(&works).Error; err != nil {
		return nil, nil, err
	}
	if len(works) == 0 {
		return nil, nil, errors.New("任务当前没有符合条件的执行")
	}
	return task, works, nil
}

func (s *TaskWorkServiceImpl) setExecutionWorkStatus(work *model.TaskWork, fromStatus, toStatus int) error {
	return common.GetDB().Transaction(func(tx *gorm.DB) error {
		update := tx.Model(&model.TaskWork{}).Where("id = ? AND history_id = ? AND status = ?", work.ID, work.HistoryID, fromStatus).
			Update("status", toStatus)
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return errors.New("执行状态已经变化，请刷新后重试")
		}
		return tx.Model(&model.TaskExecutionLog{}).
			Where("history_id = ? AND task_work_id = ?", work.HistoryID, work.ID).
			Update("status", toStatus).Error
	})
}

func (s *TaskWorkServiceImpl) refreshExecutionAggregate(task *model.Task, historyID uint) error {
	var runningCount int64
	if err := common.GetDB().Model(&model.TaskWork{}).
		Where("task_id = ? AND history_id = ? AND status = ?", task.ID, historyID, model.TaskStatusRunning).
		Count(&runningCount).Error; err != nil {
		return err
	}
	status := model.TaskStatusPaused
	if runningCount > 0 {
		status = model.TaskStatusRunning
	}
	return common.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.TaskExecutionHistory{}).Where("id = ? AND finished_at IS NULL", historyID).
			Update("status", status).Error; err != nil {
			return err
		}
		if task.Type == model.TaskTypeImmediate {
			return tx.Model(&model.Task{}).Where("id = ?", task.ID).Update("status", status).Error
		}
		return nil
	})
}

// getTaskPriority 获取任务优先级
func (s *TaskWorkServiceImpl) getTaskPriority(job *model.TaskWork) string {
	// 获取父任务信息
	parentTask, err := s.dao.GetParentTask(job.TaskID)
	if err != nil {
		return "normal" // 默认优先级
	}

	// 根据任务类型确定优先级
	switch parentTask.Type {
	case model.TaskTypeImmediate: // 立即执行任务 - 高优先级
		return "high"
	case model.TaskTypeScheduled: // 定时任务 - 普通优先级
		return "normal"
	case model.TaskTypeAnsible: // Ansible任务 - 低优先级（可能耗时较长）
		return "low"
	default:
		return "normal"
	}
}

func (s *TaskWorkServiceImpl) executeJob(job *model.TaskWork) error {
	historyService := NewTaskHistoryService(common.GetDB())
	shouldRun, err := s.waitUntilExecutionRunnable(job)
	if err != nil {
		return err
	}
	if !shouldRun {
		return historyService.FinalizeExecution(job.HistoryID)
	}

	startTime := time.Now()
	_ = historyService.MarkLogStarted(job.HistoryID, job.ID, startTime)

	parentTask, err := s.dao.GetParentTask(job.TaskID)
	if err != nil {
		return s.finishFailedJob(historyService, job, startTime, fmt.Errorf("获取父任务状态失败: %v", err))
	}
	if parentTask.Type == model.TaskTypeImmediate && parentTask.Status == model.TaskStatusFailed {
		return s.finishFailedJob(historyService, job, startTime, errors.New("父任务已停止"))
	}

	template, host, err := s.getJobDetails(job)
	if err != nil {
		return s.finishFailedJob(historyService, job, startTime, fmt.Errorf("获取任务详情失败: %v", err))
	}

	logDir := filepath.Join(fmt.Sprintf("logs/task_%d", job.TaskID), fmt.Sprintf("history_%d", job.HistoryID))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return s.finishFailedJob(historyService, job, startTime, fmt.Errorf("创建日志目录失败: %v", err))
	}

	var logContent string
	var sshErr error
	maxRetries := 3
	for i := 0; i <= maxRetries; i++ {
		logContent, sshErr = s.executeSSHTask(job, template, host)
		if sshErr == nil {
			break
		}
		if !isSSHConnectionError(sshErr) || i == maxRetries {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if sshErr != nil {
		logContent = fmt.Sprintf("执行失败: %v\n%s", sshErr, logContent)
	}
	stoppedByUser := s.wasExecutionStopped(job)
	if stoppedByUser {
		sshErr = errors.New("任务已被手动停止")
		logContent = strings.TrimSpace(logContent) + "\n任务已被手动停止"
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("work_%d.log", job.ID))
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		sshErr = errors.Join(sshErr, fmt.Errorf("写入日志文件失败: %v", err))
		logPath = ""
	}

	endTime := time.Now()
	status := model.TaskStatusSuccess
	if stoppedByUser {
		status = model.TaskStatusStopped
	} else if sshErr != nil {
		status = model.TaskStatusFailed
	}
	if err := historyService.CompleteLog(job, status, logContent, logPath, startTime, endTime); err != nil {
		return err
	}
	return historyService.FinalizeExecution(job.HistoryID)
}

// waitUntilExecutionRunnable 让队列中的工作项遵守暂停/停止状态。
func (s *TaskWorkServiceImpl) waitUntilExecutionRunnable(job *model.TaskWork) (bool, error) {
	for {
		if job.HistoryID > 0 {
			var history model.TaskExecutionHistory
			if err := common.GetDB().Select("status", "finished_at").First(&history, job.HistoryID).Error; err != nil {
				return false, fmt.Errorf("读取执行记录状态失败: %w", err)
			}
			if history.FinishedAt != nil || history.Status == model.TaskStatusStopped {
				return false, nil
			}
		}
		current, err := s.dao.GetByID(job.ID)
		if err != nil {
			return false, fmt.Errorf("读取工作项状态失败: %w", err)
		}
		if job.HistoryID > 0 && current.HistoryID != job.HistoryID {
			return false, nil
		}
		switch current.Status {
		case model.TaskStatusPaused:
			time.Sleep(time.Second)
			continue
		case model.TaskStatusFailed, model.TaskStatusSuccess, model.TaskStatusStopped:
			return false, nil
		case model.TaskStatusPending, model.TaskStatusRunning:
			*job = *current
			return true, nil
		default:
			return false, fmt.Errorf("不支持的工作项状态: %d", current.Status)
		}
	}
}

func (s *TaskWorkServiceImpl) finishFailedJob(historyService *TaskHistoryService, job *model.TaskWork, startedAt time.Time, executionErr error) error {
	finishedAt := time.Now()
	content := executionErr.Error()
	status := model.TaskStatusFailed
	if s.wasExecutionStopped(job) {
		status = model.TaskStatusStopped
		content += "\n任务已被手动停止"
	}
	if err := historyService.CompleteLog(job, status, content, "", startedAt, finishedAt); err != nil {
		return errors.Join(executionErr, err)
	}
	if err := historyService.FinalizeExecution(job.HistoryID); err != nil {
		return errors.Join(executionErr, err)
	}
	// 已记录为业务失败，避免队列对脚本错误重复执行。
	return nil
}

func (s *TaskWorkServiceImpl) wasExecutionStopped(job *model.TaskWork) bool {
	currentWork, err := s.dao.GetByID(job.ID)
	if err == nil && currentWork.HistoryID == job.HistoryID && currentWork.Status == model.TaskStatusStopped {
		return true
	}
	if job.HistoryID == 0 {
		return false
	}
	var history model.TaskExecutionHistory
	return common.GetDB().Select("status").First(&history, job.HistoryID).Error == nil && history.Status == model.TaskStatusStopped
}

func (s *TaskWorkServiceImpl) StopJob(taskID, templateID uint) error {
	// 1. 获取任务详情并检查状态
	job, err := s.dao.GetByTaskAndTemplateID(taskID, templateID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %v", err)
	}
	if job.HistoryID > 0 {
		return s.StopExecution(taskID, "System")
	}

	// 2. 如果任务不是运行中状态(2)，直接返回错误
	if job.Status != 2 {
		return fmt.Errorf("任务当前状态为%d，无法停止非运行中的任务", job.Status)
	}

	// 3. 执行远程停止操作
	s.killRemoteProcess(job)

	// 4. 更新状态为异常(4)并记录结束时间
	now := time.Now()
	if err := s.dao.UpdateStatus(job.ID, 4); err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}

	// 5. 更新结束时间和耗时
	s.dao.UpdateTiming(job.ID, nil, &now, 0)

	// 6. 停止所有关联的子任务
	jobs, err := s.dao.GetByTaskID(taskID)
	if err == nil {
		// 首先停止当前任务
		s.dao.UpdateStatus(job.ID, 4)

		// 停止所有子任务
		for _, j := range jobs {
			if j.ID == job.ID {
				continue // 跳过当前任务，已经处理过
			}

			// 只停止运行中或等待中的任务
			if j.Status == 1 || j.Status == 2 {
				s.dao.UpdateStatus(j.ID, 4)
			}
		}

		// 更新父任务状态为已停止
		taskDao := dao.NewTaskDao(common.GetDB())
		taskDao.UpdateStatus(taskID, 4)
	}

	// 7. 记录操作日志
	logContent := fmt.Sprintf("任务被手动停止于 %s", now.Format(time.RFC3339))
	s.dao.UpdateLog(job.ID, logContent)

	return nil
}

func (s *TaskWorkServiceImpl) GetJobLog(taskID, templateID uint) (string, error) {
	job, err := s.dao.GetByTaskAndTemplateID(taskID, templateID)
	if err != nil {
		return "", fmt.Errorf("获取任务失败: %v", err)
	}

	if job.LogPath == "" {
		return "", fmt.Errorf("任务日志路径为空")
	}

	content, err := os.ReadFile(job.LogPath)
	if err != nil {
		return "", fmt.Errorf("读取日志文件失败: %v", err)
	}

	return string(content), nil
}

func (s *TaskWorkServiceImpl) GetJobStatus(taskID, templateID uint) (*model.TaskWorkStatus, error) {
	job, err := s.dao.GetByTaskAndTemplateID(taskID, templateID)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %v", err)
	}

	return &model.TaskWorkStatus{
		Status:    job.Status,
		Log:       job.Log,
		StartTime: job.StartTime.Format(time.RFC3339),
		EndTime:   job.EndTime.Format(time.RFC3339),
		Duration:  job.Duration,
	}, nil
}

func (s *TaskWorkServiceImpl) ScheduleJob(job *model.TaskWork) error {
	// 获取关联的TaskJob记录
	taskJob, err := s.dao.GetTaskJobByID(job.TaskID)
	if err != nil {
		return fmt.Errorf("获取任务信息失败: %v", err)
	}

	// 立即执行任务 (普通任务)
	if taskJob.Type == 1 {
		// 更新状态为运行中
		if err := s.dao.UpdateStatus(job.ID, 2); err != nil {
			return fmt.Errorf("更新任务状态失败: %v", err)
		}

		// 获取任务队列服务
		taskQueue := GetTaskQueue()
		if taskQueue == nil {
			// 如果队列服务不可用，回退到原始方式
			go func() {
				s.executeJob(job)
			}()
			return nil
		}

		// 提交到 Redis 队列
		priority := s.getTaskPriority(job)
		if err := taskQueue.Enqueue(job, priority); err != nil {
			return fmt.Errorf("提交任务到队列失败: %v", err)
		}

		return nil
	}

	// 定时任务处理
	if taskJob.Type == 2 {
		// 从task_job获取cron表达式并计算下次执行时间
		if taskJob.CronExpr == "" {
			return fmt.Errorf("定时任务必须包含cron表达式")
		}

		// 使用cron库解析cron表达式（标准5位格式：分 时 日 月 周）
		schedule, err := cron.ParseStandard(taskJob.CronExpr)
		if err != nil {
			return fmt.Errorf("解析cron表达式失败: %v", err)
		}

		// 计算下次执行时间
		nextTime := schedule.Next(time.Now())
		if nextTime.IsZero() {
			return fmt.Errorf("无法计算有效的下次执行时间")
		}

		job.ScheduledTime = &nextTime

		// 验证执行时间格式
		if job.ScheduledTime.Before(time.Now()) {
			return fmt.Errorf("定时任务执行时间(%v)不能早于当前时间(%v)",
				job.ScheduledTime.Format("2006-01-02 15:04:05"),
				time.Now().Format("2006-01-02 15:04:05"))
		}

		// 计算延迟时间
		delay := time.Until(*job.ScheduledTime)
		if delay <= 0 {
			return fmt.Errorf("定时任务执行时间已过期，请设置未来的时间")
		}

		// 先注册到全局 cron 调度器（在数据库操作之前，避免死锁）
		scheduler := GetGlobalScheduler()
		if scheduler == nil {
			return fmt.Errorf("全局调度器未初始化")
		}

		// 检查任务是否已在调度器中（防止重复注册）
		entries := scheduler.GetEntries()
		if _, exists := entries[job.TaskID]; exists {
			// 任务已注册，只更新数据库状态即可
			taskDao := dao.NewTaskDao(common.GetDB())
			if err := taskDao.UpdateStatus(job.TaskID, 2); err != nil {
				return fmt.Errorf("更新父任务状态失败: %v", err)
			}
			return nil
		}

		// 注册定时任务（使用统一的回调函数）
		if err := scheduler.AddScheduledTask(job.TaskID, taskJob.CronExpr, createScheduledTaskHandler(job.TaskID)); err != nil {
			return fmt.Errorf("注册定时任务失败: %v", err)
		}

		// 注册成功后，再更新数据库状态（避免在持有锁的情况下操作数据库）
		taskDao := dao.NewTaskDao(common.GetDB())
		if err := taskDao.UpdateStatus(job.TaskID, 2); err != nil {
			// 如果状态更新失败，移除已注册的任务
			scheduler.RemoveScheduledTask(job.TaskID)
			return fmt.Errorf("更新父任务状态失败: %v", err)
		}

		// 更新子任务的计划执行时间
		if err := s.dao.UpdateScheduledJob(job.ID, 1, job.ScheduledTime); err != nil {
			return fmt.Errorf("更新子任务状态失败: %v", err)
		}

		return nil
	}

	return fmt.Errorf("不支持的任务类型: %d (有效类型: 1=普通任务,2=定时任务,3=ansible任务)", taskJob.Type)
}

// Helper methods
func (s *TaskWorkServiceImpl) getJobDetails(job *model.TaskWork) (*model.TaskTemplate, *cmdbmodel.CmdbHost, error) {
	var host cmdbmodel.CmdbHost
	if err := common.GetDB().First(&host, job.HostID).Error; err != nil {
		return nil, nil, fmt.Errorf("获取主机失败: %v", err)
	}
	if job.HistoryID > 0 {
		executionLog, err := NewTaskHistoryService(common.GetDB()).GetExecutionLog(job.HistoryID, job.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("获取执行快照失败: %v", err)
		}
		return &model.TaskTemplate{
			ID: job.TemplateID, Name: executionLog.TemplateName,
			Type: executionLog.CommandType, Content: executionLog.CommandSnapshot,
		}, &host, nil
	}

	var template model.TaskTemplate
	if err := common.GetDB().First(&template, job.TemplateID).Error; err != nil {
		return nil, nil, fmt.Errorf("获取模板失败: %v", err)
	}

	return &template, &host, nil
}

func isSSHConnectionError(err error) bool {
	// 判断是否为SSH连接错误(包括超时)
	return err != nil && (strings.Contains(err.Error(), "connection") ||
		strings.Contains(err.Error(), "authentication") ||
		strings.Contains(err.Error(), "timeout"))
}

func (s *TaskWorkServiceImpl) executeSSHTask(job *model.TaskWork, template *model.TaskTemplate, host *cmdbmodel.CmdbHost) (string, error) {
	sshUtil := util.NewSSHUtil()
	sshConfig, err := taskSSHConfig(host)
	if err != nil {
		return "", err
	}

	basePath := remoteExecutionBase(job)
	extension, runner, err := commandRunner(template.Type, basePath, template.Content)
	if err != nil {
		return "", err
	}
	scriptFile := basePath + extension
	pidFile := basePath + ".pid"
	remoteLog := basePath + ".log"
	exitFile := basePath + ".exit"
	payload := base64.StdEncoding.EncodeToString([]byte(template.Content))
	wrappedRunner := fmt.Sprintf(
		"echo \"[$(date '+%%F %%T')] 任务开始\"; %s; code=$?; echo \"[$(date '+%%F %%T')] 任务结束，退出码: $code\"; echo \"$code\" > %s; exit \"$code\"",
		runner, shellQuote(exitFile),
	)
	bootstrap := fmt.Sprintf(
		"command -v setsid >/dev/null 2>&1 || { echo '目标服务器缺少 setsid，无法安全控制任务进程组'; exit 1; }; printf %%s %s | base64 -d > %s && chmod 700 %s || exit 1; rm -f %s %s %s; nohup setsid sh -c %s > %s 2>&1 & echo $! > %s",
		shellQuote(payload), shellQuote(scriptFile), shellQuote(scriptFile), shellQuote(pidFile), shellQuote(remoteLog), shellQuote(exitFile),
		shellQuote(wrappedRunner), shellQuote(remoteLog), shellQuote(pidFile),
	)
	_, err = sshUtil.ExecuteRemoteCommand(sshConfig, bootstrap)
	if err != nil {
		return "", fmt.Errorf("SSH执行失败: %v", err)
	}

	var pid string
	for i := 0; i < 3; i++ {
		checkCmd := fmt.Sprintf("test -f %s && cat %s", shellQuote(pidFile), shellQuote(pidFile))
		checkOutput, err := sshUtil.ExecuteRemoteCommand(sshConfig, checkCmd)
		if err == nil && checkOutput != "" {
			pid = strings.TrimSpace(checkOutput)
			break
		}
		time.Sleep(time.Second)
	}
	if pid == "" {
		return "", fmt.Errorf("无法获取有效的进程PID")
	}

	waitCmd := fmt.Sprintf("while kill -0 %s >/dev/null 2>&1; do sleep 1; done; for i in 1 2 3; do test -f %s && break; sleep 1; done", shellQuote(pid), shellQuote(exitFile))
	if _, err := sshUtil.ExecuteRemoteCommand(sshConfig, waitCmd); err != nil {
		return "", fmt.Errorf("等待任务完成失败: %v", err)
	}
	marker := "__AUTOOPS_EXIT_CODE__="
	finalLogCmd := fmt.Sprintf("cat %s 2>/dev/null; printf '\\n%s'; cat %s 2>/dev/null || printf 255", shellQuote(remoteLog), marker, shellQuote(exitFile))
	finalOutput, err := sshUtil.ExecuteRemoteCommand(sshConfig, finalLogCmd)
	if err != nil {
		return "", fmt.Errorf("获取完整日志失败: %v", err)
	}
	markerIndex := strings.LastIndex(finalOutput, marker)
	exitCode := 255
	logOutput := finalOutput
	if markerIndex >= 0 {
		logOutput = strings.TrimRight(finalOutput[:markerIndex], "\r\n")
		exitCodeText := strings.TrimSpace(finalOutput[markerIndex+len(marker):])
		if parsed, parseErr := strconv.Atoi(exitCodeText); parseErr == nil {
			exitCode = parsed
		}
	}
	cleanupCmd := fmt.Sprintf("rm -f %s %s %s %s", shellQuote(scriptFile), shellQuote(pidFile), shellQuote(exitFile), shellQuote(remoteLog))
	_, _ = sshUtil.ExecuteRemoteCommand(sshConfig, cleanupCmd)
	if exitCode != 0 {
		return logOutput, fmt.Errorf("远程命令退出码: %d", exitCode)
	}
	return logOutput, nil
}

func remoteExecutionBase(job *model.TaskWork) string {
	return fmt.Sprintf("/tmp/autoops_task_%d_%d_%d", job.TaskID, job.HistoryID, job.ID)
}

func commandRunner(templateType int, basePath, content string) (string, string, error) {
	switch templateType {
	case model.TemplateTypeShell:
		path := basePath + ".sh"
		return ".sh", "bash " + shellQuote(path), nil
	case model.TemplateTypePython:
		path := basePath + ".py"
		return ".py", "python3 " + shellQuote(path), nil
	case model.TemplateTypeAnsible:
		path := basePath + ".yml"
		return ".yml", "ansible-playbook " + shellQuote(path), nil
	case model.TemplateTypeJavaScript:
		path := basePath + ".js"
		return ".js", "node " + shellQuote(path), nil
	case model.TemplateTypePowerShell:
		path := basePath + ".ps1"
		return ".ps1", "pwsh -NoProfile -File " + shellQuote(path), nil
	case model.TemplateTypeGo:
		path := basePath + ".go"
		return ".go", "go run " + shellQuote(path), nil
	case model.TemplateTypeRuby:
		path := basePath + ".rb"
		return ".rb", "ruby " + shellQuote(path), nil
	case model.TemplateTypePHP:
		path := basePath + ".php"
		return ".php", "php " + shellQuote(path), nil
	case model.TemplateTypePerl:
		path := basePath + ".pl"
		return ".pl", "perl " + shellQuote(path), nil
	case model.TemplateTypeCommand:
		return ".command", "bash -lc " + shellQuote(content), nil
	default:
		return "", "", fmt.Errorf("不支持的模板语言: %d", templateType)
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// killRemoteProcess 保留旧接口，由新的进程组控制实现停止。
func (s *TaskWorkServiceImpl) killRemoteProcess(job *model.TaskWork) error {
	_, err := s.signalRemoteProcess(job, "stop")
	return err
}

// signalRemoteProcess 控制该工作项的整个远端进程组。
// processFound=false 表示工作项仍在队列中，尚未创建远端 PID。
func (s *TaskWorkServiceImpl) signalRemoteProcess(job *model.TaskWork, action string) (bool, error) {
	_, host, err := s.getJobDetails(job)
	if err != nil {
		return false, fmt.Errorf("获取主机信息失败: %w", err)
	}
	sshConfig, err := taskSSHConfig(host)
	if err != nil {
		return false, err
	}

	pidFile := remoteExecutionBase(job) + ".pid"
	var operation string
	switch action {
	case "pause":
		operation = `kill -STOP "-$PID" 2>/dev/null || kill -STOP "$PID"`
	case "resume":
		operation = `kill -CONT "-$PID" 2>/dev/null || kill -CONT "$PID"`
	case "stop":
		operation = `
			kill -TERM "-$PID" 2>/dev/null || kill -TERM "$PID" 2>/dev/null || true
			for i in 1 2 3 4 5; do
				kill -0 "$PID" 2>/dev/null || break
				sleep 1
			done
			if kill -0 "$PID" 2>/dev/null; then
				kill -KILL "-$PID" 2>/dev/null || kill -KILL "$PID"
			fi`
	default:
		return false, fmt.Errorf("不支持的进程控制操作: %s", action)
	}

	command := fmt.Sprintf(`
		for i in 1 2 3; do
			test -s %s && break
			sleep 1
		done
		if [ ! -s %s ]; then
			echo __AUTOOPS_PROCESS_NOT_STARTED__
			exit 0
		fi
		PID=$(cat %s)
		case "$PID" in (*[!0-9]*|'') echo "无效PID: $PID"; exit 1;; esac
		if ! kill -0 "$PID" 2>/dev/null; then
			echo __AUTOOPS_PROCESS_NOT_STARTED__
			exit 0
		fi
		%s
		echo __AUTOOPS_SIGNAL_OK__`, shellQuote(pidFile), shellQuote(pidFile), shellQuote(pidFile), operation)
	output, err := util.NewSSHUtil().ExecuteRemoteCommand(sshConfig, command)
	if err != nil {
		return false, fmt.Errorf("远端进程%s失败: %w; 输出: %s", action, err, strings.TrimSpace(output))
	}
	if strings.Contains(output, "__AUTOOPS_PROCESS_NOT_STARTED__") {
		return false, nil
	}
	if !strings.Contains(output, "__AUTOOPS_SIGNAL_OK__") {
		return false, fmt.Errorf("远端进程%s未返回确认信息", action)
	}
	return true, nil
}

func taskSSHConfig(host *cmdbmodel.CmdbHost) (*util.SSHConfig, error) {
	var ecsAuth configcentermodel.EcsAuth
	if err := common.GetDB().First(&ecsAuth, host.SSHKeyID).Error; err != nil {
		return nil, fmt.Errorf("获取SSH认证凭证失败: %w", err)
	}
	config := &util.SSHConfig{IP: host.SSHIP, Port: host.SSHPort, Username: host.SSHName, Type: ecsAuth.Type}
	switch ecsAuth.Type {
	case 1:
		if ecsAuth.Password == "" {
			return nil, errors.New("密码认证类型但密码为空")
		}
		config.Password = ecsAuth.Password
	case 2:
		if ecsAuth.PublicKey == "" {
			return nil, errors.New("密钥认证类型但公钥为空")
		}
		config.PublicKey = ecsAuth.PublicKey
	case 3:
	default:
		return nil, fmt.Errorf("不支持的SSH认证类型: %d", ecsAuth.Type)
	}
	return config, nil
}

// 辅助函数：获取任务类型名称
func getJobType(jobType int) string {
	switch jobType {
	case 1:
		return "立即执行"
	case 2:
		return "定时任务"
	default:
		return "未知类型"
	}
}

// 创建任务服务实例
func NewTaskWorkService() ITaskWorkService {
	return &TaskWorkServiceImpl{
		dao:            dao.TaskWorkDao(),
		taskJobService: NewTaskService(common.GetDB()),
	}
}
