package service

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/common/util"
	"dodevops-api/pkg/db"
	"github.com/gin-gonic/gin"
)

func TestWindowsAndDeferredHostRegistration(t *testing.T) {
	database := connectionTestDB(t)
	if err := database.AutoMigrate(&model.CmdbGroup{}, &model.CmdbHost{}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.CmdbGroup{ID: 1, Name: "servers", CreateTime: util.HTime{Time: time.Now()}}).Error; err != nil {
		t.Fatal(err)
	}
	previous := db.Db
	db.Db = database
	t.Cleanup(func() { db.Db = previous })
	service := GetCmdbHostService()
	for _, input := range []model.CreateCmdbHostDto{
		{HostName: "windows", GroupID: 1, OSType: "windows", OS: "Windows Server", PrivateIP: "192.0.2.30"},
		{HostName: "linux-desktop", GroupID: 1, OSType: "linux", SkipSSH: true, PrivateIP: "192.0.2.31"},
	} {
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		service.CreateCmdbHost(ctx, &input)
		var result struct {
			Code int `json:"code"`
			Data struct {
				ID uint `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Code != 200 || result.Data.ID == 0 {
			t.Fatalf("registration failed: %s", response.Body.String())
		}
		var host model.CmdbHost
		if err := database.First(&host, result.Data.ID).Error; err != nil {
			t.Fatal(err)
		}
		if host.OSType != input.OSType || host.PrivateIP != input.PrivateIP || host.Status != 2 || host.SSHKeyID != 0 || host.SSHIP != "" {
			t.Fatalf("unexpected inventory: %+v", host)
		}
		response = httptest.NewRecorder()
		ctx, _ = gin.CreateTestContext(response)
		service.UpdateCmdbHost(ctx, host.ID, &model.UpdateCmdbHostDto{ID: host.ID, HostName: input.HostName, GroupID: 1, OSType: input.OSType, OS: input.OS, PrivateIP: input.PrivateIP, SkipSSH: true})
		var updateResult struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &updateResult); err != nil {
			t.Fatal(err)
		}
		if updateResult.Code != 200 {
			t.Fatalf("deferred inventory update failed: %s", response.Body.String())
		}
	}
}
