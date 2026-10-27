package dao

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"dodevops-api/api/cmdb/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func permissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "permissions.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { raw.Close() })
	if err = db.AutoMigrate(&model.CmdbAssetPermission{}, &model.CmdbUserGroup{}, &model.CmdbUserGroupMember{}, &model.CmdbCredentialPermission{}, &model.CmdbGroup{}, &model.CmdbHost{}, &model.CmdbIDC{}, &model.CmdbCabinet{}, &model.CmdbNetworkDevice{}, &model.CmdbPhysicalMachine{}); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"CREATE TABLE sys_admin (id INTEGER PRIMARY KEY, status INTEGER)", "INSERT INTO sys_admin VALUES (1,1),(2,1),(3,1),(10,1),(100,1),(101,1)", "CREATE TABLE sys_role (id INTEGER PRIMARY KEY, role_key TEXT, status INTEGER)", "CREATE TABLE sys_admin_role (admin_id INTEGER, role_id INTEGER)", "INSERT INTO sys_role VALUES (1, 'admin', 1), (2, 'admin', 0)", "INSERT INTO sys_admin_role VALUES (100, 1), (101, 2)"} {
		if err = db.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestPermissionSubjectExactIDsAndValidity(t *testing.T) {
	d := NewCmdbAssetPermissionDao(permissionTestDB(t))
	rules := []model.CmdbAssetPermission{
		{Name: "wrong user", UserIDs: "[10,21]", IsActive: 1},
		{Name: "correct user", UserIDs: "[1]", IsActive: 1},
		{Name: "wrong group", GroupIDs: "[20]", IsActive: 1},
		{Name: "correct group", GroupIDs: "[2]", IsActive: 1},
		{Name: "expired", UserIDs: "[1]", IsActive: 1, DateExpired: "2000-01-01"},
		{Name: "future", UserIDs: "[1]", IsActive: 1, DateStart: "2099-01-01"},
		{Name: "malformed date", UserIDs: "[1]", IsActive: 1, DateExpired: "invalid"},
		{Name: "disabled", UserIDs: "[1]", IsActive: 0},
		{Name: "malformed subjects", UserIDs: "[1", IsActive: 1},
	}
	for i := range rules {
		if err := d.Create(&rules[i]); err != nil {
			t.Fatal(err)
		}
	}
	got, err := d.GetUserPermissions(1, []uint{2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected only direct and group grants, got %+v", got)
	}
	for _, p := range got {
		if p.Name != "correct user" && p.Name != "correct group" {
			t.Fatalf("unexpected grant %q", p.Name)
		}
	}
}

func TestPermissionAssetAndActionCannotBeCombinedAcrossRules(t *testing.T) {
	d := NewCmdbAssetPermissionDao(permissionTestDB(t))
	rules := []model.CmdbAssetPermission{
		{AssetTypes: `["host"]`, HostIDs: `[11]`, PermissionActions: `["list"]`},
		{AssetTypes: `["host"]`, HostIDs: `[12]`, PermissionActions: `["connect_ssh","update"]`},
		{AssetTypes: `["network"]`, HostIDs: `[11]`, NetworkIDs: `[31]`, PermissionActions: `["connect_vnc"]`},
	}
	cases := []struct {
		kind, action string
		want         uint
	}{{"host", "list", 11}, {"host", "connect_ssh", 12}, {"host", "update", 12}, {"network", "connect_vnc", 31}, {"host", "connect_vnc", 0}, {"host", "delete", 0}}
	for _, tc := range cases {
		t.Run(tc.kind+tc.action, func(t *testing.T) {
			ids, err := d.AllowedAssetIDs(rules, tc.kind, tc.action)
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == 0 {
				if len(ids) != 0 {
					t.Fatalf("unexpected scope %v", ids)
				}
			} else if len(ids) != 1 || !ids[tc.want] {
				t.Fatalf("unexpected scope %v", ids)
			}
		})
	}
}

func TestLegacyConnectOnlyGrantsSSH(t *testing.T) {
	p := model.CmdbAssetPermission{AssetTypes: `["host"]`, HostIDs: `[7]`, PermissionActions: `["connect"]`}
	for _, action := range PermissionActions {
		want := action == "connect" || action == "connect_ssh"
		if got := RuleAllowsAction(p, "host", action); got != want {
			t.Errorf("%s: got %v, want %v", action, got, want)
		}
	}
	if RuleAllowsAction(p, "network", "connect_ssh") {
		t.Fatal("wrong asset type allowed")
	}
	p.PermissionActions = `["admin"]`
	if RuleAllowsAction(p, "host", "made_up_action") {
		t.Fatal("unknown actions must be denied")
	}
}

func TestPermissionGroupsDisabledRolesAndCredentials(t *testing.T) {
	db := permissionTestDB(t)
	d := NewCmdbAssetPermissionDao(db)
	for _, g := range []model.CmdbUserGroup{{ID: 1, Name: "enabled", Status: 1}, {ID: 2, Name: "disabled", Status: 1}} {
		if err := db.Create(&g).Error; err != nil {
			t.Fatal(err)
		}
	}
	db.Model(&model.CmdbUserGroup{}).Where("id = ?", 2).Update("status", 0)
	for _, m := range []model.CmdbUserGroupMember{{GroupID: 1, UserID: 2}, {GroupID: 2, UserID: 3}} {
		if err := db.Create(&m).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i, g := range []uint{1, 2} {
		p := model.CmdbAssetPermission{Name: fmt.Sprint("group", i), GroupIDs: fmt.Sprintf("[%d]", g), AssetTypes: `["host"]`, HostIDs: `[7]`, PermissionActions: `["connect_rdp"]`, IsActive: 1}
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		user uint
		want bool
	}{{2, true}, {3, false}, {1, false}, {100, true}, {101, false}} {
		got, err := d.CanUserAccess(tc.user, "host", 7, "connect_rdp")
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("user %d: got %v want %v", tc.user, got, tc.want)
		}
	}
	rule := model.CmdbCredentialPermission{Name: "credentials", CredentialID: 9, UserIDs: `[10]`, GroupIDs: `[1,2]`, IsActive: 1}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		user uint
		want bool
	}{{1, false}, {10, true}, {2, true}, {3, false}, {100, true}, {101, false}} {
		got, err := d.CanUseCredential(tc.user, 9)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("credential user %d: got %v want %v", tc.user, got, tc.want)
		}
	}
}

