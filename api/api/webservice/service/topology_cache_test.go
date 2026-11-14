package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"testing"
	"time"
)

func TestTopologyReadsCollectorSnapshotWithoutRemoteProbes(t *testing.T) {
	s := fixture(t)
	now := time.Now().UTC()
	spec := &model.VIPSpec{CIDR: "192.0.2.50/24", Interface: "eth0", RouterID: 51, InstanceIDs: []uint{1}}
	state := VIPState{Config: spec, Status: "healthy", OwnerInstanceID: 1, CheckedAt: now, Nodes: map[uint]VIPObservation{1: {Bound: true, Active: true, Healthy: true}}}
	s.DB.Table(model.Tables["clusters"]).Where("id=1").Updates(map[string]any{"spec": JSON(model.ClusterSpec{VIP: spec}), "observed": JSON(map[string]any{"vip": state}), "collected_at": now, "status": "healthy"})
	s.DB.Table(model.Tables["instances"]).Where("id=1").Updates(map[string]any{"observed": JSON(Observation{Identity: "nginx", Digest: "base"}), "collected_at": now, "status": "healthy"})
	out, e := s.Topology(context.Background(), 1, 1)
	if e != nil || out.VIP == nil || out.VIP.Status != "healthy" || out.VIP.OwnerInstanceID != 1 {
		t.Fatal(out, e)
	}
	if len(s.Remote.(*fakeRuntime).requests) != 0 {
		t.Fatal("topology performed SSH")
	}
	if out.CheckedAt == nil || !out.CheckedAt.Equal(now) || len(out.Cluster.Observed) > 0 || len(out.Members[0].Instance.Observed) > 0 {
		t.Fatal("snapshot timestamp or payload incorrect")
	}
	state.CheckedAt = now.Add(-91 * time.Second)
	s.DB.Table(model.Tables["clusters"]).Where("id=1").Update("observed", JSON(map[string]any{"vip": state}))
	out, e = s.Topology(context.Background(), 1, 1)
	if e != nil || out.VIP.Status != "unknown" || out.VIP.OwnerInstanceID != 0 || len(out.VIP.Nodes) != 0 {
		t.Fatal("stale VIP presented as current", out, e)
	}
	state.Config = &model.VIPSpec{CIDR: "192.0.2.99/24"}
	state.CheckedAt = now
	s.DB.Table(model.Tables["clusters"]).Where("id=1").Update("observed", JSON(map[string]any{"vip": state}))
	out, e = s.Topology(context.Background(), 1, 1)
	if e != nil || out.VIP.Status != "unknown" || out.VIP.Config.CIDR != spec.CIDR {
		t.Fatal("previous VIP config reused", out, e)
	}
}
func TestEnvironmentOptionsRespectResourceVisibility(t *testing.T) {
	s := fixture(t)
	s.DB.Table(model.Tables["sources"]).Where("id=1").Update("environment", "prod-private")
	all, e := s.Environments(context.Background(), 1)
	if e != nil || len(all) != 1 || all[0] != "prod-private" {
		t.Fatal(all, e)
	}
	hidden, e := s.Environments(context.Background(), 2)
	if e != nil || len(hidden) != 0 {
		t.Fatal("unauthorized environment leaked", hidden, e)
	}
}
