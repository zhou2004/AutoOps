package dao

import (
	"dodevops-api/api/monitor/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"path/filepath"
	"testing"
)

func monitorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "monitor.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	if err := db.AutoMigrate(&model.MonitorCategory{}, &model.DomainCert{}, &model.MonitorAPIEndpoint{}, &model.MonitorIncident{}); err != nil {
		t.Fatal(err)
	}
	return db
}
func TestCategoryAssignmentIsolationAndDelete(t *testing.T) {
	db := monitorTestDB(t)
	categories := &MonitorCategoryDao{db: db}
	for _, scope := range []string{"domain_cert", "api_endpoint"} {
		if err := categories.Save(&model.MonitorCategoryReq{Scope: scope, Name: "生产"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := categories.Save(&model.MonitorCategoryReq{Scope: "domain_cert", Name: " 生产 "}); err == nil {
		t.Fatal("duplicate names must be rejected within a scope")
	}
	cert := model.DomainCert{Domain: "example.test"}
	if err := db.Create(&cert).Error; err != nil {
		t.Fatal(err)
	}
	if err := categories.Assign(&model.MonitorCategoryAssignReq{Scope: "domain_cert", IDs: []uint{cert.ID}, CategoryID: 2}); err == nil {
		t.Fatal("cross-scope category accepted")
	}
	if err := categories.Assign(&model.MonitorCategoryAssignReq{Scope: "domain_cert", IDs: []uint{cert.ID, 999}, CategoryID: 1}); err == nil {
		t.Fatal("partial assignment accepted")
	}
	db.First(&cert, cert.ID)
	if cert.CategoryID != 0 {
		t.Fatal("failed assignment changed target")
	}
	if err := categories.Assign(&model.MonitorCategoryAssignReq{Scope: "domain_cert", IDs: []uint{cert.ID}, CategoryID: 1}); err != nil {
		t.Fatal(err)
	}
	stale := model.DomainCert{}
	db.First(&stale, cert.ID)
	if err := categories.Delete("domain_cert", 1); err != nil {
		t.Fatal(err)
	}
	stale.Status = 4
	stale.ErrorMsg = "TLS error"
	if err := (&DomainCertDao{db: db}).UpdateCheck(&stale); err != nil {
		t.Fatal(err)
	}
	db.First(&cert, cert.ID)
	if cert.CategoryID != 0 || cert.Status != 4 {
		t.Fatalf("check overwrote category deletion: %+v", cert)
	}
	list, err := categories.List("api_endpoint")
	if err != nil || len(list) != 1 {
		t.Fatal("deleting domain category affected API scope")
	}
}
func TestCategoryFiltersAndZeroUpdates(t *testing.T) {
	db := monitorTestDB(t)
	d := &APIEndpointDao{db: db}
	categories := &MonitorCategoryDao{db: db}
	if err := categories.Save(&model.MonitorCategoryReq{Scope: "api_endpoint", Name: "生产"}); err != nil {
		t.Fatal(err)
	}
	targets := []model.MonitorAPIEndpoint{{Name: "health", URL: "https://prod.test/health", CategoryID: 1, Status: 2}, {Name: "other", URL: "https://test.test/health", Status: 1}}
	for i := range targets {
		if err := d.Create(&targets[i]); err != nil {
			t.Fatal(err)
		}
	}
	one := uint(1)
	zero := uint(0)
	list, total, err := d.GetList(&model.APIEndpointListReq{Name: "prod.test", CategoryID: &one, Status: 2})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("combined URL/category/status query failed: %v %d", err, total)
	}
	_, total, err = d.GetList(&model.APIEndpointListReq{CategoryID: &zero})
	if err != nil || total != 1 {
		t.Fatal("zero must filter uncategorized, not all records")
	}
	targets[0].CategoryID = 0
	if err := d.Update(&targets[0]); err != nil {
		t.Fatal(err)
	}
	_, total, err = d.GetList(&model.APIEndpointListReq{CategoryID: &zero})
	if err != nil || total != 2 {
		t.Fatal("updating category to zero was ignored")
	}
	_, total, err = d.GetList(&model.APIEndpointListReq{})
	if err != nil || total != 2 {
		t.Fatal("no category should include all records")
	}
}
