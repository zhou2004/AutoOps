package dao

import (
	"dodevops-api/api/cmdb/model"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type CmdbAssetPermissionDao struct {
	db *gorm.DB
}

func NewCmdbAssetPermissionDao(db *gorm.DB) *CmdbAssetPermissionDao {
	return &CmdbAssetPermissionDao{db: db}
}

func (d *CmdbAssetPermissionDao) Create(data *model.CmdbAssetPermission) error {
	return d.db.Create(data).Error
}

func (d *CmdbAssetPermissionDao) Update(data *model.CmdbAssetPermission) error {
	return d.db.Save(data).Error
}

func (d *CmdbAssetPermissionDao) Delete(id uint) error {
	return d.db.Delete(&model.CmdbAssetPermission{}, id).Error
}

func (d *CmdbAssetPermissionDao) GetByID(id uint) (*model.CmdbAssetPermission, error) {
	var data model.CmdbAssetPermission
	err := d.db.First(&data, id).Error
	return &data, err
}

func (d *CmdbAssetPermissionDao) GetList(query model.CmdbAssetPermissionQuery) ([]model.CmdbAssetPermission, int64, error) {
	db := d.db.Model(&model.CmdbAssetPermission{})
	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.IsActive > 0 {
		db = db.Where("is_active = ?", query.IsActive)
	}
	var candidates []model.CmdbAssetPermission
	if err := db.Order("id DESC").Find(&candidates).Error; err != nil {
		return nil, 0, err
	}
	filtered := make([]model.CmdbAssetPermission, 0)
	for _, p := range candidates {
		if query.SubjectID > 0 && query.SubjectType == "user" && !JSONContainsID(p.UserIDs, query.SubjectID) {
			continue
		}
		if query.SubjectID > 0 && query.SubjectType == "group" && !JSONContainsID(p.GroupIDs, query.SubjectID) {
			continue
		}
		if query.AssetTypes != "" {
			var types []string
			json.Unmarshal([]byte(p.AssetTypes), &types)
			found := false
			for _, kind := range types {
				if kind == query.AssetTypes {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		filtered = append(filtered, p)
	}
	total := int64(len(filtered))
	page, size := query.Page, query.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	if size > 1000 {
		size = 1000
	}
	start := (page - 1) * size
	if start < 0 || start >= len(filtered) {
		return []model.CmdbAssetPermission{}, total, nil
	}
	end := start + size
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], total, nil
}

// GetUserPermissions matches complete JSON array members; substring matching can grant another user's rules.
func (d *CmdbAssetPermissionDao) GetUserPermissions(userID uint, userGroupIDs []uint) ([]model.CmdbAssetPermission, error) {
	active, err := d.UserActive(userID)
	if err != nil {
		return nil, err
	}
	if !active {
		return []model.CmdbAssetPermission{}, nil
	}
	return d.getUserPermissions(userID, userGroupIDs)
}

func (d *CmdbAssetPermissionDao) getUserPermissions(userID uint, userGroupIDs []uint) ([]model.CmdbAssetPermission, error) {
	var candidates []model.CmdbAssetPermission
	if err := d.db.Where("is_active = 1").Order("id DESC").Find(&candidates).Error; err != nil {
		return nil, err
	}
	list := make([]model.CmdbAssetPermission, 0)
	for _, p := range candidates {
		if !ruleEffective(p, time.Now()) {
			continue
		}
		matches := JSONContainsID(p.UserIDs, userID)
		for _, gid := range userGroupIDs {
			matches = matches || JSONContainsID(p.GroupIDs, gid)
		}
		if matches {
			list = append(list, p)
		}
	}
	return list, nil
}

// GetAll 获取所有授权
func (d *CmdbAssetPermissionDao) GetAll() ([]model.CmdbAssetPermission, error) {
	var list []model.CmdbAssetPermission
	err := d.db.Order("id DESC").Find(&list).Error
	return list, err
}
