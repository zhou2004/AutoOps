package authorization

import "testing"

func TestWebTrafficReadDraftAndPublishPermissions(t *testing.T) {
	for _, tc := range []struct{ method, path, permission string }{
		{"GET", "/api/v1/web/traffic", "web:read"},
		{"GET", "/api/v1/web/clusters/:id/traffic-policy", "web:read"},
		{"POST", "/api/v1/web/clusters/:id/traffic-policy/preview", "web:write"},
		{"POST", "/api/v1/web/clusters/:id/traffic-policy/draft", "web:write"},
		{"POST", "/api/v1/web/operations", "web:execute"},
	} {
		got := Route(tc.method, tc.path)
		if got.Permission != tc.permission || got.AdminOnly {
			t.Errorf("%s %s = %+v; expected %s", tc.method, tc.path, got, tc.permission)
		}
	}
}
