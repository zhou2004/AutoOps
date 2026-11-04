package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"dodevops-api/api/database/model"
)

func TestEndpointConvergenceWaitsForStableRouting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	unavailable := errors.New("write entry unavailable")
	wrongRole := errors.New("VIP returned replica")
	results := []error{unavailable, nil, wrongRole, nil, nil}
	calls := 0
	err := waitEndpointConvergence(ctx, time.Millisecond, func(probe context.Context) error {
		if probe != ctx {
			t.Fatal("probe must inherit cancellation/deadline")
		}
		if calls >= len(results) {
			t.Fatal("continued probing after convergence")
		}
		result := results[calls]
		calls++
		return result
	})
	if err != nil || calls != len(results) {
		t.Fatalf("ended before consecutive successful role checks: calls=%d error=%v", calls, err)
	}
}

func TestEndpointConvergenceDeadlinePreservesFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	wrongRole := errors.New("VIP returned replica")
	err := waitEndpointConvergence(ctx, time.Millisecond, func(probe context.Context) error {
		<-probe.Done()
		return wrongRole
	})
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, wrongRole) {
		t.Fatalf("timeout lost the failed entry or was accepted as success: %v", err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	err = waitEndpointConvergence(ctx, time.Millisecond, func(context.Context) error {
		t.Fatal("started a probe after cancellation")
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation ignored: %v", err)
	}
}

func TestUnknownOperationAllowsCollectionWithoutUnlocking(t *testing.T) {
	s := testService(t)
	if !s.collectionAllowed(1) {
		t.Fatal("unlocked cluster should be collected")
	}
	op := model.Operation{ClusterID: 1, Status: "unknown", IdempotencyKey: "unknown-operation"}
	if err := s.DB.Create(&op).Error; err != nil {
		t.Fatal(err)
	}
	lock := model.OperationLock{ClusterID: 1, OperationID: op.ID}
	if err := s.DB.Create(&lock).Error; err != nil {
		t.Fatal(err)
	}
	if !s.collectionAllowed(1) {
		t.Fatal("unresolved operation must not make live health stale")
	}
	var retained model.OperationLock
	if err := s.DB.First(&retained, "cluster_id = ?", 1).Error; err != nil || retained.OperationID != op.ID {
		t.Fatalf("read-only collection released the mutation lock: %v", err)
	}
	if err := s.DB.First(&op, op.ID).Error; err != nil || op.Status != "unknown" {
		t.Fatalf("collector changed the unresolved operation: %v", err)
	}
	for _, status := range []string{"preflight", "running", "verifying", "succeeded", "failed"} {
		if err := s.DB.Model(&op).Update("status", status).Error; err != nil {
			t.Fatal(err)
		}
		if s.collectionAllowed(1) {
			t.Fatalf("collector bypassed a %s operation lock", status)
		}
	}
	if err := s.DB.Delete(&op).Error; err != nil {
		t.Fatal(err)
	}
	if s.collectionAllowed(1) {
		t.Fatal("collector bypassed an orphaned operation lock")
	}
}

func TestQueuedAndIsolatedRecoveryKeepHealthVisible(t *testing.T) {
	s := testService(t)
	op := model.Operation{ClusterID: 1, Action: "restore", Status: "queued", IdempotencyKey: "recovery-health"}
	if e := s.DB.Create(&op).Error; e != nil {
		t.Fatal(e)
	}
	if e := s.DB.Create(&model.OperationLock{ClusterID: 1, OperationID: op.ID}).Error; e != nil {
		t.Fatal(e)
	}
	if !s.collectionAllowed(1) {
		t.Fatal("queued recovery blocks health")
	}
	claimed, e := s.claim("test-worker")
	if e != nil || claimed == nil || claimed.ID != op.ID {
		t.Fatal("failed to claim", e)
	}
	if s.collectionAllowed(1) {
		t.Fatal("preflight must drain and exclude collectors")
	}
	for _, action := range []string{"restore", "verify-backup"} {
		for _, status := range []string{"running", "verifying"} {
			s.DB.Model(&op).Updates(map[string]any{"action": action, "status": status})
			if !s.collectionAllowed(1) {
				t.Fatalf("isolated %s/%s hides source health", action, status)
			}
		}
	}
	s.DB.Model(&op).Updates(map[string]any{"action": "switchover", "status": "running"})
	if s.collectionAllowed(1) {
		t.Fatal("topology mutation must retain exclusion")
	}
	s.DB.Model(&op).Update("action", nil)
	if s.collectionAllowed(1) {
		t.Fatal("missing action must fail closed")
	}
	var count int64
	s.DB.Model(&model.OperationLock{}).Where("cluster_id = ?", 1).Count(&count)
	if count != 1 {
		t.Fatal("health read released operation lock")
	}
	if claimed, e = s.claim("another-worker"); e != nil || claimed != nil {
		t.Fatal("job was replayed", e)
	}
}
