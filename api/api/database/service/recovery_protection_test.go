package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/database/model"
	"dodevops-api/api/database/policy"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRetentionProtectsWindowDependencyAndLastValidatedCopy(t *testing.T) {
	s := testService(t)
	now := time.Now()
	old := now.AddDate(0, 0, -30)
	base := model.Backup{ClusterID: 1, Method: "physical", Status: "complete", CreatedAt: old}
	s.DB.Create(&base)
	if _, e := s.expirableBackup(1, base.ID); e == nil {
		t.Fatal("last backup allowed to expire")
	}
	newer := model.Backup{ClusterID: 1, Method: "physical", Status: "complete", CreatedAt: now, VerifiedAt: &now}
	s.DB.Create(&newer)
	if _, e := s.expirableBackup(1, newer.ID); e == nil {
		t.Fatal("retention window ignored")
	}
	child := model.Backup{ClusterID: 1, ParentID: base.ID, Method: "incremental", Status: "complete", CreatedAt: old}
	s.DB.Create(&child)
	if _, e := s.expirableBackup(1, base.ID); e == nil {
		t.Fatal("incremental parent allowed to expire")
	}
	s.DB.Model(&child).Update("status", "expired")
	if _, e := s.expirableBackup(1, base.ID); e != nil {
		t.Fatal("independent obsolete backup rejected", e)
	}
	if _, e := s.expirableBackup(2, base.ID); e == nil {
		t.Fatal("cross-cluster backup authorized")
	}
}
func TestStorageThresholdNearestAncestorAndCycle(t *testing.T) {
	s := testService(t)
	s.DB.Create(&model.StorageThreshold{GroupID: 10, WarningPercent: 70, CriticalPercent: 85})
	if got := policy.StorageThresholds(s.DB, 11); got.WarningPercent != 70 || got.GroupID != 10 {
		t.Fatalf("parent override lost: %+v", got)
	}
	s.DB.Create(&model.StorageThreshold{GroupID: 11, WarningPercent: 75, CriticalPercent: 92})
	if got := policy.StorageThresholds(s.DB, 11); got.WarningPercent != 75 {
		t.Fatalf("nearest override lost: %+v", got)
	}
	s.DB.Model(&cmdb.CmdbGroup{}).Where("id = ?", 12).Update("parent_id", 12)
	if got := policy.StorageThresholds(s.DB, 12); got.WarningPercent != 80 || got.CriticalPercent != 90 {
		t.Fatalf("cycle fallback wrong: %+v", got)
	}
}
func TestNativeCooldownIsNotProofOfRecoveryCompletion(t *testing.T) {
	if (nativeRecovery{IsActive: false}).finished() {
		t.Fatal("inactive incomplete recovery treated as complete")
	}
	if !(nativeRecovery{IsActive: true, RecoveryEndTimestamp: "2026-11-01 00:00:00"}).finished() {
		t.Fatal("finished recovery in cooldown treated as running")
	}
}
func TestInstanceIdentityDisambiguatesPatroniOnSharedHost(t *testing.T) {
	c := &model.Cluster{Members: []model.Member{{ID: 1, Kind: "data", HostID: 5, Address: "host", Port: 5432}, {ID: 2, Kind: "data", HostID: 5, Address: "host", Port: 5433}, {ID: 3, Kind: "patroni", HostID: 5, Address: "host", Port: 8008, DataMemberID: 1}, {ID: 4, Kind: "patroni", HostID: 5, Address: "host", Port: 8009, DataMemberID: 2}}}
	control, e := patroniFor(c, c.Members[1])
	if e != nil || control.ID != 4 {
		t.Fatalf("wrong Patroni instance: %+v %v", control, e)
	}
	c.Members[2].DataMemberID = 0
	c.Members[3].DataMemberID = 0
	if _, e = patroniFor(c, c.Members[1]); e == nil {
		t.Fatal("ambiguous host-only match allowed")
	}
}
func TestRemoteMutationRequiresDurableExecutionIdentity(t *testing.T) {
	s := testService(t)
	if _, e := s.remoteMutation(context.Background(), 530, "", nil); e == nil {
		t.Fatal("untracked mutation allowed")
	}
}
func TestParameterComparisonUsesNativeUnits(t *testing.T) {
	if !parameterMatches("slow_query_log", "1", "ON") || !parameterMatches("long_query_time", "1", "1.000000") || parameterMatches("work_mem", "4096", "4") {
		t.Fatal("parameter comparison confused units or booleans")
	}
}

func TestCompatibilityPreservesObservationWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		engine, version string
		allowed         bool
	}{{"mysql", "8.0.24", true}, {"mysql", "8.4.1", false}, {"mysql", "8.0.1-MariaDB", false}, {"postgresql", "15.14 (Debian)", true}, {"postgresql", "16.0", false}, {"redis", "7.0.15", true}, {"redis", "8.0.1", false}, {"redis", "", false}} {
		c := &model.Cluster{Engine: tc.engine, Topology: "standalone", Management: "operate", Members: []model.Member{{Kind: "data", Version: tc.version}}}
		if allowed, _ := capability(c, "backup"); allowed != tc.allowed {
			t.Fatalf("unsafe version gate for %s %s", tc.engine, tc.version)
		}
		if allowed, _ := capability(c, "refresh"); !allowed {
			t.Fatal("unsupported version cannot be observed")
		}
	}
	if validateTLS("redis", "require") == nil || validateTLS("mysql", "prefer") == nil || validateTLS("postgresql", "unknown") == nil {
		t.Fatal("invalid TLS silently accepted")
	}
	if validateTLS("redis", "verify-full") != nil || validateTLS("postgresql", "prefer") != nil {
		t.Fatal("supported TLS rejected")
	}
}
func TestFencingIdentityIncludesInstanceAndRetirement(t *testing.T) {
	c := &model.Cluster{Members: []model.Member{{ID: 1, HostID: 5, Kind: "data", Port: 13306}, {ID: 2, HostID: 5, Kind: "data", Port: 13307}}}
	before := fencingMembersHash(c)
	c.Members[0], c.Members[1] = c.Members[1], c.Members[0]
	if fencingMembersHash(c) != before {
		t.Fatal("ordering changed identity")
	}
	c.Members = c.Members[:1]
	if fencingMembersHash(c) == before {
		t.Fatal("retired colocated instance left in fencing target hash")
	}
}

