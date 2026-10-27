package apikey

import (
	"context"
	"dodevops-api/api/system/model"
	"encoding/json"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestKeyLifecycleAndScopes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "keys.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	if err = db.AutoMigrate(&model.APIKey{}); err != nil {
		t.Fatal(err)
	}
	db.Exec("CREATE TABLE sys_admin(id INTEGER PRIMARY KEY,status INTEGER)")
	db.Exec("INSERT INTO sys_admin VALUES(1,1),(2,0)")
	ctx := context.Background()
	key, token, err := Create(ctx, db, 1, "Harness", []string{"aiops:read", "aiops:chat"}, 30)
	if err != nil || key.ID == 0 {
		t.Fatalf("create: %v", err)
	}
	serialized, _ := json.Marshal(key)
	if strings.Contains(string(serialized), token) || strings.Contains(string(serialized), key.Hash) || key.Hash == token {
		t.Fatal("credential exposed")
	}
	if auth, err := Authenticate(ctx, db, token); err != nil || auth.UserID != 1 || Has(auth, "aiops:operate") {
		t.Fatalf("auth: %+v %v", auth, err)
	}
	if _, err = Authenticate(ctx, db, token+"x"); err == nil {
		t.Fatal("invalid length accepted")
	}
	if _, _, err = Create(ctx, db, 2, "disabled", []string{"aiops:read"}, 30); err == nil {
		t.Fatal("inactive owner accepted")
	}
	if _, _, err = Create(ctx, db, 1, "scope", []string{"*"}, 30); err == nil {
		t.Fatal("wildcard accepted")
	}
	db.Model(&key).Update("expires_at", time.Now().Add(-time.Second))
	if _, err = Authenticate(ctx, db, token); err == nil {
		t.Fatal("expired key accepted")
	}
	db.Model(&key).Updates(map[string]any{"expires_at": time.Now().Add(time.Hour), "revoked_at": time.Now()})
	if _, err = Authenticate(ctx, db, token); err == nil {
		t.Fatal("revoked key accepted")
	}
	db.Model(&key).Update("revoked_at", nil)
	db.Exec("UPDATE sys_admin SET status=0 WHERE id=1")
	if _, err = Authenticate(ctx, db, token); err == nil {
		t.Fatal("disabled owner accepted")
	}
	for _, route := range []string{"/api/v1/admin/list", "/api/v1/system/api-keys", "/api/v1/aiops/configs", "/api/v1/cmdb/docker/:assetType/:assetId/containers/:container/action"} {
		for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
			if RouteScope(method, route) != "" {
				t.Fatalf("broad route permitted: %s %s", method, route)
			}
		}
	}
	if RouteScope("POST", "/api/v1/aiops/sessions/:id/messages") != "aiops:chat" || RouteScope("PUT", "/api/v1/aiops/sessions/:id/grants") != "aiops:operate" {
		t.Fatal("scopes not mapped")
	}
}
