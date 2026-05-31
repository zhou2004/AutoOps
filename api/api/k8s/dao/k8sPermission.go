package dao

import (
	"dodevops-api/api/k8s/model"

	"gorm.io/gorm"
)

type K8sPermissionDao struct {
	db *gorm.DB
}

type K8sGroupPermissionDao struct {
	db *gorm.DB
}

func NewK8sPermissionDao(db *gorm.DB) *K8sPermissionDao {
	return &K8sPermissionDao{db: db}
}

func NewK8sGroupPermissionDao(db *gorm.DB) *K8sGroupPermissionDao {
	return &K8sGroupPermissionDao{db: db}
}

// ======================= 用户组权限 DAO =======================

// Create 创建用户组权限
func (d *K8sGroupPermissionDao) Create(perm *model.K8sGroupPermission) error {
	return d.db.Create(perm).Error
}

// BatchCreate 批量创建用户组权限
func (d *K8sGroupPermissionDao) BatchCreate(perms []model.K8sGroupPermission) error {
	return d.db.CreateInBatches(perms, 100).Error
}

// Update 更新用户组权限
func (d *K8sGroupPermissionDao) Update(id uint, permType string) error {
	return d.db.Model(&model.K8sGroupPermission{}).Where("id = ?", id).Update("permission_type", permType).Error
}

// Delete 删除用户组权限
func (d *K8sGroupPermissionDao) Delete(id uint) error {
	return d.db.Delete(&model.K8sGroupPermission{}, id).Error
}

// GetByID 获取用户组权限
func (d *K8sGroupPermissionDao) GetByID(id uint) (*model.K8sGroupPermission, error) {
	var perm model.K8sGroupPermission
	err := d.db.First(&perm, id).Error
	return &perm, err
}

// GetByGroup 获取用户组的所有权限
func (d *K8sGroupPermissionDao) GetByGroup(groupID uint) ([]model.K8sGroupPermissionVo, error) {
	var vos []model.K8sGroupPermissionVo
	err := d.db.Table("k8s_group_permission gp").
		Select("gp.id, gp.group_id, gp.cluster_id, gp.namespace, gp.permission_type, gp.created_at, gp.updated_at, ug.name as group_name, kc.name as cluster_name").
		Joins("LEFT JOIN k8s_user_group ug ON ug.id = gp.group_id").
		Joins("LEFT JOIN k8s_cluster kc ON kc.id = gp.cluster_id").
		Where("gp.group_id = ?", groupID).
		Order("gp.id DESC").
		Scan(&vos).Error
	return vos, err
}

// CheckGroupPermission 检查用户组在指定集群命名空间的权限
func (d *K8sGroupPermissionDao) CheckGroupPermission(groupID, clusterID uint, namespace string) (*model.K8sGroupPermission, error) {
	var perm model.K8sGroupPermission
	err := d.db.Where("group_id = ? AND cluster_id = ? AND namespace = ?", groupID, clusterID, namespace).First(&perm).Error
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

// GetGroupAllowedNamespaces 获取用户组允许访问的命名空间（按集群分组）
func (d *K8sGroupPermissionDao) GetGroupAllowedNamespaces(groupIDs []uint) (map[uint][]string, error) {
	var perms []model.K8sGroupPermission
	err := d.db.Where("group_id IN ?", groupIDs).Find(&perms).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint][]string)
	for _, p := range perms {
		result[p.ClusterID] = append(result[p.ClusterID], p.Namespace)
	}
	return result, nil
}

// GetGroupAllowedClusterIDs 获取用户组有权限的集群ID
func (d *K8sGroupPermissionDao) GetGroupAllowedClusterIDs(groupIDs []uint) ([]uint, error) {
	var ids []uint
	err := d.db.Model(&model.K8sGroupPermission{}).
		Select("DISTINCT cluster_id").
		Where("group_id IN ?", groupIDs).
		Pluck("cluster_id", &ids).Error
	return ids, err
}