func TestRedisMasterPlacementPreservesShardAndHostMajority(t *testing.T) {
	c := &model.Cluster{Topology: "redis-cluster"}
	for shard := 1; shard <= 3; shard++ {
		for host := 1; host <= 3; host++ {
			m := model.Member{ID: uint((shard-1)*3 + host), Kind: "data", HostID: uint(host), State: "healthy", Role: "replica", ParentID: fmt.Sprint("shard", shard), NativeID: fmt.Sprintf("replica-%d-%d", shard, host), Metrics: JSON(map[string]any{"master_link_status": "up"})}
			if host == 1 {
				m.Role = "primary"
				m.NativeID = fmt.Sprint("shard", shard)
				m.ParentID = ""
			}
			c.Members = append(c.Members, m)
		}
	}
	if len(placementWarnings(c)) == 0 {
		t.Fatal("colocated master majority not surfaced")
	}
	chosen, e := primaryPlacement(c)
	if e != nil {
		t.Fatal(e)
	}
	hosts := map[uint]bool{}
	for _, m := range chosen {
		if hosts[m.HostID] {
			t.Fatal("primary host collision")
		}
		hosts[m.HostID] = true
	}
	if len(hosts) != 3 {
		t.Fatal("not all shards assigned")
	}
	c.Members[2].State = "unreachable"
	if _, e = primaryPlacement(c); e == nil {
		t.Fatal("unsafe placement during fault allowed")
	}
}

func TestPatroniSwitchRequiresVerifiedControlVersion(t *testing.T) {
	c := &model.Cluster{Engine: "postgresql", Topology: "postgresql-patroni", Management: "operate", Status: "healthy", Members: []model.Member{{Kind: "data", Version: "15.19"}, {Kind: "patroni", Version: "3.0.2"}}}
	if ok, _ := capability(c, "switchover"); ok {
		t.Fatal("unverified controller allowed to switch")
	}
	if ok, reason := capability(c, "update-patroni"); !ok {
		t.Fatal(reason)
	}
	c.Members[1].Version = "4.1.5"
	if ok, reason := capability(c, "switchover"); !ok {
		t.Fatal(reason)
	}
}

func TestEtcdIdentityAcceptsUnsigned64BitLeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"health":"true"}`)
			return
		}
		fmt.Fprint(w, `{"header":{"cluster_id":"12136804078886546352","member_id":"15207183031352232858"},"leader":"15207183031352232858","raftTerm":"101","version":"3.4.23"}`)
	}))
	defer server.Close()
	host, port, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	n, _ := strconv.Atoi(port)
	m := model.Member{Kind: "etcd", Address: host, Port: n}
	if err := testService(t).probeMember(context.Background(), &model.Cluster{}, &m); err != nil {
		t.Fatal(err)
	}
	if m.Role != "leader" || m.NativeID != "15207183031352232858" || m.State != "healthy" {
		t.Fatalf("unsigned identity lost: %+v", m)
	}
}

func TestEtcdHealthAndGatewayDoNotShareClosedConnection(t *testing.T) {
	var healthConnections sync.Map
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			healthConnections.Store(r.RemoteAddr, true)
			fmt.Fprint(w, `{"health":"true"}`)
			return
		}
		if _, reused := healthConnections.Load(r.RemoteAddr); reused {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
			return
		}
		fmt.Fprint(w, `{"header":{"cluster_id":"1","member_id":"2"},"leader":"2","version":"3.4.23"}`)
	}))
	defer server.Close()
	host, port, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	n, _ := strconv.Atoi(port)
	s := testService(t)
	for i := 0; i < 3; i++ {
		member := model.Member{Kind: "etcd", Address: host, Port: n}
		if err := s.probeMember(context.Background(), &model.Cluster{}, &member); err != nil {
			t.Fatal("healthy gateway marked unreachable after health check", err)
		}
	}
}

func TestConfigurationVersionScopesAndDecimals(t *testing.T) {
	revisions := []model.ConfigRevision{{MemberID: 7, Desired: JSON(map[string]string{"wait_timeout": "31"})}, {Desired: JSON(map[string]string{"wait_timeout": "30"})}}
	if desiredForMember(revisions, 7)["wait_timeout"] != "31" || desiredForMember(revisions, 8)["wait_timeout"] != "30" {
		t.Fatal("per-instance configuration leaked")
	}
	if validateChanges("mysql", map[string]string{"long_query_time": "0.25"}) != nil {
		t.Fatal("valid fractional slow query threshold rejected")
	}
	for _, v := range []string{"NaN", "Inf", "0;DROP TABLE x", "1.2"} {
		if validateChanges("mysql", map[string]string{"wait_timeout": v}) == nil {
			t.Fatal("invalid bounded parameter accepted", v)
		}
	}
}
