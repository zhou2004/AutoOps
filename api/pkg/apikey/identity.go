package apikey

import (
	"context"
	"dodevops-api/api/system/model"
	"dodevops-api/pkg/authorization"
	"errors"
	"gorm.io/gorm"
	"strings"
)

type keyContext struct{}

// WithKey propagates the credential to nested services and deferred Harness callbacks.
func WithKey(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, keyContext{}, id)
}

// ID persists the originating credential for queued operations; workers must
// revalidate this key instead of silently inheriting the owner's full session.
func ID(ctx context.Context) uint { id, _ := ctx.Value(keyContext{}).(uint); return id }

// Restrict preserves the owner's resource identity but removes the module admin bypass.
func Restrict(owner authorization.Identity, key model.APIKey) (authorization.Identity, error) {
	if key.UserID != owner.UserID {
		return authorization.Identity{}, errors.New("API Key 所属用户不匹配")
	}
	if key.GrantMode == "" {
		return owner, nil
	}
	if key.GrantMode != Scoped {
		return authorization.Identity{}, errors.New("API Key 授权模式无效")
	}
	owner.Admin = false
	owner.Permissions = strings.Split(key.Scopes, ",")
	for _, p := range owner.Permissions {
		if !authorization.Known(p) {
			return authorization.Identity{}, errors.New("API Key 权限范围无效")
		}
	}
	return owner, nil
}

// EffectiveIdentity revalidates both account and credential on every service operation.
func EffectiveIdentity(ctx context.Context, db *gorm.DB, user uint) (authorization.Identity, error) {
	owner, err := authorization.Load(ctx, db, user)
	if err != nil {
		return owner, err
	}
	id, ok := ctx.Value(keyContext{}).(uint)
	if !ok {
		return owner, nil
	}
	key, err := Valid(ctx, db, id)
	if err != nil {
		return authorization.Identity{}, err
	}
	return Restrict(owner, key)
}