// GetGroupPermissionList 分页查询用户组权限列表
func (d *K8sGroupPermissionDao) GetGroupPermissionList(query model.GroupPermissionQuery) ([]model.K8sGroupPermissionVo, int64, error) {
	var total int64
	var vos []model.K8sGroupPermissionVo
	db := d.db.Table("k8s_group_permission gp").
		Select("gp.id, gp.group_id, gp.cluster_id, gp.namespace, gp.permission_type, gp.created_at, gp.updated_at, ug.name as group_name, kc.name as cluster_name").
		Joins("LEFT JOIN k8s_user_group ug ON ug.id = gp.group_id").
		Joins("LEFT JOIN k8s_cluster kc ON kc.id = gp.cluster_id")

	if query.GroupID > 0 {
		db = db.Where("gp.group_id = ?", query.GroupID)
	}
	if query.ClusterID > 0 {
		db = db.Where("gp.cluster_id = ?", query.ClusterID)
	}
	if query.Namespace != "" {
		db = db.Where("gp.namespace LIKE ?", "%"+query.Namespace+"%")
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
	err := db.Order("gp.id DESC").Offset(offset).Limit(size).Scan(&vos).Error
	return vos, total, err
}

// Create 创建权限记录
func (d *K8sPermissionDao) Create(perm *model.K8sPermission) error {
	return d.db.Create(perm).Error
}

// BatchCreate 批量创建权限记录
func (d *K8sPermissionDao) BatchCreate(perms []model.K8sPermission) error {
	return d.db.CreateInBatches(perms, 100).Error
}

// Update 更新权限记录
func (d *K8sPermissionDao) Update(id uint, permType string) error {
	return d.db.Model(&model.K8sPermission{}).Where("id = ?", id).Update("permission_type", permType).Error
}

// Delete 删除权限记录
func (d *K8sPermissionDao) Delete(id uint) error {
	return d.db.Delete(&model.K8sPermission{}, id).Error
}

// DeleteByUserAndCluster 删除用户在某集群的所有权限
func (d *K8sPermissionDao) DeleteByUserAndCluster(userID, clusterID uint) error {
	return d.db.Where("user_id = ? AND cluster_id = ?", userID, clusterID).Delete(&model.K8sPermission{}).Error
}

// GetByID 根据ID获取权限
func (d *K8sPermissionDao) GetByID(id uint) (*model.K8sPermission, error) {
	var perm model.K8sPermission
	err := d.db.First(&perm, id).Error
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

// GetByUserAndCluster 获取用户在指定集群的所有权限
func (d *K8sPermissionDao) GetByUserAndCluster(userID, clusterID uint) ([]model.K8sPermission, error) {
	var perms []model.K8sPermission
	err := d.db.Where("user_id = ? AND cluster_id = ?", userID, clusterID).Find(&perms).Error
	return perms, err
}

// GetByUser 获取用户的所有权限
func (d *K8sPermissionDao) GetByUser(userID uint) ([]model.K8sPermission, error) {
	var perms []model.K8sPermission
	err := d.db.Where("user_id = ?", userID).Find(&perms).Error
	return perms, err
}

// GetByCluster 获取集群的所有权限分配
func (d *K8sPermissionDao) GetByCluster(clusterID uint) ([]model.K8sPermission, error) {
	var perms []model.K8sPermission
	err := d.db.Where("cluster_id = ?", clusterID).Find(&perms).Error
	return perms, err
}

// CheckPermission 检查用户是否有指定集群命名空间的权限
func (d *K8sPermissionDao) CheckPermission(userID, clusterID uint, namespace string) (*model.K8sPermission, error) {
	var perm model.K8sPermission
	err := d.db.Where("user_id = ? AND cluster_id = ? AND namespace = ?", userID, clusterID, namespace).First(&perm).Error
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

// IsAdmin 检查用户是否为管理员（拥有所有权限）
func (d *K8sPermissionDao) IsAdmin(userID uint) (bool, error) {
	var count int64
	// 通过角色判断是否为管理员 - 角色ID为1通常是管理员
	d.db.Table("sys_admin_role").
		Joins("JOIN sys_role ON sys_role.id = sys_admin_role.role_id").
		Where("sys_admin_role.admin_id = ? AND sys_role.role_key = ?", userID, "admin").
		Count(&count)
	return count > 0, nil
}

// GetUserAllowedNamespaces 获取用户允许访问的命名空间列表（按集群分组）
func (d *K8sPermissionDao) GetUserAllowedNamespaces(userID uint) (map[uint][]string, error) {
	var perms []model.K8sPermission
	err := d.db.Where("user_id = ?", userID).Find(&perms).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint][]string)
	for _, p := range perms {
		result[p.ClusterID] = append(result[p.ClusterID], p.Namespace)
	}
	return result, nil
}

// GetUserAllowedClusterIDs 获取用户有权限访问的集群ID列表
func (d *K8sPermissionDao) GetUserAllowedClusterIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := d.db.Model(&model.K8sPermission{}).
		Select("DISTINCT cluster_id").
		Where("user_id = ?", userID).
		Pluck("cluster_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// GetList 分页查询权限列表（带关联信息）
func (d *K8sPermissionDao) GetList(query model.K8sPermissionQuery) ([]model.K8sPermissionVo, int64, error) {
	var total int64
	var vos []model.K8sPermissionVo

	db := d.db.Table("k8s_permission kp").
		Select(`kp.id, kp.user_id, kp.cluster_id, kp.namespace, kp.permission_type, 
				kp.created_at, kp.updated_at,
				sa.username, sa.nickname,
				kc.name as cluster_name`).
		Joins("LEFT JOIN sys_admin sa ON sa.id = kp.user_id").
		Joins("LEFT JOIN k8s_cluster kc ON kc.id = kp.cluster_id")

	if query.UserID > 0 {
		db = db.Where("kp.user_id = ?", query.UserID)
	}
	if query.ClusterID > 0 {
		db = db.Where("kp.cluster_id = ?", query.ClusterID)
	}
	if query.Namespace != "" {
		db = db.Where("kp.namespace LIKE ?", "%"+query.Namespace+"%")
	}

	// 计数
	db.Count(&total)

	// 分页
	page := query.Page
	if page <= 0 {
		page = 1
	}
	size := query.Size
	if size <= 0 {
		size = 10
	}
	offset := (page - 1) * size

	err := db.Order("kp.id DESC").Offset(offset).Limit(size).Scan(&vos).Error
	return vos, total, err
}
