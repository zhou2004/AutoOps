package service

import (
	"context"
	"dodevops-api/api/database/model"
	"testing"
	"time"
)

func TestDeleteRepositoryRequiresAdminConfirmationAndNoReferences(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	r := model.Repository{Name: "fixture", GroupID: 10, Kind: "local", HostID: 1, Path: "/var/lib/autoops-backup-repository/fixture", Status: "healthy"}
	if e := s.DB.Create(&r).Error; e != nil {
		t.Fatal(e)
	}
	if e := s.DeleteRepository(ctx, 2, r.ID, true); e != ErrForbidden {
		t.Fatal("nonadmin deletion", e)
	}
	if e := s.DeleteRepository(ctx, 1, r.ID, false); e == nil {
		t.Fatal("unconfirmed deletion")
	}
	b := model.Backup{ClusterID: 1, RepositoryID: r.ID, Status: "complete"}
	s.DB.Create(&b)
	if e := s.DeleteRepository(ctx, 1, r.ID, true); e == nil {
		t.Fatal("live backup orphaned")
	}
	s.DB.Model(&b).Update("status", "expired")
	p := model.BackupPolicy{ClusterID: 1, RepositoryID: r.ID, Enabled: false}
	s.DB.Create(&p)
	if e := s.DeleteRepository(ctx, 1, r.ID, true); e == nil {
		t.Fatal("paused policy orphaned")
	}
	s.DB.Delete(&p)
	op := model.Operation{ClusterID: 1, Action: "backup", Status: "queued", Parameters: JSON(Parameters{RepositoryID: r.ID}), IdempotencyKey: "repository-test"}
	s.DB.Create(&op)
	if e := s.DeleteRepository(ctx, 1, r.ID, true); e == nil {
		t.Fatal("queued task orphaned")
	}
	s.DB.Model(&op).Update("status", "unknown")
	if e := s.DeleteRepository(ctx, 1, r.ID, true); e == nil {
		t.Fatal("unknown task orphaned")
	}
	s.DB.Model(&op).Update("status", "failed")
	if e := s.DeleteRepository(ctx, 1, r.ID, true); e != nil {
		t.Fatal("unreferenced registration rejected", e)
	}
	var n int64
	s.DB.Model(&model.Repository{}).Where("id = ?", r.ID).Count(&n)
	if n != 0 {
		t.Fatal("registration retained")
	}
	if e := s.DB.First(&b, b.ID).Error; e != nil {
		t.Fatal("history removed", e)
	}
}

func TestSubmitBackupRechecksDeletedRepository(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	s.DB.Model(&model.Cluster{}).Where("id = ?", 1).Update("management", "operate")
	c, _ := s.Cluster(1)
	p := model.OperationPlan{ClusterID: 1, UserID: 1, Action: "backup", Revision: c.Revision, Parameters: JSON(Parameters{Method: "logical", RepositoryID: 999}), Executable: true, ExpiresAt: time.Now().Add(time.Minute)}
	if e := s.DB.Create(&p).Error; e != nil {
		t.Fatal(e)
	}
	if _, e := s.SubmitConfirmed(ctx, 1, 1, p.ID, "deleted-repository-plan", true); e == nil {
		t.Fatal("stale repository plan submitted")
	}
	var n int64
	s.DB.Model(&model.Operation{}).Count(&n)
	if n != 0 {
		t.Fatal("failed submission persisted operation")
	}
}
