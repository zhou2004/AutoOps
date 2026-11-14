package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/webservice/model"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func targetFixture(t *testing.T) *Service {
	t.Helper()
	s := fixture(t)
	source, err := s.Get("sources", 1)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := decode[model.SourceSpec](source.Spec)
	if err != nil {
		t.Fatal(err)
	}
	for id := uint(1); id <= 3; id++ {
		spec.HostID = id
		source := model.Resource{ID: id, Name: fmt.Sprintf("source-%d", id), GroupID: 10, Revision: 1, Spec: JSON(spec)}
		instance := model.Resource{ID: id, Name: fmt.Sprintf("node-%d", id), GroupID: 10, SourceID: id, ClusterID: 1, Revision: 1, Spec: source.Spec, Observed: JSON(s.Remote.(*fakeRuntime).ob)}
		if err := s.DB.Table(model.Tables["sources"]).Save(&source).Error; err != nil {
			t.Fatal(err)
		}
		if err := s.DB.Table(model.Tables["instances"]).Save(&instance).Error; err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func targetDraft(t *testing.T, s *Service, ids []uint) model.Resource {
	t.Helper()
	draft := model.Resource{Name: "native-draft", GroupID: 10, ClusterID: 1, Revision: 1, Status: "draft", Spec: JSON(model.ConfigSpec{InstanceIDs: ids, Files: []model.File{{Path: "/etc/nginx/conf.d/native.conf", Content: "server { listen 8080; }"}}})}
	if err := s.DB.Table(model.Tables["configurations"]).Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	return draft
}

func targetSnapshot(t *testing.T, p model.Plan) []Target {
	t.Helper()
	snapshot, err := decode[map[string]string](p.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := unseal(snapshot["sealed"])
	if err != nil {
		t.Fatal(err)
	}
	targets, err := decode[[]Target]([]byte(plaintext))
	if err != nil {
		t.Fatal(err)
	}
	return targets
}

func assertOnlyHost(t *testing.T, s *Service, hostID uint) {
	t.Helper()
	for _, request := range s.Remote.(*fakeRuntime).requests {
		if request.Spec.HostID != hostID {
			t.Fatalf("%s unexpectedly reached host %d, wanted only %d", request.Action, request.Spec.HostID, hostID)
		}
	}
}

func TestNativeDraftPublishFreezesSelectedInstance(t *testing.T) {
	s := targetFixture(t)
	draft := targetDraft(t, s, []uint{2})
	ctx := context.Background()
	p, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "publish", Parameters: Parameters{ConfigurationID: draft.ID}})
	if err != nil || !p.Executable {
		t.Fatalf("plan: executable=%v error=%v checks=%s", p.Executable, err, p.Checks)
	}
	targets := targetSnapshot(t, p)
	if len(targets) != 1 || targets[0].Instance.ID != 2 {
		t.Fatalf("unexpected target snapshot: %+v", targets)
	}
	assertOnlyHost(t, s, 2)
	// Editing a draft after confirmation must not expand the frozen operation.
	spec, _ := decode[model.ConfigSpec](draft.Spec)
	spec.InstanceIDs = []uint{1, 3}
	if err := s.DB.Table(model.Tables["configurations"]).Where("id = ?", draft.ID).Updates(map[string]any{"spec": JSON(spec), "revision": 2}).Error; err != nil {
		t.Fatal(err)
	}
	op, err := s.Submit(ctx, 1, p.ID, p.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	s.execute(ctx, op)
	if err := s.DB.First(&op, op.ID).Error; err != nil || op.Status != "succeeded" {
		t.Fatalf("operation: %s %s %v", op.Status, op.Message, err)
	}
	assertOnlyHost(t, s, 2)
	current, err := s.Get("configurations", draft.ID)
	if err != nil || current.Status != "draft" {
		t.Fatalf("newer draft was incorrectly marked published: %s %v", current.Status, err)
	}
	var revisions []model.Revision
	if err := s.DB.Where("cluster_id = ? AND kind = ?", op.ClusterID, "configuration").Find(&revisions).Error; err != nil || len(revisions) != 2 {
		t.Fatalf("expected before/after revisions: %d %v", len(revisions), err)
	}
	for _, revision := range revisions {
		spec, err := decode[model.ConfigSpec](revision.Content)
		if err != nil || !reflect.DeepEqual(spec.InstanceIDs, []uint{2}) || len(spec.Nodes) != 1 || len(spec.Nodes[2]) == 0 {
			t.Fatalf("revision expanded beyond selected instance: %+v %v", spec, err)
		}
	}
}

func TestNativeDraftTargetSelectionRejectsInvalidMembers(t *testing.T) {
	for _, tc := range []struct {
		name string
		ids  []uint
	}{
		{"zero", []uint{0}},
		{"duplicate", []uint{2, 2}},
		{"outside-cluster", []uint{2, 99}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := targetFixture(t)
			draft := targetDraft(t, s, tc.ids)
			if _, err := s.Plan(context.Background(), 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "publish", Parameters: Parameters{ConfigurationID: draft.ID}}); err == nil {
				t.Fatal("invalid target selection accepted")
			}
			if len(s.Remote.(*fakeRuntime).requests) != 0 {
				t.Fatal("invalid targets reached remote discovery")
			}
		})
	}
}

