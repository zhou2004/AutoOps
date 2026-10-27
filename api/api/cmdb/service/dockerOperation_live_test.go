package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/common/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Opt-in live check. It reads the existing CMDB asset and pinned SSH profile,
// then runs only Docker inventory commands on the target machine.
func TestDockerInventoryLive(t *testing.T) {
	if os.Getenv("CMDB_DOCKER_LIVE") != "1" {
		t.Skip("set CMDB_DOCKER_LIVE=1 to run the read-only Docker inventory check")
	}
	configPath := os.Getenv("CMDB_CONFIG_PATH")
	if configPath == "" {
		t.Fatal("CMDB_CONFIG_PATH is required")
	}
	if err := config.LoadConfig(configPath); err != nil {
		t.Fatal(err)
	}
	cfg := config.GetConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Db, cfg.Charset)
	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot open CMDB database")
	}
	var asset model.CmdbPhysicalMachine
	if err := database.Where("LOWER(host_name) = ?", "lenovo").First(&asset).Error; err != nil {
		t.Fatal("lenovo physical asset is missing")
	}
	var profile model.CmdbAssetConnection
	if err := database.Where("asset_type = ? AND asset_id = ? AND protocol = ? AND enabled = ?", "physical", asset.ID, "ssh", true).Order("id ASC").First(&profile).Error; err != nil {
		t.Fatal("lenovo has no enabled SSH connection profile")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	service := NewDockerOperationService(database)
	info, err := service.Info(ctx, "physical", asset.ID, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if info["ServerVersion"] == nil || info["Name"] == nil {
		t.Fatalf("incomplete Docker daemon inventory: %+v", info)
	}
	checks := []struct {
		name string
		fn   func() error
	}{
		{"containers", func() error { _, err := service.Containers(ctx, "physical", asset.ID, profile.ID); return err }},
		{"images", func() error { _, err := service.Images(ctx, "physical", asset.ID, profile.ID); return err }},
		{"networks", func() error { _, err := service.Networks(ctx, "physical", asset.ID, profile.ID); return err }},
		{"volumes", func() error { _, err := service.Volumes(ctx, "physical", asset.ID, profile.ID); return err }},
	}
	for _, check := range checks {
		if err := check.fn(); err != nil {
			t.Fatalf("%s inventory failed: %v", check.name, err)
		}
	}
}
