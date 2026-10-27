package service

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dodevops-api/api/cmdb/model"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/common/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func connectionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "connections.db")+"?_busy_timeout=5000&_journal_mode=WAL"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.CmdbConnectionTicket{}, &model.CmdbAssetConnection{}, &model.CmdbNetworkDevice{}, &configmodel.EcsAuth{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestConnectionTicketExpiryAndSingleUse(t *testing.T) {
	db := connectionTestDB(t)
	token, err := CreateConnectionTicket(db, 7, 31, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(token) != 43 {
		t.Fatal("unexpected ticket entropy encoding")
	}
	var stored model.CmdbConnectionTicket
	if err := db.First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Hash == token || len(stored.Hash) != 64 || stored.Hash != ticketHash(token) {
		t.Fatal("ticket must be stored hashed")
	}
	if remaining := time.Until(stored.ExpiresAt); remaining < 50*time.Second || remaining > time.Minute {
		t.Fatalf("unexpected ticket TTL: %s", remaining)
	}
	used, err := ConsumeConnectionTicket(db, token, func(origin string) bool { return origin == "" })
	if err != nil || used.UserID != 7 || used.ConnectionID != 31 || used.ConsumedAt == nil {
		t.Fatalf("first redemption failed: %+v %v", used, err)
	}
	if _, err := ConsumeConnectionTicket(db, token, func(origin string) bool { return origin == "" }); err == nil {
		t.Fatal("replayed ticket accepted")
	}
	expired, err := CreateConnectionTicket(db, 8, 32, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.CmdbConnectionTicket{}).Where("hash = ?", ticketHash(expired)).Update("expires_at", time.Now().UTC().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumeConnectionTicket(db, expired, func(origin string) bool { return origin == "" }); err == nil {
		t.Fatal("expired ticket accepted")
	}
	for _, invalid := range []string{"", "short", strings.Repeat("x", 43), strings.Repeat("x", 44)} {
		if _, err := ConsumeConnectionTicket(db, invalid, func(origin string) bool { return origin == "" }); err == nil {
			t.Fatal("invalid ticket accepted")
		}
	}
}

func TestConnectionTicketConcurrentRedemption(t *testing.T) {
	db := connectionTestDB(t)
	token, err := CreateConnectionTicket(db, 7, 31, "")
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := ConsumeConnectionTicket(db, token, func(origin string) bool { return origin == "" }); err == nil {
				winners.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatalf("expected one redemption, got %d", winners.Load())
	}
}

func TestAssetConnectionValidationAndScope(t *testing.T) {
	fingerprint := "SHA256:" + base64.RawStdEncoding.EncodeToString(make([]byte, 32))
	base := model.CmdbAssetConnection{Name: " management ", AssetType: "network", Protocol: "ssh", Address: " 192.0.2.10 ", CredentialID: 1, HostKey: fingerprint}
	if err := ValidateAssetConnection(&base); err != nil {
		t.Fatal(err)
	}
	if base.Port != 22 || base.Address != "192.0.2.10" || base.Name != "management" {
		t.Fatal("normalization failed")
	}
	for _, mutate := range []func(*model.CmdbAssetConnection){
		func(p *model.CmdbAssetConnection) { p.Address = "127.0.0.1; id" },
		func(p *model.CmdbAssetConnection) { p.Address = "https://example.org/path" },
		func(p *model.CmdbAssetConnection) { p.Address = "0.0.0.0" },
		func(p *model.CmdbAssetConnection) { p.Address = "224.0.0.1" },
		func(p *model.CmdbAssetConnection) { p.Port = 65536 },
		func(p *model.CmdbAssetConnection) { p.CredentialID = 0 },
		func(p *model.CmdbAssetConnection) { p.HostKey = "" },
		func(p *model.CmdbAssetConnection) { p.HostKey = "SHA256:invalid" },
		func(p *model.CmdbAssetConnection) { p.ReadOnly = true },
		func(p *model.CmdbAssetConnection) { p.Protocol = "rdp" },
	} {
		candidate := base
		mutate(&candidate)
		if err := ValidateAssetConnection(&candidate); err == nil {
			t.Fatalf("invalid connection accepted: %+v", candidate)
		}
	}
	desktop := model.CmdbAssetConnection{Name: "desktop", AssetType: "host", Protocol: "rdp", Address: "2001:db8::1", CredentialID: 1}
	if err := ValidateAssetConnection(&desktop); err != nil || desktop.Port != 3389 || desktop.Security != "nla" {
		t.Fatalf("RDP defaults: %+v %v", desktop, err)
	}
	db := connectionTestDB(t)
	for _, device := range []model.CmdbNetworkDevice{{ID: 1, SN: "ONE", Name: "one", DeviceType: 1}, {ID: 2, SN: "TWO", Name: "two", DeviceType: 1}} {
		if err := db.Create(&device).Error; err != nil {
			t.Fatal(err)
		}
	}
	base.ID, base.AssetID = 31, 1
	if err := db.Create(&base).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := GetAssetConnection(db, "network", 1, 31); err != nil {
		t.Fatal(err)
	}
	if _, err := GetAssetConnection(db, "network", 2, 31); err == nil {
		t.Fatal("cross-asset connection returned")
	}
	if err := AssetExists(db, "network", 999); err == nil {
		t.Fatal("missing asset accepted")
	}
}

// Opt-in smoke test: credentials are supplied only through the process environment.
// The only authenticated remote command is a read-only operating-system query.
func TestAssetConnectionLive(t *testing.T) {
	host, user, password := os.Getenv("CMDB_TEST_HOST"), os.Getenv("CMDB_TEST_USER"), os.Getenv("CMDB_TEST_PASSWORD")
	if host == "" || user == "" || password == "" {
		t.Skip("set CMDB_TEST_HOST, CMDB_TEST_USER and CMDB_TEST_PASSWORD for a read-only live check")
	}
	port := 22
	if raw := os.Getenv("CMDB_TEST_PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 65535 {
			t.Fatal("invalid CMDB_TEST_PORT")
		}
		port = parsed
	}
	fingerprint, err := DiscoverAssetHostKey(host, port)
	if err != nil {
		t.Fatal("SSH host-key discovery failed")
	}
	db := connectionTestDB(t)
	credential := configmodel.EcsAuth{Name: "live-test", Type: 1, Username: user, Password: password, CreateTime: util.HTime{Time: time.Now()}}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal("test credential initialization failed")
	}
	profile := model.CmdbAssetConnection{Name: "live-test", Protocol: "ssh", Address: host, Port: port, Username: user, CredentialID: credential.ID, Enabled: true, HostKey: fingerprint}
	client, err := DialAssetSSH(db, &profile)
	if err != nil {
		t.Fatal("verified SSH login failed")
	}
	defer client.Close()
	content, err := runNetworkConfigCommand(t.Context(), client, "uname -s", "")
	if err != nil {
		t.Fatal("read-only system query failed")
	}
	if strings.TrimSpace(content) == "" {
		t.Fatal("system query returned no data")
	}
	t.Logf("verified SSH login and read-only query: %s", strings.TrimSpace(content))
	if os.Getenv("CMDB_TEST_PLATFORM") == "openwrt" {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			t.Fatal("test encryption initialization failed")
		}
		t.Setenv("CMDB_CONFIG_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))
		if err := db.AutoMigrate(&model.CmdbNetworkConfigSnapshot{}, &model.CmdbNetworkConfigAudit{}); err != nil {
			t.Fatal("test schema initialization failed")
		}
		for _, statement := range []string{
			"CREATE TABLE sys_admin (id integer PRIMARY KEY, status integer)",
			"INSERT INTO sys_admin (id, status) VALUES (7, 1)",
			"CREATE TABLE sys_role (id integer PRIMARY KEY, role_key text, status integer)",
			"CREATE TABLE sys_admin_role (admin_id integer, role_id integer)",
			"INSERT INTO sys_role (id, role_key, status) VALUES (1, 'admin', 1)",
			"INSERT INTO sys_admin_role (admin_id, role_id) VALUES (7, 1)",
		} {
			if err := db.Exec(statement).Error; err != nil {
				t.Fatal("test authorization initialization failed")
			}
		}
		device := model.CmdbNetworkDevice{Name: "live-readonly-test", SN: "live-readonly-test", DeviceType: 1, ManageIP: host}
		if err := db.Create(&device).Error; err != nil {
			t.Fatal("test asset initialization failed")
		}
		profile.AssetType, profile.AssetID = "network", device.ID
		if err := db.Create(&profile).Error; err != nil {
			t.Fatal("test connection initialization failed")
		}
		svc := NewNetworkConfigService(db)
		snapshot, err := svc.Backup(t.Context(), 7, device.ID, NetworkConfigBackupRequest{ConnectionID: profile.ID, Platform: "openwrt", Remark: "Read-only live integration test"})
		if err != nil {
			t.Fatal("read-only UCI backup failed")
		}
		detail, err := svc.Snapshot(device.ID, snapshot.ID)
		if err != nil || len(detail.Content) == 0 || snapshot.ByteSize == 0 || len(snapshot.Checksum) != 64 || snapshot.ContentCipher == "" {
			t.Fatal("encrypted backup validation failed")
		}
		if strings.Contains(detail.Content, password) {
			t.Fatal("redacted snapshot contains the login credential")
		}
		t.Logf("OpenWrt read-only backup verified: %d bytes, integrity hash present, encrypted storage and redacted viewing successful", snapshot.ByteSize)
	}
}
