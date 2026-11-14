package controller

import (
	system "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDNSDiagnosticsRejectUnauthorizedAndMalformedRequestsWithoutCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, method, path, body string
		authenticated            bool
		code                     int
	}{
		{"credentials-anonymous", "GET", "/dns-credentials?provider=cloudflare", "", false, 401},
		{"zones-anonymous", "GET", "/dns-accounts/1/zones", "", false, 401},
		{"check-anonymous", "POST", "/dns-accounts/1/check", "{}", false, 401},
		{"check-malformed", "POST", "/dns-accounts/1/check", "{", true, 400},
		{"check-extra-fields", "POST", "/dns-accounts/1/check", `{"token":"must-not-be-accepted"}`, true, 400},
		{"check-trailing-json", "POST", "/dns-accounts/1/check", `{} {}`, true, 400},
		{"check-oversized", "POST", "/dns-accounts/1/check", `{"domain":"` + strings.Repeat("a", 4096) + `"}`, true, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Controller{}
			r := gin.New()
			if tc.authenticated {
				r.Use(func(ctx *gin.Context) { ctx.Set(constant.ContextKeyUserObj, &system.JwtAdmin{ID: 1}); ctx.Next() })
			}
			r.GET("/dns-credentials", c.DNSCredentials)
			r.GET("/dns-accounts/:id/zones", c.DNSAccountZones)
			r.POST("/dns-accounts/:id/check", c.CheckDNSAccount)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			var result struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Code != tc.code {
				t.Fatal(result.Code, err)
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("cache header missing")
			}
		})
	}
}
