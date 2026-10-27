package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"dodevops-api/api/cmdb/model"
	"golang.org/x/crypto/ssh"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func networkConfigFixture(t *testing.T) (*NetworkConfigService, *gorm.DB) {
	t.Helper()
	t.Setenv("CMDB_CONFIG_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.CmdbNetworkDevice{}, &model.CmdbNetworkConfigSnapshot{}, &model.CmdbNetworkConfigAudit{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.CmdbNetworkDevice{ID: 11, SN: "NET-11", Name: "router-one", DeviceType: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.CmdbNetworkDevice{ID: 12, SN: "NET-12", Name: "router-two", DeviceType: 1}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewNetworkConfigService(db)
	svc.dial = func(uint, uint) (*ssh.Client, error) { return nil, nil }
	svc.authorizeConnection = func(_, _, connectionID uint, _ bool) (uint, error) { return connectionID, nil }
	return svc, db
}

func TestNetworkConfigBackupEncryptedAndScoped(t *testing.T) {
	svc, db := networkConfigFixture(t)
	content := "package wireless\nconfig wifi-iface 'default'\n\toption ssid 'office'\n\toption key 'never-show-this'\n"
	svc.run = func(_ context.Context, _ *ssh.Client, command, input string) (string, error) {
		if input != "" || !strings.Contains(command, "uci export") {
			t.Fatal("unexpected backup command")
		}
		return content, nil
	}
	snapshot, err := svc.Backup(context.Background(), 7, 11, NetworkConfigBackupRequest{Platform: "openwrt"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(snapshot.ContentCipher, "never-show-this") {
		t.Fatal("stored plaintext")
	}
	encoded, _ := json.Marshal(snapshot)
	if strings.Contains(string(encoded), "content") || strings.Contains(string(encoded), "never-show-this") {
		t.Fatal("snapshot JSON leaks configuration")
	}
	detail, err := svc.Snapshot(11, snapshot.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(detail.Content, "never-show-this") || !strings.Contains(detail.Content, "office") {
		t.Fatalf("unexpected redacted config: %s", detail.Content)
	}
	if _, err := svc.Snapshot(12, snapshot.ID); err == nil {
		t.Fatal("cross-device snapshot access allowed")
	}
	list, _, err := svc.Snapshots(11, 1, 20)
	if err != nil || len(list) != 1 || list[0].ContentCipher != "" {
		t.Fatal("list loads secret content")
	}
	var audit model.CmdbNetworkConfigAudit
	if err := db.First(&audit).Error; err != nil || audit.Status != "success" || audit.SnapshotID != snapshot.ID {
		t.Fatalf("missing backup audit: %+v %v", audit, err)
	}
	copy := *snapshot
	copy.NetworkID = 12
	if _, err := decryptNetworkConfig(&copy); err == nil {
		t.Fatal("ciphertext accepted for different asset")
	}
	copy = *snapshot
	copy.ContentCipher = copy.ContentCipher[:12] + "AAAA" + copy.ContentCipher[16:]
	if _, err := decryptNetworkConfig(&copy); err == nil {
		t.Fatal("tampered ciphertext accepted")
	}
}

func TestNetworkConfigRestoreCreatesRollbackBeforeRemoteWrite(t *testing.T) {
	svc, db := networkConfigFixture(t)
	saved := "package network\nconfig interface 'lan'\n\toption ipaddr '10.2.0.1'\n"
	current := "package network\nconfig interface 'lan'\n\toption ipaddr '10.3.0.1'\n"
	snapshot, err := svc.saveSnapshot(7, 11, 3, "openwrt", saved, "manual", "")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	svc.run = func(_ context.Context, _ *ssh.Client, command, input string) (string, error) {
		calls++
		if calls == 1 {
			return current, nil
		}
		if command != networkConfigRestoreCommand || input != saved {
			t.Fatal("restore does not use exact encrypted version as stdin")
		}
		var count int64
		db.Model(&model.CmdbNetworkConfigSnapshot{}).Where("source = ?", "pre_restore").Count(&count)
		if count != 1 {
			t.Fatal("remote write occurred before rollback snapshot")
		}
		return "", nil
	}
	output, err := svc.Restore(context.Background(), 7, 11, NetworkConfigRestoreRequest{SnapshotID: snapshot.ID, ConfirmName: "router-one"})
	if err != nil || calls != 2 {
		t.Fatalf("restore: %v, calls=%d", err, calls)
	}
	rollback, err := svc.dao.Snapshot(11, output.RollbackSnapshotID)
	if err != nil || rollback.Source != "pre_restore" {
		t.Fatal("missing rollback snapshot")
	}
	plain, err := decryptNetworkConfig(rollback)
	if err != nil || plain != current {
		t.Fatal("rollback content mismatch")
	}
	for _, forbidden := range []string{"reboot", "network restart", "network reload"} {
		if strings.Contains(networkConfigRestoreCommand, forbidden) {
			t.Fatal("restore automatically disrupts service")
		}
	}
}

func TestNetworkConfigRestoreRejectsWrongConfirmationAndOwnership(t *testing.T) {
	svc, db := networkConfigFixture(t)
	snapshot, err := svc.saveSnapshot(7, 11, 0, "openwrt", "package network\n", "manual", "")
	if err != nil {
		t.Fatal(err)
	}
	svc.dial = func(uint, uint) (*ssh.Client, error) { t.Fatal("invalid request dialed device"); return nil, nil }
	for _, tc := range []struct {
		id   uint
		name string
	}{{11, "wrong"}, {12, "router-two"}} {
		if _, err := svc.Restore(context.Background(), 7, tc.id, NetworkConfigRestoreRequest{SnapshotID: snapshot.ID, ConfirmName: tc.name}); err == nil {
			t.Fatal("invalid restore allowed")
		}
	}
	var audits []model.CmdbNetworkConfigAudit
	db.Find(&audits)
	if len(audits) != 2 || audits[0].Status != "failed" || audits[1].Status != "failed" {
		t.Fatalf("failed attempts not audited: %+v", audits)
	}
}

func TestNetworkConfigBackupFailureDoesNotSavePartialOutput(t *testing.T) {
	svc, db := networkConfigFixture(t)
	svc.run = func(context.Context, *ssh.Client, string, string) (string, error) {
		return "partial-secret", errors.New("command timed out")
	}
	if _, err := svc.Backup(context.Background(), 7, 11, NetworkConfigBackupRequest{Platform: "openwrt"}); err == nil {
		t.Fatal("failed command saved")
	}
	var count int64
	db.Model(&model.CmdbNetworkConfigSnapshot{}).Count(&count)
	if count != 0 {
		t.Fatal("partial version created")
	}
	var audit model.CmdbNetworkConfigAudit
	db.First(&audit)
	if audit.Status != "failed" {
		t.Fatal("missing failed audit")
	}
}

func TestNetworkConfigDiffAndKeyRequirement(t *testing.T) {
	svc, _ := networkConfigFixture(t)
	a, _ := svc.saveSnapshot(7, 11, 0, "openwrt", "package wireless\n\toption key 'old-secret'\n\toption ssid 'old'\n", "manual", "")
	b, _ := svc.saveSnapshot(7, 11, 0, "openwrt", "package wireless\n\toption key 'new-secret'\n\toption ssid 'new'\n", "manual", "")
	diff, err := svc.Diff(11, a.ID, b.ID)
	if err != nil || !diff.Changed || !strings.Contains(diff.Diff, "+\toption ssid 'new'") || strings.Contains(diff.Diff, "secret'") {
		t.Fatalf("unsafe or missing diff: %+v %v", diff, err)
	}
	t.Setenv("CMDB_CONFIG_ENCRYPTION_KEY", "")
	if _, ready := svc.Capabilities(); ready {
		t.Fatal("claims encryption is ready without key")
	}
	if _, err := svc.Snapshot(11, a.ID); err == nil {
		t.Fatal("decrypt without key succeeded")
	}
}

func TestNetworkConfigValidationAndOutputBound(t *testing.T) {
	for _, content := range []string{"", "Error: Unrecognized command", "package network\n--More--", "login failed"} {
		if err := validateNetworkConfig("openwrt", content); err == nil {
			t.Errorf("accepted invalid configuration %q", content)
		}
	}
	if _, err := networkConfigBackupCommand("openwrt; reboot"); err == nil {
		t.Fatal("accepted command injection")
	}
	buffer := &configBoundedBuffer{max: 4}
	if _, err := fmt.Fprint(buffer, "12345"); err == nil || buffer.buffer.Len() != 0 {
		t.Fatal("output limit not enforced")
	}
	unlocked, err := lockNetworkConfig(9876)
	if err != nil {
		t.Fatal(err)
	}
	defer unlocked()
	if _, err := lockNetworkConfig(9876); err == nil {
		t.Fatal("concurrent operation permitted")
	}
}

func TestNetworkConfigRedactsMultilineSecrets(t *testing.T) {
	content := "package test\n\toption password 'first-line\nsecret-continuation\nlast-line'\n\toption ssid 'office'\n\toption apikey 'opaque-value'\n-----BEGIN RSA PRIVATE KEY-----\nencoded-private-data\n-----END RSA PRIVATE KEY-----\n"
	redacted := redactNetworkConfig(content)
	for _, secret := range []string{"first-line", "secret-continuation", "last-line", "opaque-value", "encoded-private-data"} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("leaked %q", secret)
		}
	}
	if !strings.Contains(redacted, "office") {
		t.Fatal("redaction hid subsequent public config")
	}
}

func TestNetworkConfigConnectionCredentialAuthorization(t *testing.T) {
	_, db := networkConfigFixture(t)
	for _, statement := range []string{
		"CREATE TABLE sys_admin (id integer PRIMARY KEY, status integer)",
		"INSERT INTO sys_admin (id, status) VALUES (7, 1), (8, 1)",
		"CREATE TABLE sys_role (id integer PRIMARY KEY, role_key text, status integer)",
		"CREATE TABLE sys_admin_role (admin_id integer, role_id integer)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.AutoMigrate(&model.CmdbAssetConnection{}, &model.CmdbAssetPermission{}, &model.CmdbUserGroup{}, &model.CmdbUserGroupMember{}); err != nil {
		t.Fatal(err)
	}
	profiles := []model.CmdbAssetConnection{
		{ID: 31, AssetType: "network", AssetID: 11, Protocol: "ssh", Enabled: true, CredentialID: 51},
		{ID: 32, AssetType: "network", AssetID: 12, Protocol: "ssh", Enabled: true, CredentialID: 51},
		{ID: 33, AssetType: "network", AssetID: 11, Protocol: "ssh", Enabled: true, CredentialID: 52, ReadOnly: true},
	}
	for _, profile := range profiles {
		if err := db.Create(&profile).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := NewNetworkConfigService(db)
	if _, err := svc.authorizeConnection(7, 11, 31, false); err == nil {
		t.Fatal("ungranted asset permission usable")
	}
	if err := db.Create(&model.CmdbAssetPermission{Name: "operators", UserIDs: "[7]", GroupIDs: "[]", AssetTypes: `["network"]`, NetworkIDs: "[11]", PermissionActions: `["config_backup","config_restore"]`, IsActive: 1}).Error; err != nil {
		t.Fatal(err)
	}
	id, err := svc.authorizeConnection(7, 11, 0, false)
	if err != nil || id != 31 {
		t.Fatalf("default authorized profile selection failed: %d %v", id, err)
	}
	if _, err := svc.authorizeConnection(7, 11, 32, false); err == nil {
		t.Fatal("another asset's profile usable")
	}
	if _, err := svc.authorizeConnection(8, 11, 31, false); err == nil {
		t.Fatal("another user's asset permission grant usable")
	}
	if _, err := svc.authorizeConnection(7, 11, 33, true); err == nil {
		t.Fatal("read-only profile permits restore")
	}
	if err := db.Model(&model.CmdbAssetPermission{}).Where("name = ?", "operators").Update("is_active", 0).Error; err != nil {
		t.Fatal(err)
	}
	svc.dial = func(uint, uint) (*ssh.Client, error) {
		t.Fatal("revoked asset permission dialed device")
		return nil, nil
	}
	if _, err := svc.Backup(context.Background(), 7, 11, NetworkConfigBackupRequest{Platform: "openwrt", ConnectionID: 31}); err == nil {
		t.Fatal("revoked asset permission still permits backup")
	}
}
