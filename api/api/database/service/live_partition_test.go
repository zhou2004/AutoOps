package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/database/model"
	"dodevops-api/pkg/rediscompat"
	"fmt"
	"testing"
	"time"
)

// Test-only network fault: exact owned ports and lab peers, private chain,
// independent timed cleanup. SSH, K8s and unrelated database ports are untouched.
const labPartition = `import json,sys,subprocess,pathlib,re
p=json.load(sys.stdin);chain=p['chain'];assert re.fullmatch(r'AUTODB[0-9]+',chain)
root=pathlib.Path('/run/autoops-db-lab-network');root.mkdir(mode=0o700,exist_ok=True)
script=root/(chain+'.py');unit='autoops-db-lab-net-'+chain.lower()
if p['install']:
 assert p['ports'] in ([12379,12380],[18379,18380,18381,28379,28380,28381],[13306,33306])
 assert all(peer in ('192.168.80.51','192.168.80.52','192.168.80.53') for peer in p['peers'])
 assert not script.exists()
 cleanup="import subprocess\nchain="+repr(chain)+"\nfor parent in ('INPUT','OUTPUT'):\n subprocess.run(['iptables','-w','5','-D',parent,'-j',chain],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)\nsubprocess.run(['iptables','-w','5','-F',chain],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)\nsubprocess.run(['iptables','-w','5','-X',chain],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)\n"
 script.write_text(cleanup);script.chmod(0o700)
 subprocess.run(['systemd-run','--quiet','--unit',unit,'--on-active=120s','/usr/bin/python3',str(script)],check=True)
 try:
  subprocess.run(['iptables','-w','5','-N',chain],check=True)
  ports=','.join(map(str,p['ports']))
  for peer in p['peers']:
   for direction in ('-s','-d'):
    subprocess.run(['iptables','-w','5','-A',chain,direction,peer,'-p','tcp','-m','multiport','--dports',ports,'-j','DROP'],check=True)
  for parent in ('INPUT','OUTPUT'):subprocess.run(['iptables','-w','5','-I',parent,'1','-j',chain],check=True)
 except Exception:
  subprocess.run(['/usr/bin/python3',str(script)],check=True);raise
else:
 assert script.exists() and script.stat().st_uid==0
 subprocess.run(['/usr/bin/python3',str(script)],check=True)
 subprocess.run(['systemctl','stop',unit+'.timer'],check=True)
 assert subprocess.run(['iptables','-w','5','-S',chain],capture_output=True).returncode!=0
 script.unlink()
print(json.dumps({'installed':p['install'],'chain':chain}))`