func TestNativeDraftEmptyTargetsRetainWholeClusterCompatibility(t *testing.T) {
	s := targetFixture(t)
	draft := targetDraft(t, s, nil)
	for _, action := range []string{"publish", "validate"} {
		p, err := s.Plan(context.Background(), 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: action, Parameters: Parameters{ConfigurationID: draft.ID}})
		if err != nil || !p.Executable {
			t.Fatalf("%s: %v checks=%s", action, err, p.Checks)
		}
		if len(targetSnapshot(t, p)) != 3 {
			t.Fatalf("%s did not retain all members", action)
		}
	}
}

func TestNativeDraftDifferentInstancesStillShareClusterLock(t *testing.T) {
	s := targetFixture(t)
	ctx := context.Background()
	plans := make([]model.Plan, 0, 2)
	for _, id := range []uint{1, 2} {
		draft := targetDraft(t, s, []uint{id})
		p, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "publish", Parameters: Parameters{ConfigurationID: draft.ID}})
		if err != nil || !p.Executable {
			t.Fatalf("plan: %v checks=%s", err, p.Checks)
		}
		plans = append(plans, p)
	}
	if _, err := s.Submit(ctx, 1, plans[0].ID, plans[0].IdempotencyKey, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Submit(ctx, 1, plans[1].ID, plans[1].IdempotencyKey, true); !errors.Is(err, ErrConflict) {
		t.Fatalf("different subsets bypassed cluster mutex: %v", err)
	}
}

func TestNativeDraftRollbackUsesHistoricalInstanceScope(t *testing.T) {
	s := targetFixture(t)
	// Legacy history has only Nodes; newly joined cluster members are excluded.
	revision := model.Revision{ClusterID: 1, Kind: "configuration", Content: JSON(model.ConfigSpec{Nodes: map[uint][]model.File{2: {{Path: "/etc/nginx/nginx.conf", Content: "events {} http {}"}}}})}
	if err := s.DB.Create(&revision).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	p, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "rollback", Parameters: Parameters{RevisionID: revision.ID}})
	if err != nil || !p.Executable {
		t.Fatalf("rollback plan: %v checks=%s", err, p.Checks)
	}
	targets := targetSnapshot(t, p)
	if len(targets) != 1 || targets[0].Instance.ID != 2 {
		t.Fatalf("rollback included unrelated instances: %+v", targets)
	}
	op, err := s.Submit(ctx, 1, p.ID, p.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	s.execute(ctx, op)
	if err := s.DB.First(&op, op.ID).Error; err != nil || op.Status != "succeeded" {
		t.Fatalf("rollback operation: %s %s %v", op.Status, op.Message, err)
	}
	assertOnlyHost(t, s, 2)
}

