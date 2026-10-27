package service

import (
	"context"
	"crypto/sha256"
	model "dodevops-api/api/aiops/model"
	cmdb "dodevops-api/api/cmdb/model"
	system "dodevops-api/api/system/model"
	"dodevops-api/pkg/apikey"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func sessionService(t *testing.T) *Service {
	s := New(aiopsTestDB(t))
	if err := s.DB.AutoMigrate(&model.Session{}, &model.Message{}, &model.ToolCall{}, &model.HarnessTicket{}, &system.APIKey{}, &system.RolePermission{}, &cmdb.CmdbAssetConnection{}, &cmdb.CmdbCredentialPermission{}); err != nil {
		t.Fatal(err)
	}
	return s
}
func TestConversationPersistsContextAndIsolation(t *testing.T) {
	s := sessionService(t)
	var requests [][]chatMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []chatMessage `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		requests = append(requests, body.Messages)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"fixture answer\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	cfg := model.Config{Name: "fixture", Kind: "model", Enabled: true, Provider: "openai", Model: "fixture", BaseURL: server.URL, TimeoutSeconds: 30}
	s.DB.Create(&cfg)
	ctx := context.Background()
	row, err := s.CreateSession(ctx, 1, model.SessionRequest{ModelID: cfg.ID, Title: "diagnose"})
	if err != nil || row.ID == 0 {
		t.Fatalf("session: %+v %v", row, err)
	}
	if _, err = s.Session(ctx, 2, row.ID); err == nil {
		t.Fatal("session ownership leaked")
	}
	if err = s.Chat(ctx, 2, row.ID, 0, true, nil, "other user", func(model.Event) error { return nil }); err == nil {
		t.Fatal("cross-user chat accepted")
	}
	for _, prompt := range []string{"first question", "follow up"} {
		done := false
		if err = s.Chat(ctx, 1, row.ID, 0, true, nil, prompt, func(e model.Event) error {
			if e.Type == "done" {
				done = true
			}
			return nil
		}); err != nil || !done {
			t.Fatalf("chat: %v done=%v", err, done)
		}
	}
	if len(requests) != 2 || len(requests[1]) != 4 || requests[1][1].Content != "first question" || requests[1][2].Content != "fixture answer" {
		t.Fatalf("lost context: %+v", requests)
	}
	var messages []model.Message
	s.DB.Where("session_id = ?", row.ID).Find(&messages)
	if len(messages) != 4 {
		t.Fatal("messages not persisted")
	}
	s.DB.Model(&row).Update("busy_until", time.Now().Add(time.Minute))
	if err = s.Chat(ctx, 1, row.ID, 0, true, nil, "duplicate", func(model.Event) error { return nil }); err == nil {
		t.Fatal("concurrent turn accepted")
	}
}
func TestToolAuthorizationAndRevocation(t *testing.T) {
	s := sessionService(t)
	s.DB.Exec("INSERT INTO sys_role VALUES (2,'operator',1)")
	s.DB.Exec("INSERT INTO sys_admin_role VALUES (2,2)")
	s.DB.Create(&system.RolePermission{RoleID: 2, Permissions: `["aiops:read","aiops:operate","docker:read","docker:execute"]`})
	ctx := context.Background()
	s.DB.Exec("INSERT INTO cmdb_host VALUES(11,'worker',1,1,1)")
	profile := cmdb.CmdbAssetConnection{AssetType: "host", AssetID: 11, Name: "worker", Protocol: "ssh", CredentialID: 1, Enabled: true}
	s.DB.Create(&profile)
	rule := cmdb.CmdbAssetPermission{Name: "worker", UserIDs: "[2]", AssetTypes: `["host"]`, HostIDs: "[11]", PermissionActions: `["docker_view","docker_logs","docker_operate"]`, IsActive: 1}
	s.DB.Create(&rule)
	cfg := model.Config{Name: "fixture", Kind: "model", Enabled: true}
	s.DB.Create(&cfg)
	row, err := s.CreateSession(ctx, 2, model.SessionRequest{ModelID: cfg.ID, Targets: []uint{profile.ID}})
	if err != nil {
		t.Fatal(err)
	}
	r := model.ToolRequest{CallID: "request-123", Tool: "docker_restart", ConnectionID: profile.ID, Container: "app"}
	if _, err = s.ExecuteTool(ctx, 2, row.ID, true, r); err == nil {
		t.Fatal("ungranted mutation permitted")
	}
	row, err = s.Grant(ctx, 2, row.ID, model.GrantRequest{Actions: []string{"docker_restart"}, Minutes: 15})
	if err != nil || row.Actions != "docker_restart" {
		t.Fatalf("grant: %v %+v", err, row)
	}
	if _, err = s.ExecuteTool(ctx, 2, row.ID, false, r); err == nil {
		t.Fatal("readonly API key permitted mutation")
	}
	wrong := r
	wrong.ConnectionID = 999
	if _, err = s.ExecuteTool(ctx, 2, row.ID, true, wrong); err == nil {
		t.Fatal("unselected target permitted")
	}
	// A persisted request must be returned without executing again.
	raw, _ := json.Marshal(r)
	record := model.ToolCall{SessionID: row.ID, CallID: r.CallID, Tool: r.Tool, Arguments: string(raw), Status: "succeeded", Result: "prior result"}
	s.DB.Create(&record)
	got, err := s.ExecuteTool(ctx, 2, row.ID, true, r)
	if err != nil || got.Result != "prior result" {
		t.Fatal("idempotent result not returned", err)
	}
	s.DB.Model(&system.RolePermission{}).Where("role_id = ?", 2).Update("permissions", `["aiops:read","docker:read"]`)
	if _, err = s.ExecuteTool(ctx, 2, row.ID, true, r); err == nil {
		t.Fatal("revoked module permission still usable")
	}
	s.DB.Model(&system.RolePermission{}).Where("role_id = ?", 2).Update("permissions", `["aiops:read","aiops:operate","docker:read","docker:execute"]`)
	s.DB.Model(&rule).Update("is_active", 0)
	if _, err = s.ExecuteTool(ctx, 2, row.ID, true, r); err == nil {
		t.Fatal("revoked asset permission still usable")
	}
	s.DB.Model(&rule).Update("is_active", 1)
	s.DB.Model(&row).Update("grant_until", time.Now().Add(-time.Second))
	if _, err = s.ExecuteTool(ctx, 2, row.ID, true, r); err == nil {
		t.Fatal("expired grant permitted")
	}
	key, _, err := apikey.Create(ctx, s.DB, 2, "harness", []string{"aiops:read", "aiops:chat", "aiops:operate"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("a", 64)
	hash := sha256.Sum256([]byte(token))
	ticket := model.HarnessTicket{Hash: hex.EncodeToString(hash[:]), UserID: 2, SessionID: row.ID, APIKeyID: key.ID, CanOperate: true, ExpiresAt: time.Now().Add(time.Minute)}
	s.DB.Create(&ticket)
	s.DB.Model(&key).Update("revoked_at", time.Now())
	if _, err = s.HarnessTool(ctx, token, r); err == nil || !strings.Contains(err.Error(), "API Key") {
		t.Fatal("revoked API key callback permitted", err)
	}
	if _, err = s.HarnessTool(ctx, strings.Repeat("b", 64), r); err == nil {
		t.Fatal("forged callback ticket permitted")
	}
	// Scoped callback credentials are independent of role module grants, but still bound to asset grants.
	scoped, _, err := apikey.CreateScoped(ctx, s.DB, 2, "scoped callback", []string{"aiops:read", "aiops:operate", "docker:read", "docker:execute"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	s.DB.Model(&ticket).Update("api_key_id", scoped.ID)
	s.DB.Model(&row).Update("grant_until", time.Now().Add(time.Minute))
	s.DB.Model(&system.RolePermission{}).Where("role_id = 2").Update("permissions", `[]`)
	got, err = s.HarnessTool(ctx, token, r)
	if err != nil || got.Result != "prior result" {
		t.Fatal("independent callback grant lost", err)
	}
	s.DB.Model(&scoped).Update("scopes", "aiops:read,aiops:operate,docker:read")
	if _, err = s.HarnessTool(ctx, token, r); err == nil {
		t.Fatal("callback bypassed missing docker:execute")
	}
	s.DB.Model(&scoped).Update("scopes", "aiops:read,aiops:operate,docker:read,docker:execute")
	s.DB.Model(&rule).Update("is_active", 0)
	if _, err = s.HarnessTool(ctx, token, r); err == nil {
		t.Fatal("scoped callback bypassed revoked asset grant")
	}
	s.DB.Model(&rule).Update("is_active", 1)
	s.DB.Model(&scoped).Update("revoked_at", time.Now())
	if _, err = s.HarnessTool(ctx, token, r); err == nil {
		t.Fatal("revoked scoped callback permitted")
	}

}
