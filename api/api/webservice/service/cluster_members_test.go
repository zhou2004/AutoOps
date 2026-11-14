package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	system "dodevops-api/api/system/model"
	"dodevops-api/api/webservice/model"
	"errors"
	"testing"
)

func memberNode(t *testing.T, s *Service, id, cid uint) {
	t.Helper()
	source, _ := s.Get("sources", 1)
	source.ID = id
	source.Name = "source"
	source.LocatorHash = nil
	if err := s.DB.Table(model.Tables["sources"]).Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Table(model.Tables["instances"]).Create(&model.Resource{ID: id, Name: "instance", Engine: "nginx", GroupID: 10, SourceID: id, ClusterID: cid, Revision: 1, Spec: source.Spec}).Error; err != nil {
		t.Fatal(err)
	}
}
func newMemberCluster() model.Resource {
	return model.Resource{Name: "selected Nginx", Engine: "nginx", GroupID: 10, Spec: JSON(model.ClusterSpec{})}
}
func memberBelongs(t *testing.T, s *Service, id, cid uint) {
	t.Helper()
	r, e := s.Get("instances", id)
	if e != nil || r.ClusterID != cid {
		t.Fatalf("instance %d membership = %d, want %d: %v", id, r.ClusterID, cid, e)
	}
}
func TestClusterCreateWithSelectedInstances(t *testing.T) {
	s := fixture(t)
	memberNode(t, s, 2, 0)
	memberNode(t, s, 3, 0)
	out, e := s.SaveCluster(context.Background(), 1, newMemberCluster(), &ClusterMemberChanges{Add: []uint{3, 2}})
	if e != nil || out.ID == 0 || out.Revision != 3 {
		t.Fatal(out, e)
	}
	memberBelongs(t, s, 2, out.ID)
	memberBelongs(t, s, 3, out.ID)
	memberBelongs(t, s, 1, 1)
	var count int64
	s.DB.Model(&model.Event{}).Where("kind = ?", "cluster-member-added").Count(&count)
	if count != 2 {
		t.Fatal("membership audit missing", count)
	}
	if len(s.Remote.(*fakeRuntime).requests) != 0 {
		t.Fatal("membership edit changed remote Nginx")
	}
}
func TestClusterCreateRollsBackMetadataAndEarlierMember(t *testing.T) {
	s := fixture(t)
	memberNode(t, s, 2, 0)
	memberNode(t, s, 3, 1)
	out, e := s.SaveCluster(context.Background(), 1, newMemberCluster(), &ClusterMemberChanges{Add: []uint{2, 3}})
	if !errors.Is(e, ErrConflict) || out.ID != 0 {
		t.Fatal(out, e)
	}
	memberBelongs(t, s, 2, 0)
	memberBelongs(t, s, 3, 1)
	var count int64
	s.DB.Table(model.Tables["clusters"]).Count(&count)
	if count != 1 {
		t.Fatal("orphan cluster", count)
	}
	s.DB.Model(&model.Event{}).Count(&count)
	if count != 0 {
		t.Fatal("rolled-back changes left audit", count)
	}
}
func TestClusterEditUsesExplicitMemberDeltas(t *testing.T) {
	s := fixture(t)
	memberNode(t, s, 2, 0)
	memberNode(t, s, 3, 1)
	old, _ := s.Get("clusters", 1)
	old.Spec = JSON(model.ClusterSpec{})
	old.Name = "updated"
	out, e := s.SaveCluster(context.Background(), 1, old, &ClusterMemberChanges{Add: []uint{2}, Remove: []uint{1}})
	if e != nil || out.Name != "updated" || out.Revision != 4 {
		t.Fatal(out, e)
	}
	memberBelongs(t, s, 1, 0)
	memberBelongs(t, s, 2, 1)
	memberBelongs(t, s, 3, 1)
	// An editor cannot remove a concurrent member by saving an outdated form.
	old.Name = "stale"
	if _, e = s.SaveCluster(context.Background(), 1, old, &ClusterMemberChanges{Remove: []uint{3}}); !errors.Is(e, ErrConflict) {
		t.Fatal("stale edit accepted", e)
	}
	memberBelongs(t, s, 3, 1)
}
func TestClusterEditPreservesVIPMembersAndMetadataOnFailure(t *testing.T) {
	s := fixture(t)
	memberNode(t, s, 2, 0)
	vip := model.ClusterSpec{VIP: &model.VIPSpec{CIDR: "192.0.2.110/24", InstanceIDs: []uint{1}}}
	s.DB.Table(model.Tables["clusters"]).Where("id=1").Update("spec", JSON(vip))
	old, _ := s.Get("clusters", 1)
	old.Name = "must roll back"
	if _, e := s.SaveCluster(context.Background(), 1, old, &ClusterMemberChanges{Remove: []uint{1}, Add: []uint{2}}); e == nil {
		t.Fatal("VIP member removed")
	}
	now, _ := s.Get("clusters", 1)
	if now.Name == old.Name || now.Revision != 1 {
		t.Fatal("metadata partially saved", now)
	}
	memberBelongs(t, s, 1, 1)
	memberBelongs(t, s, 2, 0)
	// Empty deltas are metadata-only and preserve protected members.
	if _, e := s.SaveCluster(context.Background(), 1, old, &ClusterMemberChanges{}); e != nil {
		t.Fatal(e)
	}
	memberBelongs(t, s, 1, 1)
}
func TestClusterMemberValidationAndLocks(t *testing.T) {
	for _, tc := range []string{"group", "engine", "source-lock", "cluster-lock", "duplicate", "both-lists", "missing"} {
		t.Run(tc, func(t *testing.T) {
			s := fixture(t)
			memberNode(t, s, 2, 0)
			old, _ := s.Get("clusters", 1)
			old.Spec = JSON(model.ClusterSpec{})
			old.Name = "must roll back"
			changes := &ClusterMemberChanges{Add: []uint{2}}
			switch tc {
			case "group":
				s.DB.Table(model.Tables["instances"]).Where("id=2").Update("group_id", 99)
			case "engine":
				s.DB.Table(model.Tables["instances"]).Where("id=2").Update("engine", "caddy")
			case "source-lock":
				s.DB.Create(&model.Lock{Key: "sources:2", OperationID: 9})
			case "cluster-lock":
				s.DB.Create(&model.Lock{Key: "cluster:1", OperationID: 9})
			case "duplicate":
				changes.Add = []uint{2, 2}
			case "both-lists":
				changes.Remove = []uint{2}
			case "missing":
				changes.Add = []uint{1000}
			}
			if _, e := s.SaveCluster(context.Background(), 1, old, changes); e == nil {
				t.Fatal("invalid selection accepted")
			}
			memberBelongs(t, s, 2, 0)
			now, _ := s.Get("clusters", 1)
			if now.Name == old.Name || now.Revision != 1 {
				t.Fatal("partial metadata update")
			}
		})
	}
}
func TestClusterMemberPermissionsAreRechecked(t *testing.T) {
	s := fixture(t)
	memberNode(t, s, 2, 0)
	for _, g := range []cmdb.CmdbAssetPermission{
		{Name: "cluster", UserIDs: "[2]", AssetTypes: `["web_cluster"]`, WebClusterIDs: "[1]", PermissionActions: `["web_view","web_manage"]`, IsActive: 1},
		{Name: "source1", UserIDs: "[2]", AssetTypes: `["web_source"]`, WebSourceIDs: "[1]", PermissionActions: `["web_view","web_manage"]`, IsActive: 1},
	} {
		if e := s.DB.Create(&g).Error; e != nil {
			t.Fatal(e)
		}
	}
	old, _ := s.Get("clusters", 1)
	old.Spec = JSON(model.ClusterSpec{})
	old.Name = "must roll back"
	if _, e := s.SaveCluster(context.Background(), 2, old, &ClusterMemberChanges{Remove: []uint{1}, Add: []uint{2}}); !errors.Is(e, ErrForbidden) {
		t.Fatal("source permission bypass", e)
	}
	memberBelongs(t, s, 1, 1)
	memberBelongs(t, s, 2, 0)
	s.DB.Model(&system.RolePermission{}).Where("role_id=2").Update("permissions", `["web:read","web:write"]`)
	if _, e := s.SaveCluster(context.Background(), 2, old, &ClusterMemberChanges{Remove: []uint{1}}); !errors.Is(e, ErrForbidden) {
		t.Fatal("execute permission bypass", e)
	}
	if _, e := s.SaveCluster(context.Background(), 2, old, nil); e != nil {
		t.Fatal("metadata-only edit unexpectedly requires execute", e)
	}
	memberBelongs(t, s, 1, 1)
}
func TestSelectedMembersInvalidateOldOperationPlan(t *testing.T) {
	s := fixture(t)
	memberNode(t, s, 2, 0)
	p, e := s.Plan(context.Background(), 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "reload"})
	if e != nil || !p.Executable {
		t.Fatal(p, e)
	}
	old, _ := s.Get("clusters", 1)
	old.Spec = JSON(model.ClusterSpec{})
	if _, e = s.SaveCluster(context.Background(), 1, old, &ClusterMemberChanges{Add: []uint{2}}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Submit(context.Background(), 1, p.ID, p.IdempotencyKey, true); !errors.Is(e, ErrConflict) {
		t.Fatal("stale target plan admitted", e)
	}
}