func TestNativeDraftUnknownRecoveryKeepsOriginalInstanceScope(t *testing.T) {
	s := targetFixture(t)
	draft := targetDraft(t, s, []uint{2})
	ctx := context.Background()
	p, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "publish", Parameters: Parameters{ConfigurationID: draft.ID}})
	if err != nil || !p.Executable {
		t.Fatalf("plan: %v checks=%s", err, p.Checks)
	}
	op, err := s.Submit(ctx, 1, p.ID, p.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	s.Remote.(*fakeRuntime).fail = true
	s.execute(ctx, op)
	if err := s.DB.First(&op, op.ID).Error; err != nil || op.Status != "unknown" {
		t.Fatalf("lost response must remain unknown: %s %v", op.Status, err)
	}
	recovery, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "reconcile", Parameters: Parameters{OperationID: op.ID}})
	if err != nil || !recovery.Executable {
		t.Fatalf("reconcile plan: %v checks=%s", err, recovery.Checks)
	}
	targets := targetSnapshot(t, recovery)
	if len(targets) != 1 || targets[0].Instance.ID != 2 {
		t.Fatalf("reconcile expanded original scope: %+v", targets)
	}
	assertOnlyHost(t, s, 2)
	var lock model.Lock
	if err := s.DB.Where("`key` = ?", "cluster:1").First(&lock).Error; err != nil || lock.OperationID != op.ID {
		t.Fatalf("unknown operation lost its cluster lock: %+v %v", lock, err)
	}
	reconcile, err := s.Submit(ctx, 1, recovery.ID, recovery.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	s.execute(ctx, reconcile)
	if err := s.DB.First(&op, op.ID).Error; err != nil || op.Status != "succeeded" {
		t.Fatalf("reconciliation did not finalize original operation: %s %s %v", op.Status, op.Message, err)
	}
	current, err := s.Get("configurations", draft.ID)
	if err != nil || current.Status != "published" {
		t.Fatalf("confirmed publication did not update matching draft: %s %v", current.Status, err)
	}
	assertOnlyHost(t, s, 2)
	applyCount := 0
	for _, request := range s.Remote.(*fakeRuntime).requests {
		if request.Action == "apply" {
			applyCount++
		}
	}
	if applyCount != 1 {
		t.Fatalf("reconciliation replayed apply: %d calls", applyCount)
	}
}

