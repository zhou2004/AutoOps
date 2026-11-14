package service

import (
	"context"
	cmdbdao "dodevops-api/api/cmdb/dao"
	"dodevops-api/pkg/apikey"
)

// A Web credential cannot silently inherit host execution beyond its API-key scope.
func (s *Service) requireHostAccess(ctx context.Context, u, host uint, action string) error {
	identity, err := apikey.EffectiveIdentity(ctx, s.DB, u)
	if err != nil || !identity.Allows("cmdb:execute") {
		return ErrForbidden
	}
	allowed, err := cmdbdao.NewCmdbAssetPermissionDao(s.DB).CanUserAccess(u, "host", host, action)
	if err != nil || !allowed {
		return ErrForbidden
	}
	return nil
}
