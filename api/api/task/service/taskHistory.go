package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cmdbmodel "dodevops-api/api/cmdb/model"
	"dodevops-api/api/task/dao"
	"dodevops-api/api/task/model"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type TaskHistoryService struct {
	db *gorm.DB
}

type TaskHistoryList struct {
	List  []model.TaskExecutionHistory `json:"list"`
	Total int64                        `json:"total"`
}

func NewTaskHistoryService(db *gorm.DB) *TaskHistoryService {
	return &TaskHistoryService{db: db}
}

// BeginExecution 固化本次执行配置，并为每个工作项创建独立日志记录。
func (s *TaskHistoryService) BeginExecution(taskID uint, trigger int, operator string) (*model.TaskExecutionHistory, []model.TaskWork, error) {
	var history model.TaskExecutionHistory
	var works []model.TaskWork
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var task model.Task
		if err := tx.First(&task, taskID).Error; err != nil {
			return err
		}
		if task.ExecutionMode == 0 {
			task.ExecutionMode = 1
		}
		if task.MaxHistoryKeep <= 0 {
			task.MaxHistoryKeep = 20
		}

		if err := tx.Where("task_id = ?", taskID).Order("id ASC").Find(&works).Error; err != nil {
			return err
		}
		if len(works) == 0 {
			return errors.New("任务没有可执行的工作项")
		}
		for _, work := range works {
			if work.Status == model.TaskStatusRunning || work.Status == model.TaskStatusPaused {
				return errors.New("任务已有工作项正在运行，请稍后重试")
			}
		}

		templates, templateMap, err := loadTaskTemplateSnapshots(tx, &task)
		if err != nil {
			return err
		}
		snapshotJSON, err := json.Marshal(templates)
		if err != nil {
			return fmt.Errorf("序列化模板快照失败: %w", err)
		}

		now := time.Now()
		history = model.TaskExecutionHistory{
			TaskID:            task.ID,
			TaskName:          task.Name,
			Trigger:           trigger,
			Operator:          operator,
			Status:            model.TaskStatusRunning,
			ExecutionMode:     task.ExecutionMode,
			Command:           task.Command,
			CommandType:       task.CommandType,
			MaxHistoryKeep:    task.MaxHistoryKeep,
			Shell:             task.Shell,
			HostIDs:           task.HostIDs,
			CronExpr:          task.CronExpr,
			Remark:            task.Remark,
			TemplateSnapshots: string(snapshotJSON),
			StartedAt:         now,
		}
		if err := tx.Create(&history).Error; err != nil {
			return err
		}

		hostNames, err := loadTaskHostNames(tx, works)
		if err != nil {
			return err
		}
		logs := make([]model.TaskExecutionLog, 0, len(works))
		for _, work := range works {
			commandType := task.CommandType
			command := task.Command
			templateName := "直接命令"
			if task.ExecutionMode != 2 {
				snapshot, ok := templateMap[work.TemplateID]
				if !ok {
					return fmt.Errorf("任务模板 %d 不存在", work.TemplateID)
				}
				commandType = snapshot.Type
				command = snapshot.Content
				templateName = snapshot.Name
			}
			logs = append(logs, model.TaskExecutionLog{
				HistoryID:       history.ID,
				TaskID:          task.ID,
				TaskWorkID:      work.ID,
				TemplateID:      work.TemplateID,
				HostID:          work.HostID,
				HostName:        hostNames[work.HostID],
				TemplateName:    templateName,
				CommandType:     commandType,
				CommandSnapshot: command,
				Status:          model.TaskStatusPending,
			})
		}
		if err := tx.CreateInBatches(logs, 100).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.TaskWork{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
			"history_id": history.ID,
			"status":     model.TaskStatusRunning,
			"log":        "",
			"log_path":   "",
			"start_time": nil,
			"end_time":   nil,
			"duration":   0,
		}).Error; err != nil {
			return err
		}
		if task.Type == model.TaskTypeImmediate {
			if err := tx.Model(&model.Task{}).Where("id = ?", taskID).Updates(map[string]interface{}{
				"status":     model.TaskStatusRunning,
				"start_time": now,
			}).Error; err != nil {
				return err
			}
		} else if err := tx.Model(&model.Task{}).Where("id = ?", taskID).Update("start_time", now).Error; err != nil {
			return err
		}
		return tx.Where("task_id = ?", taskID).Order("id ASC").Find(&works).Error
	})
	return &history, works, err
}