func TestCredentialPermissionSeparatesSources(t *testing.T) {
	db := permissionTestDB(t)
	rules := []model.CmdbCredentialPermission{
		{Name: "ssh", CredentialType: "ssh", CredentialID: 9, UserIDs: `[2]`, IsActive: 1},
		{Name: "database", CredentialType: "database", CredentialID: 9, UserIDs: `[3]`, IsActive: 1},
	}
	for _, rule := range rules {
		if err := db.Create(&rule).Error; err != nil {
			t.Fatal(err)
		}
	}
	dao := NewCmdbAssetPermissionDao(db)
	for _, tc := range []struct {
		user   uint
		source string
		want   bool
	}{{2, "ssh", true}, {2, "database", false}, {3, "ssh", false}, {3, "database", true}} {
		got, err := dao.CanUseCredentialSource(tc.user, 9, tc.source)
		if err != nil || got != tc.want {
			t.Fatalf("user=%d source=%s got=%v want=%v err=%v", tc.user, tc.source, got, tc.want, err)
		}
	}
}

func TestPermissionDescendantGroupsAndIDCScope(t *testing.T) {
	db := permissionTestDB(t)
	d := NewCmdbAssetPermissionDao(db)
	for _, q := range []string{
		"INSERT INTO cmdb_group (id,parent_id,name,create_time) VALUES (1,0,'one',CURRENT_TIMESTAMP),(2,1,'two',CURRENT_TIMESTAMP),(3,2,'three',CURRENT_TIMESTAMP)",
		"INSERT INTO cmdb_host (id,host_name,name,group_id,ssh_ip,create_time) VALUES (10,'h','h',3,'127.0.0.1',CURRENT_TIMESTAMP)",
		"INSERT INTO cmdb_network_device (id,name,idc_id,device_type) VALUES (20,'n',5,1),(21,'outside',6,1)",
	} {
		if err := db.Exec(q).Error; err != nil {
			t.Fatal(err)
		}
	}
	rules := []model.CmdbAssetPermission{{AssetTypes: `["host","network"]`, HostGroupIDs: `[1]`, IDCIDs: `[5]`, PermissionActions: `["list"]`}}
	ids, err := d.AllowedAssetIDs(rules, "host", "list")
	if err != nil || len(ids) != 1 || !ids[10] {
		t.Fatalf("host scope %v, error %v", ids, err)
	}
	ids, err = d.AllowedAssetIDs(rules, "network", "list")
	if err != nil || len(ids) != 1 || !ids[20] {
		t.Fatalf("IDC scope %v, error %v", ids, err)
	}
}

