package controller

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	model "dodevops-api/api/aiops/model"
	"dodevops-api/api/aiops/service"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func controllerForTest(t *testing.T) *Controller {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ai.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	for _, sql := range []string{"CREATE TABLE sys_admin(id INTEGER,status INTEGER)", "CREATE TABLE sys_role(id INTEGER,role_key TEXT,status INTEGER)", "CREATE TABLE sys_admin_role(admin_id INTEGER,role_id INTEGER)", "INSERT INTO sys_admin VALUES(1,1),(2,1),(3,0)", "INSERT INTO sys_role VALUES(1,'admin',1)", "INSERT INTO sys_admin_role VALUES(1,1)"} {
		if err = db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err = db.AutoMigrate(&model.Config{}, &model.Run{}); err != nil {
		t.Fatal(err)
	}
	return &Controller{Service: service.New(db)}
}
func invoke(t *testing.T, user uint, path, body string, fn gin.HandlerFunc) map[string]interface{} {
	t.Helper()
	r := httptest.NewRecorder()
	g, _ := gin.CreateTestContext(r)
	g.Request = httptest.NewRequest("POST", path, strings.NewReader(body))
	g.Request.Header.Set("Content-Type", "application/json")
	if user > 0 {
		g.Set(constant.ContextKeyUserObj, &systemmodel.JwtAdmin{ID: user})
	}
	if strings.Contains(path, "/1") {
		g.Params = gin.Params{{Key: "id", Value: "1"}}
	}
	fn(g)
	var result map[string]interface{}
	if err := json.Unmarshal(r.Body.Bytes(), &result); err != nil {
		t.Fatalf("response: %s", r.Body.String())
	}
	return result
}
func TestConfigAdministrationAndAuthenticatedCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := controllerForTest(t)
	body := `{"kind":"model","name":"local","provider":"ollama","baseUrl":"http://127.0.0.1:11434","model":"test","enabled":true}`
	for _, user := range []uint{0, 2, 3} {
		r := invoke(t, user, "/configs", body, c.Save)
		if r["code"] == float64(200) {
			t.Fatalf("user %d could edit configuration", user)
		}
	}
	r := invoke(t, 1, "/configs", body, c.Save)
	if r["code"] != float64(200) {
		t.Fatalf("admin save failed: %+v", r)
	}
	r = invoke(t, 2, "/catalog", "", c.Catalog)
	if r["code"] != float64(200) || r["data"].(map[string]interface{})["admin"] != false {
		t.Fatal("nonadmin catalog failed")
	}
	data := r["data"].(map[string]interface{})["configs"].([]interface{})[0].(map[string]interface{})
	if data["baseUrl"] != "" {
		t.Fatal("endpoint exposed to nonadmin")
	}
	if r = invoke(t, 2, "/configs/1", "", c.Delete); r["code"] != float64(403) {
		t.Fatal("delete authorization")
	}
	if r = invoke(t, 2, "/models/1/test", "", c.Test); r["code"] != float64(403) {
		t.Fatal("test authorization")
	}
}
func TestHistoryIsolationAndStreamPreflight(t *testing.T) {
	c := controllerForTest(t)
	c.Service.DB.Create(&model.Run{UserID: 1, Prompt: "private", Output: "private", Status: "succeeded"})
	if r := invoke(t, 2, "/runs/1", "", c.Detail); r["code"] != float64(404) {
		t.Fatal("history ownership leak")
	}
	if r := invoke(t, 1, "/diagnose", `{"prompt":"","modules":["cmdb"],"days":7}`, c.Stream); r["code"] != float64(400) {
		t.Fatal("invalid stream request should fail before SSE")
	}
	if r := invoke(t, 0, "/diagnose", `{}`, c.Stream); r["code"] != float64(401) {
		t.Fatal("stream requires authentication")
	}
	if r := invoke(t, 3, "/catalog", "", c.Catalog); r["code"] != float64(403) {
		t.Fatal("inactive account catalog")
	}
}
