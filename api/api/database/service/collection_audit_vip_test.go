package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/database/model"
	system "dodevops-api/api/system/model"
	"dodevops-api/pkg/apikey"
	"strings"
	"testing"
	"time"
)

func TestAuditRetentionPreservesReceiptsAndUnresolvedTasks(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	old := time.Now().AddDate(0, 0, -100)
	if _, e := s.SaveAuditPolicy(ctx, 2, 30); e != ErrForbidden {
		t.Fatal("nonadmin changed retention", e)
	}
	if _, e := s.SaveAuditPolicy(ctx, 1, 30); e != nil {
		t.Fatal(e)
	}
	for i, status := range []string{"succeeded", "failed", "unknown", "running"} {
		op := model.Operation{ClusterID: 1, Status: status, IdempotencyKey: status, RequestHash: "preserve-receipt", Parameters: JSON(map[string]any{"details": "log"}), FinishedAt: &old}
		if e := s.DB.Create(&op).Error; e != nil {
			t.Fatal(e)
		}
		s.DB.Create(&model.OperationStep{OperationID: op.ID, Name: "log", Log: "sensitive operational log"})
		if i == 1 {
			s.DB.Create(&model.OperationLock{ClusterID: 1, OperationID: op.ID})
		}
	}
	s.PruneAudit(ctx)
	var rows []model.Operation
	s.DB.Order("id").Find(&rows)
	if rows[0].AuditDeletedAt == nil || rows[0].RequestHash != "preserve-receipt" || rows[0].IdempotencyKey != "succeeded" {
		t.Fatal("receipt removed or log not expired")
	}
	for _, op := range rows[1:] {
		if op.AuditDeletedAt != nil {
			t.Fatal("active/locked task deleted")
		}
	}
	var steps int64
	s.DB.Model(&model.OperationStep{}).Where("operation_id = ?", rows[0].ID).Count(&steps)
	if steps != 0 {
		t.Fatal("logs retained after expiry")
	}
	if e := s.DeleteAudit(ctx, 2, "operations", rows[2].ID, true); e != ErrForbidden {
		t.Fatal("unauthorized deletion", e)
	}
	if e := s.DeleteAudit(ctx, 1, "operations", rows[2].ID, true); e == nil {
		t.Fatal("unresolved task deleted")
	}
	if e := s.DeleteAudit(ctx, 1, "operations", rows[0].ID, false); e == nil {
		t.Fatal("missing confirmation accepted")
	}
}
func TestCollectionLeaseExclusionAndExpiry(t *testing.T) {
	s := testService(t)
	if !s.acquireCollection(1, "api") || s.acquireCollection(1, "worker") {
		t.Fatal("lease not exclusive")
	}
	s.DB.Model(&model.CollectionLease{}).Where("cluster_id = ?", 1).Update("until", time.Now().Add(-time.Second))
	if !s.acquireCollection(1, "worker") {
		t.Fatal("expired readonly lease did not recover")
	}
}
func TestReplicationEvidenceDoesNotInventSynchronization(t *testing.T) {
	now := time.Now()
	c := model.Cluster{Engine: "mysql", Members: []model.Member{
		{ID: 1, Kind: "data", Role: "primary", NativeID: "uuid", State: "healthy", CollectedAt: &now},
		{ID: 2, Kind: "data", Role: "replica", ParentID: "uuid", State: "healthy", CollectedAt: &now, Metrics: JSON(map[string]any{"replicationIO": "Yes", "replicationSQL": "Yes", "replicationError": 0, "read_only": 1, "super_read_only": 1, "sampleCaughtUp": true})}}}
	if l := ReplicationLinks(&c); len(l) != 1 || l[0].State != "healthy" {
		t.Fatalf("valid replication: %+v", l)
	}
	metrics := object(c.Members[1].Metrics)
	delete(metrics, "sampleCaughtUp")
	c.Members[1].Metrics = JSON(metrics)
	if ReplicationLinks(&c)[0].State != "unknown" {
		t.Fatal("unknown replay was reported synchronized")
	}
	metrics["sampleCaughtUp"] = true
	metrics["replicationIO"] = "No"
	c.Members[1].Metrics = JSON(metrics)
	if ReplicationLinks(&c)[0].State != "unavailable" {
		t.Fatal("stopped thread hidden")
	}
	past := now.Add(-2 * time.Minute)
	c.Members[1].CollectedAt = &past
	if ReplicationLinks(&c)[0].State != "unknown" {
		t.Fatal("stale green replication")
	}
}
func TestVIPInputsAndExclusiveAddress(t *testing.T) {
	s := testService(t)
	s.DB.AutoMigrate(&cmdb.CmdbHost{})
	c := model.Cluster{ID: 1, Engine: "mysql", Members: []model.Member{{ID: 8, Kind: "proxy", Managed: true, HostID: 1, Address: "192.0.2.51", Port: 23306, Service: "test-proxy-a"}, {ID: 9, Kind: "proxy", Managed: true, HostID: 2, Address: "192.0.2.52", Port: 23306, Service: "test-proxy-b"}}}
	p := VIPInput{CIDR: "192.0.2.60/24", Interface: "ens33", RouterID: 84, MemberIDs: []uint{8, 9}}
	if _, e := s.vipMembers(&c, p); e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{"192.0.2.51/24", "192.0.2.255/24", "192.0.2.0/24", "203.0.113.60/24", "not-ip"} {
		q := p
		q.CIDR = bad
		if _, e := s.vipMembers(&c, q); e == nil {
			t.Fatal("invalid VIP accepted", bad)
		}
	}
	q := p
	q.Interface = "eth0;shutdown"
	if _, e := s.vipMembers(&c, q); e == nil {
		t.Fatal("interface injection accepted")
	}
	s.DB.Create(&model.VirtualIP{ClusterID: 2, CIDR: p.CIDR, RouterID: 85})
	if _, e := s.vipMembers(&c, p); e == nil || !strings.Contains(e.Error(), "登记") {
		t.Fatal("duplicate VIP accepted", e)
	}
}

func TestAuditCannotInheritAdminFromRestrictedKey(t *testing.T) {
	s := testService(t)
	s.DB.AutoMigrate(&system.APIKey{})
	key, _, err := apikey.CreateScoped(context.Background(), s.DB, 1, "audit-read", []string{"database:read"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := apikey.WithKey(context.Background(), key.ID)
	if _, err = s.SaveAuditPolicy(ctx, 1, 1); err != ErrForbidden {
		t.Fatal("restricted key changed retention", err)
	}
	if err = s.DeleteAudit(ctx, 1, "events", 1, true); err != ErrForbidden {
		t.Fatal("restricted key deleted audit", err)
	}
}

func TestFutureObservationIsNotFresh(t *testing.T) {
	future := time.Now().Add(8 * time.Hour)
	c := model.Cluster{Status: "healthy", CollectedAt: &future}
	MarkStale(&c)
	if c.Status != "stale" || collectedFresh(&future, 90*time.Second) {
		t.Fatal("future timestamp hid a stale collector")
	}
}

func TestVIPAddressReservationCannotRaceAcrossPrefixes(t *testing.T) {
	s := testService(t)
	address := "192.0.2.60"
	a := model.VirtualIP{ClusterID: 1, CIDR: address + "/24", Address: &address, RouterID: 84}
	b := model.VirtualIP{ClusterID: 2, CIDR: address + "/25", Address: &address, RouterID: 85}
	if e := s.DB.Create(&a).Error; e != nil {
		t.Fatal(e)
	}
	if e := s.DB.Create(&b).Error; e == nil {
		t.Fatal("same VIP reserved twice across clusters")
	}
}
