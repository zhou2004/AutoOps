package middleware

import (
	"dodevops-api/api/k8s/dao"
	"dodevops-api/api/k8s/model"
	system "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestK8sResourceScopesAndSideEffects(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "k8s.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	if err = db.AutoMigrate(&model.K8sPermission{}, &model.K8sUserGroup{}, &model.K8sUserGroupMember{}, &model.K8sGroupPermission{}, &model.K8sRbacRole{}, &model.K8sRbacBinding{}); err != nil {
		t.Fatal(err)
	}
	db.Exec("CREATE TABLE sys_role(id INTEGER,role_key TEXT,status INTEGER)")
	db.Exec("CREATE TABLE sys_admin_role(admin_id INTEGER,role_id INTEGER)")
	db.Create(&model.K8sPermission{UserID: 1, ClusterID: 7, Namespace: "team", PermissionType: "readonly"})
	rules, _ := json.Marshal([]model.K8sRule{{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"list", "get", "update"}}, {APIGroups: []string{""}, Resources: []string{"pods/exec"}, Verbs: []string{"create"}}})
	role := model.K8sRbacRole{ClusterID: 7, Namespace: "team", Name: "operator", Rules: string(rules)}
	db.Create(&role)
	db.Create(&model.K8sRbacBinding{ClusterID: 7, Namespace: "team", SubjectType: "User", SubjectID: 2, RoleID: role.ID})
	m := &K8sPermissionMiddleware{permDao: dao.NewK8sPermissionDao(db), groupDao: dao.NewK8sUserGroupDao(db), gPermDao: dao.NewK8sGroupPermissionDao(db), rbacDao: dao.NewK8sRbacDao(db)}
	for _, tc := range []struct {
		user                     uint
		method, route, url, body string
		want                     int
	}{
		{1, "GET", "/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/terminal", "/k8s/cluster/7/namespaces/team/pods/a/terminal", "", 403},
		{1, "GET", "/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/files/list", "/k8s/cluster/7/namespaces/team/pods/a/files/list", "", 403},
		{1, "POST", "/k8s/cluster/:id/nodes/:nodeName/cordon", "/k8s/cluster/7/nodes/a/cordon", "", 403},
		{1, "GET", "/k8s/cluster/:id/nodes", "/k8s/cluster/7/nodes", "", 403},
		{1, "POST", "/k8s/cluster/:id/namespaces", "/k8s/cluster/7/namespaces", "", 403},
		{1, "GET", "/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/logs", "/k8s/cluster/7/namespaces/team/pods/a/logs", "", 200},
		{2, "GET", "/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/terminal", "/k8s/cluster/7/namespaces/team/pods/a/terminal", "", 200},
		{2, "GET", "/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/terminal", "/k8s/cluster/7/namespaces/other/pods/a/terminal", "", 403},
		{2, "PUT", "/k8s/cluster/:id/namespaces/:namespaceName/workload-yaml", "/k8s/cluster/7/namespaces/team/workload-yaml", `{"workloadType":"deployment"}`, 200},
		{2, "PUT", "/k8s/cluster/:id/namespaces/:namespaceName/workload-yaml", "/k8s/cluster/7/namespaces/team/workload-yaml", `{"workloadType":"statefulset"}`, 403},
		{2, "GET", "/k8s/cluster/:id/namespaces/:namespaceName/workloads", "/k8s/cluster/7/namespaces/team/workloads?type=statefulset", "", 403},
		{2, "GET", "/k8s/cluster/:id/namespaces/:namespaceName/workloads", "/k8s/cluster/7/namespaces/team/workloads?type=all", "", 200},
	} {
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set(constant.ContextKeyUserObj, &system.JwtAdmin{ID: tc.user}); c.Next() })
		r.Handle(tc.method, tc.route, m.checkResourcePermission(), func(c *gin.Context) {
			if strings.HasSuffix(c.FullPath(), "/workloads") {
				kinds := c.MustGet("k8s_workload_kinds").([]string)
				if len(kinds) != 1 || kinds[0] != "deployments" {
					t.Errorf("workload leak: %v", kinds)
				}
			}
			c.Status(200)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body)))
		if w.Code != tc.want {
			t.Errorf("user=%d %s: want %d got %d %s", tc.user, tc.url, tc.want, w.Code, w.Body.String())
		}
	}
}
