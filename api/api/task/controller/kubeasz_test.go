package controller

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestKubeaszRequiresEnabledAdministrator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "auth.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{"CREATE TABLE sys_role (id integer, role_key text, status integer)", "CREATE TABLE sys_admin_role (admin_id integer, role_id integer)", "INSERT INTO sys_role VALUES (1, 'admin', 1), (2, 'admin', 2), (3, 'operator', 1)", "INSERT INTO sys_admin_role VALUES (1, 1), (2, 2), (3, 3)"} {
		if err = db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	controller := &KubeaszController{db: db}
	for _, tc := range []struct {
		name string
		id   uint
		code int
	}{{"enabled-admin", 1, 200}, {"disabled-admin", 2, 403}, {"operator", 3, 403}, {"no-role", 4, 403}, {"anonymous", 0, 401}} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) {
				if tc.id > 0 {
					c.Set(constant.ContextKeyUserObj, &systemmodel.JwtAdmin{ID: tc.id, Username: "test"})
				}
				c.Next()
			})
			r.GET("/test", controller.RequireAdmin(), func(c *gin.Context) { c.JSON(200, gin.H{"code": 200}) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
			var body struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != tc.code {
				t.Fatalf("got %d want %d: %s", body.Code, tc.code, w.Body.String())
			}
		})
	}
	if err = db.Exec("DROP TABLE sys_role").Error; err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(constant.ContextKeyUserObj, &systemmodel.JwtAdmin{ID: 1}); c.Next() })
	r.GET("/test", controller.RequireAdmin(), func(c *gin.Context) { t.Fatal("database error granted access") })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/test", nil))
}
func TestKubeaszRejectsUnknownJSONFields(t *testing.T) {
	for _, body := range []string{`{"action":"backup","cluster":"prod","command":"id"}`, `{"action":"backup","cluster":"prod"} {"action":"stop"}`} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/preview", strings.NewReader(body))
		if _, err := kubeaszBody(c); err == nil {
			t.Fatal("accepted unsafe JSON", body)
		}
	}
}
