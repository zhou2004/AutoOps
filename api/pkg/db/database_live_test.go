package db

import (
	"fmt"
	"os"
	"testing"
	"time"

	cmdb "dodevops-api/api/cmdb/model"
	database "dodevops-api/api/database/model"
	"dodevops-api/common/config"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestLiveDatabaseLegacyMigration(t *testing.T) {
	if os.Getenv("AUTOOPS_DB_LIVE_MIGRATION") != "1" {
		t.Skip("explicit opt-in for existing CMDB metadata migration")
	}
	if e := config.LoadConfig("../../config.yaml"); e != nil {
		t.Fatal(e)
	}
	cfg := config.Config.Db
	dsn := mysqlDriver.NewConfig()
	dsn.User = cfg.Username
	dsn.Passwd = cfg.Password
	dsn.Net = "tcp"
	dsn.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	dsn.DBName = cfg.Db
	dsn.ParseTime = true
	dsn.Loc = time.Local
	db, e := gorm.Open(mysql.Open(dsn.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), DisableForeignKeyConstraintWhenMigrating: true})
	if e != nil {
		t.Fatal("metadata unavailable")
	}
	raw, _ := db.DB()
	defer raw.Close()
	if e = db.AutoMigrate(database.Models()...); e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&cmdb.CmdbSQL{}); e != nil {
		t.Fatal(e)
	}
	var existing []database.Cluster
	db.Find(&existing)
	management := map[uint]string{}
	for _, c := range existing {
		management[c.ID] = c.Management
	}
	var before []cmdb.CmdbSQL
	db.Find(&before)
	for i := 0; i < 2; i++ {
		if e = migrateDatabaseClusters(db); e != nil {
			t.Fatal(e)
		}
	}
	for _, b := range before {
		var after cmdb.CmdbSQL
		if db.First(&after, b.ID).Error != nil || after.AccountID != b.AccountID || after.GroupID != b.GroupID {
			t.Fatal("source identity or authorization association changed")
		}
	}
	var clusters []database.Cluster
	db.Where("deployment_id IS NOT NULL").Preload("Members").Find(&clusters)
	for _, c := range clusters {
		want := "observe"
		if old, ok := management[c.ID]; ok {
			want = old
		}
		if c.Management != want {
			t.Fatal("migration changed management level")
		}
		if _, ok := management[c.ID]; !ok && c.HAValidatedAt != nil {
			t.Fatal("newly migrated asset falsely HA validated")
		}
		t.Logf("legacy cluster %d %s: %d members, management preserved (%s)", c.ID, c.Name, len(c.Members), c.Management)
	}
}
