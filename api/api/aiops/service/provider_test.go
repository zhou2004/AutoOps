package service

import (
	"context"
	model "dodevops-api/api/aiops/model"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStreamingProtocolsAndTruncation(t *testing.T) {
	cases := []struct {
		name, provider, body, want string
		fails                      bool
	}{
		{"openai unicode", "openai", "data: {\"choices\":[{\"delta\":{\"content\":\"中文🙂\"},\"finish_reason\":null}]}\r\n\r\ndata: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\r\n\r\ndata: [DONE]\r\n\r\n", "中文🙂", false},
		{"openai truncated", "openai", "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n", "partial", true},
		{"openai token limit", "openai", "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"length\"}]}\n\ndata: [DONE]\n\n", "", true},
		{"openai error", "openai", "data: {\"error\":{\"message\":\"TOP_SECRET\"}}\n\n", "", true},
		{"ollama", "ollama", "{\"message\":{\"content\":\"本地\"},\"done\":false}\n{\"message\":{\"content\":\"模型\"},\"done\":true,\"done_reason\":\"stop\"}\n", "本地模型", false},
		{"ollama truncated", "ollama", "{\"message\":{\"content\":\"partial\"},\"done\":false}\n", "partial", true},
		{"ollama empty", "ollama", "{\"done\":true}\n", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["stream"] != true || body["model"] != "test" {
					t.Error("wrong provider payload")
				}
				path := "/chat/completions"
				if tc.provider == "ollama" {
					path = "/api/chat"
				}
				if r.URL.Path != path {
					t.Error("wrong provider endpoint")
				}
				for _, b := range []byte(tc.body) {
					_, _ = w.Write([]byte{b})
					w.(http.Flusher).Flush()
				}
			}))
			defer server.Close()
			var out strings.Builder
			err := streamModel(context.Background(), model.Config{Provider: tc.provider, BaseURL: server.URL, Model: "test"}, []chatMessage{{Role: "user", Content: "hello"}}, func(s string) error { out.WriteString(s); return nil })
			if (err != nil) != tc.fails || out.String() != tc.want {
				t.Fatalf("got %q, %v", out.String(), err)
			}
			if err != nil && strings.Contains(err.Error(), "TOP_SECRET") {
				t.Fatal("upstream error leaked")
			}
		})
	}
}
func TestStreamCancellationAndHTTPError(t *testing.T) {
	cancelled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(cancelled)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := streamModel(ctx, model.Config{Provider: "openai", BaseURL: server.URL}, nil, func(string) error { cancel(); return context.Canceled })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel not propagated: %v", err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("upstream not cancelled")
	}
	failure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401); _, _ = w.Write([]byte("TOP_SECRET")) }))
	defer failure.Close()
	err = streamModel(context.Background(), model.Config{BaseURL: failure.URL}, nil, func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "TOP_SECRET") {
		t.Fatalf("unsafe error: %v", err)
	}
}
func TestWorkflowAdapters(t *testing.T) {
	for _, provider := range []string{"dify", "webhook"} {
		t.Run(provider, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				if provider == "dify" {
					if r.URL.Path != "/workflows/run" || body["response_mode"] != "streaming" {
						t.Error("invalid Dify request")
					}
					_, _ = fmt.Fprint(w, "data: {\"event\":\"text_chunk\",\"data\":{\"text\":\"diagnosis\"}}\n\ndata: {\"event\":\"workflow_finished\",\"data\":{\"status\":\"succeeded\",\"outputs\":{\"text\":\"diagnosis\"}}}\n\n")
				} else {
					_, _ = fmt.Fprint(w, `{"output":"diagnosis"}`)
				}
			}))
			defer server.Close()
			output := ""
			err := externalWorkflow(context.Background(), model.Config{Provider: provider, BaseURL: server.URL}, 1, "question", "en-US", "{}", func(e model.Event) error { output += e.Text; return nil })
			if err != nil || output != "diagnosis" {
				t.Fatalf("got %q %v", output, err)
			}
		})
	}
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "data: {\"event\":\"workflow_finished\",\"data\":{\"status\":\"failed\"}}\n\n")
	}))
	defer failed.Close()
	if externalWorkflow(context.Background(), model.Config{Provider: "dify", BaseURL: failed.URL}, 1, "q", "en-US", "{}", func(model.Event) error { return nil }) == nil {
		t.Fatal("failed workflow treated as success")
	}
}
func TestEndpointPolicy(t *testing.T) {
	for _, address := range []string{"http://127.0.0.1:11434", "http://192.168.1.1:3000/v1", "https://example.com/v1"} {
		if validateURL(address) != nil {
			t.Fatalf("local/private model rejected %s", address)
		}
	}
	for _, address := range []string{"file:///etc/passwd", "http://169.254.169.254/latest", "http://[fe80::1]/", "http://user:pass@example.com/v1", "http://example.com?key=secret", "http://0.0.0.0"} {
		if validateURL(address) == nil {
			t.Fatalf("unsafe address allowed %s", address)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest", http.StatusFound)
	}))
	defer server.Close()
	if _, err := post(context.Background(), model.Config{BaseURL: server.URL}, "", nil); err == nil {
		t.Fatal("redirect accepted")
	}
}
func TestSecretsEncryptedAndKeyPersisted(t *testing.T) {
	t.Setenv("AIOPS_ENCRYPTION_KEY", "")
	path := filepath.Join(t.TempDir(), "keys", "aiops.key")
	t.Setenv("AIOPS_KEY_FILE", path)
	a, err := sealSecret("TOP_SECRET")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := sealSecret("TOP_SECRET")
	if a == b || strings.Contains(a, "TOP_SECRET") {
		t.Fatal("encryption needs unique nonces")
	}
	decoded, err := openSecret(a)
	if err != nil || decoded != "TOP_SECRET" {
		t.Fatal("secret round trip failed")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("key permissions")
	}
	raw, _ := os.ReadFile(path)
	raw[0] ^= 1
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = openSecret(a); err == nil {
		t.Fatal("wrong key accepted")
	}
}
