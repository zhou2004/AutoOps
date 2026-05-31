package dao

import (
	"dodevops-api/api/k8s/model"

	"gorm.io/gorm"
)

type K8sUserGroupDao struct {
	db *gorm.DB
}

func NewK8sUserGroupDao(db *gorm.DB) *K8sUserGroupDao {
	return &K8sUserGroupDao{db: db}
}

// Create 创建用户组
func (d *K8sUserGroupDao) Create(group *model.K8sUserGroup) error {
	return d.db.Create(group).Error
}

// Update 更新用户组
func (d *K8sUserGroupDao) Update(id uint, updates map[string]interface{}) error {
	return d.db.Model(&model.K8sUserGroup{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除用户组
func (d *K8sUserGroupDao) Delete(id uint) error {
	// 开启事务删除组和组成员关系
	tx := d.db.Begin()
	if err := tx.Where("group_id = ?", id).Delete(&model.K8sUserGroupMember{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Delete(&model.K8sUserGroup{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// GetByID 获取用户组
func (d *K8sUserGroupDao) GetByID(id uint) (*model.K8sUserGroup, error) {
	var group model.K8sUserGroup
	err := d.db.First(&group, id).Error
	return &group, err
}

// GetList 分页查询用户组列表（含成员数）
func (d *K8sUserGroupDao) GetList(query model.UserGroupQuery) ([]model.UserGroupVo, int64, error) {
	var total int64
	var vos []model.UserGroupVo

	db := d.db.Table("k8s_user_group ug")
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

	err := db.Select(`ug.*, (SELECT COUNT(*) FROM k8s_user_group_member m WHERE m.group_id = ug.id) as member_count`).
		Order("ug.id DESC").
		Offset(offset).Limit(size).
		Scan(&vos).Error
	return vos, total, err
}

// AddMembers 批量添加组成员
func (d *K8sUserGroupDao) AddMembers(groupID uint, userIDs []uint) error {
	var members []model.K8sUserGroupMember
	for _, uid := range userIDs {
		members = append(members, model.K8sUserGroupMember{
			GroupID: groupID,
			UserID:  uid,
		})
	}
	return d.db.CreateInBatches(members, 100).Error
}

// RemoveMember 移除组成员
func (d *K8sUserGroupDao) RemoveMember(groupID, userID uint) error {
	return d.db.Where("group_id = ? AND user_id = ?", groupID, userID).Delete(&model.K8sUserGroupMember{}).Error
}

// GetMembers 获取组成员列表
func (d *K8sUserGroupDao) GetMembers(groupID uint) ([]model.GroupMemberVo, error) {
	var vos []model.GroupMemberVo
	err := d.db.Table("k8s_user_group_member m").
		Select("m.id, m.user_id, sa.username, sa.nickname, m.created_at").
		Joins("LEFT JOIN sys_admin sa ON sa.id = m.user_id").
		Where("m.group_id = ?", groupID).
		Order("m.id DESC").
		Scan(&vos).Error
	return vos, err
}

// GetUserGroupIDs 获取用户所属的所有用户组ID
func (d *K8sUserGroupDao) GetUserGroupIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := d.db.Model(&model.K8sUserGroupMember{}).
		Select("DISTINCT group_id").
		Where("user_id = ?", userID).
		Pluck("group_id", &ids).Error
	return ids, err
}