package dao

import (
	"dodevops-api/api/task/model"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 引用taskwork.go中的TaskWorkDaoInterface接口
var _ TaskWorkDaoInterface = (*TaskWorkDaoImpl)(nil)

type TaskDao interface {
	Create(task *model.Task) error
	GetById(id uint) (*model.Task, error)
	Update(task *model.Task) error
	Delete(id uint) error
	List(offset, limit int, name string, status int) ([]model.Task, int64, error)
	ExistsByName(name string) (bool, error)
	GetTasksByName(name string) ([]model.Task, error)
	GetTasksByType(taskType int) ([]model.Task, error)
	GetTasksByStatus(status int) ([]model.Task, error)
	GetTaskTemplatesWithStatus(taskId uint) ([]map[string]interface{}, error)
	UpdateStartTime(id uint, startTime *time.Time) error
	UpdateEndTimeAndDuration(id uint, endTime *time.Time, duration int) error
	UpdateStatus(id uint, status int) error
	GetParentTask(taskID uint) (*model.Task, error)
	ReplaceTaskWorks(tx *gorm.DB, task *model.Task) error
}

type taskDaoImpl struct {
	db *gorm.DB
}

func NewTaskDao(db *gorm.DB) TaskDao {
	return &taskDaoImpl{db: db}
}

func (d *taskDaoImpl) Create(task *model.Task) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		// 创建主任务
		if err := tx.Create(task).Error; err != nil {
			return err
		}

		return createTaskWorks(tx, task)
	})
}

func createTaskWorks(tx *gorm.DB, task *model.Task) error {
	hostIDs := strings.Split(strings.TrimSpace(task.HostIDs), ",")
	if len(hostIDs) == 0 || (len(hostIDs) == 1 && hostIDs[0] == "") {
		return nil
	}

	templateIDs := strings.Split(strings.TrimSpace(task.Shell), ",")
	if task.ExecutionMode == 2 {
		templateIDs = []string{"0"}
	}
	if len(templateIDs) == 0 || (len(templateIDs) == 1 && templateIDs[0] == "") {
		return nil
	}

	var taskWorks []model.TaskWork
	for _, hostIDStr := range hostIDs {
		hostID, err := strconv.ParseUint(strings.TrimSpace(hostIDStr), 10, 32)
		if err != nil || hostID == 0 {
			return fmt.Errorf("无效的主机ID: %q", hostIDStr)
		}
		for _, templateIDStr := range templateIDs {
			templateID, err := strconv.ParseUint(strings.TrimSpace(templateIDStr), 10, 32)
			if err != nil || (templateID == 0 && task.ExecutionMode != 2) {
				return fmt.Errorf("无效的模板ID: %q", templateIDStr)
			}
			taskWorks = append(taskWorks, model.TaskWork{
				TaskID:     task.ID,
				HostID:     uint(hostID),
				TemplateID: uint(templateID),
				Type:       task.Type,
				Status:     model.TaskStatusPending,
				CreatedAt:  time.Now(),
			})
		}
	}
	if len(taskWorks) == 0 {
		return nil
	}
	return tx.CreateInBatches(taskWorks, 100).Error
}

func (d *taskDaoImpl) GetById(id uint) (*model.Task, error) {
	var task model.Task
	err := d.db.Where("id = ?", id).First(&task).Error
	return &task, err
}

func (d *taskDaoImpl) Update(task *model.Task) error {
	return d.db.Save(task).Error
}

func (d *taskDaoImpl) Delete(id uint) error {
	return d.db.Where("id = ?", id).Delete(&model.Task{}).Error
}

func (d *taskDaoImpl) List(offset, limit int, name string, status int) ([]model.Task, int64, error) {
	var tasks []model.Task
	var count int64

	query := d.db.Model(&model.Task{})
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if status > 0 {
		query = query.Where("status = ?", status)
	}

	err := query.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&tasks).Error
	return tasks, count, err
}

func (d *taskDaoImpl) ExistsByName(name string) (bool, error) {
	var count int64
	err := d.db.Model(&model.Task{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}

func (d *taskDaoImpl) GetTasksByName(name string) ([]model.Task, error) {
	var tasks []model.Task
	err := d.db.Where("name LIKE ?", "%"+name+"%").Find(&tasks).Error
	return tasks, err
}

func (d *taskDaoImpl) GetTasksByType(taskType int) ([]model.Task, error) {
	var tasks []model.Task
	err := d.db.Where("type = ?", taskType).Find(&tasks).Error
	return tasks, err
}

func (d *taskDaoImpl) GetTasksByStatus(status int) ([]model.Task, error) {
	var tasks []model.Task
	err := d.db.Where("status = ?", status).Find(&tasks).Error
	return tasks, err
}

// GetTaskTemplatesWithStatus 根据任务ID获取模板信息和任务状态
func (d *taskDaoImpl) GetTaskTemplatesWithStatus(taskId uint) ([]map[string]interface{}, error) {
	var result []map[string]interface{}

	// 查询任务关联的所有模板
	err := d.db.Table("task_template tt").
		Select("tt.id as template_id, tt.name as template_name, tt.remark as template_remark, tw.status as task_status, tw.task_id").
		Joins("JOIN task_work tw ON tt.id = tw.template_id").
		Where("tw.task_id = ?", taskId).
		Group("tt.id, tt.name, tt.remark, tw.status, tw.task_id").
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (d *taskDaoImpl) UpdateStartTime(id uint, startTime *time.Time) error {
	return d.db.Model(&model.Task{}).
		Where("id = ?", id).
		Update("start_time", startTime).Error
}

func (d *taskDaoImpl) UpdateEndTimeAndDuration(id uint, endTime *time.Time, duration int) error {
	return d.db.Model(&model.Task{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"end_time": endTime,
			"duration": duration,
		}).Error
}

func (d *taskDaoImpl) UpdateStatus(id uint, status int) error {
	return d.db.Model(&model.Task{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (d *taskDaoImpl) GetParentTask(taskID uint) (*model.Task, error) {
	var task model.Task
	err := d.db.Where("id = ?", taskID).First(&task).Error
	return &task, err
}

func (d *taskDaoImpl) ReplaceTaskWorks(tx *gorm.DB, task *model.Task) error {
	if tx == nil {
		tx = d.db
	}
	if err := tx.Where("task_id = ?", task.ID).Delete(&model.TaskWork{}).Error; err != nil {
		return err
	}
	return createTaskWorks(tx, task)
}
