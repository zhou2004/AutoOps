package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	cmdb "dodevops-api/api/cmdb/model"
	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/database/model"
)

func exerciseSQL(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	var source cmdb.CmdbSQL
	if s.DB.Where("cluster_id = ? AND connection_mode = ?", c.ID, "cluster").First(&source).Error != nil {
		t.Fatal("missing logical SQL entrypoint")
	}
	connector := cmdbservice.NewDatabaseOperationService(s.DB)
	databaseName, restorePort := "postgres", 19433
	query := func(q string) string {
		if c.Engine == "mysql" {
			q = strings.ReplaceAll(q, "public.autoops_ha_probe", "autoops_ha_probe")
			q = strings.ReplaceAll(q, "$1", "?")
			q = strings.ReplaceAll(q, "$2", "?")
		}
		return q
	}
	if c.Engine == "mysql" {
		databaseName = "autoops_database_lab"
		restorePort = 14307
		initDB, err := connector.OpenSQL(ctx, source.ID, "")
		if err != nil {
			t.Fatal(err)
		}
		_, err = initDB.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS autoops_database_lab")
		initDB.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	db, e := connector.OpenSQL(ctx, source.ID, databaseName)
	if e != nil {
		t.Fatal(e)
	}
	_, e = db.ExecContext(ctx, query("CREATE TABLE IF NOT EXISTS public.autoops_ha_probe (run_key varchar(191) PRIMARY KEY, payload text NOT NULL)"))
	db.Close()
	if e != nil {
		t.Fatal(e)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_EXERCISE") != "restore-only" {
		for round := 1; round <= 3; round++ {
			if c.Engine == "mysql" {
				if e = s.waitMySQLRecovery(ctx, c); e != nil {
					t.Fatal(e)
				}
				// Explicitly acknowledge a completed, repaired lab recovery before
				// the next deliberately injected fault. Native auto-recovery itself
				// remains independent of AutoOps.
				runLiveOperation(t, ctx, s, c, user, "acknowledge-recovery", Parameters{})
			}
			c, e = s.Refresh(ctx, c.ID)
			if e != nil {
				t.Fatal(e)
			}
			old, e := primary(c)
			if e != nil {
				t.Fatal(e)
			}
			var mu sync.Mutex
			ack := map[string]string{}
			unknown := []string{}
			last := time.Now()
			var outage, recovered time.Time
			consecutive := 0
			stop, done := make(chan struct{}), make(chan struct{})
			prefix := fmt.Sprintf("pg-%d-%d-", time.Now().UnixNano(), round)
			go func() {
				defer close(done)
				for i := 0; ; i++ {
					select {
					case <-stop:
						return
					default:
					}
					key := prefix + fmt.Sprint(i)
					value := fmt.Sprint(i)
					queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
					db, err := connector.OpenSQL(queryCtx, source.ID, databaseName)
					acknowledged := false
					if err == nil {
						_, err = db.ExecContext(queryCtx, query("INSERT INTO public.autoops_ha_probe(run_key,payload) VALUES ($1,$2)"), key, value)
						acknowledged = err == nil
						if err == nil {
							var got string
							err = db.QueryRowContext(queryCtx, query("SELECT payload FROM public.autoops_ha_probe WHERE run_key=$1"), key).Scan(&got)
							if err == nil && got != value {
								err = fmt.Errorf("read-after-write mismatch")
							}
						}
						db.Close()
					}
					cancel()
					mu.Lock()
					if acknowledged {
						ack[key] = value
					} else {
						unknown = append(unknown, key)
					}
					if err == nil {
						last = time.Now()
						if !outage.IsZero() {
							consecutive++
							if consecutive == 10 && recovered.IsZero() {
								recovered = last
							}
						}
					} else {
						if outage.IsZero() {
							outage = last
						}
						consecutive = 0
					}
					mu.Unlock()
					select {
					case <-stop:
						return
					case <-time.After(100 * time.Millisecond):
					}
				}
			}()
			_, e = s.ansible(ctx, c, []model.Member{*old}, "service.yml", map[string]any{"service_action": "stopped"}, nil)
			if e == nil {
				e = waitUntil(ctx, 90*time.Second, func() bool { mu.Lock(); defer mu.Unlock(); return !recovered.IsZero() })
			}
			close(stop)
			<-done
			var restartErr error
			if c.Engine == "mysql" {
				c, _ = s.Refresh(ctx, c.ID)
				if e == nil {
					runLiveOperation(t, ctx, s, c, user, "rejoin", Parameters{MemberID: old.ID})
				}
			} else {
				_, restartErr = s.ansible(ctx, c, []model.Member{*old}, "service.yml", map[string]any{"service_action": "started"}, nil)
			}
			if restartErr != nil {
				t.Fatal("old primary restart", restartErr)
			}
			if e != nil {
				t.Fatal("logical SQL entrypoint failed to recover", e)
			}
			if e = s.waitHealthy(ctx, c.ID, 120*time.Second, 0); e != nil {
				t.Fatal(e)
			}
			db, e := connector.OpenSQL(ctx, source.ID, databaseName)
			if e != nil {
				t.Fatal(e)
			}
			lost, present := 0, 0
			for key, value := range ack {
				var got string
				if e := db.QueryRowContext(ctx, query("SELECT payload FROM public.autoops_ha_probe WHERE run_key=$1"), key).Scan(&got); e != nil || got != value {
					lost++
				}
			}
			for _, key := range unknown {
				var count int
				if db.QueryRowContext(ctx, query("SELECT count(*) FROM public.autoops_ha_probe WHERE run_key=$1"), key).Scan(&count) == nil {
					present += count
				}
			}
			db.Close()
			rto := recovered.Sub(outage).Seconds()
			report := map[string]any{"round": round, "failedMemberId": old.ID, "rtoSeconds": rto, "acknowledged": len(ack), "acknowledgedLost": lost, "unknownWrites": len(unknown), "unknownPresent": present}
			t.Logf("%s entrypoint failover: %s", c.Topology, JSON(report))
			s.event(c.ID, user, 0, "integration-report", "info", string(JSON(report)))
			if rto > 60 || lost != 0 {
				t.Errorf("RTO or acknowledged-write protection failed: %s", JSON(report))
			}
		}
	}
	runLiveOperation(t, ctx, s, c, user, "collect-storage", Parameters{})
	bop := runLiveOperation(t, ctx, s, c, user, "backup", Parameters{Method: "logical"})
	var backup model.Backup
	s.DB.Where("operation_id = ?", bop.ID).First(&backup)
	runLiveOperation(t, ctx, s, c, user, "verify-backup", Parameters{BackupID: backup.ID, HostID: 530, Port: restorePort})
	t.Logf("%s logical backup %d restored to isolated port %d", c.Engine, backup.ID, restorePort)
}

func exerciseArchive(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	if !flag(object(c.Policy)["archiveConfigured"]) || os.Getenv("AUTOOPS_DB_LIVE_ARCHIVE_RECONFIGURE") == "1" {
		runLiveOperation(t, ctx, s, c, user, "configure-archive", Parameters{})
		c, _ = s.Cluster(c.ID)
	}
	op := runLiveOperation(t, ctx, s, c, user, "backup", Parameters{Method: "physical"})
	var base model.Backup
	s.DB.Where("operation_id = ?", op.ID).First(&base)
	runLiveOperation(t, ctx, s, c, user, "verify-backup", Parameters{BackupID: base.ID, HostID: 530, Port: 19434})
	var source cmdb.CmdbSQL
	s.DB.Where("cluster_id = ? AND connection_mode = ?", c.ID, "cluster").First(&source)
	db, e := cmdbservice.NewDatabaseOperationService(s.DB).OpenSQL(ctx, source.ID, "postgres")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	_, e = db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS public.autoops_pitr_probe(id text PRIMARY KEY)")
	if e != nil {
		t.Fatal(e)
	}
	before := fmt.Sprintf("before-%d", time.Now().UnixNano())
	after := fmt.Sprintf("after-%d", time.Now().UnixNano())
	if _, e = db.ExecContext(ctx, "INSERT INTO public.autoops_pitr_probe VALUES($1)", before); e != nil {
		t.Fatal(e)
	}
	var target time.Time
	if e = db.QueryRowContext(ctx, "SELECT clock_timestamp()").Scan(&target); e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, "INSERT INTO public.autoops_pitr_probe VALUES($1)", after); e != nil {
		t.Fatal(e)
	}
	archiveOp := runLiveOperation(t, ctx, s, c, user, "backup", Parameters{Method: "wal-archive", ParentBackupID: base.ID})
	var archive model.Backup
	s.DB.Where("operation_id = ?", archiveOp.ID).First(&archive)
	restore := runLiveOperation(t, ctx, s, c, user, "restore", Parameters{BackupID: archive.ID, HostID: 530, Port: 19435, TargetTime: target.UTC().Format(time.RFC3339Nano)})
	// Reopen only the retained loopback instance to verify the business boundary,
	// then stop it in a finally block. Never connects it to the original Patroni.
	var manifest struct {
		Members []model.Member `json:"members"`
	}
	json.Unmarshal(archive.Manifest, &manifest)
	root := fmt.Sprintf("/var/lib/autoops-database-restores/%d/%d/%d", c.ID, restore.ID, manifest.Members[0].ID)
	raw, e := s.remoteJSON(ctx, 530, `import json,sys,subprocess,time
p=json.load(sys.stdin);unit='autoops-pitr-boundary-'+str(p['operation']);root=p['root'];port=str(p['port'])
subprocess.run(['systemd-run','--unit',unit,'--property=User=autoopsdb','--property=RuntimeMaxSec=120','/usr/lib/postgresql/15/bin/postgres','-D',root+'/data'],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 import psycopg2
 for attempt in range(30):
  try: conn=psycopg2.connect(host='127.0.0.1',port=port,user='postgres',dbname='postgres',connect_timeout=1);break
  except Exception:time.sleep(1)
 else:raise RuntimeError('restore did not restart')
 with conn.cursor() as cur:
  cur.execute('SELECT id FROM autoops_pitr_probe WHERE id IN (%s,%s)',(p['before'],p['after']));rows={r[0] for r in cur.fetchall()}
 assert p['before'] in rows and p['after'] not in rows
 print(json.dumps({'pitrBoundaryVerified':True,'beforePresent':True,'afterPresent':False}))
finally:subprocess.run(['systemctl','stop',unit],check=True,stdout=subprocess.DEVNULL)`, map[string]any{"operation": restore.ID, "root": root, "port": 19435, "before": before, "after": after})
	if e != nil {
		t.Fatal("PITR business boundary not verified", e)
	}
	t.Logf("pgBackRest full backup=%d archive=%d PITR=%s report=%s", base.ID, archive.ID, target.UTC().Format(time.RFC3339Nano), raw)
}
