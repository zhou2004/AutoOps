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
	var list []model.CmdbAssetPermission
	var total int64
	db := d.db.Model(&model.CmdbAssetPermission{})

	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.IsActive > 0 {
		db = db.Where("is_active = ?", query.IsActive)
	}
	if query.AssetTypes != "" {
		db = db.Where("asset_types LIKE ?", "%"+query.AssetTypes+"%")
	}
	// 按授权主体筛选
	if query.SubjectType == "user" && query.SubjectID > 0 {
		idJSON, _ := json.Marshal(query.SubjectID)
		db = db.Where("user_ids LIKE ?", "%"+string(idJSON)+"%")
	}
	if query.SubjectType == "group" && query.SubjectID > 0 {
		idJSON, _ := json.Marshal(query.SubjectID)
		db = db.Where("group_ids LIKE ?", "%"+string(idJSON)+"%")
	}

	db.Count(&total)
	page, size := query.Page, query.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	err := db.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// GetUserPermissions 获取用户的所有有效授权
func (d *CmdbAssetPermissionDao) GetUserPermissions(userID uint, userGroupIDs []uint) ([]model.CmdbAssetPermission, error) {
	var list []model.CmdbAssetPermission
	db := d.db.Model(&model.CmdbAssetPermission{}).
		Where("is_active = 1").
		Where("(date_start IS NULL OR date_start = '' OR date_start <= ?)", time.Now().Format("2006-01-02")).
		Where("(date_expired IS NULL OR date_expired = '' OR date_expired >= ?)", time.Now().Format("2006-01-02"))

	// 构建查询条件：用户直接授权 OR 用户组授权
	userJSON, _ := json.Marshal(userID)
	userCond := "user_ids LIKE ?"
	args := []interface{}{"%" + string(userJSON) + "%"}

	for _, gid := range userGroupIDs {
		groupJSON, _ := json.Marshal(gid)
		userCond += " OR group_ids LIKE ?"
		args = append(args, "%"+string(groupJSON)+"%")
	}

	db = db.Where("("+userCond+")", args...)
	err := db.Order("id DESC").Find(&list).Error
	return list, err
}

// GetAll 获取所有授权
func (d *CmdbAssetPermissionDao) GetAll() ([]model.CmdbAssetPermission, error) {
	var list []model.CmdbAssetPermission
	err := d.db.Order("id DESC").Find(&list).Error
	return list, err
}
