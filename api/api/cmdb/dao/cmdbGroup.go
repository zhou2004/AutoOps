package dao

import (
	"dodevops-api/api/cmdb/model"
	"dodevops-api/common"

	"gorm.io/gorm"
)

type CmdbGroupDao struct {
	db *gorm.DB
}

func NewCmdbGroupDao() CmdbGroupDao {
	return CmdbGroupDao{
		db: common.GetDB(),
	}
}

func (d *CmdbGroupDao) GetCmdbGroupById(id uint) (model.CmdbGroup, error) {
	var group model.CmdbGroup
	err := d.db.Where("id = ?", id).First(&group).Error
	return group, err
}

func (d *CmdbGroupDao) GetCmdbGroupByName(name string) (model.CmdbGroup, error) {
	var group model.CmdbGroup
	err := d.db.Where("name LIKE ?", "%"+name+"%").First(&group).Error
	return group, err
}

func (d *CmdbGroupDao) CheckNameExists(name string) bool {
	var count int64
	d.db.Model(&model.CmdbGroup{}).Where("name LIKE ?", "%"+name+"%").Count(&count)
	return count > 0
}

func (d *CmdbGroupDao) CreateCmdbGroup(group *model.CmdbGroup) error {
	return d.db.Create(group).Error
}

func (d *CmdbGroupDao) UpdateCmdbGroup(id uint, group *model.CmdbGroup) error {
	return d.db.Model(&model.CmdbGroup{}).Where("id = ?", id).Updates(group).Error
}

func (d *CmdbGroupDao) DeleteCmdbGroup(id uint) error {
	return d.db.Delete(&model.CmdbGroup{}, id).Error
}

func (d *CmdbGroupDao) GetCmdbGroupList() []model.CmdbGroup {
	var list []model.CmdbGroup
	d.db.Find(&list)
	return list
}

// GetUserGroupIDs 获取用户所属的资产分组ID列表（通过 sys_admin_role）
func (d *CmdbGroupDao) GetUserGroupIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := d.db.Table("sys_admin_role").
		Joins("JOIN sys_role ON sys_role.id = sys_admin_role.role_id").
		Where("sys_admin_role.admin_id = ? AND sys_role.role_key = ?", userID, "admin").
		Pluck("sys_role.id", &ids).Error
	return ids, err
}

// IsAdmin 检查用户是否为管理员
func (d *CmdbGroupDao) IsAdmin(userID uint) (bool, error) {
	var count int64
	err := d.db.Table("sys_admin_role").
		Joins("JOIN sys_role ON sys_role.id = sys_admin_role.role_id").
		Where("sys_admin_role.admin_id = ? AND sys_role.role_key = ?", userID, "admin").
		Count(&count).Error
	return count > 0, err
}
