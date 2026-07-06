package dao

import (
	"dodevops-api/api/task/model"

	"gorm.io/gorm"
)

type TaskAnsibleViewDao struct {
	DB *gorm.DB
}

func NewTaskAnsibleViewDao(db *gorm.DB) *TaskAnsibleViewDao {
	return &TaskAnsibleViewDao{DB: db}
}

// Create 创建视图
func (d *TaskAnsibleViewDao) Create(view *model.TaskAnsibleView) error {
	return d.DB.Create(view).Error
}

// Update 更新视图
func (d *TaskAnsibleViewDao) Update(view *model.TaskAnsibleView) error {
	return d.DB.Save(view).Error
}

// Delete 删除视图
func (d *TaskAnsibleViewDao) Delete(id uint) error {
	// 事务：删除视图同时将引用该视图的任务的 view_id 置空
	tx := d.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := tx.Model(&model.TaskAnsible{}).Where("view_id = ?", id).Update("view_id", nil).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&model.TaskAnsibleView{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// GetByID 根据ID获取视图
func (d *TaskAnsibleViewDao) GetByID(id uint) (*model.TaskAnsibleView, error) {
	var view model.TaskAnsibleView
	err := d.DB.First(&view, id).Error
	return &view, err
}

// GetAll 获取所有视图
func (d *TaskAnsibleViewDao) GetAll() ([]model.TaskAnsibleView, error) {
	var views []model.TaskAnsibleView
	err := d.DB.Order("id ASC").Find(&views).Error
	return views, err
}

// GetByName 根据名称查找视图
func (d *TaskAnsibleViewDao) GetByName(name string) (*model.TaskAnsibleView, error) {
	var view model.TaskAnsibleView
	err := d.DB.Where("name = ?", name).First(&view).Error
	return &view, err
}
