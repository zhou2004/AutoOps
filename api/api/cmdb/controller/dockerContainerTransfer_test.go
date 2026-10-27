package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dodevops-api/api/cmdb/model"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"github.com/gin-gonic/gin"
)

func TestDockerTransferPermissionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, actions string
		targetGrant   bool
		credential    uint
		profile       uint
		migrate       bool
		code          int
	}{
		{"target denied", `["docker_operate"]`, false, 51, 42, false, 403},
		{"migration requires source delete", `["docker_operate"]`, true, 51, 42, true, 403},
		{"profile belongs to source", `["docker_operate"]`, true, 51, 41, false, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, db := assetControllerFixture(t)
			if err := db.AutoMigrate(&model.CmdbPhysicalMachine{}); err != nil {
				t.Fatal(err)
			}
			for _, id := range []uint{1, 2} {
				if err := db.Create(&model.CmdbPhysicalMachine{ID: id, SN: "physical-" + string(rune('0'+id))}).Error; err != nil {
					t.Fatal(err)
				}
			}
			for _, profile := range []model.CmdbAssetConnection{
				{ID: 41, AssetType: "physical", AssetID: 1, Name: "source", Protocol: "ssh", Enabled: true, CredentialID: 51},
				{ID: 42, AssetType: "physical", AssetID: 2, Name: "target", Protocol: "ssh", Enabled: true, CredentialID: tc.credential},
			} {
				if err := db.Create(&profile).Error; err != nil {
					t.Fatal(err)
				}
			}
			grant := model.CmdbAssetPermission{Name: "source-docker", UserIDs: "[7]", GroupIDs: "[]", AssetTypes: `["physical"]`, PhysicalIDs: "[1]", PermissionActions: tc.actions, IsActive: 1}
			if err := db.Create(&grant).Error; err != nil {
				t.Fatal(err)
			}
			if tc.targetGrant {
				grant.ID = 0
				grant.Name = "target-docker"
				grant.PhysicalIDs = "[2]"
				if err := db.Create(&grant).Error; err != nil {
					t.Fatal(err)
				}
			}
			router := gin.New()
			router.Use(func(c *gin.Context) { c.Set(constant.ContextKeyUserObj, &systemmodel.JwtAdmin{ID: 7}); c.Next() })
			NewDockerOperationController(db).Register(router.Group("/cmdb"))
			body, _ := json.Marshal(map[string]any{"connectionId": 41, "targetAssetType": "physical", "targetAssetId": 2, "targetConnectionId": tc.profile, "name": "copy", "migrate": tc.migrate, "confirmName": "web", "start": true})
			req := httptest.NewRequest(http.MethodPost, "/cmdb/docker/physical/1/containers/web/transfer", strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			var response struct{ Code int }
			_ = json.Unmarshal(recorder.Body.Bytes(), &response)
			if response.Code != tc.code {
				t.Fatalf("got %s, want %d", recorder.Body.String(), tc.code)
			}
		})
	}
}
