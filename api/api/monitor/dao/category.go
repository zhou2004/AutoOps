package dao

import (
	"dodevops-api/api/monitor/model"
	"dodevops-api/common"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

type MonitorCategoryDao struct{ db *gorm.DB }

func NewMonitorCategoryDao() *MonitorCategoryDao { return &MonitorCategoryDao{db: common.GetDB()} }
func categoryTarget(scope string) (interface{}, error) {
	switch scope {
	case "domain_cert":
		return &model.DomainCert{}, nil
	case "api_endpoint":
		return &model.MonitorAPIEndpoint{}, nil
	default:
		return nil, errors.New("无效的监控类型")
	}
}
func (d *MonitorCategoryDao) List(scope string) ([]model.MonitorCategory, error) {
	if _, err := categoryTarget(scope); err != nil {
		return nil, err
	}
	list := []model.MonitorCategory{}
	err := d.db.Where("scope = ?", scope).Order("id ASC").Find(&list).Error
	return list, err
}
func (d *MonitorCategoryDao) Validate(scope string, id uint) error {
	return validateCategory(d.db, scope, id)
}
func validateCategory(tx *gorm.DB, scope string, id uint) error {
	if _, err := categoryTarget(scope); err != nil {
		return err
	}
	if id == 0 {
		return nil
	}
	var category model.MonitorCategory
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND scope = ?", id, scope).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("分类不存在或不属于此监控类型")
		}
		return err
	}
	return nil
}
func (d *MonitorCategoryDao) Save(req *model.MonitorCategoryReq) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return errors.New("分类名称不能为空")
	}
	if _, err := categoryTarget(req.Scope); err != nil {
		return err
	}
	return d.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.MonitorCategory{}).Where("scope = ? AND name = ? AND id <> ?", req.Scope, req.Name, req.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("该分类名称已存在")
		}
		if req.ID == 0 {
			return tx.Create(&model.MonitorCategory{Scope: req.Scope, Name: req.Name, Description: req.Description}).Error
		}
		if err := validateCategory(tx, req.Scope, req.ID); err != nil {
			return err
		}
		return tx.Model(&model.MonitorCategory{}).Where("id = ?", req.ID).Updates(map[string]interface{}{"name": req.Name, "description": req.Description}).Error
	})
}

// 删除分类仅解除归类，不删除监控目标。
func (d *MonitorCategoryDao) Delete(scope string, id uint) error {
	target, err := categoryTarget(scope)
	if err != nil {
		return err
	}
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := validateCategory(tx, scope, id); err != nil {
			return err
		}
		if err := tx.Model(target).Where("category_id = ?", id).Update("category_id", 0).Error; err != nil {
			return err
		}
		return tx.Delete(&model.MonitorCategory{}, id).Error
	})
}
func (d *MonitorCategoryDao) Assign(req *model.MonitorCategoryAssignReq) error {
	target, err := categoryTarget(req.Scope)
	if err != nil {
		return err
	}
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := validateCategory(tx, req.Scope, req.CategoryID); err != nil {
			return err
		}
		unique := make(map[uint]bool)
		for _, id := range req.IDs {
			unique[id] = true
		}
		var count int64
		if err := tx.Model(target).Where("id IN ?", req.IDs).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(unique)) {
			return errors.New("部分监控目标已不存在，请刷新列表")
		}
		return tx.Model(target).Where("id IN ?", req.IDs).Update("category_id", req.CategoryID).Error
	})
}
