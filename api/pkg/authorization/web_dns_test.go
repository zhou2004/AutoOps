package authorization

import "testing"

func TestWebDNSDiagnosticsRetainAdminAndModuleBoundaries(t *testing.T) {
	for _, tc := range []struct {
		method, path, permission string
		admin                    bool
	}{
		{"GET", "/api/v1/web/dns-credentials", "web:read", true},
		{"GET", "/api/v1/web/dns-accounts/:id/zones", "web:read", true},
		{"POST", "/api/v1/web/dns-accounts/:id/check", "web:write", true},
		{"POST", "/api/v1/web/sources/:id/check", "web:read", false},
	} {
		req := Route(tc.method, tc.path)
		if req.Permission != tc.permission || req.AdminOnly != tc.admin {
			t.Errorf("%s %s got %+v", tc.method, tc.path, req)
		}
	}
}