func exercisePartition(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	if c.Topology != "postgresql-patroni" && c.Topology != "redis-cluster" && c.Topology != "mysql-gtid" {
		t.Fatal("unsupported isolated network fixture")
	}
	if c.Engine == "mysql" {
		runLiveOperation(t, ctx, s, c, user, "acknowledge-recovery", Parameters{})
	}
	if c.Topology == "redis-cluster" {
		runLiveOperation(t, ctx, s, c, user, "balance-primaries", Parameters{})
		var err error
		c, err = s.Refresh(ctx, c.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	root, e := primary(c)
	if e != nil {
		t.Fatal(e)
	}
	peers := []string{}
	ports := []int{12379, 12380}
	for _, addr := range []string{"192.168.80.51", "192.168.80.52", "192.168.80.53"} {
		if addr != root.Address {
			peers = append(peers, addr)
		}
	}
	if len(peers) != 2 || !root.Managed {
		t.Fatal("not an authorized lab member")
	}
	if c.Engine == "redis" {
		ports = []int{18379, 18380, 18381, 28379, 28380, 28381}
	}
	if c.Engine == "mysql" {
		ports = []int{13306, 33306}
		peers = append(peers, root.Address)
	}
	probeKey := ""
	mysqlPromoted := false
	confirmedRow := fmt.Sprintf("partition-%d", time.Now().UnixNano())
	if c.Engine == "redis" {
		client, e := s.redis(c, *root)
		if e != nil {
			t.Fatal(e)
		}
		ranges, e := rediscompat.Slots(ctx, client, root.Address)
		if e != nil {
			client.Close()
			t.Fatal(e)
		}
		for n := 0; n < 1000 && probeKey == ""; n++ {
			key := fmt.Sprintf("{autoops-partition-%d}:proof", n)
			slot, e := client.ClusterKeySlot(ctx, key).Result()
			if e != nil {
				client.Close()
				t.Fatal(e)
			}
			for _, r := range ranges {
				if int(slot) >= r.Start && int(slot) <= r.End && len(r.Nodes) > 0 && r.Nodes[0].ID == root.NativeID {
					probeKey = key
					break
				}
			}
		}
		client.Close()
		if probeKey == "" {
			t.Fatal("could not pin probe to failed shard")
		}
	}
	chain := fmt.Sprintf("AUTODB%d%d", c.ID, time.Now().Unix())
	cleanup := func() {
		sub, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, e := s.remoteJSON(sub, root.HostID, labPartition, map[string]any{"chain": chain, "install": false}); e != nil {
			t.Error("network fixture cleanup", e)
		}
	}
	if _, e = s.remoteJSON(ctx, root.HostID, labPartition, map[string]any{"chain": chain, "install": true, "ports": ports, "peers": peers}); e != nil {
		t.Fatal("partition setup failed; independent timed cleanup is armed", e)
	}
	cleaned := false
	defer func() {
		if !cleaned {
			cleanup()
		}
	}()
	var source cmdb.CmdbSQL
	if s.DB.Where("cluster_id = ? AND connection_mode = ?", c.ID, "cluster").First(&source).Error != nil {
		t.Fatal("logical source absent")
	}
	connector := cmdbservice.NewDatabaseOperationService(s.DB)
	if c.Engine == "postgresql" {
		if e = waitUntil(ctx, 60*time.Second, func() bool {
			db, e := s.sql(ctx, c, *root, "postgres")
			if e != nil {
				return true
			}
			defer db.Close()
			var ro bool
			e = db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&ro)
			return e == nil && ro
		}); e != nil {
			t.Fatal("isolated PostgreSQL primary did not demote", e)
		}
		if e = waitUntil(ctx, 40*time.Second, func() bool {
			db, e := connector.OpenSQL(ctx, source.ID, "postgres")
			if e != nil {
				return false
			}
			defer db.Close()
			var address string
			if e = db.QueryRowContext(ctx, "SELECT inet_server_addr()::text").Scan(&address); e != nil || address == root.Address {
				return false
			}
			_, e = db.ExecContext(ctx, "INSERT INTO public.autoops_protection_probe VALUES($1)", confirmedRow)
			return e == nil
		}); e != nil {
			t.Fatal("majority PostgreSQL entry did not restore writes", e)
		}
	} else if c.Engine == "mysql" {
		if e = waitUntil(ctx, 45*time.Second, func() bool {
			if s.confirmStopped(ctx, *root) != nil {
				raw, err := s.remoteJSON(ctx, root.HostID, `import json,sys,pymysql
p=json.load(sys.stdin);allowed=json.load(open('/etc/autoops-fencing/'+str(p['cluster'])+'.json'))
with pymysql.connect(**allowed[p['service']],connect_timeout=2,read_timeout=2) as conn:
 with conn.cursor() as cur:
  cur.execute('SELECT @@global.read_only,@@global.super_read_only');protected=cur.fetchone()==(1,1)
print(json.dumps({'protected':protected}))`, map[string]any{"cluster": c.ID, "service": root.Service})
				if err != nil || !flag(object(raw)["protected"]) {
					return false
				}
			}
			db, err := connector.OpenSQL(ctx, source.ID, "autoops_database_lab")
			if err != nil {
				return false
			}
			defer db.Close()
			_, err = db.ExecContext(ctx, "INSERT INTO autoops_ha_probe(run_key,payload) VALUES (?, 'network-partition')", confirmedRow)
			return err == nil
		}); e != nil {
			// Native Orchestrator may conservatively refuse promotion when
			// replica evidence does not establish a dead primary. Verify the
			// protected business path instead of forcing a recovery.
			for attempt := 0; attempt < 3; attempt++ {
				sub, cancel := context.WithTimeout(ctx, 3*time.Second)
				db, err := connector.OpenSQL(sub, source.ID, "autoops_database_lab")
				if err == nil {
					_, err = db.ExecContext(sub, "INSERT INTO autoops_ha_probe(run_key,payload) VALUES (?, 'partition-block-probe')", fmt.Sprintf("%s-%d", confirmedRow, attempt))
					db.Close()
				}
				cancel()
				if err == nil {
					t.Fatal("unfenced primary accepted business-path writes during isolated network")
				}
			}
			fresh, err := s.Refresh(ctx, c.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range fresh.Members {
				if m.Kind == "data" && m.ID != root.ID && m.State == "healthy" {
					v := object(m.Metrics)
					if m.Role != "replica" || !flag(v["read_only"]) || !flag(v["super_read_only"]) {
						t.Fatal("unconfirmed alternative write primary", m.ID)
					}
				}
			}
			t.Log("native manager did not force promotion: business path blocked and surviving replicas remained protected")
		} else {
			mysqlPromoted = true
		}
	} else {
		client, e := s.redis(c, *root)
		if e != nil {
			t.Fatal(e)
		}
		defer client.Close()
		if e = waitUntil(ctx, 30*time.Second, func() bool {
			state, e := client.ClusterInfo(ctx).Result()
			return e == nil && infoMap(state)["cluster_state"] == "fail"
		}); e != nil {
			t.Fatal("minority Redis did not stop serving cluster", e)
		}
		logical, e := connector.OpenRedis(source.ID)
		if e != nil {
			t.Fatal(e)
		}
		defer logical.Close()
		if e = waitUntil(ctx, 40*time.Second, func() bool {
			key := probeKey
			return logical.Set(ctx, key, "verified", 0).Err() == nil && logical.Get(ctx, key).Val() == "verified"
		}); e != nil {
			t.Fatal("majority Redis client did not recover", e)
		}
	}
	cleanup()
	cleaned = true
	if c.Engine == "mysql" && mysqlPromoted {
		runLiveOperation(t, ctx, s, c, user, "rejoin", Parameters{MemberID: root.ID})
	}
	if e = s.waitHealthy(ctx, c.ID, 90*time.Second, 0); e != nil {
		t.Fatal("network fixture did not converge after cleanup", e)
	}
	if c.Engine == "postgresql" {
		// A brief healthy reading before the old controller refreshes DCS is
		// insufficient. Observe beyond the 30s leader TTL and verify the ACK
		// on every actual data copy after the formerly isolated host returns.
		deadline := time.Now().Add(40 * time.Second)
		for time.Now().Before(deadline) {
			fresh, err := s.Refresh(ctx, c.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range fresh.Members {
				if m.Kind != "data" {
					continue
				}
				db, err := s.sql(ctx, fresh, m, "postgres")
				if err != nil {
					t.Fatal("rejoined member unavailable", m.ID, err)
				}
				var n int
				err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM public.autoops_protection_probe WHERE id=$1", confirmedRow).Scan(&n)
				db.Close()
				if err != nil || n != 1 {
					t.Fatal("acknowledged partition write missing after rejoin", m.ID, err)
				}
			}
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(2 * time.Second):
			}
		}
		t.Log("acknowledged partition write retained on all three instances for > leader TTL after network healing")
	}
	if c.Engine == "mysql" {
		if e = s.waitEndpoints(ctx, c); e != nil {
			t.Fatal("SQL endpoint did not recover after network healing", e)
		}
		if !mysqlPromoted {
			db, err := connector.OpenSQL(ctx, source.ID, "autoops_database_lab")
			if err != nil {
				t.Fatal(err)
			}
			_, err = db.ExecContext(ctx, "INSERT INTO autoops_ha_probe(run_key,payload) VALUES (?, 'network-healed')", confirmedRow)
			db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
		fresh, err := s.Refresh(ctx, c.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range fresh.Members {
			if m.Kind != "data" {
				continue
			}
			db, err := s.sql(ctx, fresh, m, "autoops_database_lab")
			if err != nil {
				t.Fatal(err)
			}
			var n int
			err = waitUntil(ctx, 30*time.Second, func() bool {
				return db.QueryRowContext(ctx, "SELECT COUNT(*) FROM autoops_ha_probe WHERE run_key=?", confirmedRow).Scan(&n) == nil && n == 1
			})
			db.Close()
			if err != nil || n != 1 {
				t.Fatal("partition ACK missing after old-primary rejoin", m.ID, err)
			}
		}
		runLiveOperation(t, ctx, s, fresh, user, "acknowledge-recovery", Parameters{})
		t.Logf("MySQL network partition: nativePromotion=%t, protected path verified, all-copy acknowledged record verified after healing", mysqlPromoted)
	}
	s.event(c.ID, user, 0, "integration-report", "info", fmt.Sprintf("独立端口网络分区通过：%s，原生恢复或保护性阻断已核验，网络恢复后业务入口可读写，防火墙测试链已移除", c.Topology))
	t.Logf("%s network partition: native recovery or protective blocking verified, healed client writes verified, exact test rules removed", c.Topology)
}
