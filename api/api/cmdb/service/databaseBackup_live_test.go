package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/common/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDatabaseBackupRestoreLive(t *testing.T) {
	if os.Getenv("AUTOOPS_DB_RESTORE_LIVE") != "1" {
		t.Skip("set AUTOOPS_DB_RESTORE_LIVE=1 for isolated backup restoration tests")
	}
	if err := config.LoadConfig(os.Getenv("CMDB_CONFIG_PATH")); err != nil {
		t.Fatal(err)
	}
	cfg := config.GetConfig()
	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Db, cfg.Charset)), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("CMDB unavailable")
	}
	s := NewDatabaseAutomationService(db)
	ops := NewDatabaseOperationService(db)
	root, err := databaseAutomationRoot()
	if err != nil {
		t.Fatal(err)
	}
	user := uint(0)
	db.Table("sys_admin").Select("id").Where("username = ?", "admin").Scan(&user)
	for _, engine := range []string{"mysql", "postgresql", "redis"} {
		t.Run(engine, func(t *testing.T) {
			var d model.DatabaseDeployment
			if db.Where("name = ? AND status = ?", "autoops-"+engine+"-validation", "ready").First(&d).Error != nil {
				t.Fatal("validation deployment not ready")
			}
			var asset model.CmdbSQL
			if db.Where("deployment_id = ? AND host_id = ?", d.ID, decodeIDs(d.Nodes)[0]).First(&asset).Error != nil {
				t.Fatal("source missing")
			}
			marker := fmt.Sprintf("restore-%d", time.Now().UnixNano())
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			if engine == "redis" {
				client, _, e := ops.redisClient(asset.ID)
				if e != nil {
					t.Fatal(e)
				}
				if e = client.Set(ctx, "autoops:restore-check", marker, 0).Err(); e != nil {
					t.Fatal(e)
				}
				client.Close()
			} else {
				conn, _, e := ops.openSQL(ctx, asset.ID, asset.DefaultDB)
				if e != nil {
					t.Fatal(e)
				}
				if engine == "mysql" {
					for _, q := range []string{"CREATE DATABASE IF NOT EXISTS autoops_validation", "CREATE TABLE IF NOT EXISTS autoops_validation.restore_check (id INT PRIMARY KEY,payload VARCHAR(100))"} {
						if _, e = conn.ExecContext(ctx, q); e != nil {
							t.Fatal(e)
						}
					}
					_, e = conn.ExecContext(ctx, "INSERT INTO autoops_validation.restore_check(id,payload) VALUES(1,?) ON DUPLICATE KEY UPDATE payload=VALUES(payload)", marker)
				} else {
					if _, e = conn.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS public.autoops_restore_check (id INT PRIMARY KEY,payload VARCHAR(100))"); e != nil {
						t.Fatal(e)
					}
					_, e = conn.ExecContext(ctx, "INSERT INTO public.autoops_restore_check(id,payload) VALUES(1,$1) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload", marker)
				}
				conn.Close()
				if e != nil {
					t.Fatal(e)
				}
			}
			run, err := s.Start(d.ID, user, "backup", d.Name)
			if err != nil {
				t.Fatal(err)
			}
			for {
				if ctx.Err() != nil {
					t.Fatal("backup timed out")
				}
				if err := db.First(run, run.ID).Error; err != nil {
					t.Fatal(err)
				}
				if run.FinishedAt != nil {
					break
				}
				time.Sleep(time.Second)
			}
			if run.Status != "success" {
				t.Fatal("backup failed")
			}
			sec, err := s.secrets(&d)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			inventory, err := s.inventory(d, dir)
			if err != nil {
				t.Fatal(err)
			}
			data, _ := json.Marshal(inventory)
			if err := os.WriteFile(filepath.Join(dir, "inventory.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			data, _ = json.Marshal(map[string]any{"database_engine": engine, "database_port": d.Port, "database_name": d.Name, "database_password": sec.Password, "database_run_id": run.ID, "mysql_version": d.Version, "validation_marker": marker})
			if err := os.WriteFile(filepath.Join(dir, "vars.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.CommandContext(ctx, "ansible-playbook", "-i", filepath.Join(dir, "inventory.json"), filepath.Join(root, "playbooks", "verify-backup.yml"), "--extra-vars", "@"+filepath.Join(dir, "vars.json"))
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "ANSIBLE_HOST_KEY_CHECKING=True", "ANSIBLE_NOCOLOR=1", "ANSIBLE_DISPLAY_ARGS_TO_STDOUT=False", "ANSIBLE_ROLES_PATH="+filepath.Join(root, "roles"))
			log := &databaseRunLog{secrets: []string{sec.Password, sec.Replication, sec.HTTP, sec.Topology}, save: func(string) {}}
			cmd.Stdout = log
			cmd.Stderr = log
			err = cmd.Run()
			if evidence := os.Getenv("AUTOOPS_DB_EVIDENCE_DIR"); evidence != "" {
				os.WriteFile(filepath.Join(evidence, "restore-"+engine+".log"), []byte(log.redacted()), 0600)
			}
			if err != nil {
				t.Fatalf("restoration failed; inspect restore-%s.log", engine)
			}
			t.Logf("backup run %d restored and sentinel value matched; isolated process cleaned up", run.ID)
		})
	}
}
