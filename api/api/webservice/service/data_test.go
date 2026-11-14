package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/webservice/model"
	"dodevops-api/common/util"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

type unavailableRuntime struct{}

func (unavailableRuntime) Run(context.Context, RemoteRequest) (json.RawMessage, error) {
	return nil, errors.New("Nginx binary not found")
}

type editedSourceRuntime struct{ service *Service }

func TestOnlyUndiscoveredSourceMayChangeLocator(t *testing.T) {
	s := fixture(t)
	if err := s.DB.AutoMigrate(&cmdb.CmdbHost{}, &cmdb.CmdbAssetConnection{}); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Omit("Group").Create(&cmdb.CmdbHost{ID: 5, HostName: "node", GroupID: 10, CreateTime: util.HTime{Time: time.Now()}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Create(&cmdb.CmdbAssetConnection{ID: 8, AssetID: 5, AssetType: "host", Protocol: "ssh", Name: "SSH", Enabled: true}).Error; err != nil {
		t.Fatal(err)
	}
	s.DB.Exec("DELETE FROM web_instance")
	r, _ := s.Get("sources", 1)
	r.Spec = JSON(model.SourceSpec{HostID: 5, ConnectionID: 8, Runtime: "host", ConfigPath: "/etc/nginx/nginx.conf"})
	r, err := s.Save(context.Background(), 1, "sources", r, "")
	if err != nil {
		t.Fatal(err)
	}
	oldLocator := *r.LocatorHash
	var spec model.SourceSpec
	json.Unmarshal(r.Spec, &spec)
	spec.ConfigPath = "/etc/autoops-web/main/current/nginx.conf"
	r.Spec = JSON(spec)
	if _, err = s.Save(context.Background(), 1, "sources", r, ""); err != nil {
		t.Fatal("undiscovered source cannot be corrected", err)
	}
	r, _ = s.Get("sources", r.ID)
	if *r.LocatorHash == oldLocator {
		t.Fatal("locator hash was not updated with spec")
	}
	if _, err = s.Discover(context.Background(), 1, r.ID); err != nil {
		t.Fatal(err)
	}
	spec.ConfigPath = "/etc/other/nginx.conf"
	r.Spec = JSON(spec)
	if _, err = s.Save(context.Background(), 1, "sources", r, ""); err == nil {
		t.Fatal("discovered instance identity was changed")
	}
}

func (r editedSourceRuntime) Run(context.Context, RemoteRequest) (json.RawMessage, error) {
	r.service.DB.Table(model.Tables["sources"]).Where("id = 1").Update("revision", 2)
	return JSON(Observation{Identity: "old-connection-result", Digest: "old"}), nil
}

func TestDiscoveryDoesNotRegisterResultsAfterSourceWasEdited(t *testing.T) {
	s := fixture(t)
	s.DB.Exec("DELETE FROM web_instance")
	s.Remote = editedSourceRuntime{s}
	if _, e := s.Discover(context.Background(), 1, 1); !errors.Is(e, ErrConflict) {
		t.Fatal("stale discovery result was accepted", e)
	}
	var count int64
	s.DB.Table(model.Tables["instances"]).Count(&count)
	if count != 0 {
		t.Fatal("stale source created an instance")
	}
}

func TestSourceDiscoveryFailureRemainsVisibleWithoutCreatingInstance(t *testing.T) {
	s := fixture(t)
	s.DB.Exec("DELETE FROM web_instance")
	s.Remote = unavailableRuntime{}
	if _, e := s.Discover(context.Background(), 1, 1); e == nil {
		t.Fatal("missing service was accepted")
	}
	r, _ := s.Get("sources", 1)
	var ob Observation
	json.Unmarshal(r.Observed, &ob)
	if r.Status != "unavailable" || r.CollectedAt == nil || ob.Message != "Nginx binary not found" {
		t.Fatal("failed discovery not available to the page")
	}
	var count int64
	s.DB.Table(model.Tables["instances"]).Count(&count)
	if count != 0 {
		t.Fatal("failed discovery fabricated an instance")
	}
	rows, total, e := s.List(1, "sources", Filter{Status: "unavailable"})
	if e != nil || total != 1 || len(rows) != 1 {
		t.Fatal("failed source filter", e)
	}
	s.Remote = &fakeRuntime{ob: Observation{Identity: "nginx", Digest: "new"}}
	if _, e = s.Discover(context.Background(), 1, 1); e != nil {
		t.Fatal(e)
	}
	r, _ = s.Get("sources", 1)
	json.Unmarshal(r.Observed, &ob)
	if r.Status != "healthy" || ob.Message != "" {
		t.Fatal("successful rediscovery retained old error")
	}
}

func TestSourceConnectionViewUsesCMDBWithoutExposingCredentials(t *testing.T) {
	s := fixture(t)
	if e := s.DB.AutoMigrate(&cmdb.CmdbHost{}, &cmdb.CmdbAssetConnection{}); e != nil {
		t.Fatal(e)
	}
	host := cmdb.CmdbHost{ID: 5, HostName: "web-node", Name: "web-node", GroupID: 10, CreateTime: util.HTime{Time: time.Now()}}
	if e := s.DB.Omit("Group").Create(&host).Error; e != nil {
		t.Fatal(e)
	}
	connection := cmdb.CmdbAssetConnection{ID: 8, AssetID: 5, AssetType: "host", Protocol: "ssh", Name: "Ops SSH", Address: "192.0.2.5", Port: 22, Username: "operator", CredentialID: 999, HostKey: "private-host-key-reference"}
	s.DB.Create(&connection)
	r, _ := s.Get("sources", 1)
	r.Spec = JSON(model.SourceSpec{HostID: 5, ConnectionID: 8})
	s.Redact(1, "sources", &r)
	if r.Connection == nil || r.Connection.Address != "192.0.2.5" || r.Connection.Alias != "Ops SSH" {
		t.Fatal("CMDB display metadata missing")
	}
	public := string(JSON(r))
	if strings.Contains(public, "credentialId") || strings.Contains(public, "private-host-key-reference") {
		t.Fatal("connection view leaked credential metadata")
	}
}

func TestAuditFiltersPermissionBeforePaginationAndKeepsUnfinishedTasks(t *testing.T) {
	s := fixture(t)
	s.DB.Create(&cmdb.CmdbGroup{ID: 20, ParentID: 10, Name: "child", CreateTime: util.HTime{Time: time.Now()}})
	s.DB.Table(model.Tables["clusters"]).Where("id = 1").Updates(map[string]any{"name": "public", "group_id": 20, "environment": "production"})
	s.DB.Table(model.Tables["clusters"]).Create(&model.Resource{ID: 2, Name: "hidden", GroupID: 20, Environment: "production"})
	s.DB.Create(&cmdb.CmdbAssetPermission{Name: "audit", UserIDs: "[2]", AssetTypes: `["web_cluster"]`, WebClusterIDs: "[1]", PermissionActions: `["web_audit"]`, IsActive: 1})
	for _, op := range []model.Operation{
		{ID: 1, PlanID: 1, IdempotencyKey: "audit1", TargetType: "clusters", TargetID: 1, ClusterID: 1, Action: "publish", Status: "failed"},
		{ID: 2, PlanID: 2, IdempotencyKey: "audit2", TargetType: "clusters", TargetID: 2, ClusterID: 2, Action: "publish", Status: "failed"},
		{ID: 3, PlanID: 3, IdempotencyKey: "audit3", TargetType: "clusters", TargetID: 1, ClusterID: 1, Action: "reload", Status: "unknown", CreatedAt: time.Now().Add(-100 * 24 * time.Hour)},
	} {
		if e := s.DB.Create(&op).Error; e != nil {
			t.Fatal(e)
		}
	}
	f := Filter{Name: "发布", GroupID: 10, Environment: "production", Status: "failed", Engine: "nginx", Size: 1}
	rows, total, e := s.Audit(2, f)
	if e != nil || total != 1 || len(rows) != 1 || rows[0].ID != 1 || rows[0].TargetName != "public" {
		t.Fatalf("audit filters: %v %d %v", rows, total, e)
	}
	f.Page = 2
	rows, total, e = s.Audit(2, f)
	if e != nil || total != 1 || len(rows) != 0 {
		t.Fatal("audit pagination leaked inaccessible count")
	}
	rows, total, e = s.Audit(2, Filter{Status: "unknown", ClusterID: 1})
	if e != nil || total != 1 || rows[0].ID != 3 {
		t.Fatal("unfinished historical task was hidden")
	}
}
