package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/database/model"
	system "dodevops-api/api/system/model"
	"dodevops-api/pkg/apikey"
	"testing"
)

func TestDatabaseLogMemberScopeAndIdentity(t *testing.T) {
	s := testService(t)
	s.DB.Model(&model.Member{}).Where("cluster_id=1").Updates(map[string]any{"host_id": 42, "service": "autoops-mysql-3306", "data_dir": "/srv/mysql"})
	var m model.Member
	s.DB.First(&m, "cluster_id=1")
	if _, e := s.LogMember(context.Background(), 1, 1, m.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := s.LogMember(context.Background(), 1, 2, m.ID); e == nil {
		t.Fatal("cross-cluster member accepted")
	}
	if _, e := s.LogMember(context.Background(), 1, 1, 0); e == nil {
		t.Fatal("zero member accepted")
	}
	if _, e := s.LogMember(context.Background(), 2, 1, m.ID); e == nil {
		t.Fatal("SQL user read host logs")
	}
	s.DB.Create(&cmdb.CmdbAssetPermission{Name: "log reader", UserIDs: "[3]", AssetTypes: "[\"database_cluster\"]", HostGroupIDs: "[10]", PermissionActions: "[\"db_cluster_view\"]", IsActive: 1})
	if e := s.Authorize(3, 1, "db_cluster_audit"); e == nil {
		t.Fatal("view-only user can read logs")
	}
	s.DB.Model(&model.Member{}).Where("id=?", m.ID).Update("service", "mysql;id")
	if _, e := s.LogMember(context.Background(), 1, 1, m.ID); e == nil {
		t.Fatal("unsafe unit accepted")
	}
	s.DB.Model(&model.Member{}).Where("id=?", m.ID).Updates(map[string]any{"service": "mysql", "host_id": 0})
	if _, e := s.LogMember(context.Background(), 1, 1, m.ID); e == nil {
		t.Fatal("unmapped host accepted")
	}
	s.DB.Exec("UPDATE sys_admin SET status=0 WHERE id=1")
	if _, e := s.LogMember(context.Background(), 1, 1, m.ID); e == nil {
		t.Fatal("disabled account accepted")
	}
}
func TestDatabaseLogStreamLimitReleasesSlots(t *testing.T) {
	releases := []func(){}
	for i := 0; i < 4; i++ {
		r, e := acquireDatabaseLogStream(987)
		if e != nil {
			t.Fatal(e)
		}
		releases = append(releases, r)
	}
	if r, e := acquireDatabaseLogStream(987); e == nil {
		r()
		t.Error("per-user limit not enforced")
	}
	for _, r := range releases {
		r()
	}
	r, e := acquireDatabaseLogStream(987)
	if e != nil {
		t.Fatal(e)
	}
	r()
}

func TestDatabaseLogReaderGrantAndKeyRevocation(t *testing.T) {
	s := testService(t)
	if e := s.DB.AutoMigrate(&system.RolePermission{}, &system.APIKey{}); e != nil {
		t.Fatal(e)
	}
	s.DB.Exec("INSERT INTO sys_role VALUES (2,'reader',1)")
	s.DB.Exec("INSERT INTO sys_admin_role VALUES (3,2)")
	s.DB.Create(&system.RolePermission{RoleID: 2, Permissions: "[\"database:read\"]"})
	grant := cmdb.CmdbAssetPermission{Name: "logs", UserIDs: "[3]", AssetTypes: "[\"database_cluster\"]", HostGroupIDs: "[10]", PermissionActions: "[\"db_cluster_view\",\"db_cluster_audit\"]", IsActive: 1}
	s.DB.Create(&grant)
	s.DB.Model(&model.Member{}).Where("cluster_id=1").Updates(map[string]any{"host_id": 42, "service": "mysql"})
	var m model.Member
	s.DB.First(&m, "cluster_id=1")
	if _, e := s.LogMember(context.Background(), 3, 1, m.ID); e != nil {
		t.Fatal("scoped reader rejected", e)
	}
	s.DB.Model(&grant).Update("permission_actions", "[\"db_cluster_view\"]")
	if _, e := s.LogMember(context.Background(), 3, 1, m.ID); e != ErrForbidden {
		t.Fatal("revoked audit grant remained active", e)
	}
	key, _, e := apikey.CreateScoped(context.Background(), s.DB, 1, "not-database", []string{"cmdb:read"}, 1)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.LogMember(apikey.WithKey(context.Background(), key.ID), 1, 1, m.ID); e != ErrForbidden {
		t.Fatal("API key scope bypass", e)
	}
}
