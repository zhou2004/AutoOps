package service

import (
	"dodevops-api/api/cmdb/dao"
	"encoding/json"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"dodevops-api/api/cmdb/model"
	"github.com/gin-gonic/gin"
)

func TestEmptyAssetScopeDoesNotBehaveAsAdministrator(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("cmdb_allowed_asset_ids", map[uint]bool{})
	if isAdminContext(c) {
		t.Fatal("empty scope must not be administrator")
	}
	if got := filterAllowedIDs(c, []uint{1, 2}); len(got) != 0 {
		t.Fatalf("IDs escaped empty scope: %v", got)
	}
	if got := filterAllowedHosts(c, []model.CmdbHost{{ID: 1}}); len(got) != 0 {
		t.Fatalf("hosts escaped empty scope: %v", got)
	}
	c.Set("cmdb_allowed_asset_ids", "invalid")
	if isAdminContext(c) || len(filterAllowedIDs(c, []uint{1})) != 0 {
		t.Fatal("malformed scope must deny")
	}
}

func TestAssetPermissionValidation(t *testing.T) {
	valid := CmdbAssetPermissionReq{Name: "operator", UserIDs: []uint{1}, AssetTypes: []string{"host"}, HostIDs: []uint{7}, PermissionActions: []string{"connect_rdp", "audit"}, IsActive: 0}
	if err := validateAssetPermission(&valid); err != nil {
		t.Fatal(err)
	}
	invalidAction := valid
	invalidAction.PermissionActions = []string{"allow_everything"}
	if validateAssetPermission(&invalidAction) == nil {
		t.Fatal("unknown action accepted")
	}
	invalidDate := valid
	invalidDate.DateStart = "2026-10-03"
	invalidDate.DateExpired = "2026-10-02"
	if validateAssetPermission(&invalidDate) == nil {
		t.Fatal("reversed dates accepted")
	}
	malformedDate := valid
	malformedDate.DateStart = "invalid"
	if validateAssetPermission(&malformedDate) == nil {
		t.Fatal("malformed date accepted")
	}
}

func TestCredentialPermissionServicePreservesDisabledState(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "credential-permissions.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	if err = db.AutoMigrate(&model.CmdbCredentialPermission{}); err != nil {
		t.Fatal(err)
	}
	svc := &CmdbCredentialPermissionServiceImpl{dao: dao.NewCmdbCredentialPermissionDao(db)}
	request := model.CmdbCredentialPermissionReq{Name: "disabled", CredentialID: 9, UserIDs: []uint{1}, IsActive: 0}
	invoke := func(update bool, state int) int {
		t.Helper()
		request.IsActive = state
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		if update {
			svc.Update(c, 1, &request)
		} else {
			svc.Create(c, &request)
		}
		var response struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response.Code
	}
	if code := invoke(false, 0); code != 200 {
		t.Fatalf("disabled create returned %d", code)
	}
	var stored model.CmdbCredentialPermission
	if err = db.First(&stored, 1).Error; err != nil || stored.IsActive != 0 {
		t.Fatalf("create enabled rule unexpectedly: %+v %v", stored, err)
	}
	for _, state := range []int{1, 0} {
		if code := invoke(true, state); code != 200 {
			t.Fatalf("update returned %d", code)
		}
		if err = db.First(&stored, 1).Error; err != nil || stored.IsActive != state {
			t.Fatalf("wrong persisted state: %+v %v", stored, err)
		}
	}
	for _, state := range []int{-1, 2, 99} {
		if code := invoke(false, state); code != 400 {
			t.Fatalf("invalid create state %d returned %d", state, code)
		}
		if code := invoke(true, state); code != 400 {
			t.Fatalf("invalid update state %d returned %d", state, code)
		}
	}
	var count int64
	if err = db.Model(&model.CmdbCredentialPermission{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("invalid state created rows: %d %v", count, err)
	}
	if err = db.First(&stored, 1).Error; err != nil || stored.IsActive != 0 {
		t.Fatalf("invalid state modified grant: %+v %v", stored, err)
	}
}