func loadTaskTemplateSnapshots(tx *gorm.DB, task *model.Task) ([]model.TaskTemplateSnapshot, map[uint]model.TaskTemplateSnapshot, error) {
	result := []model.TaskTemplateSnapshot{}
	byID := make(map[uint]model.TaskTemplateSnapshot)
	if task.ExecutionMode == 2 {
		return result, byID, nil
	}
	ids, err := parseUintList(task.Shell)
	if err != nil || len(ids) == 0 {
		return nil, nil, errors.New("任务未配置有效模板")
	}
	var templates []model.TaskTemplate
	if err := tx.Where("id IN ?", ids).Find(&templates).Error; err != nil {
		return nil, nil, err
	}
	found := make(map[uint]model.TaskTemplate, len(templates))
	for _, template := range templates {
		found[template.ID] = template
	}
	for _, id := range ids {
		template, ok := found[id]
		if !ok {
			return nil, nil, fmt.Errorf("任务模板 %d 不存在", id)
		}
		snapshot := model.TaskTemplateSnapshot{
			ID: template.ID, Name: template.Name, Type: template.Type,
			Content: template.Content, Remark: template.Remark,
		}
		result = append(result, snapshot)
		byID[id] = snapshot
	}
	return result, byID, nil
}

func loadTaskHostNames(tx *gorm.DB, works []model.TaskWork) (map[uint]string, error) {
	ids := make([]uint, 0, len(works))
	seen := make(map[uint]struct{})
	for _, work := range works {
		if _, ok := seen[work.HostID]; !ok {
			seen[work.HostID] = struct{}{}
			ids = append(ids, work.HostID)
		}
	}
	var hosts []cmdbmodel.CmdbHost
	if err := tx.Where("id IN ?", ids).Find(&hosts).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]string, len(hosts))
	for _, host := range hosts {
		name := host.HostName
		if name == "" {
			name = host.SSHIP
		}
		result[host.ID] = name
	}
	return result, nil
}

func (s *TaskHistoryService) MarkLogStarted(historyID, taskWorkID uint, startedAt time.Time) error {
	return s.db.Model(&model.TaskExecutionLog{}).
		Where("history_id = ? AND task_work_id = ?", historyID, taskWorkID).
		Updates(map[string]interface{}{"status": model.TaskStatusRunning, "started_at": startedAt}).Error
}

func (s *TaskHistoryService) GetExecutionLog(historyID, taskWorkID uint) (*model.TaskExecutionLog, error) {
	var executionLog model.TaskExecutionLog
	if err := s.db.Where("history_id = ? AND task_work_id = ?", historyID, taskWorkID).First(&executionLog).Error; err != nil {
		return nil, err
	}
	return &executionLog, nil
}

func (s *TaskHistoryService) CompleteLog(job *model.TaskWork, status int, content, logPath string, startedAt, finishedAt time.Time) error {
	duration := int(finishedAt.Sub(startedAt).Seconds())
	if duration < 0 {
		duration = 0
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.TaskExecutionLog{}).
			Where("history_id = ? AND task_work_id = ?", job.HistoryID, job.ID).
			Updates(map[string]interface{}{
				"status": status, "log": content, "log_path": logPath,
				"started_at": startedAt, "finished_at": finishedAt, "duration": duration,
			}).Error; err != nil {
			return err
		}
		return tx.Model(&model.TaskWork{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
			"status": status, "log": content, "log_path": logPath,
			"start_time": startedAt, "end_time": finishedAt, "duration": duration,
		}).Error
	})
}

