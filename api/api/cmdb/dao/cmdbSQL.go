package dao

import (
	"dodevops-api/api/cmdb/model"
	configmodel "dodevops-api/api/configcenter/model"
	"gorm.io/gorm"
)

type CmdbSQLDao struct {
	db *gorm.DB
}

func NewCmdbSQLDao(db *gorm.DB) *CmdbSQLDao {
	return &CmdbSQLDao{db: db}
}

// Create 创建数据库记录
func (d *CmdbSQLDao) Create(db *model.CmdbSQL) error {
	return d.db.Create(db).Error
}

// Update 更新数据库记录
func (d *CmdbSQLDao) Update(db *model.CmdbSQL) error {
	return d.db.Save(db).Error
}

// Delete 删除数据库记录
func (d *CmdbSQLDao) Delete(id uint) error {
	return d.db.Delete(&model.CmdbSQL{}, id).Error
}

// GetByID 根据ID获取数据库记录
func (d *CmdbSQLDao) GetByID(id uint) (*model.CmdbSQL, error) {
	var db model.CmdbSQL
	err := d.db.First(&db, id).Error
	return &db, err
}

func (d *CmdbSQLDao) GetAccount(id uint) (*configmodel.AccountAuth, error) {
	var account configmodel.AccountAuth
	if err := d.db.First(&account, id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (d *CmdbSQLDao) GroupExists(id uint) (bool, error) {
	var count int64
	err := d.db.Model(&model.CmdbGroup{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

// List 分页查询数据库列表
func (d *CmdbSQLDao) List(page, pageSize int, scopes ...map[uint]bool) ([]model.CmdbSQL, int64, error) {
	var dbs []model.CmdbSQL
	var count int64

	query := ApplyAssetIDScope(d.db.Model(&model.CmdbSQL{}), scopes...)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&dbs).Error
	return dbs, count, err
}

// GetByAccountID 根据账号ID查询数据库
func (d *CmdbSQLDao) GetByAccountID(accountID uint, scopes ...map[uint]bool) ([]model.CmdbSQL, error) {
	var dbs []model.CmdbSQL
	err := ApplyAssetIDScope(d.db, scopes...).Where("account_id = ?", accountID).Find(&dbs).Error
	return dbs, err
}

// GetByGroupID 根据业务组ID查询数据库
func (d *CmdbSQLDao) GetByGroupID(groupID uint, scopes ...map[uint]bool) ([]model.CmdbSQL, error) {
	var dbs []model.CmdbSQL
	err := ApplyAssetIDScope(d.db, scopes...).Where("group_id = ?", groupID).Find(&dbs).Error
	return dbs, err
}

// GetByName 根据名称查询数据库
func (d *CmdbSQLDao) GetByName(name string, scopes ...map[uint]bool) ([]model.CmdbSQL, error) {
	var dbs []model.CmdbSQL
	err := ApplyAssetIDScope(d.db, scopes...).Where("name = ?", name).Find(&dbs).Error
	return dbs, err
}

// GetByType 根据类型查询数据库
func (d *CmdbSQLDao) GetByType(dbType int, scopes ...map[uint]bool) ([]model.CmdbSQL, error) {
	var dbs []model.CmdbSQL
	err := ApplyAssetIDScope(d.db, scopes...).Where("type = ?", dbType).Find(&dbs).Error
	return dbs, err
}
