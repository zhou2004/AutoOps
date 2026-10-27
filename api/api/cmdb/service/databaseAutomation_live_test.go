package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/common/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Explicitly opt-in: installs isolated database services on the provided CMDB
// hosts and preserves them for UI inspection. Never touches existing instances.
func TestDatabaseAutomationLive(t *testing.T) {
	if os.Getenv("AUTOOPS_DB_LIVE") != "1" {
		t.Skip("set AUTOOPS_DB_LIVE=1 and AUTOOPS_DB_TEST_HOST_IDS to authorize live installation")
	}
	if err := config.LoadConfig(os.Getenv("CMDB_CONFIG_PATH")); err != nil {
		t.Fatal(err)
	}
	cfg := config.GetConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Db, cfg.Charset)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("CMDB connection failed")
	}
	if err = db.AutoMigrate(&model.DatabaseDeployment{}, &model.DatabaseAutomationRun{}, &model.DatabaseAutomationLock{}, &model.CmdbSQL{}); err != nil {
		t.Fatal(err)
	}
	ids := []uint{}
	for _, value := range strings.Split(os.Getenv("AUTOOPS_DB_TEST_HOST_IDS"), ",") {
		id, e := strconv.ParseUint(value, 10, 32)
		if e != nil || id == 0 {
			t.Fatal("explicit test host IDs required")
		}
		ids = append(ids, uint(id))
	}
	if len(ids) < 2 {
		t.Fatal("at least two authorized hosts required")
	}
	s := NewDatabaseAutomationService(db)
	first, err := s.hostConnection(ids[0])
	if err != nil {
		t.Fatal(err)
	}
	_ = first
	var host model.CmdbHost
	if db.First(&host, ids[0]).Error != nil {
		t.Fatal("host missing")
	}
	user := uint(0)
	db.Table("sys_admin").Select("id").Where("username = ?", "admin").Scan(&user)
	evidence := os.Getenv("AUTOOPS_DB_EVIDENCE_DIR")
	if evidence == "" {
		evidence = t.TempDir()
	}
	_ = os.MkdirAll(evidence, 0700)
	awaitRun := func(t *testing.T, id uint) model.DatabaseAutomationRun {
		deadline := time.Now().Add(80 * time.Minute)
		for time.Now().Before(deadline) {
			var run model.DatabaseAutomationRun
			if e := db.First(&run, id).Error; e != nil {
				t.Fatal(e)
			}
			if run.FinishedAt != nil {
				os.WriteFile(filepath.Join(evidence, fmt.Sprintf("run-%d.log", id)), []byte(run.Log), 0600)
				if run.Status != "success" {
					t.Fatalf("run %d %s failed (%s), log saved to evidence directory", id, run.Action, run.Status)
				}
				return run
			}
			time.Sleep(2 * time.Second)
		}
		t.Fatal("live run timed out")
		return model.DatabaseAutomationRun{}
	}
	secret, err := randomDatabaseSecret()
	if err != nil {
		t.Fatal(err)
	}
	managers := []uint{ids[0]}
	if len(ids) == 3 {
		managers = ids
	}
	requests := []DatabaseDeployRequest{
		{Name: "autoops-mysql-validation", Engine: "mysql", Mode: "replication", GroupID: host.GroupID, HostIDs: ids, ManagerIDs: managers, Port: 13306, Password: secret, AutoRecovery: true},
		{Name: "autoops-postgresql-validation", Engine: "postgresql", Mode: "standalone", GroupID: host.GroupID, HostIDs: []uint{ids[0]}, Port: 15432, Password: secret},
		{Name: "autoops-redis-validation", Engine: "redis", Mode: "standalone", GroupID: host.GroupID, HostIDs: []uint{ids[len(ids)-1]}, Port: 16379, Password: secret},
	}
	ops := NewDatabaseOperationService(db)
	for _, request := range requests {
		t.Run(request.Engine, func(t *testing.T) {
			var d model.DatabaseDeployment
			if e := db.Where("name = ?", request.Name).First(&d).Error; e != nil {
				created, run, e := s.Create(request, user)
				if e != nil {
					t.Fatal(e)
				}
				d = *created
				t.Logf("submitted deployment %d, run %d", d.ID, run.ID)
				awaitRun(t, run.ID)
			} else if d.Status != "ready" {
				var active model.DatabaseAutomationRun
				if db.Where("deployment_id = ? AND finished_at IS NULL", d.ID).Order("id DESC").First(&active).Error == nil {
					awaitRun(t, active.ID)
				} else {
					run, e := s.Start(d.ID, user, "retry-install", d.Name)
					if e != nil {
						t.Fatal(e)
					}
					t.Logf("retry deployment %d, run %d", d.ID, run.ID)
					awaitRun(t, run.ID)
				}
			}
			var assets []model.CmdbSQL
			if e := db.Where("deployment_id = ?", d.ID).Find(&assets).Error; e != nil {
				t.Fatal(e)
			}
			if len(assets) != len(request.HostIDs) {
				t.Fatalf("asset count %d, expected %d", len(assets), len(request.HostIDs))
			}
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			for _, asset := range assets {
				overview, e := ops.Overview(ctx, asset.ID)
				if e != nil {
					t.Fatal(e)
				}
				t.Logf("registered source %d: %s %s", asset.ID, overview.Address, overview.Version)
				if asset.Type != 3 {
					if _, _, e := ops.Query(ctx, asset.ID, asset.DefaultDB, "SELECT 42 AS live_validation"); e != nil {
						t.Fatal(e)
					}
				} else {
					client, _, e := ops.redisClient(asset.ID)
					if e != nil {
						t.Fatal(e)
					}
					key := "autoops:validation:" + fmt.Sprint(time.Now().UnixNano())
					if e := client.Set(ctx, key, "verified", time.Minute).Err(); e != nil {
						t.Fatal(e)
					}
					if value, e := client.Get(ctx, key).Result(); e != nil || value != "verified" {
						t.Fatal("redis roundtrip failed")
					}
					client.Del(ctx, key)
					client.Close()
				}
			}
			run, e := s.Start(d.ID, user, "backup", d.Name)
			if e != nil {
				t.Fatal(e)
			}
			awaitRun(t, run.ID)
			t.Logf("backup verified run %d", run.ID)
			if request.Engine == "mysql" {
				for _, view := range []string{"health", "topology", "analysis", "recoveries", "recovery-state"} {
					data, e := s.HAView(ctx, d.ID, view)
					if e != nil {
						t.Fatalf("%s: %v", view, e)
					}
					raw, _ := json.MarshalIndent(data, "", "  ")
					os.WriteFile(filepath.Join(evidence, view+".json"), raw, 0600)
				}
			}
		})
	}
}
