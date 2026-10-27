package controller

import (
	"testing"

	"dodevops-api/api/cmdb/model"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"github.com/gin-gonic/gin"
)

func TestDockerTargetsUseAssetDockerGrantsWithoutInventoryOrCredentialGrants(t *testing.T) {
	_, db := assetControllerFixture(t)
	for _, sql := range []string{
		"CREATE TABLE cmdb_host(id INTEGER, host_name TEXT)",
		"INSERT INTO cmdb_host VALUES (1,'allowed'), (2,'outside'), (3,'disabled')",
		"CREATE TABLE cmdb_physical_machine(id INTEGER, host_name TEXT)",
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	grant := model.CmdbAssetPermission{Name: "docker-only", UserIDs: "[7]", AssetTypes: `["host"]`, HostIDs: "[1,3]", PermissionActions: `["docker_view"]`, IsActive: 1}
	if err := db.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint{1, 2, 3} {
		if err := db.Create(&model.CmdbAssetConnection{ID: 40 + id, AssetType: "host", AssetID: id, Name: "SSH", Protocol: "ssh", Enabled: id != 3, CredentialID: 51}).Error; err != nil {
			t.Fatal(err)
		}
	}
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(constant.ContextKeyUserObj, &systemmodel.JwtAdmin{ID: 7}); c.Next() })
	NewDockerOperationController(db).Register(router.Group("/cmdb"))
	code, data := assetControllerRequest(t, router, "GET", "/cmdb/docker/targets", "")
	if code != 200 {
		t.Fatalf("Docker target discovery failed: %d", code)
	}
	hosts := data["hosts"].([]interface{})
	if len(hosts) != 1 || hosts[0].(map[string]interface{})["id"] != float64(1) {
		t.Fatalf("unexpected targets: %+v", data)
	}
	code, data = assetControllerRequest(t, router, "GET", "/cmdb/docker/assets/host/1/connections", "")
	if code != 200 {
		t.Fatalf("Docker connection discovery failed: %d", code)
	}
	profile := data["connections"].([]interface{})[0].(map[string]interface{})
	if profile["canDocker"] != true || profile["canConnect"] != false || profile["canManage"] != false {
		t.Fatalf("Docker grant expanded permissions: %+v", profile)
	}
	if err := db.Model(&grant).Update("is_active", 0).Error; err != nil {
		t.Fatal(err)
	}
	code, data = assetControllerRequest(t, router, "GET", "/cmdb/docker/targets", "")
	if code != 200 || len(data["hosts"].([]interface{})) != 0 {
		t.Fatal("revoked targets still visible")
	}
	if code, _ := assetControllerRequest(t, router, "GET", "/cmdb/docker/assets/host/1/connections", ""); code != 403 {
		t.Fatal("revoked connection still visible")
	}
}
