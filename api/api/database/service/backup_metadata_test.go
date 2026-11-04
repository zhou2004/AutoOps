package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	accounts "dodevops-api/api/configcenter/model"
	"dodevops-api/api/database/model"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBackupNameSurvivesPlanAndQueueAndRejectsInvalidInput(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	now := time.Now()
	s.DB.Model(&model.Cluster{}).Where("id = ?", 1).Updates(map[string]any{"management": "operate", "collected_at": now})
	name := "上线前备份 · 中文 2026"
	plan, e := s.Plan(ctx, 1, 1, "backup", Parameters{Method: "logical", BackupName: name})
	if e != nil || !plan.Executable {
		t.Fatalf("named backup rejected: %v %+v", e, plan)
	}
	op, e := s.SubmitConfirmed(ctx, 1, 1, plan.ID, "named-backup-plan", true)
	if e != nil {
		t.Fatal(e)
	}
	var p Parameters
	if e = json.Unmarshal(op.Parameters, &p); e != nil || p.BackupName != name {
		t.Fatal("backup name lost in persisted queue", e)
	}
	claimed, e := s.claim("backup-name-test")
	if e != nil || claimed == nil {
		t.Fatal(e)
	}
	var c model.Cluster
	s.DB.First(&c, 1)
	got, e := backupName(&c, p, now)
	if e != nil || got != name {
		t.Fatal("worker name changed", got, e)
	}
	automatic, e := backupName(&c, Parameters{Method: "logical"}, now)
	if e != nil || !strings.HasPrefix(automatic, "one-logical-") {
		t.Fatal("legacy unnamed request invalid", automatic, e)
	}
	for _, bad := range []string{strings.Repeat("备", 101), "backup\nname", "backup\x00name", string([]byte{0xff})} {
		plan, e = s.Plan(ctx, 1, 1, "backup", Parameters{Method: "logical", BackupName: bad})
		if e != nil || plan.Executable {
			t.Fatal("invalid name allowed by preflight", e)
		}
	}
}

func TestRepositoryDetailScopesStatisticsAndDoesNotExposeKeys(t *testing.T) {
	s := testService(t)
	if e := s.DB.AutoMigrate(&cmdb.CmdbHost{}, &accounts.KeyManage{}); e != nil {
		t.Fatal(e)
	}
	s.DB.Create(&cmdb.CmdbHost{ID: 20, HostName: "repository-host", SSHIP: "192.0.2.53"})
	s.DB.Create(&accounts.KeyManage{ID: 30, Remark: "S3 backup credential", KeyID: "never-return-access-key", KeySecret: "never-return-secret"})
	repo := model.Repository{GroupID: 10, HostID: 20, KeyID: 30, Name: "s3-details", Kind: "s3", Endpoint: "https://s3.example.test", Bucket: "backups", Region: "us-east-1", Status: "healthy", CheckReport: JSON(map[string]any{"probe": "HeadBucket", "healthy": true})}
	if e := s.DB.Create(&repo).Error; e != nil {
		t.Fatal(e)
	}
	manifest := JSON(map[string]any{"remoteComplete": true, "artifacts": map[string]any{"member_1": map[string]any{"size": 100, "sha256": "a"}}, "repositoryReport": map[string]any{"member_1": map[string]any{"verified": true, "sha256": "a"}}})
	now := time.Now()
	for _, b := range []model.Backup{{ClusterID: 1, RepositoryID: repo.ID, Name: "visible", Status: "complete", VerifiedAt: &now, Manifest: manifest}, {ClusterID: 3, RepositoryID: repo.ID, Name: "other-group", Status: "complete", Manifest: manifest}, {ClusterID: 1, RepositoryID: repo.ID, Status: "expired", Manifest: manifest}} {
		if e := s.DB.Create(&b).Error; e != nil {
			t.Fatal(e)
		}
	}
	s.DB.Create(&cmdb.CmdbAssetPermission{Name: "repository observer", UserIDs: `[3]`, AssetTypes: `["database_cluster"]`, HostGroupIDs: `[10]`, PermissionActions: `["db_cluster_view"]`, IsActive: 1})
	d, e := s.RepositoryDetail(context.Background(), 3, repo.ID)
	if e != nil {
		t.Fatal(e)
	}
	stats := d["statistics"].(map[string]any)
	if stats["backupCount"] != 1 || stats["managedBytes"] != int64(100) || stats["verifiedBackupCount"] != 1 {
		t.Fatal("unauthorized/cleaned backup counted", stats)
	}
	raw, _ := json.Marshal(d)
	if strings.Contains(string(raw), "never-return") {
		t.Fatal("credential leaked")
	}
	if d["credentialName"] != "S3 backup credential" {
		t.Fatal("credential reference missing")
	}
	if _, e = s.RepositoryDetail(context.Background(), 2, repo.ID); e != ErrForbidden {
		t.Fatal("SQL-only user saw repository details", e)
	}
	admin, e := s.RepositoryDetail(context.Background(), 1, repo.ID)
	if e != nil || admin["statistics"].(map[string]any)["backupCount"] != 2 {
		t.Fatal("admin history missing", e)
	}
}
