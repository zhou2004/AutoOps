package service

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"dodevops-api/api/cmdb/model"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/common/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Opt-in live check for the MySQL and Redis endpoints already configured for AutoOps.
// It uses a temporary asset registry and performs read-only operations on both services.
func TestConfiguredDatabaseServicesLive(t *testing.T) {
	if os.Getenv("CMDB_DATABASE_LIVE") != "1" {
		t.Skip("set CMDB_DATABASE_LIVE=1 to run configured database checks")
	}
	configPath := os.Getenv("CMDB_CONFIG_PATH")
	if configPath == "" {
		t.Fatal("CMDB_CONFIG_PATH is required")
	}
	if err := config.LoadConfig(configPath); err != nil {
		t.Fatal(err)
	}
	registry, err := gorm.Open(sqlite.Open(t.TempDir()+"/database-live.db"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.AutoMigrate(&model.CmdbSQL{}, &configmodel.AccountAuth{}); err != nil {
		t.Fatal(err)
	}
	mysqlConfig := config.GetConfig()
	mysqlAccount := configmodel.AccountAuth{ID: 1, Alias: "live-mysql", Host: mysqlConfig.Host, Port: mysqlConfig.Port, Name: mysqlConfig.Username, Password: mysqlConfig.Password, Type: 1}
	if err := mysqlAccount.EncryptPassword(); err != nil {
		t.Fatal(err)
	}
	redisConfig := config.GetRedisConfig()
	redisHost, redisPortText, err := net.SplitHostPort(redisConfig.Address)
	if err != nil {
		t.Fatalf("configured Redis address is invalid: %v", err)
	}
	redisPort, err := strconv.Atoi(redisPortText)
	if err != nil {
		t.Fatal("configured Redis port is invalid")
	}
	redisAccount := configmodel.AccountAuth{ID: 2, Alias: "live-redis", Host: redisHost, Port: redisPort, Password: redisConfig.Password, Type: 3}
	if err := redisAccount.EncryptPassword(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []any{&mysqlAccount, &redisAccount, &model.CmdbSQL{ID: 1, Name: "live-mysql", Type: 1, AccountID: 1, DefaultDB: mysqlConfig.Db, TLSMode: "disable"}, &model.CmdbSQL{ID: 2, Name: "live-redis", Type: 3, AccountID: 2, DefaultDB: "0", TLSMode: "disable"}} {
		if err := registry.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	service := NewDatabaseOperationService(registry)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if overview, err := service.Overview(ctx, 1); err != nil || overview.Version == "" {
		t.Fatalf("MySQL overview failed: %v", err)
	}
	if result, operation, err := service.Query(ctx, 1, mysqlConfig.Db, "SELECT 1 AS connection_ok"); err != nil || operation != "SELECT" || result.ReturnedRows != 1 {
		t.Fatalf("MySQL query failed: operation=%s result=%+v error=%v", operation, result, err)
	}
	if overview, err := service.Overview(ctx, 2); err != nil || overview.Version == "" {
		t.Fatalf("Redis overview failed: %v", err)
	}
}
