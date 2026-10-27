package middleware

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"dodevops-api/api/k8s/dao"
	"dodevops-api/api/k8s/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// This uses a private temporary database, never the application's configured DB.
func TestCephPermissionNamespaceAndVerbIsolation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "permissions.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if err = db.AutoMigrate(&model.K8sPermission{}, &model.K8sGroupPermission{}, &model.K8sUserGroup{}, &model.K8sUserGroupMember{}, &model.K8sRbacRole{}, &model.K8sRbacBinding{}); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{"CREATE TABLE sys_role (id INTEGER PRIMARY KEY, role_key TEXT, status INTEGER)", "CREATE TABLE sys_admin_role (admin_id INTEGER, role_id INTEGER)", "INSERT INTO sys_role VALUES (1, 'admin', 1), (2, 'admin', 2)", "INSERT INTO sys_admin_role VALUES (10, 1), (16, 2)"} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []model.K8sPermission{
		{UserID: 1, ClusterID: 7, Namespace: "rook-ceph", PermissionType: "readonly"},
		{UserID: 2, ClusterID: 7, Namespace: "application", PermissionType: "admin"},
		{UserID: 3, ClusterID: 7, Namespace: "rook-ceph", PermissionType: "write"},
	} {
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
	}
	addRole := func(userID uint, namespace, group, resource string, verbs ...string) {
		t.Helper()
		rules, _ := json.Marshal([]model.K8sRule{{APIGroups: []string{group}, Resources: []string{resource}, Verbs: verbs}})
		role := model.K8sRbacRole{ClusterID: 7, Namespace: namespace, Name: "test", Rules: string(rules)}
		if err := db.Create(&role).Error; err != nil {
			t.Fatal(err)
		}
		binding := model.K8sRbacBinding{ClusterID: 7, Namespace: namespace, RoleID: role.ID, SubjectType: "User", SubjectID: userID}
		if err := db.Create(&binding).Error; err != nil {
			t.Fatal(err)
		}
	}
	addRole(4, "rook-ceph", "autoops.io", "cephmaintenance", "get")
	addRole(5, "application", "autoops.io", "cephmaintenance", "*")
	addRole(6, "rook-ceph", "batch", "jobs", "*")
	addRole(7, "", "autoops.io", "cephmaintenance", "get", "update")
	addRole(8, "rook-ceph", "autoops.io", "cephmaintenance", "create")
	if err := db.Create(&model.K8sUserGroup{ID: 22, Name: "enabled", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.K8sUserGroupMember{GroupID: 22, UserID: 9}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.K8sGroupPermission{GroupID: 22, ClusterID: 7, Namespace: "rook-ceph", PermissionType: "write"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.K8sUserGroup{ID: 23, Name: "disabled", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.K8sUserGroup{}).Where("id = ?", 23).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.K8sUserGroupMember{GroupID: 23, UserID: 12}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.K8sGroupPermission{GroupID: 23, ClusterID: 7, Namespace: "rook-ceph", PermissionType: "admin"}).Error; err != nil {
		t.Fatal(err)
	}
	addRole(13, "application", "autoops.io", "cephmaintenance", "*")
	if err := db.Model(&model.K8sRbacBinding{}).Where("subject_id = ?", 13).Update("namespace", "rook-ceph").Error; err != nil {
		t.Fatal(err)
	}
	addRole(14, "rook-ceph", "autoops.io", "cephmaintenance", "*")
	var binding model.K8sRbacBinding
	if err := db.Where("subject_id = ?", 14).First(&binding).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.K8sRbacRole{}).Where("id = ?", binding.RoleID).Update("cluster_id", 8).Error; err != nil {
		t.Fatal(err)
	}
	addRole(15, "rook-ceph", "autoops.io", "cephmaintenance", "*")
	if err := db.Model(&model.K8sRbacBinding{}).Where("subject_id = ?", 15).Updates(map[string]any{"subject_type": "Group", "subject_id": 23}).Error; err != nil {
		t.Fatal(err)
	}
	m := &K8sPermissionMiddleware{permDao: dao.NewK8sPermissionDao(db), groupDao: dao.NewK8sUserGroupDao(db), gPermDao: dao.NewK8sGroupPermissionDao(db), rbacDao: dao.NewK8sRbacDao(db)}
	cases := []struct {
		name          string
		user, cluster uint
		verb          string
		want          bool
	}{
		{"read-only can inspect", 1, 7, "get", true},
		{"read-only cannot mutate", 1, 7, "update", false},
		{"unrelated namespace cannot inspect", 2, 7, "get", false},
		{"unrelated namespace cannot mutate", 2, 7, "update", false},
		{"storage write can mutate", 3, 7, "update", true},
		{"other cluster isolated", 3, 8, "update", false},
		{"RBAC get can inspect", 4, 7, "get", true},
		{"RBAC get cannot mutate", 4, 7, "update", false},
		{"RBAC unrelated namespace isolated", 5, 7, "update", false},
		{"jobs privilege is insufficient", 6, 7, "update", false},
		{"cluster storage RBAC accepted", 7, 7, "update", true},
		{"create does not imply maintenance", 8, 7, "update", false},
		{"group storage permission accepted", 9, 7, "update", true},
		{"system administrator accepted", 10, 8, "update", true},
		{"ungranted user denied", 11, 7, "get", false},
		{"disabled administrator denied", 16, 7, "update", false},
		{"disabled group grants ignored", 12, 7, "update", false},
		{"role namespace cannot be widened by binding", 13, 7, "update", false},
		{"role cannot be bound to a different cluster", 14, 7, "update", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := m.cephVerbAllowed(tc.user, tc.cluster, tc.verb); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	if err := db.Exec("DROP TABLE sys_role").Error; err != nil {
		t.Fatal(err)
	}
	if allowed, err := m.permDao.IsAdmin(10); err == nil || allowed {
		t.Fatal("administrator lookup must report database failure without granting access")
	}
}
