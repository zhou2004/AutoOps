package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"dodevops-api/pkg/authorization"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	cmdbdao "dodevops-api/api/cmdb/dao"
	"dodevops-api/api/system/model"
	"gorm.io/gorm"
)

const ContextKey = "autoops.apiKey"

const Inherit = "user:inherit"
const Scoped = "scoped"

// Legacy restricted keys retain their original ceiling until an administrator reissues them.
var Scopes = []string{"aiops:read", "aiops:chat", "aiops:operate", Inherit}

func Has(k model.APIKey, scope string) bool {
	for _, s := range strings.Split(k.Scopes, ",") {
		if s == scope || k.GrantMode == "" && s == Inherit {
			return true
		}
	}
	return false
}
func Create(ctx context.Context, db *gorm.DB, user uint, name string, scopes []string, days int) (model.APIKey, string, error) {
	return create(ctx, db, user, name, scopes, days, "")
}

// CreateScoped grants only explicit module actions, independently of the owner's roles.
func CreateScoped(ctx context.Context, db *gorm.DB, user uint, name string, scopes []string, days int) (model.APIKey, string, error) {
	return create(ctx, db, user, name, scopes, days, Scoped)
}

func create(ctx context.Context, db *gorm.DB, user uint, name string, scopes []string, days int, mode string) (model.APIKey, string, error) {
	k := model.APIKey{}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 || days < 1 || days > 365 || len(scopes) < 1 || len(scopes) > 256 || mode == "" && len(scopes) > 3 {
		return k, "", errors.New("名称、权限范围或有效期无效（1–365 天）")
	}
	seen := map[string]bool{}
	for _, scope := range scopes {
		valid := authorization.Known(scope)
		if mode == "" {
			valid = scope == Scopes[0] || scope == Scopes[1] || scope == Scopes[2] || scope == Inherit
		}
		if !valid || seen[scope] {
			return k, "", errors.New("API Key 权限范围无效")
		}
		seen[scope] = true
	}
	scopes = append([]string(nil), scopes...)
	sort.Strings(scopes)
	active, err := cmdbdao.NewCmdbAssetPermissionDao(db.WithContext(ctx)).UserActive(user)
	if err != nil || !active {
		return k, "", errors.New("账号不可用")
	}
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return k, "", err
	}
	token := "aops_" + hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	k = model.APIKey{GrantMode: mode, UserID: user, Name: name, Prefix: token[:13], Hash: hex.EncodeToString(hash[:]), Scopes: strings.Join(scopes, ","), ExpiresAt: time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour)}
	err = db.WithContext(ctx).Create(&k).Error
	if err != nil {
		return model.APIKey{}, "", errors.New("创建 API Key 失败")
	}
	return k, token, nil
}
func Valid(ctx context.Context, db *gorm.DB, id uint) (model.APIKey, error) {
	var k model.APIKey
	err := db.WithContext(ctx).Where("id = ? AND revoked_at IS NULL AND expires_at > ?", id, time.Now().UTC()).First(&k).Error
	if err != nil || !k.ExpiresAt.After(time.Now()) || k.RevokedAt != nil {
		return k, errors.New("API Key 无效或已过期")
	}
	active, err := cmdbdao.NewCmdbAssetPermissionDao(db.WithContext(ctx)).UserActive(k.UserID)
	if err != nil || !active {
		return k, errors.New("账号不可用")
	}
	return k, nil
}
func Authenticate(ctx context.Context, db *gorm.DB, token string) (model.APIKey, error) {
	var k model.APIKey
	if len(token) != 69 || !strings.HasPrefix(token, "aops_") {
		return k, errors.New("API Key 无效")
	}
	h := sha256.Sum256([]byte(token))
	if db.WithContext(ctx).Where("hash = ?", hex.EncodeToString(h[:])).First(&k).Error != nil {
		return k, errors.New("API Key 无效")
	}
	k, err := Valid(ctx, db, k.ID)
	if err == nil {
		db.WithContext(ctx).Model(&k).Where("last_used_at IS NULL OR last_used_at < ?", time.Now().Add(-time.Minute)).Update("last_used_at", time.Now().UTC())
	}
	return k, err
}

// Explicit allowlist for legacy AI-only keys; scoped keys use the platform catalog.
func RouteScope(method, route string) string {
	switch {
	case method == "GET" && (route == "/api/v1/aiops/catalog" || route == "/api/v1/aiops/targets" || route == "/api/v1/aiops/sessions" || route == "/api/v1/aiops/sessions/:id"):
		return "aiops:read"
	case method == "POST" && (route == "/api/v1/aiops/sessions" || route == "/api/v1/aiops/sessions/:id/messages"):
		return "aiops:chat"
	case method == "PUT" && route == "/api/v1/aiops/sessions/:id/grants":
		return "aiops:operate"
	case method == "POST" && route == "/api/v1/aiops/sessions/:id/tools":
		return "aiops:read" // Tool controller separately requires operate for mutations.
	}
	return ""
}
