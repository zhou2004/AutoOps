package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/common/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSQLOperationGuard(t *testing.T) {
	valid := map[string]string{
		" SELECT 1; ": "SELECT",
		"/* report */ WITH x AS (SELECT 1) SELECT * FROM x": "WITH",
		"-- comment\nUPDATE users SET active=1 WHERE id=7":  "UPDATE",
	}
	for statement, want := range valid {
		got, err := sqlOperation(statement)
		if err != nil || got != want {
			t.Fatalf("%q: got %q, %v", statement, got, err)
		}
	}
	for _, statement := range []string{"", "SELECT 1; DROP TABLE users", strings.Repeat("x", 256*1024+1)} {
		if _, err := sqlOperation(statement); err == nil {
			t.Fatalf("unsafe SQL accepted: %.40q", statement)
		}
	}
}

func TestDatabaseAssetValidation(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.CmdbSQL{}, &model.CmdbGroup{}, &configmodel.AccountAuth{}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.CmdbGroup{ID: 1, Name: "data", CreateTime: util.HTime{Time: time.Now()}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&configmodel.AccountAuth{ID: 1, Alias: "mysql", Host: "127.0.0.1", Port: 3306, Name: "tester", Password: "encrypted", Type: 1}).Error; err != nil {
		t.Fatal(err)
	}
	service := NewCmdbSQLService(dao.NewCmdbSQLDao(database))
	asset := model.CmdbSQL{Name: " primary ", Type: 1, AccountID: 1, GroupID: 1}
	if err := service.CreateDatabase(&asset); err != nil {
		t.Fatal(err)
	}
	if asset.Name != "primary" || asset.TLSMode != "disable" {
		t.Fatalf("asset was not normalized: %+v", asset)
	}
	wrongType := model.CmdbSQL{Name: "redis", Type: 3, AccountID: 1, GroupID: 1, DefaultDB: "0"}
	if err := service.CreateDatabase(&wrongType); err == nil {
		t.Fatal("mismatched credential type accepted")
	}
	badRedisDB := model.CmdbSQL{Name: "redis", Type: 3, AccountID: 1, GroupID: 1, DefaultDB: "all"}
	if err := service.CreateDatabase(&badRedisDB); err == nil {
		t.Fatal("invalid Redis DB accepted")
	}
}

func TestHostVendorRemainsNumericInAPI(t *testing.T) {
	payload, err := json.Marshal(model.CmdbHostVo{Vendor: 2})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"id":0,"hostName":"","sshIp":"","sshPort":0,"sshKeyId":0,"groupId":0,"status":0,"cpu":"","cpuNum":0,"memory":0,"disk":0,"os":"","osType":"","image":"","privateIp":"","publicIp":"","remoteProtocols":null,"agent":0,"agentType":"","agentId":0,"agentVersion":"","agentInstallTime":"","agentLastHeartbeat":"","agentStatus":"","agentErrorMessage":"","remark":"","createTime":"0001-01-01T00:00:00Z","updateTime":"0001-01-01T00:00:00Z","vendor":2}` {
		if !strings.Contains(string(payload), `"vendor":2`) || strings.Contains(string(payload), `"vendor":"2"`) {
			t.Fatalf("vendor changed type in API response: %s", payload)
		}
	}
}