// FinalizeExecution 在全部工作项结束后汇总状态；重复调用是幂等的。
func (s *TaskHistoryService) FinalizeExecution(historyID uint) error {
	if historyID == 0 {
		return nil
	}
	var taskID uint
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var history model.TaskExecutionHistory
		if err := tx.First(&history, historyID).Error; err != nil {
			return err
		}
		if history.FinishedAt != nil {
			return nil
		}
		var unfinished int64
		if err := tx.Model(&model.TaskExecutionLog{}).Where("history_id = ? AND status IN ?", historyID, []int{model.TaskStatusPending, model.TaskStatusRunning, model.TaskStatusPaused}).Count(&unfinished).Error; err != nil {
			return err
		}
		if unfinished > 0 {
			return nil
		}
		var failed int64
		if err := tx.Model(&model.TaskExecutionLog{}).Where("history_id = ? AND status = ?", historyID, model.TaskStatusFailed).Count(&failed).Error; err != nil {
			return err
		}
		var stopped int64
		if err := tx.Model(&model.TaskExecutionLog{}).Where("history_id = ? AND status = ?", historyID, model.TaskStatusStopped).Count(&stopped).Error; err != nil {
			return err
		}
		status := model.TaskStatusSuccess
		if stopped > 0 {
			status = model.TaskStatusStopped
		} else if failed > 0 {
			status = model.TaskStatusFailed
		}
		finishedAt := time.Now()
		duration := int(finishedAt.Sub(history.StartedAt).Seconds())
		update := tx.Model(&model.TaskExecutionHistory{}).
			Where("id = ? AND finished_at IS NULL", historyID).
			Updates(map[string]interface{}{"status": status, "finished_at": finishedAt, "duration": duration})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return nil
		}

		var task model.Task
		if err := tx.First(&task, history.TaskID).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{
			"execute_count": gorm.Expr("execute_count + 1"),
			"duration":      duration,
			"end_time":      finishedAt,
		}
		if task.Type == model.TaskTypeImmediate {
			updates["status"] = status
		} else {
			if schedule, err := cron.ParseStandard(task.CronExpr); err == nil {
				updates["next_run_time"] = schedule.Next(finishedAt)
			}
			if err := tx.Model(&model.TaskWork{}).Where("task_id = ? AND history_id = ?", task.ID, historyID).Update("status", model.TaskStatusPending).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&model.Task{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
			return err
		}
		taskID = task.ID
		return nil
	})
	if err != nil || taskID == 0 {
		return err
	}
	return s.cleanupHistory(taskID)
}

func (s *TaskHistoryService) List(taskID uint, page, pageSize, status int) (*TaskHistoryList, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 10
	}
	query := s.db.Model(&model.TaskExecutionHistory{}).Where("task_id = ?", taskID)
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []model.TaskExecutionHistory
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &TaskHistoryList{List: list, Total: total}, nil
}

