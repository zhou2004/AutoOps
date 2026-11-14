package controller

import (
	system "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCertificateSensitiveEndpointsRejectMalformedRequestsWithoutCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, method, path, body string
		authenticated            bool
		code                     int
	}{
		{"new-import-unauthenticated", "POST", "/certificates/import", "{}", false, 401},
		{"version-import-unauthenticated", "POST", "/certificates/1/import", "{}", false, 401},
		{"private-export-unauthenticated", "GET", "/certificates/1/versions/0/export?key=true", "", false, 401},
		{"invalid-import-json", "POST", "/certificates/import", "{", true, 400},
		{"invalid-version", "GET", "/certificates/1/versions/not-a-version/export?key=true", "", true, 400},
		{"overflow-version", "GET", "/certificates/1/versions/4294967296/export?key=true", "", true, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// These requests must fail before the domain layer or database is used.
			c := &Controller{}
			r := gin.New()
			if tc.authenticated {
				r.Use(func(ctx *gin.Context) {
					ctx.Set(constant.ContextKeyUserObj, &system.JwtAdmin{ID: 1})
					ctx.Next()
				})
			}
			r.POST("/certificates/import", c.ImportNewCertificate)
			r.POST("/certificates/:id/import", c.ImportCertificate)
			r.GET("/certificates/:id/versions/:versionId/export", c.ExportCertificate)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			var body struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != tc.code {
				t.Fatalf("unexpected rejection: code=%d error=%v", body.Code, err)
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("sensitive endpoint did not disable response caching")
			}
		})
	}
}
