package dao

import (
	"strings"

	"dodevops-api/api/k8s/model"

	"gorm.io/gorm"
)

type K8sRbacDao struct {
	db *gorm.DB
}

func NewK8sRbacDao(db *gorm.DB) *K8sRbacDao {
	return &K8sRbacDao{db: db}
}

// ======================= Role DAO =======================

func (d *K8sRbacDao) CreateRole(role *model.K8sRbacRole) error {
	return d.db.Create(role).Error
}

func (d *K8sRbacDao) UpdateRole(role *model.K8sRbacRole) error {
	return d.db.Save(role).Error
}

func (d *K8sRbacDao) DeleteRole(id uint) error {
	return d.db.Delete(&model.K8sRbacRole{}, id).Error
}

func (d *K8sRbacDao) GetRoleByID(id uint) (*model.K8sRbacRole, error) {
	var role model.K8sRbacRole
	err := d.db.First(&role, id).Error
	return &role, err
}

func (d *K8sRbacDao) GetRolesByIDs(ids []uint) ([]model.K8sRbacRole, error) {
	var roles []model.K8sRbacRole
	err := d.db.Where("id IN ?", ids).Find(&roles).Error
	return roles, err
}

func (d *K8sRbacDao) GetRoleList(clusterID uint, namespace, name string, page, size int) ([]model.K8sRbacRoleVo, int64, error) {
	var vos []model.K8sRbacRoleVo
	var total int64
	db := d.db.Table("k8s_rbac_role r").
		Select("r.*, kc.name as cluster_name").
		Joins("LEFT JOIN k8s_cluster kc ON kc.id = r.cluster_id")

	countDb := d.db.Model(&model.K8sRbacRole{})
	if clusterID > 0 {
		db = db.Where("r.cluster_id = ?", clusterID)
		countDb = countDb.Where("cluster_id = ?", clusterID)
	}
	if namespace != "" {
		nsList := strings.Split(namespace, ",")
		db = db.Where("r.namespace IN ?", nsList)
		countDb = countDb.Where("namespace IN ?", nsList)
	}
	if name != "" {
		db = db.Where("r.name LIKE ?", "%"+name+"%")
		countDb = countDb.Where("name LIKE ?", "%"+name+"%")
	}
	countDb.Count(&total)
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	offset := (page - 1) * size
	err := db.Order("r.id DESC").Offset(offset).Limit(size).Scan(&vos).Error
	return vos, total, err
}

// ======================= Binding DAO =======================

func (d *K8sRbacDao) CreateBinding(binding *model.K8sRbacBinding) error {
	return d.db.Create(binding).Error
}

func (d *K8sRbacDao) UpdateBinding(binding *model.K8sRbacBinding) error {
	return d.db.Save(binding).Error
}

func (d *K8sRbacDao) DeleteBinding(id uint) error {
	return d.db.Delete(&model.K8sRbacBinding{}, id).Error
}

func (d *K8sRbacDao) GetBindingByID(id uint) (*model.K8sRbacBinding, error) {
	var binding model.K8sRbacBinding
	err := d.db.First(&binding, id).Error
	return &binding, err
}

// GetBindingsByUser 获取用户所有绑定（直接绑定 + 通过用户组继承）
func (d *K8sRbacDao) GetBindingsByUser(userID uint, groupIDs []uint) ([]model.K8sRbacBinding, error) {
	var bindings []model.K8sRbacBinding
	query := d.db.Where("(subject_type = 'User' AND subject_id = ?)", userID)
	if len(groupIDs) > 0 {
		for i, gid := range groupIDs {
			if i == 0 {
				query = query.Or("(subject_type = 'Group' AND subject_id = ?)", gid)
			} else {
				query = query.Or("(subject_type = 'Group' AND subject_id = ?)", gid)
			}
		}
	}
	err := query.Find(&bindings).Error
	return bindings, err
}

// GetBindingsByClusterNamespace 获取指定集群命名空间的所有绑定
func (d *K8sRbacDao) GetBindingsByClusterNamespace(clusterID uint, namespace string) ([]model.K8sRbacBinding, error) {
	var bindings []model.K8sRbacBinding
	err := d.db.Where("cluster_id = ? AND namespace = ?", clusterID, namespace).Find(&bindings).Error
	return bindings, err
}

func (d *K8sRbacDao) GetBindingList(clusterID uint, namespace, subjectType, subjectName string, page, size int) ([]model.K8sRbacBindingVo, int64, error) {
	var vos []model.K8sRbacBindingVo
	var total int64
	query := d.db.Table("k8s_rbac_binding rb").
		Select("rb.*, kc.name as cluster_name, rr.name as role_name, " +
			"CASE WHEN rb.subject_type = 'User' THEN u.username " +
			"     WHEN rb.subject_type = 'Group' THEN ug.name " +
			"END as subject_name").
		Joins("LEFT JOIN k8s_cluster kc ON kc.id = rb.cluster_id").
		Joins("LEFT JOIN k8s_rbac_role rr ON rr.id = rb.role_id").
		Joins("LEFT JOIN sys_admin u ON rb.subject_type = 'User' AND u.id = rb.subject_id").
		Joins("LEFT JOIN k8s_user_group ug ON rb.subject_type = 'Group' AND ug.id = rb.subject_id")

	countQuery := d.db.Table("k8s_rbac_binding rb")

	if clusterID > 0 {
		query = query.Where("rb.cluster_id = ?", clusterID)
		countQuery = countQuery.Where("rb.cluster_id = ?", clusterID)
	}
	if namespace != "" {
		nsList := strings.Split(namespace, ",")
		query = query.Where("rb.namespace IN ?", nsList)
		countQuery = countQuery.Where("rb.namespace IN ?", nsList)
	}
	if subjectType != "" {
		query = query.Where("rb.subject_type = ?", subjectType)
		countQuery = countQuery.Where("rb.subject_type = ?", subjectType)
	}
	if subjectName != "" {
		query = query.Where("(CASE WHEN rb.subject_type = 'User' THEN u.username WHEN rb.subject_type = 'Group' THEN ug.name END) LIKE ?", "%"+subjectName+"%")
		countQuery = countQuery.Joins("LEFT JOIN sys_admin u2 ON rb.subject_type = 'User' AND u2.id = rb.subject_id").
			Joins("LEFT JOIN k8s_user_group ug2 ON rb.subject_type = 'Group' AND ug2.id = rb.subject_id").
			Where("(CASE WHEN rb.subject_type = 'User' THEN u2.username WHEN rb.subject_type = 'Group' THEN ug2.name END) LIKE ?", "%"+subjectName+"%")
	}

	countQuery.Count(&total)

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	offset := (page - 1) * size
	err := query.Order("rb.id DESC").Offset(offset).Limit(size).Scan(&vos).Error
	return vos, total, err
}

func (d *K8sRbacDao) GetBindingsBySubjects(subjects []struct {
	Type string
	ID   uint
}) ([]model.K8sRbacBinding, error) {
	var bindings []model.K8sRbacBinding
	if len(subjects) == 0 {
		return bindings, nil
	}

	query := d.db
	for i, s := range subjects {
		if i == 0 {
			query = query.Where("(subject_type = ? AND subject_id = ?)", s.Type, s.ID)
		} else {
			query = query.Or("(subject_type = ? AND subject_id = ?)", s.Type, s.ID)
		}
	}

	err := query.Find(&bindings).Error
	return bindings, err
}
