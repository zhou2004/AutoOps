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

func TestAutoscalingExactPermissionScopes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "autoscaling.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	if err = db.AutoMigrate(&model.K8sPermission{}, &model.K8sGroupPermission{}, &model.K8sUserGroup{}, &model.K8sUserGroupMember{}, &model.K8sRbacRole{}, &model.K8sRbacBinding{}); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{"CREATE TABLE sys_role (id INTEGER PRIMARY KEY, role_key TEXT, status INTEGER)", "CREATE TABLE sys_admin_role (admin_id INTEGER, role_id INTEGER)", "INSERT INTO sys_role VALUES (1, 'admin', 1), (2, 'admin', 2)", "INSERT INTO sys_admin_role VALUES (9, 1), (10, 2)"} {
		if err = db.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	add := func(user, roleCluster, bindingCluster uint, roleNS, bindingNS, group, resource, verb string) {
		t.Helper()
		rules, _ := json.Marshal([]model.K8sRule{{APIGroups: []string{group}, Resources: []string{resource}, Verbs: []string{verb}}})
		role := model.K8sRbacRole{ClusterID: roleCluster, Namespace: roleNS, Name: "test", Rules: string(rules)}
		if err = db.Create(&role).Error; err != nil {
			t.Fatal(err)
		}
		if err = db.Create(&model.K8sRbacBinding{ClusterID: bindingCluster, Namespace: bindingNS, RoleID: role.ID, SubjectType: "User", SubjectID: user}).Error; err != nil {
			t.Fatal(err)
		}
	}
	add(1, 7, 7, "team", "team", "autoscaling", "horizontalpodautoscalers", "list")
	add(2, 7, 7, "team", "team", "", "services", "update")
	add(3, 7, 7, "team", "team", "serving.knative.dev", "services", "update")
	add(4, 8, 7, "team", "team", "autoscaling", "horizontalpodautoscalers", "*")
	add(5, 7, 7, "other", "team", "autoscaling", "horizontalpodautoscalers", "*")
	add(6, 7, 7, "", "", "autoscaling", "horizontalpodautoscalers", "update")
	// A namespaced binding must not shadow a valid cluster binding.
	add(6, 7, 7, "team", "team", "autoscaling", "horizontalpodautoscalers", "get")
	group := model.K8sUserGroup{Name: "disabled", Status: 1}
	db.Create(&group)
	db.Model(&group).Update("status", 0)
	db.Create(&model.K8sUserGroupMember{UserID: 7, GroupID: group.ID})
	db.Create(&model.K8sGroupPermission{GroupID: group.ID, ClusterID: 7, Namespace: "team", PermissionType: "admin"})
	db.Create(&model.K8sPermission{UserID: 8, ClusterID: 7, Namespace: "team", PermissionType: "readonly"})
	m := &K8sPermissionMiddleware{permDao: dao.NewK8sPermissionDao(db), groupDao: dao.NewK8sUserGroupDao(db), gPermDao: dao.NewK8sGroupPermissionDao(db), rbacDao: dao.NewK8sRbacDao(db)}
	cases := []struct {
		user, cluster             uint
		ns, group, resource, verb string
		want                      bool
	}{
		{1, 7, "team", "autoscaling", "horizontalpodautoscalers", "list", true},
		{1, 7, "team", "autoscaling", "horizontalpodautoscalers", "delete", false},
		{1, 7, "other", "autoscaling", "horizontalpodautoscalers", "list", false},
		{1, 8, "team", "autoscaling", "horizontalpodautoscalers", "list", false},
		{2, 7, "team", "serving.knative.dev", "services", "update", false},
		{3, 7, "team", "serving.knative.dev", "services", "update", true},
		{4, 7, "team", "autoscaling", "horizontalpodautoscalers", "update", false},
		{5, 7, "team", "autoscaling", "horizontalpodautoscalers", "update", false},
		{6, 7, "team", "autoscaling", "horizontalpodautoscalers", "update", true},
		{7, 7, "team", "autoscaling", "horizontalpodautoscalers", "update", false},
		{8, 7, "team", "autoscaling.k8s.io", "verticalpodautoscalers", "list", true},
		{8, 7, "team", "autoscaling.k8s.io", "verticalpodautoscalers", "update", false},
		{9, 7, "team", "autoscaling.k8s.io", "verticalpodautoscalers", "delete", true},
		{10, 7, "team", "autoscaling.k8s.io", "verticalpodautoscalers", "delete", false},
	}
	for _, tc := range cases {
		if got := m.autoscalingAccess(tc.user, tc.cluster, tc.ns)(tc.group, tc.resource, tc.verb); got != tc.want {
			t.Errorf("user=%d cluster=%d ns=%s group=%s verb=%s: got %v want %v", tc.user, tc.cluster, tc.ns, tc.group, tc.verb, got, tc.want)
		}
	}
}