func TestPermissionScopeBeforePaginationAndEmptyScope(t *testing.T) {
	db := permissionTestDB(t)
	d := NewCmdbNetworkDeviceDao(db)
	for i := uint(1); i <= 5; i++ {
		device := model.CmdbNetworkDevice{ID: i, Name: fmt.Sprint("device", i), SN: fmt.Sprint("SN", i)}
		if err := db.Create(&device).Error; err != nil {
			t.Fatal(err)
		}
	}
	list, total, err := d.GetList(model.CmdbNetworkDeviceQuery{Page: 2, Size: 2}, map[uint]bool{1: true, 2: true, 3: true})
	if err != nil || total != 3 || len(list) != 1 || list[0].ID != 1 {
		t.Fatalf("page: total=%d list=%v error=%v", total, list, err)
	}
	list, total, err = d.GetList(model.CmdbNetworkDeviceQuery{Page: 1, Size: 20}, map[uint]bool{})
	if err != nil || total != 0 || len(list) != 0 {
		t.Fatalf("empty scope leaked: total=%d list=%v error=%v", total, list, err)
	}
	all, err := d.GetAll(map[uint]bool{})
	if err != nil || len(all) != 0 {
		t.Fatalf("empty GetAll scope leaked: %v %v", all, err)
	}
}

func TestRuleEffectiveInclusiveDates(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	if !ruleEffective(model.CmdbAssetPermission{IsActive: 1, DateStart: "2026-10-02", DateExpired: "2026-10-02"}, now) {
		t.Fatal("inclusive date boundaries should authorize")
	}
}

func TestAssetPlacementRequiresSameRuleForSourceAndDestination(t *testing.T) {
	db := permissionTestDB(t)
	d := NewCmdbAssetPermissionDao(db)
	for _, query := range []string{
		"INSERT INTO cmdb_group (id,parent_id,name,create_time) VALUES (1,0,'one',CURRENT_TIMESTAMP),(2,0,'two',CURRENT_TIMESTAMP),(3,0,'three',CURRENT_TIMESTAMP)",
		"INSERT INTO cmdb_host (id,host_name,name,group_id,ssh_ip,create_time) VALUES (7,'h','h',1,'127.0.0.1',CURRENT_TIMESTAMP)",
	} {
		if err := db.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	direct := model.CmdbAssetPermission{AssetTypes: `["host"]`, HostIDs: `[7]`, PermissionActions: `["update"]`}
	unrelated := model.CmdbAssetPermission{AssetTypes: `["host"]`, HostGroupIDs: `[2]`, PermissionActions: `["update"]`}
	combined := model.CmdbAssetPermission{AssetTypes: `["host"]`, HostIDs: `[7]`, HostGroupIDs: `[2]`, PermissionActions: `["update","create"]`}
	for _, tc := range []struct {
		name               string
		rules              []model.CmdbAssetPermission
		asset, destination uint
		action             string
		want               bool
	}{
		{"in place", []model.CmdbAssetPermission{direct}, 7, 1, "update", true},
		{"direct cannot move", []model.CmdbAssetPermission{direct}, 7, 2, "update", false},
		{"no stitching", []model.CmdbAssetPermission{direct, unrelated}, 7, 2, "update", false},
		{"same rule move", []model.CmdbAssetPermission{combined}, 7, 2, "update", true},
		{"create in scope", []model.CmdbAssetPermission{combined}, 0, 2, "create", true},
		{"create outside scope", []model.CmdbAssetPermission{combined}, 0, 3, "create", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := d.CanPlaceAsset(tc.rules, "host", tc.asset, tc.destination, tc.action)
			if err != nil || got != tc.want {
				t.Fatalf("got %v error %v want %v", got, err, tc.want)
			}
		})
	}
}

