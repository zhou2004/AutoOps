package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	model "dodevops-api/api/dashboard/model1"
	"dodevops-api/common/config"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Explicitly opt in; performs SELECTs in a read-only transaction, without migrations.
func TestDashboardConfiguredDatabaseReadOnly(t *testing.T) {
	if os.Getenv("DASHBOARD_LIVE") != "1" {
		t.Skip("set DASHBOARD_LIVE=1 and DASHBOARD_CONFIG_PATH for a read-only check")
	}
	if err := config.LoadConfig(os.Getenv("DASHBOARD_CONFIG_PATH")); err != nil {
		t.Fatal(err)
	}
	cfg := config.GetConfig()
	dsn := mysqlDriver.NewConfig()
	dsn.User, dsn.Passwd, dsn.DBName = cfg.Username, cfg.Password, cfg.Db
	dsn.Net, dsn.Addr = "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	dsn.ParseTime, dsn.Loc = true, dashboardLocation
	dsn.Timeout, dsn.ReadTimeout = 5*time.Second, 10*time.Second
	db, err := gorm.Open(mysql.Open(dsn.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx := db.WithContext(ctx).Begin(&sql.TxOptions{ReadOnly: true})
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	var userID uint
	if err := tx.Table("sys_admin u").Joins("JOIN sys_admin_role ar ON ar.admin_id = u.id").
		Joins("JOIN sys_role r ON r.id = ar.role_id").Where("u.status = 1 AND r.status = 1 AND r.role_key = ?", "admin").
		Order("u.id").Limit(1).Pluck("u.id", &userID).Error; err != nil || userID == 0 {
		t.Fatalf("no active administrator for scope verification: %v", err)
	}
	data, err := (&DashboardService{db: tx}).GetOverview(ctx, userID, nil, 30)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		table string
		value *model.Inventory
	}{
		{"cmdb_host", data.Hosts}, {"cmdb_sql", data.Databases}, {"cmdb_physical_machine", data.Physical},
		{"cmdb_network_device", data.Network}, {"k8s_cluster", data.Clusters}, {"app_application", data.Applications},
		{"monitor_api_endpoint", data.Endpoints}, {"monitor_domain_cert", data.Certificates},
	} {
		if entry.value == nil {
			continue
		}
		var expected int64
		if err := tx.Table(entry.table).Count(&expected).Error; err != nil || expected != entry.value.Total {
			t.Fatalf("%s count mismatch: expected=%d actual=%d error=%v", entry.table, expected, entry.value.Total, err)
		}
		t.Logf("%s: %d", entry.table, entry.value.Total)
	}
	for _, section := range []struct {
		name  string
		value *model.ExecutionOverview
	}{{"deployments", data.Deployments}, {"jobs", data.Jobs}, {"ansible", data.Ansible}} {
		if section.value != nil {
			t.Logf("%s: retained=%d; 30 days: total=%d success=%d failed=%d", section.name, section.value.AllTimeTotal, section.value.Total, section.value.Success, section.value.Failed)
		}
	}
	if len(data.Errors) > 0 {
		t.Fatalf("unavailable sources: %v", data.Errors)
	}
	if outputDir := os.Getenv("DASHBOARD_SNAPSHOT_DIR"); outputDir != "" {
		for _, days := range []int{7, 30, 90} {
			snapshot, err := (&DashboardService{db: tx}).GetOverview(ctx, userID, nil, days)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(outputDir, "dashboard-"+strconv.Itoa(days)+".json"), encoded, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
}
