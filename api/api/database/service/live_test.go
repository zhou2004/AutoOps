package service

import (
	"context"
	cmdbservice "dodevops-api/api/cmdb/service"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/database/model"
	"dodevops-api/common/config"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Explicit opt-in: never runs during the normal test suite. Only uses the three
// CMDB hosts authorized for this project, and newly named services/ports.
func TestLiveDatabasePlatform(t *testing.T) {
	topology := os.Getenv("AUTOOPS_DB_LIVE_TOPOLOGY")
	if topology == "" {
		t.Skip("set AUTOOPS_DB_LIVE_TOPOLOGY to opt into authorized CMDB integration")
	}
	if e := config.LoadConfig("../../../config.yaml"); e != nil {
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
		t.Fatal("metadata connection failed")
	}
	raw, _ := db.DB()
	defer raw.Close()
	if e = db.AutoMigrate(model.Models()...); e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&cmdb.CmdbSQL{}, &cmdb.CmdbAssetPermission{}); e != nil {
		t.Fatal(e)
	}
	s := New(db)
	var user uint
	db.Table("sys_admin u").Select("u.id").Joins("JOIN sys_admin_role ar ON ar.admin_id = u.id JOIN sys_role r ON r.id = ar.role_id").Where("u.status = 1 AND r.role_key = ? AND r.status = 1", "admin").Order("u.id").Limit(1).Scan(&user)
	if !s.IsAdmin(user) {
		t.Fatal("integration identity is not an administrator")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	engine := "redis"
	port := 17379
	if topology == "postgresql-patroni" {
		engine = "postgresql"
		port = 15433
	}
	if topology == "redis-cluster" {
		port = 18379
	}
	name := "db-lab-" + topology
	if topology == "mysql-gtid-legacy" {
		engine = "mysql"
		port = 13306
		name = "autoops-mysql-validation"
	}
	var c *model.Cluster
	var existing model.Cluster
	if db.Where("name = ?", name).First(&existing).Error == nil {
		c, e = s.Cluster(existing.ID)
		if e != nil {
			t.Fatal(e)
		}
	} else {
		if topology == "mysql-gtid-legacy" {
			t.Fatal("migrate the existing authorized MySQL deployment first")
		}
		ids := []uint{530, 528, 531}
		members := []MemberInput{}
		count := 3
		if topology == "redis-cluster" {
			count = 9
		}
		for i := 0; i < count; i++ {
			host := i % 3
			if topology == "redis-cluster" {
				host = (i%3 + i/3) % 3
			}
			members = append(members, MemberInput{HostID: ids[host], Port: port + i/3, Kind: "data"})
		}
		c, e = s.Create(ctx, ClusterRequest{Name: name, Engine: engine, Topology: topology, GroupID: 18, Environment: "integration", Provision: true, Members: members, Policy: JSON(map[string]any{"memoryMB": 64})}, user)
		if e != nil {
			t.Fatal(e)
		}
	}
	if value := os.Getenv("AUTOOPS_DB_LIVE_QUEUED_RUNTIME"); value != "" {
		id, err := strconv.ParseUint(value, 10, 32)
		if err != nil || c.ID != 4 {
			t.Fatal("invalid authorized API task fixture")
		}
		var op model.Operation
		if s.DB.Where("id = ? AND cluster_id = ? AND user_id = ? AND action = ? AND status = ?", id, c.ID, user, "configure", "queued").First(&op).Error != nil {
			t.Fatal("queued runtime task absent")
		}
		var p Parameters
		if json.Unmarshal(op.Parameters, &p) != nil || !p.RuntimeOnly || p.MemberID == 0 {
			t.Fatal("not a runtime configuration task")
		}
		op.WorkerID = "integration-workbench-api"
		if s.DB.Model(&model.Operation{}).Where("id = ? AND status = ?", op.ID, "queued").Updates(map[string]any{"worker_id": op.WorkerID, "status": "preflight", "heartbeat_at": time.Now()}).RowsAffected != 1 {
			t.Fatal("task already claimed")
		}
		s.run(ctx, op)
		s.DB.First(&op, op.ID)
		if op.Status != "succeeded" {
			t.Fatal(op.Status, op.Message)
		}
		t.Logf("workbench API task #%d executed via durable worker with member scope", op.ID)
		return
	}
	if os.Getenv("AUTOOPS_DB_LIVE_PG_REPAIR") == "preserve-confirmed-branch" {
		repairPGTestBranch(t, ctx, s, c)
		return
	}
	if os.Getenv("AUTOOPS_DB_LIVE_PG_REPAIR") == "update-and-rebuild" {
		finishPGTestRepair(t, ctx, s, c, user)
		return
	}
	if assignment := os.Getenv("AUTOOPS_DB_LIVE_ASSIGN"); assignment != "" {
		var node, target uint
		if c.ID != 3 {
			t.Fatal("wrong replica repair fixture")
		}
		if _, err := fmt.Sscanf(assignment, "%d:%d", &node, &target); err != nil {
			t.Fatal(err)
		}
		runLiveOperation(t, ctx, s, c, user, "assign-replica", Parameters{MemberID: node, CandidateID: target})
		return
	}
	if pending := os.Getenv("AUTOOPS_DB_LIVE_RECONCILE"); pending != "" {
		id, e := strconv.ParseUint(pending, 10, 32)
		if e != nil {
			t.Fatal(e)
		}
		runLiveOperation(t, ctx, s, c, user, "reconcile", Parameters{OperationID: uint(id)})
		c, e = s.Cluster(c.ID)
		if e != nil {
			t.Fatal(e)
		}
	}
	if os.Getenv("AUTOOPS_DB_LAB_REPAIR") == "finalize-installed" {
		if c.Environment != "integration" || c.Name != "db-lab-postgresql-patroni" {
			t.Fatal("finalization is restricted to the isolated PostgreSQL lab")
		}
		var old model.Operation
		if e := s.DB.Where("cluster_id = ? AND action = ? AND status = ?", c.ID, "provision", "unknown").First(&old).Error; e != nil || old.FinishedAt == nil {
			t.Fatal("original installer must have terminated")
		}
		for _, m := range c.Members {
			_, e := s.remoteJSON(ctx, m.HostID, `import json,sys,pathlib
p=json.load(sys.stdin);d=pathlib.Path(p['dataDir']);assert (d/'.autoops-owner').read_text()==str(p['clusterId'])+':'+str(p['id'])
for proc in pathlib.Path('/proc').glob('[0-9]*/cmdline'):
 try: cmd=proc.read_bytes()
 except (OSError,PermissionError): continue
 assert (b'Ansiball'+b'Z_') not in cmd
print(json.dumps({'owned':True,'installerIdle':True}))`, m)
			if e != nil {
				t.Fatal("cannot prove installer quiescence and directory ownership", e)
			}
		}
		fresh, e := s.Refresh(ctx, c.ID)
		if e != nil {
			t.Fatal(e)
		}
		h := object(fresh.Health)
		if h["database"] != "healthy" || h["replication"] != "healthy" || h["control"] != "healthy" {
			t.Fatalf("data/control bootstrap not complete: %s", fresh.Health)
		}
		proxies := []model.Member{}
		for _, m := range fresh.Members {
			if m.Kind == "proxy" {
				proxies = append(proxies, m)
			}
		}
		if _, e = s.ansible(ctx, fresh, proxies, "endpoint.yml", nil, func(v string) { t.Log(v) }); e != nil {
			t.Fatal(e)
		}
		if e = s.registerSources(fresh); e != nil {
			t.Fatal(e)
		}
		fresh, e = s.Refresh(ctx, c.ID)
		if e != nil || fresh.Status != "healthy" {
			t.Fatalf("final verification failed: %v %s", e, fresh.Health)
		}
		s.complete(old, "succeeded", "安装器已退出；核验目录归属、三节点数据与控制服务，补齐独立代理并验证逻辑入口")
		c = fresh
	}
	if c.Status == "planned" {
		if os.Getenv("AUTOOPS_DB_LAB_REPAIR") == "empty-provision" {
			if c.Environment != "integration" || c.Name != "db-lab-"+topology {
				t.Fatal("not an isolated integration cluster")
			}
			for _, m := range c.Members {
				_, e := s.remoteJSON(ctx, m.HostID, `import json,sys,os,socket
p=json.load(sys.stdin);d=p['dataDir'];sock=socket.socket();sock.settimeout(1)
assert sock.connect_ex(('127.0.0.1',p['port']))!=0
assert not os.path.exists('/etc/systemd/system/'+p['service']+'.service')
if os.path.isdir(d):
 assert set(os.listdir(d))=={'.autoops-owner'}
 assert open(d+'/.autoops-owner').read()==str(p['clusterId'])+':'+str(p['id'])
 os.unlink(d+'/.autoops-owner');os.rmdir(d)
print(json.dumps({'emptyOwnedDirectoryRemoved':True}))`, m)
				if e != nil {
					t.Fatal("cannot verify empty failed installation", e)
				}
			}
			var failed []model.Operation
			db.Where("cluster_id = ? AND status = ?", c.ID, "unknown").Find(&failed)
			for _, op := range failed {
				s.complete(op, "failed", "已检查远端未启动服务且仅存在本次安装的空目录；清理该空目录后重新生成安装计划")
			}
		}
		var abandoned model.Operation
		if db.Where("cluster_id = ? AND status = ? AND message = ?", c.ID, "unknown", "数据库自动化目录不存在").First(&abandoned).Error == nil {
			for _, check := range s.preflightHosts(ctx, c) {
				if !check.Passed {
					t.Fatalf("cannot reconcile untouched deployment: %s", check.Name)
				}
			}
			s.complete(abandoned, "failed", "执行器在启动 Ansible 前失败；已核验远端目录和端口未变更")
		}
		plan, e := s.Plan(ctx, c.ID, user, "provision", Parameters{})
		if e != nil {
			t.Fatal(e)
		}
		for _, check := range planChecks(plan) {
			if !check.Passed {
				t.Logf("blocked: %s: %s", check.Name, check.Message)
			}
		}
		if !plan.Executable {
			t.Fatal("preflight rejected deployment")
		}
		op, e := s.Submit(ctx, c.ID, user, plan.ID, fmt.Sprintf("integration-%d", time.Now().UnixNano()), c.Name)
		if e != nil {
			t.Fatal(e)
		}
		claimed, e := s.claim("integration")
		if e != nil || claimed == nil || claimed.ID != op.ID {
			t.Fatal("worker claim failed")
		}
		s.run(ctx, *claimed)
		var done model.Operation
		db.Preload("Steps").First(&done, op.ID)
		for _, step := range done.Steps {
			t.Logf("%s: %s\n%s", step.Name, step.Status, step.Log)
		}
		if done.Status != "succeeded" {
			t.Fatalf("deployment %s: %s", done.Status, done.Message)
		}
	}
	c, e = s.Refresh(ctx, c.ID)
	if e != nil {
		t.Fatal(e)
	}
	if topology == "mysql-gtid-legacy" {
		for _, m := range c.Members {
			ownedExpansion := os.Getenv("AUTOOPS_DB_LIVE_MEMBERSHIP") != "" && m.HostID == 531 && m.Port >= 13307 && m.Port <= 13309 && m.Service == fmt.Sprintf("autoops-db-4-531-%d", m.Port) && m.DataDir == fmt.Sprintf("/var/lib/autoops-database/autoops-mysql-validation/531-%d", m.Port)
			if m.Kind == "data" && (m.Port != 13306 || m.Service != "autoops-mysql") && !ownedExpansion {
				t.Fatal("not the isolated authorized MySQL instance")
			}
		}
		if c.Management == "observe" {
			runLiveOperation(t, ctx, s, c, user, "adopt", Parameters{Management: "operate"})
			c, _ = s.Cluster(c.ID)
		}
		if value := os.Getenv("AUTOOPS_DB_LIVE_REPAIR_ROOT"); value != "" {
			id, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				t.Fatal(err)
			}
			runLiveOperation(t, ctx, s, c, user, "repair-replication", Parameters{CandidateID: uint(id)})
			c, _ = s.Refresh(ctx, c.ID)
		}
		if value := os.Getenv("AUTOOPS_DB_LIVE_REJOIN"); value != "" {
			id, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				t.Fatal(err)
			}
			runLiveOperation(t, ctx, s, c, user, "rejoin", Parameters{MemberID: uint(id)})
			c, _ = s.Refresh(ctx, c.ID)
		}
		if !flag(object(c.Policy)["fencingInstalled"]) || os.Getenv("AUTOOPS_DB_LIVE_FENCING_RECONFIGURE") == "1" {
			runLiveOperation(t, ctx, s, c, user, "configure-fencing", Parameters{})
			c, _ = s.Cluster(c.ID)
		}
		root, rootErr := primary(c)
		if rootErr == nil && flag(object(root.Metrics)["read_only"]) {
			runLiveOperation(t, ctx, s, c, user, "resume-writes", Parameters{})
			c, _ = s.Refresh(ctx, c.ID)
		}
		proxies := 0
		for _, m := range c.Members {
			if m.Kind == "proxy" {
				proxies++
			}
		}
		if proxies == 0 || os.Getenv("AUTOOPS_DB_LIVE_ENDPOINT_RECONFIGURE") == "1" {
			runLiveOperation(t, ctx, s, c, user, "configure-endpoint", Parameters{Members: []MemberInput{{HostID: 530, Port: 23306, Kind: "proxy"}, {HostID: 528, Port: 23306, Kind: "proxy"}}})
		}
		c, e = s.Refresh(ctx, c.ID)
		if e != nil {
			t.Fatal(e)
		}
		if os.Getenv("AUTOOPS_DB_LIVE_EXIT_MAINTENANCE") == "1" {
			runLiveOperation(t, ctx, s, c, user, "exit-maintenance", Parameters{})
		}
	}
	t.Logf("cluster=%d status=%s topology=%s health=%s", c.ID, c.Status, c.Topology, c.Health)
	for _, m := range c.Members {
		t.Logf("%s %s:%d %s %s", m.Name, m.Address, m.Port, m.Role, m.State)
	}
	healthForAcceptance := c.Health
	if os.Getenv("AUTOOPS_DB_LIVE_MEMBERSHIP") == "resume" {
		existing := *c
		existing.Members = nil
		for _, m := range c.Members {
			if !m.BaselinePending {
				existing.Members = append(existing.Members, m)
			}
		}
		healthForAcceptance = health(&existing)
	}
	if h := object(healthForAcceptance); h["database"] != "healthy" || h["replication"] == "degraded" || h["control"] == "degraded" || h["endpoint"] == "unavailable" || h["endpoint"] == "conflict" {
		t.Fatal("database, replication or access entrypoint is not healthy")
	}
	if os.Getenv("AUTOOPS_DB_LIVE_EXERCISE") != "" && c.Engine == "redis" {
		if os.Getenv("AUTOOPS_DB_LAB_REPAIR") == "restore-before-start" {
			var pending []model.Operation
			s.DB.Preload("Steps").Where("cluster_id = ? AND action = ? AND status = ?", c.ID, "verify-backup", "unknown").Find(&pending)
			for _, op := range pending {
				known := false
				for _, step := range op.Steps {
					if strings.Contains(step.Log, "unexpected keyword argument 'filter'") || strings.Contains(step.Log, "Unsupported parameters for (ansible.builtin.apt) module: lock_timeout") {
						known = true
					}
				}
				if !known {
					t.Fatal("unknown failure is not a pre-start extraction compatibility failure")
				}
				var p Parameters
				json.Unmarshal(op.Parameters, &p)
				var backup model.Backup
				s.DB.First(&backup, p.BackupID)
				var manifest struct {
					Members []model.Member `json:"members"`
				}
				json.Unmarshal(backup.Manifest, &manifest)
				for _, m := range manifest.Members {
					_, e := s.remoteJSON(ctx, p.HostID, `import json,sys,os,socket
p=json.load(sys.stdin);s=socket.socket();s.settimeout(1);assert s.connect_ex(('127.0.0.1',p['port']))!=0
assert not os.path.exists(p['path']+'/payload')
print(json.dumps({'notStarted':True}))`, map[string]any{"port": p.Port, "path": fmt.Sprintf("/var/lib/autoops-database-restores/%d/%d/%d", c.ID, op.ID, m.ID)})
					if e != nil {
						t.Fatal(e)
					}
				}
				s.complete(op, "failed", "已核验恢复在解包前失败，未启动隔离实例；保留失败现场并使用新目录重试")
			}
		}
		exerciseRedis(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_EXERCISE") != "" && (c.Engine == "postgresql" || c.Engine == "mysql") {
		exerciseSQL(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_ARCHIVE") == "1" && c.Engine == "postgresql" {
		exerciseArchive(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_PROTECTION") == "1" && (c.Engine == "postgresql" || c.Engine == "mysql") {
		exerciseProtection(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_CONTROL") == "1" && (c.Topology == "redis-sentinel" || c.Engine == "mysql") {
		exerciseControlQuorum(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_MEMBERSHIP") != "" && c.Topology != "redis-cluster" {
		exerciseMembership(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_BALANCE") == "1" {
		runLiveOperation(t, ctx, s, c, user, "balance-primaries", Parameters{})
	}
	if os.Getenv("AUTOOPS_DB_LIVE_RUNTIME_CONFIG") == "1" {
		exerciseRuntimeConfiguration(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_DISCOVERY") == "1" {
		exerciseDiscovery(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_REBUILD") == "1" && c.Engine == "postgresql" {
		for _, m := range c.Members {
			if m.Kind == "data" && m.Role == "replica" {
				runLiveOperation(t, ctx, s, c, user, "rebuild-replica", Parameters{MemberID: m.ID})
				break
			}
		}
	}
	if os.Getenv("AUTOOPS_DB_LIVE_STORAGE_FAILURE") == "1" {
		exerciseStorageFailure(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_CONFIGURATION") == "1" {
		exerciseConfiguration(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_RETENTION") == "1" {
		exerciseRetention(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_PARTITION") == "1" {
		exercisePartition(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_REPOSITORY") == "1" && c.Topology == "redis-sentinel" {
		exerciseRepository(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_MYSQL_ARCHIVE") == "1" && c.Engine == "mysql" {
		exerciseMySQLArchive(t, ctx, s, c, user)
	}
	if os.Getenv("AUTOOPS_DB_LIVE_FENCE_FAILURE") == "1" && c.Engine == "mysql" {
		exerciseFencingFailure(t, ctx, s, c, user)
	}

	if os.Getenv("AUTOOPS_DB_LIVE_SLOTS") != "" && c.Topology == "redis-cluster" {
		exerciseSlots(t, ctx, s, c, user)
	}
}

func exerciseRedis(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	var source cmdb.CmdbSQL
	if s.DB.Where("cluster_id = ? AND connection_mode = ?", c.ID, "cluster").First(&source).Error != nil {
		t.Fatal("logical cluster source not registered")
	}
	client, e := cmdbservice.NewDatabaseOperationService(s.DB).OpenRedis(source.ID)
	if e != nil {
		t.Fatal(e)
	}
	defer client.Close()
	if e = client.Ping(ctx).Err(); e != nil {
		t.Fatal("cluster-aware client failed", e)
	}
	reports := []map[string]any{}
	rounds := 3
	if os.Getenv("AUTOOPS_DB_LIVE_EXERCISE") == "restore-only" {
		rounds = 0
	}
	for round := 1; round <= rounds; round++ {
		if os.Getenv("AUTOOPS_DB_LIVE_EXERCISE") == "host-domain" {
			runLiveOperation(t, ctx, s, c, user, "balance-primaries", Parameters{})
		}
		c, e = s.Refresh(ctx, c.ID)
		if e != nil {
			t.Fatal(e)
		}
		leader, e := primary(c)
		if e != nil {
			t.Fatal(e)
		}
		prefix := fmt.Sprintf("autoops:exercise:%d:%d:", time.Now().UnixNano(), round)
		if c.Topology == "redis-cluster" {
			probe, e := s.redis(c, *leader)
			if e != nil {
				t.Fatal(e)
			}
			found := false
			for n := 0; n < 100; n++ {
				tag := fmt.Sprintf("exercise-%d-%d", round, n)
				slot, e := probe.ClusterKeySlot(ctx, "{"+tag+"}").Result()
				if e != nil {
					probe.Close()
					t.Fatal(e)
				}
				var ranges []string
				json.Unmarshal(JSON(object(leader.Metrics)["slots"]), &ranges)
				for _, r := range ranges {
					var start, end int
					if _, e := fmt.Sscanf(r, "%d-%d", &start, &end); e != nil {
						fmt.Sscanf(r, "%d", &start)
						end = start
					}
					if int(slot) >= start && int(slot) <= end {
						prefix = "{" + tag + "}:" + prefix
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			probe.Close()
			if !found {
				t.Fatal("could not pin workload to failed shard")
			}
		}
		var mu sync.Mutex
		ack := map[string]string{}
		unknown := []string{}
		lastSuccess := time.Now()
		var firstFailure, recovered time.Time
		successes := 0
		stopWriter := make(chan struct{})
		writerDone := make(chan struct{})
		go func() {
			defer close(writerDone)
			n := 0
			for {
				select {
				case <-stopWriter:
					return
				default:
				}
				n++
				key := fmt.Sprintf("%s%d", prefix, n)
				value := fmt.Sprint(n)
				err := client.Set(ctx, key, value, 0).Err()
				writeAcknowledged := err == nil
				if err == nil {
					got, readErr := client.Get(ctx, key).Result()
					if readErr != nil || got != value {
						err = fmt.Errorf("read-after-write failed")
					}
				}
				mu.Lock()
				now := time.Now()
				if writeAcknowledged {
					ack[key] = value
				}
				if err == nil {
					lastSuccess = now
					if !firstFailure.IsZero() {
						successes++
						if successes == 10 && recovered.IsZero() {
							recovered = now
						}
					}
				} else {
					if !writeAcknowledged {
						unknown = append(unknown, key)
					}
					successes = 0
					if firstFailure.IsZero() {
						firstFailure = lastSuccess
					}
				}
				mu.Unlock()
				select {
				case <-stopWriter:
					return
				case <-time.After(100 * time.Millisecond):
				}
			}
		}()
		// Native Sentinel/Cluster failover runs without an AutoOps API/worker request.
		failed := []model.Member{*leader}
		if os.Getenv("AUTOOPS_DB_LIVE_EXERCISE") == "host-domain" {
			if c.Topology != "redis-cluster" {
				t.Fatal("host-domain fixture requires Redis Cluster")
			}
			failed = nil
			for _, m := range c.Members {
				if m.Kind == "data" && m.HostID == leader.HostID {
					failed = append(failed, m)
				}
			}
			if len(failed) != 3 {
				t.Fatal("requires exactly three owned cluster data processes per test host")
			}
		}
		_, e = s.ansible(ctx, c, failed, "service.yml", map[string]any{"service_action": "stopped"}, nil)
		if e == nil {
			e = waitUntil(ctx, 90*time.Second, func() bool { mu.Lock(); defer mu.Unlock(); return !recovered.IsZero() })
		}
		close(stopWriter)
		<-writerDone
		cleanup, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		_, restartErr := s.ansible(cleanup, c, failed, "service.yml", map[string]any{"service_action": "started"}, nil)
		cleanupCancel()
		if restartErr != nil {
			t.Fatal("old primary restart failed", restartErr)
		}
		if e != nil {
			t.Fatal("client did not recover", e)
		}
		if e = s.waitHealthy(ctx, c.ID, 90*time.Second, 0); e != nil {
			t.Fatal(e)
		}
		lost := 0
		for key, value := range ack {
			got, e := client.Get(ctx, key).Result()
			if e != nil || got != value {
				lost++
			}
		}
		present := 0
		for _, key := range unknown {
			if client.Exists(ctx, key).Val() == 1 {
				present++
			}
		}
		rto := recovered.Sub(firstFailure).Seconds()
		report := map[string]any{"round": round, "rtoSeconds": rto, "acknowledged": len(ack), "acknowledgedLost": lost, "unknownWrites": len(unknown), "unknownPresent": present, "failedMemberId": leader.ID}
		report["failedProcesses"] = len(failed)
		reports = append(reports, report)
		t.Logf("native failover: %s", JSON(report))
		if rto > 60 {
			t.Errorf("client RTO target exceeded: %.2fs", rto)
		}
	}
	t.Logf("all failover reports: %s", JSON(reports))
	s.event(c.ID, user, 0, "integration-report", "info", string(JSON(reports)))
	run := func(action string, p Parameters) *model.Operation {
		t.Helper()
		fresh, e := s.Refresh(ctx, c.ID)
		if e != nil {
			t.Fatal(e)
		}
		plan, e := s.Plan(ctx, c.ID, user, action, p)
		if e != nil {
			t.Fatal(e)
		}
		if !plan.Executable {
			t.Fatalf("%s preflight blocked: %s", action, plan.Checks)
		}
		op, e := s.Submit(ctx, c.ID, user, plan.ID, fmt.Sprintf("exercise-%d", time.Now().UnixNano()), fresh.Name)
		if e != nil {
			t.Fatal(e)
		}
		worker := "integration-exercise"
		s.DB.Model(op).Updates(map[string]any{"status": "preflight", "worker_id": worker, "heartbeat_at": time.Now()})
		op.WorkerID = worker
		s.run(ctx, *op)
		s.DB.Preload("Steps").First(op, op.ID)
		if op.Status != "succeeded" {
			for _, step := range op.Steps {
				t.Log(step.Log)
			}
			t.Fatalf("%s %s: %s", action, op.Status, op.Message)
		}
		return op
	}
	run("collect-storage", Parameters{})
	backupOp := run("backup", Parameters{Method: "rdb"})
	var backup model.Backup
	s.DB.Where("operation_id = ?", backupOp.ID).First(&backup)
	run("verify-backup", Parameters{BackupID: backup.ID, HostID: 530, Port: 19379})
	t.Logf("backup %d restored and verified on loopback port 19379", backup.ID)
}
func planChecks(p *model.OperationPlan) []Check {
	var checks []Check
	_ = json.Unmarshal(p.Checks, &checks)
	return checks
}

func runLiveOperation(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint, action string, p Parameters) *model.Operation {
	t.Helper()
	if action != "reconcile" {
		if _, e := s.Refresh(ctx, c.ID); e != nil {
			t.Fatal(e)
		}
	}
	plan, e := s.Plan(ctx, c.ID, user, action, p)
	if e != nil {
		t.Fatal(e)
	}
	if !plan.Executable {
		t.Fatalf("%s preflight rejected: %s", action, plan.Checks)
	}
	op, e := s.Submit(ctx, c.ID, user, plan.ID, fmt.Sprintf("live-%s-%d", action, time.Now().UnixNano()), c.Name)
	if e != nil {
		t.Fatal(e)
	}
	op.WorkerID = "integration-operations"
	s.DB.Model(op).Updates(map[string]any{"worker_id": op.WorkerID, "status": "preflight", "heartbeat_at": time.Now()})
	s.run(ctx, *op)
	s.DB.Preload("Steps").First(op, op.ID)
	if op.Status != "succeeded" {
		for _, step := range op.Steps {
			t.Logf("%s: %s", step.Name, step.Log)
		}
		t.Fatalf("%s task %d %s: %s", action, op.ID, op.Status, op.Message)
	}
	return op
}