func TestDisabledAndDeletedUsersLoseAllAssetAndCredentialAccess(t *testing.T) {
	for _, administrator := range []bool{false, true} {
		for _, deleted := range []bool{false, true} {
			t.Run(fmt.Sprintf("administrator=%v/deleted=%v", administrator, deleted), func(t *testing.T) {
				db := permissionTestDB(t)
				d := NewCmdbAssetPermissionDao(db)
				userID := uint(1)
				if administrator {
					userID = 100
				}
				p := model.CmdbAssetPermission{Name: "direct access", UserIDs: fmt.Sprintf("[%d]", userID), AssetTypes: `["host"]`, HostIDs: `[7]`, PermissionActions: `["connect_rdp"]`, IsActive: 1}
				if err := db.Create(&p).Error; err != nil {
					t.Fatal(err)
				}
				credential := model.CmdbCredentialPermission{Name: "direct credential", CredentialID: 9, UserIDs: p.UserIDs, IsActive: 1}
				if err := db.Create(&credential).Error; err != nil {
					t.Fatal(err)
				}
				if allowed, err := d.CanUserAccess(userID, "host", 7, "connect_rdp"); err != nil || !allowed {
					t.Fatalf("active user denied: %v %v", allowed, err)
				}
				if allowed, err := d.CanUseCredential(userID, 9); err != nil || !allowed {
					t.Fatalf("active credential denied: %v %v", allowed, err)
				}
				mutation := db.Exec("UPDATE sys_admin SET status = 2 WHERE id = ?", userID)
				if deleted {
					mutation = db.Exec("DELETE FROM sys_admin WHERE id = ?", userID)
				}
				if mutation.Error != nil {
					t.Fatal(mutation.Error)
				}
				for name, check := range map[string]func() (bool, error){
					"active":        func() (bool, error) { return d.UserActive(userID) },
					"administrator": func() (bool, error) { return d.IsAdministrator(userID) },
					"asset":         func() (bool, error) { return d.CanUserAccess(userID, "host", 7, "connect_rdp") },
					"credential":    func() (bool, error) { return d.CanUseCredential(userID, 9) },
				} {
					if allowed, err := check(); err != nil || allowed {
						t.Errorf("%s: got %v %v", name, allowed, err)
					}
				}
				if permissions, err := d.EffectivePermissions(userID); err != nil || len(permissions) != 0 {
					t.Fatalf("effective permissions remain: %v %v", permissions, err)
				}
				if permissions, err := d.GetUserPermissions(userID, nil); err != nil || len(permissions) != 0 {
					t.Fatalf("direct permissions remain: %v %v", permissions, err)
				}
				creds, err := (&CmdbCredentialPermissionDao{db: db}).GetUserCredentials(userID, nil)
				if err != nil || len(creds) != 0 {
					t.Fatalf("credential list remains: %v %v", creds, err)
				}
			})
		}
	}
}

func TestAccountStatusLookupFailureDeniesAuthorization(t *testing.T) {
	db := permissionTestDB(t)
	d := NewCmdbAssetPermissionDao(db)
	if err := db.Exec("DROP TABLE sys_admin").Error; err != nil {
		t.Fatal(err)
	}
	for name, check := range map[string]func() (bool, error){
		"active":     func() (bool, error) { return d.UserActive(100) },
		"admin":      func() (bool, error) { return d.IsAdministrator(100) },
		"asset":      func() (bool, error) { return d.CanUserAccess(100, "host", 7, "connect_rdp") },
		"credential": func() (bool, error) { return d.CanUseCredential(100, 9) },
	} {
		if allowed, err := check(); allowed || err == nil {
			t.Errorf("%s must fail closed: %v %v", name, allowed, err)
		}
	}
}

func TestCredentialPermissionCreateDisabledDoesNotGrantAccess(t *testing.T) {
	db := permissionTestDB(t)
	credentials := NewCmdbCredentialPermissionDao(db)
	grant := model.CmdbCredentialPermission{Name: "disabled at creation", CredentialID: 9, UserIDs: `[1]`, IsActive: 0}
	if err := credentials.Create(&grant); err != nil {
		t.Fatal(err)
	}
	stored, err := credentials.GetByID(grant.ID)
	if err != nil || stored.IsActive != 0 {
		t.Fatalf("disabled grant activated: %+v %v", stored, err)
	}
	allowed, err := NewCmdbAssetPermissionDao(db).CanUseCredential(1, 9)
	if err != nil || allowed {
		t.Fatalf("disabled credential granted access: %v %v", allowed, err)
	}
	grant.IsActive = 1
	if err := credentials.Update(&grant); err != nil {
		t.Fatal(err)
	}
	allowed, err = NewCmdbAssetPermissionDao(db).CanUseCredential(1, 9)
	if err != nil || !allowed {
		t.Fatalf("enabled credential not granted: %v %v", allowed, err)
	}
	grant.IsActive = 0
	if err := credentials.Update(&grant); err != nil {
		t.Fatal(err)
	}
	allowed, err = NewCmdbAssetPermissionDao(db).CanUseCredential(1, 9)
	if err != nil || allowed {
		t.Fatalf("disabled update retained access: %v %v", allowed, err)
	}
}