func (s *TaskHistoryService) Detail(taskID, historyID uint) (*model.TaskExecutionHistory, error) {
	var history model.TaskExecutionHistory
	if err := s.db.Preload("Logs", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		Where("id = ? AND task_id = ?", historyID, taskID).First(&history).Error; err != nil {
		return nil, err
	}
	return &history, nil
}

func (s *TaskHistoryService) Rollback(taskID, historyID uint, operator string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var history model.TaskExecutionHistory
		if err := tx.Where("id = ? AND task_id = ?", historyID, taskID).First(&history).Error; err != nil {
			return err
		}
		var task model.Task
		if err := tx.First(&task, taskID).Error; err != nil {
			return err
		}
		if task.Status == model.TaskStatusRunning {
			return errors.New("运行中或已启用调度的任务不能回滚，请先停止或暂停")
		}
		var activeWorkCount int64
		if err := tx.Model(&model.TaskWork{}).
			Where("task_id = ? AND history_id > 0 AND status IN ?", taskID, []int{model.TaskStatusRunning, model.TaskStatusPaused}).
			Count(&activeWorkCount).Error; err != nil {
			return err
		}
		if activeWorkCount > 0 {
			return errors.New("任务当前仍有运行中或已暂停的执行，不能回滚")
		}

		shell := history.Shell
		if history.ExecutionMode != 2 {
			var snapshots []model.TaskTemplateSnapshot
			if err := json.Unmarshal([]byte(history.TemplateSnapshots), &snapshots); err != nil {
				return fmt.Errorf("解析历史模板快照失败: %w", err)
			}
			mappedIDs := make([]string, 0, len(snapshots))
			for _, snapshot := range snapshots {
				var existing model.TaskTemplate
				err := tx.First(&existing, snapshot.ID).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					existing = model.TaskTemplate{Name: snapshot.Name, Type: snapshot.Type, Content: snapshot.Content, Remark: snapshot.Remark, CreatedBy: operator, UpdatedBy: operator}
					if err := tx.Create(&existing).Error; err != nil {
						return err
					}
				} else if err != nil {
					return err
				} else if err := tx.Model(&existing).Updates(map[string]interface{}{
					"name": snapshot.Name, "type": snapshot.Type, "content": snapshot.Content,
					"remark": snapshot.Remark, "updated_by": operator,
				}).Error; err != nil {
					return err
				}
				mappedIDs = append(mappedIDs, strconv.FormatUint(uint64(existing.ID), 10))
			}
			shell = strings.Join(mappedIDs, ",")
		}

		task.Name = history.TaskName
		task.ExecutionMode = history.ExecutionMode
		task.Command = history.Command
		task.CommandType = history.CommandType
		task.Shell = shell
		task.HostIDs = history.HostIDs
		task.CronExpr = history.CronExpr
		task.Remark = history.Remark
		if history.MaxHistoryKeep > 0 {
			task.MaxHistoryKeep = history.MaxHistoryKeep
		}
		if err := tx.Model(&task).Select("name", "execution_mode", "command", "command_type", "shell", "host_ids", "cron_expr", "remark", "max_history_keep").Updates(&task).Error; err != nil {
			return err
		}
		if err := dao.NewTaskDao(tx).ReplaceTaskWorks(tx, &task); err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&model.TaskExecutionHistory{}).Where("id = ?", historyID).Updates(map[string]interface{}{
			"rolled_back_at": now, "rolled_back_by": operator,
		}).Error
	})
}

func (s *TaskHistoryService) cleanupHistory(taskID uint) error {
	var task model.Task
	if err := s.db.First(&task, taskID).Error; err != nil {
		return err
	}
	keep := task.MaxHistoryKeep
	if keep <= 0 {
		keep = 20
	}
	var old []model.TaskExecutionHistory
	if err := s.db.Where("task_id = ?", taskID).Order("id DESC").Offset(keep).Find(&old).Error; err != nil {
		return err
	}
	if len(old) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(old))
	for _, history := range old {
		ids = append(ids, history.ID)
	}
	var logs []model.TaskExecutionLog
	if err := s.db.Where("history_id IN ?", ids).Find(&logs).Error; err != nil {
		return err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("history_id IN ?", ids).Delete(&model.TaskExecutionLog{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&model.TaskExecutionHistory{}).Error
	}); err != nil {
		return err
	}
	allowedPrefix := filepath.Clean(fmt.Sprintf("logs/task_%d", taskID)) + string(os.PathSeparator)
	for _, logRecord := range logs {
		cleanPath := filepath.Clean(logRecord.LogPath)
		if strings.HasPrefix(cleanPath, allowedPrefix) {
			_ = os.Remove(cleanPath)
		}
	}
	return nil
}

func parseUintList(value string) ([]uint, error) {
	parts := strings.Split(strings.TrimSpace(value), ",")
	result := make([]uint, 0, len(parts))
	seen := make(map[uint]struct{})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseUint(part, 10, 32)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("无效ID: %q", part)
		}
		uid := uint(id)
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		result = append(result, uid)
	}
	return result, nil
}