func TestNativeDraftSelectedInstancePermissionIsRecheckedBeforeApply(t *testing.T) {
	s := targetFixture(t)
	clusterGrant := cmdb.CmdbAssetPermission{Name: "cluster publication", UserIDs: "[2]", AssetTypes: `["web_cluster"]`, WebClusterIDs: "[1]", PermissionActions: `["web_view","web_publish"]`, IsActive: 1}
	instanceGrant := cmdb.CmdbAssetPermission{Name: "selected instance publication", UserIDs: "[2]", AssetTypes: `["web_source"]`, WebSourceIDs: "[2]", PermissionActions: `["web_publish"]`, IsActive: 1}
	for _, grant := range []*cmdb.CmdbAssetPermission{&clusterGrant, &instanceGrant} {
		if err := s.DB.Create(grant).Error; err != nil {
			t.Fatal(err)
		}
	}
	draft := targetDraft(t, s, []uint{2})
	ctx := context.Background()
	p, err := s.Plan(ctx, 2, PlanInput{TargetType: "clusters", TargetID: 1, Action: "publish", Parameters: Parameters{ConfigurationID: draft.ID}})
	if err != nil || !p.Executable {
		t.Fatalf("authorized subset must not require excluded instance grants: %v checks=%s", err, p.Checks)
	}
	op, err := s.Submit(ctx, 2, p.ID, p.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Delete(&instanceGrant).Error; err != nil {
		t.Fatal(err)
	}
	s.execute(ctx, op)
	if err := s.DB.First(&op, op.ID).Error; err != nil || op.Status != "failed" {
		t.Fatalf("revoked instance publication did not fail: %s %v", op.Status, err)
	}
	assertOnlyHost(t, s, 2)
	for _, request := range s.Remote.(*fakeRuntime).requests {
		if request.Action == "apply" {
			t.Fatal("queued operation changed the instance after permission revocation")
		}
	}
}

func TestNativeDraftPublicationStatusRequiresFrozenRevisionAndContent(t *testing.T) {
	for _, mutation := range []string{"none", "revision", "content", "legacy-plan"} {
		t.Run(mutation, func(t *testing.T) {
			s := targetFixture(t)
			draft := targetDraft(t, s, []uint{2})
			ctx := context.Background()
			p, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "publish", Parameters: Parameters{ConfigurationID: draft.ID}})
			if err != nil || !p.Executable {
				t.Fatalf("plan: %v checks=%s", err, p.Checks)
			}
			switch mutation {
			case "revision":
				err = s.DB.Table(model.Tables["configurations"]).Where("id = ?", draft.ID).Update("revision", 2).Error
			case "content":
				spec, _ := decode[model.ConfigSpec](draft.Spec)
				spec.Files[0].Content = "server { listen 9090; }"
				err = s.DB.Table(model.Tables["configurations"]).Where("id = ?", draft.ID).Update("spec", JSON(spec)).Error
			case "legacy-plan":
				snapshot, _ := decode[map[string]string](p.Snapshot)
				delete(snapshot, "configurationRevision")
				delete(snapshot, "configurationDigest")
				err = s.DB.Model(&p).Update("snapshot", JSON(snapshot)).Error
			}
			if err != nil {
				t.Fatal(err)
			}
			op, err := s.Submit(ctx, 1, p.ID, p.IdempotencyKey, true)
			if err != nil {
				t.Fatal(err)
			}
			s.execute(ctx, op)
			if err := s.DB.First(&op, op.ID).Error; err != nil || op.Status != "succeeded" {
				t.Fatalf("publication failed: %s %s %v", op.Status, op.Message, err)
			}
			current, err := s.Get("configurations", draft.ID)
			want := "draft"
			if mutation == "none" {
				want = "published"
			}
			if err != nil || current.Status != want {
				t.Fatalf("draft status=%s want=%s error=%v", current.Status, want, err)
			}
		})
	}
}

type targetMutationRuntime struct {
	delegate   RuntimeConnector
	onValidate func()
}

func (r *targetMutationRuntime) Run(ctx context.Context, request RemoteRequest) (json.RawMessage, error) {
	if request.Action == "validate" && r.onValidate != nil {
		mutate := r.onValidate
		r.onValidate = nil
		mutate()
	}
	return r.delegate.Run(ctx, request)
}

func TestNativeDraftConcurrentEditDuringPreflightRequiresNewPlan(t *testing.T) {
	s := targetFixture(t)
	draft := targetDraft(t, s, []uint{2})
	s.Remote = &targetMutationRuntime{delegate: s.Remote, onValidate: func() {
		if err := s.DB.Table(model.Tables["configurations"]).Where("id = ?", draft.ID).Update("revision", 2).Error; err != nil {
			t.Fatal(err)
		}
	}}
	if _, err := s.Plan(context.Background(), 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "publish", Parameters: Parameters{ConfigurationID: draft.ID}}); !errors.Is(err, ErrConflict) {
		t.Fatalf("concurrently edited draft produced a plan: %v", err)
	}
	var count int64
	if err := s.DB.Model(&model.Plan{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("mixed revision plan persisted: %d %v", count, err)
	}
}
