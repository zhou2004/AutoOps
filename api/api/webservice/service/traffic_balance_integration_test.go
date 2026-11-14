package service

import (
	"dodevops-api/api/webservice/model"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Exercises generated native policy files against an isolated Nginx process.
// No system configuration, existing listener, or service manager is touched.
func TestTrafficPolicyRealWeightedAndBackupRouting(t *testing.T) {
	binary, err := exec.LookPath("nginx")
	if err != nil {
		t.Skip("native nginx unavailable")
	}
	backend := func(name string) *httptest.Server {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, name+":"+r.URL.Path)
		}))
		t.Cleanup(server.Close)
		return server
	}
	a, b := backend("A"), backend("B")
	dead := backend("unavailable")
	dead.Close()
	address := func(server *httptest.Server) string { return strings.TrimPrefix(server.URL, "http://") }
	for _, tc := range []struct {
		name    string
		members []UpstreamMember
		wantA   int
	}{
		{"weighted", []UpstreamMember{{Address: address(a), Weight: 3}, {Address: address(b), Weight: 1}}, 30},
		{"down-member-excluded", []UpstreamMember{{Address: address(a), Weight: 1, Down: true}, {Address: address(b), Weight: 1}}, 0},
		{"down-primary-uses-backup", []UpstreamMember{{Address: address(a), Weight: 1, Down: true}, {Address: address(b), Weight: 1, Backup: true}}, 0},
		{"unavailable-primary-uses-backup", []UpstreamMember{{Address: address(dead), Weight: 1, MaxFails: 1, FailTimeout: 30}, {Address: address(b), Weight: 1, Backup: true}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			port := listener.Addr().(*net.TCPAddr).Port
			listener.Close()
			root := t.TempDir()
			main, site := filepath.Join(root, "nginx.conf"), filepath.Join(root, "site.conf")
			original := []model.File{
				{Path: main, Mode: 0644, Content: fmt.Sprintf("worker_processes 1; pid %s/nginx.pid; error_log %s/error.log; events {} http { access_log %s/access.log; include %s; }\n", root, root, root, site)},
				{Path: site, Mode: 0644, Content: fmt.Sprintf("# native comment stays\nserver { listen 127.0.0.1:%d; server_name traffic.test; location / { proxy_pass %s/preserved; } }\n", port, a.URL)},
			}
			snapshot := InstanceConfig{InstanceID: 1, ClusterID: 1, ConfigPath: main, Prefix: root, ManagedPaths: []string{root}, Files: original}
			input := TrafficPolicyInput{Path: site, ServerIndex: 0, Policy: TrafficPolicySpec{Enabled: true, Upstream: &UpstreamSpec{Strategy: "round_robin", Members: tc.members}, Routes: []TrafficRoute{{Path: "/", Scheme: "http"}}}}
			changed, _, err := buildTrafficPolicy(snapshot, 1, input)
			if err != nil {
				t.Fatal(err)
			}
			files := map[string]string{}
			for _, f := range original {
				files[f.Path] = f.Content
			}
			for _, f := range changed {
				files[f.Path] = f.Content
			}
			for path, content := range files {
				if err = os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if !strings.Contains(files[site], "# native comment stays") {
				t.Fatal("native comment was lost")
			}
			check := exec.Command(binary, "-p", root, "-c", main, "-t")
			if out, e := check.CombinedOutput(); e != nil {
				t.Fatalf("nginx validation: %s: %v", out, e)
			}
			cmd := exec.Command(binary, "-p", root, "-c", main, "-g", "daemon off;")
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = cmd.Process.Signal(syscall.SIGQUIT)
				done := make(chan struct{})
				go func() { _ = cmd.Wait(); close(done) }()
				select {
				case <-done:
				case <-time.After(3 * time.Second):
					_ = cmd.Process.Kill()
					<-done
				}
			})
			addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
			ready := false
			for attempt := 0; attempt < 100; attempt++ {
				conn, e := net.DialTimeout("tcp", addr, 50*time.Millisecond)
				if e == nil {
					conn.Close()
					ready = true
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !ready {
				t.Fatal("isolated nginx did not start")
			}
			client := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}}
			defer client.CloseIdleConnections()
			countA := 0
			for n := 0; n < 40; n++ {
				res, e := client.Get("http://" + addr + "/")
				if e != nil {
					t.Fatal(e)
				}
				body, _ := io.ReadAll(res.Body)
				res.Body.Close()
				if res.StatusCode != 200 || (string(body) != "A:/preserved" && string(body) != "B:/preserved") {
					t.Fatalf("upstream route or original URI changed: %d %q", res.StatusCode, body)
				}
				if string(body) == "A:/preserved" {
					countA++
				}
			}
			if countA != tc.wantA {
				t.Fatalf("unexpected upstream selection A=%d B=%d", countA, 40-countA)
			}
		})
	}
}
