package dao

import (
	"dodevops-api/api/cmdb/model"
	"dodevops-api/common"
	"encoding/json"

	"gorm.io/gorm"
)

type CmdbUserGroupDao struct {
	db *gorm.DB
}

func NewCmdbUserGroupDao() *CmdbUserGroupDao {
	return &CmdbUserGroupDao{db: common.GetDB()}
}

func (d *CmdbUserGroupDao) Create(group *model.CmdbUserGroup) error {
	return d.db.Create(group).Error
}

func (d *CmdbUserGroupDao) Update(id uint, updates map[string]interface{}) error {
	return d.db.Model(&model.CmdbUserGroup{}).Where("id = ?", id).Updates(updates).Error
}

func (d *CmdbUserGroupDao) Delete(id uint) error {
	tx := d.db.Begin()
	if err := tx.Where("group_id = ?", id).Delete(&model.CmdbUserGroupMember{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Delete(&model.CmdbUserGroup{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (d *CmdbUserGroupDao) GetByID(id uint) (*model.CmdbUserGroup, error) {
	var group model.CmdbUserGroup
	err := d.db.First(&group, id).Error
	return &group, err
}

func (d *CmdbUserGroupDao) GetList(query model.CmdbUserGroupQuery) ([]model.CmdbUserGroupVo, int64, error) {
	var total int64
	var vos []model.CmdbUserGroupVo

	db := d.db.Table("cmdb_user_group ug")
	if query.Name != "" {
		db = db.Where("ug.name LIKE ?", "%"+query.Name+"%")
	}
	if query.Code != "" {
		db = db.Where("ug.code LIKE ?", "%"+query.Code+"%")
	}

	db.Count(&total)

	page := query.Page
	if page <= 0 {
		page = 1
	}
	size := query.Size
	if size <= 0 {
		size = 10
	}
	offset := (page - 1) * size

	err := db.Select(`ug.*, (SELECT COUNT(*) FROM cmdb_user_group_member m WHERE m.group_id = ug.id) as member_count`).
		Order("ug.id DESC").
		Offset(offset).Limit(size).
		Scan(&vos).Error
	return vos, total, err
}

func (d *CmdbUserGroupDao) AddMembers(groupID uint, userIDs []uint) error {
	for _, uid := range userIDs {
		member := &model.CmdbUserGroupMember{GroupID: groupID, UserID: uid}
		if err := d.db.Create(member).Error; err != nil {
			// 跳过重复
			continue
		}
	}
	return nil
}

func (d *CmdbUserGroupDao) RemoveMember(groupID, userID uint) error {
	return d.db.Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&model.CmdbUserGroupMember{}).Error
}

func (d *CmdbUserGroupDao) GetMembers(groupID uint) ([]model.CmdbGroupMemberVo, error) {
	var members []model.CmdbGroupMemberVo
	err := d.db.Table("cmdb_user_group_member m").
		Select("m.id, m.user_id, a.username, a.nickname, m.created_at").
		Joins("JOIN sys_admin a ON a.id = m.user_id").
		Where("m.group_id = ?", groupID).
		Order("m.id ASC").
		Scan(&members).Error
	return members, err
}

func (d *CmdbUserGroupDao) GetUserGroupIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := d.db.Table("cmdb_user_group_member").
		Where("user_id = ?", userID).
		Pluck("group_id", &ids).Error
	return ids, err
}

func (d *CmdbUserGroupDao) GetUserGroups(userID uint) ([]model.CmdbUserGroup, error) {
	var groups []model.CmdbUserGroup
	err := d.db.Table("cmdb_user_group ug").
		Joins("JOIN cmdb_user_group_member m ON m.group_id = ug.id").
		Where("m.user_id = ?", userID).
		Find(&groups).Error
	return groups, err
}

func (d *CmdbUserGroupDao) GetAll() ([]model.CmdbUserGroup, error) {
	var list []model.CmdbUserGroup
	err := d.db.Where("status = 1").Order("name ASC").Find(&list).Error
	return list, err
}

// CmdbCredentialPermissionDao 凭据授权 DAO
type CmdbCredentialPermissionDao struct {
	db *gorm.DB
}

func NewCmdbCredentialPermissionDao() *CmdbCredentialPermissionDao {
	return &CmdbCredentialPermissionDao{db: common.GetDB()}
}

func (d *CmdbCredentialPermissionDao) Create(data *model.CmdbCredentialPermission) error {
	return d.db.Create(data).Error
}

func (d *CmdbCredentialPermissionDao) Update(data *model.CmdbCredentialPermission) error {
	return d.db.Save(data).Error
}

func (d *CmdbCredentialPermissionDao) Delete(id uint) error {
	return d.db.Delete(&model.CmdbCredentialPermission{}, id).Error
}

func (d *CmdbCredentialPermissionDao) GetByID(id uint) (*model.CmdbCredentialPermission, error) {
	var data model.CmdbCredentialPermission
	err := d.db.First(&data, id).Error
	return &data, err
}

func (d *CmdbCredentialPermissionDao) GetList(query model.CmdbCredentialPermissionQuery) ([]model.CmdbCredentialPermission, int64, error) {
	var list []model.CmdbCredentialPermission
	var total int64
	db := d.db.Model(&model.CmdbCredentialPermission{})

	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.CredentialID > 0 {
		db = db.Where("credential_id = ?", query.CredentialID)
	}
	if query.IsActive > 0 {
		db = db.Where("is_active = ?", query.IsActive)
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

// GetUserCredentials 获取用户有权限使用的凭据
func (d *CmdbCredentialPermissionDao) GetUserCredentials(userID uint, userGroupIDs []uint) ([]model.CmdbCredentialPermission, error) {
	var list []model.CmdbCredentialPermission
	db := d.db.Model(&model.CmdbCredentialPermission{}).
		Where("is_active = 1")

	userJSON, _ := json.Marshal(userID)
	cond := "user_ids LIKE ?"
	args := []interface{}{"%" + string(userJSON) + "%"}

	for _, gid := range userGroupIDs {
		groupJSON, _ := json.Marshal(gid)
		cond += " OR group_ids LIKE ?"
		args = append(args, "%"+string(groupJSON)+"%")
	}

	db = db.Where("("+cond+")", args...)
	err := db.Order("id DESC").Find(&list).Error
	return list, err
}
