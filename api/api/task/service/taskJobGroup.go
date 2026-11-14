package service

import (
	"errors"
	"strings"
	"unicode/utf8"

	"dodevops-api/api/task/model"
	"gorm.io/gorm"
)

type TaskJobGroupService struct{ db *gorm.DB }

func NewTaskJobGroupService(db *gorm.DB) *TaskJobGroupService { return &TaskJobGroupService{db: db} }

func (s *TaskJobGroupService) List() ([]model.TaskJobGroupSummary, error) {
	groups := []model.TaskJobGroupSummary{}
	err := s.db.Table("task_job_group g").
		Select("g.id, g.name, g.created_at, g.updated_at, COUNT(t.id) AS task_count").
		Joins("LEFT JOIN task_job t ON t.group_id = g.id").
		Group("g.id, g.name, g.created_at, g.updated_at").Order("g.id ASC").Scan(&groups).Error
	return groups, err
}

func (s *TaskJobGroupService) Save(id uint, name string) (*model.TaskJobGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return nil, errors.New("分组名称须为 1–100 个字符")
	}
	var group model.TaskJobGroup
	if id > 0 {
		if err := s.db.First(&group, id).Error; err != nil {
			return nil, errors.New("分组不存在")
		}
	}
	var count int64
	if err := s.db.Model(&model.TaskJobGroup{}).Where("name = ? AND id <> ?", name, id).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("分组名称已存在")
	}
	group.Name = name
	if err := s.db.Save(&group).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// Delete retains jobs and their execution/history; only the grouping is cleared.
func (s *TaskJobGroupService) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var group model.TaskJobGroup
		if err := tx.First(&group, id).Error; err != nil {
			return errors.New("分组不存在")
		}
		if err := tx.Model(&model.Task{}).Where("group_id = ?", id).Update("group_id", nil).Error; err != nil {
			return err
		}
		return tx.Delete(&group).Error
	})
}
