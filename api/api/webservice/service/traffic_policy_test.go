package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func trafficTestSnapshot(root string) InstanceConfig {
	main := filepath.Join(root, "nginx.conf")
	site := filepath.Join(root, "conf.d/site.conf")
	return InstanceConfig{InstanceID: 1, ClusterID: 1, Management: "operate", ConfigPath: main, Prefix: root, ManagedPaths: []string{root}, Digest: "base", Identity: "nginx", Files: []model.File{
		{Path: main, Mode: 0644, Content: "# untouched main\nerror_log " + root + "/error.log;\npid " + root + "/nginx.pid;\nevents {}\nhttp {\n    access_log " + root + "/access.log;\n    include conf.d/*.conf;\n}\n"},
		{Path: site, Mode: 0644, Content: "# untouched native comment\nserver {\n listen 18081;\n server_name web.example.test;\n set $literal 'quoted { # }';\n location / {\n   proxy_set_header Host $host; # keep comment\n   proxy_pass http://127.0.0.1:18082/original/;\n }\n}\n"},
	}}
}
func trafficTestInput(snapshot InstanceConfig) TrafficPolicyInput {
	return TrafficPolicyInput{InstanceIDs: []uint{1}, Path: snapshot.Files[1].Path, ServerIndex: 0, Baselines: map[uint]string{1: "base"}, Policy: TrafficPolicySpec{Enabled: true, Allow: []string{"127.0.0.1", "10.20.0.5/24"}, Deny: []string{"192.0.2.7"}, RequestsPerSecond: 10, Burst: 20, Nodelay: true, MaxConnections: 5, ResponseBytesPerSecond: 64000, TrafficLog: true, Upstream: &UpstreamSpec{Strategy: "least_conn", Members: []UpstreamMember{{Address: "127.0.0.1:18082", Weight: 2, MaxFails: 2, FailTimeout: 10}, {Address: "127.0.0.1:18083", Weight: 1, Backup: true}}}, Routes: []TrafficRoute{{Path: "/", Scheme: "http"}}}}
}
func trafficApplySnapshot(snapshot InstanceConfig, files []model.File) InstanceConfig {
	snapshot.Files = append([]model.File(nil), snapshot.Files...)
	for index := range snapshot.Files {
		for _, f := range files {
			if snapshot.Files[index].Path == f.Path {
				snapshot.Files[index] = f
			}
		}
	}
	return snapshot
}
func TestTrafficPolicyPreservesNativeConfigurationAndExactDisable(t *testing.T) {
	snapshot := trafficTestSnapshot("/etc/nginx")
	input := trafficTestInput(snapshot)
	files, preview, err := buildTrafficPolicy(snapshot, 1, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || len(preview.Files) != 2 || preview.UpstreamName == "" {
		t.Fatalf("bad candidate: %+v", preview)
	}
	joined := files[0].Content + files[1].Content
	if !strings.Contains(joined, "proxy_pass http://"+preview.UpstreamName+"/original/;") {
		t.Fatal("upstream replacement lost original URI suffix")
	}
	for _, want := range []string{"# untouched native comment", "set $literal 'quoted { # }';", "proxy_set_header Host $host; # keep comment", "limit_req_status 429;", "limit_conn_status 429;", "allow 10.20.0.0/24;", "limit_rate 64000;", "least_conn;", "max_fails=2 fail_timeout=10s", "backup;", "escape=json", "\"$limit_req_status\""} {
		if !strings.Contains(joined, want) {
			t.Fatalf("lost %q", want)
		}
	}
	next := trafficApplySnapshot(snapshot, files)
	// Reading metadata retains the policy and route restore evidence.
	native, err := newTrafficNativeConfig(next)
	if err != nil {
		t.Fatal(err)
	}
	node := native.Servers[input.Path][0]
	metadata, err := readTrafficMetadata(native.Files[input.Path].Content[node.Open+1:node.Close], trafficPolicyKey(1, input.Path, 0))
	if err != nil || metadata == nil {
		t.Fatalf("read policy: %+v %v", metadata, err)
	}
	if len(metadata.OriginalProxies) != 1 || metadata.OriginalProxies[0] != "proxy_pass http://127.0.0.1:18082/original/;" {
		t.Fatal("route restore evidence missing")
	}
	input.Policy = metadata.Policy
	input.Policy.Enabled = false
	restored, _, err := buildTrafficPolicy(next, 1, input)
	if err != nil {
		t.Fatal(err)
	}
	final := trafficApplySnapshot(next, restored)
	for index := range snapshot.Files {
		if snapshot.Files[index].Content != final.Files[index].Content {
			t.Fatalf("disable did not restore original bytes for %s", snapshot.Files[index].Path)
		}
	}
}
func TestTrafficPolicyRejectsDriftInjectionAndUnsupportedNativeRules(t *testing.T) {
	snapshot := trafficTestSnapshot("/etc/nginx")
	input := trafficTestInput(snapshot)
	files, _, err := buildTrafficPolicy(snapshot, 1, input)
	if err != nil {
		t.Fatal(err)
	}
	drifted := trafficApplySnapshot(snapshot, files)
	for index := range drifted.Files {
		drifted.Files[index].Content = strings.ReplaceAll(drifted.Files[index].Content, "limit_req_status 429;", "limit_req_status 503;")
	}
	if _, _, err = buildTrafficPolicy(drifted, 1, input); err == nil || !strings.Contains(err.Error(), "手工修改") {
		t.Fatalf("drift accepted: %v", err)
	}
	cases := []struct {
		name   string
		change func(*TrafficPolicyInput, *InstanceConfig)
		want   string
	}{
		{"directive injection", func(p *TrafficPolicyInput, _ *InstanceConfig) { p.Policy.Deny = []string{"127.0.0.1; deny all"} }, "CIDR"},
		{"unknown route", func(p *TrafficPolicyInput, _ *InstanceConfig) { p.Policy.Routes[0].Path = "/other" }, "location"},
		{"unregistered upstream", func(p *TrafficPolicyInput, _ *InstanceConfig) { p.Policy.Routes[0].Upstream = "other" }, "具名上游"},
		{"access override", func(_ *TrafficPolicyInput, s *InstanceConfig) {
			s.Files[1].Content = strings.Replace(s.Files[1].Content, "location / {", "location / { allow all;", 1)
		}, "手工 allow"},
		{"limit override", func(_ *TrafficPolicyInput, s *InstanceConfig) {
			s.Files[1].Content = strings.Replace(s.Files[1].Content, "location / {", "location / { limit_req zone=manual;", 1)
		}, "手工 limit_req"},
		{"hidden routing", func(_ *TrafficPolicyInput, s *InstanceConfig) {
			s.Files[1].Content = strings.Replace(s.Files[1].Content, "location / {", "location / { rewrite ^ /x break;", 1)
		}, "rewrite"},
		{"location log", func(_ *TrafficPolicyInput, s *InstanceConfig) {
			s.Files[1].Content = strings.Replace(s.Files[1].Content, "location / {", "location / { access_log /tmp/other.log;", 1)
		}, "独立 access_log"},
		{"missing explicit log", func(_ *TrafficPolicyInput, s *InstanceConfig) {
			s.Files[0].Content = strings.Replace(s.Files[0].Content, "    access_log /etc/nginx/access.log;\n", "", 1)
		}, "显式配置"},
		{"ip hash backup", func(p *TrafficPolicyInput, _ *InstanceConfig) { p.Policy.Upstream.Strategy = "ip_hash" }, "backup"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			s := trafficTestSnapshot("/etc/nginx")
			p := trafficTestInput(s)
			test.change(&p, &s)
			if _, _, e := buildTrafficPolicy(s, 1, p); e == nil || !strings.Contains(e.Error(), test.want) {
				t.Fatalf("expected %s, got %v", test.want, e)
			}
		})
	}
}
func TestTrafficPolicyFindsOnlyLoadedHTTPBlocksAndNestedIncludes(t *testing.T) {
	snapshot := trafficTestSnapshot("/etc/nginx")
	snapshot.Files[0].Content = "events {} http { include /etc/nginx/http.conf; } stream { server {listen 9;} }"
	snapshot.Files = append(snapshot.Files, model.File{Path: "/etc/nginx/http.conf", Content: "access_log /tmp/access.log; include conf.d/*.conf;"}, model.File{Path: "/etc/nginx/unloaded.conf", Content: "server {listen 10;}"})
	parsed, err := newTrafficNativeConfig(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Servers) != 1 || len(parsed.Servers["/etc/nginx/conf.d/site.conf"]) != 1 {
		t.Fatalf("unloaded or stream server leaked: %+v", parsed.Servers)
	}
	snapshot.Files = append(snapshot.Files, model.File{Path: "/etc/nginx/conf.d/loop.conf", Content: "include /etc/nginx/http.conf;"})
	if _, err = newTrafficNativeConfig(snapshot); err == nil {
		t.Fatal("include cycle accepted")
	}
}
func trafficServiceFixture(t *testing.T) (*Service, *configRuntime) {
	s, rt := configFixture(t)
	snapshot := trafficTestSnapshot("/etc/nginx")
	rt.ob = Observation{Digest: "base", Identity: "nginx", Files: snapshot.Files}
	s.DB.Table(model.Tables["instances"]).Where("id=1").Updates(map[string]any{"management": "operate", "observed": JSON(rt.ob)})
	return s, rt
}
func TestTrafficPolicyDraftValidatesAndLeavesRemoteUnchanged(t *testing.T) {
	s, rt := trafficServiceFixture(t)
	ctx := context.Background()
	view, e := s.TrafficPolicy(ctx, 1, 1, 0)
	if e != nil || len(view.Targets) != 1 {
		t.Fatalf("targets: %+v %v", view, e)
	}
	input := trafficTestInput(trafficTestSnapshot("/etc/nginx"))
	preview, e := s.PreviewTrafficPolicy(ctx, 1, 1, input)
	if e != nil || !preview.Valid || len(preview.Checks) != 1 {
		t.Fatalf("preview: %+v %v", preview, e)
	}
	saved, e := s.DraftTrafficPolicy(ctx, 1, 1, input)
	if e != nil || saved.Draft.ID == 0 {
		t.Fatalf("draft: %+v %v", saved, e)
	}
	for _, req := range rt.requests {
		if req.Action != "discover" && req.Action != "validate" {
			t.Fatalf("draft changed remote: %s", req.Action)
		}
	}
	var spec model.ConfigSpec
	if e = json.Unmarshal(saved.Draft.Spec, &spec); e != nil {
		t.Fatal(e)
	}
	if spec.Baselines[1] != "base" || len(spec.InstanceIDs) != 1 {
		t.Fatal("draft lost target baseline")
	}
	input.Baselines[1] = "old"
	if _, e = s.DraftTrafficPolicy(ctx, 1, 1, input); e != ErrConflict {
		t.Fatalf("stale baseline accepted: %v", e)
	}
	input.Baselines[1] = "base"
	input.InstanceIDs = []uint{999}
	if _, e = s.PreviewTrafficPolicy(ctx, 1, 1, input); e == nil {
		t.Fatal("cross-cluster instance accepted")
	}
	rt.err = errorsForTrafficTest("nginx invalid")
	input.InstanceIDs = []uint{1}
	if _, e = s.DraftTrafficPolicy(ctx, 1, 1, input); e == nil {
		t.Fatal("failed nginx validation created draft")
	}
	var count int64
	s.DB.Table(model.Tables["configurations"]).Count(&count)
	if count != 1 {
		t.Fatalf("invalid drafts persisted: %d", count)
	}
}

type errorsForTrafficTest string

func (e errorsForTrafficTest) Error() string { return string(e) }
func TestTrafficPolicyUnauthorizedTargetsAreNeverRead(t *testing.T) {
	s, rt := trafficServiceFixture(t)
	if _, e := s.TrafficPolicy(context.Background(), 2, 1, 0); e == nil {
		t.Fatal("unauthorized read accepted")
	}
	if _, e := s.PreviewTrafficPolicy(context.Background(), 2, 1, trafficTestInput(trafficTestSnapshot("/etc/nginx"))); e == nil {
		t.Fatal("unauthorized edit accepted")
	}
	if len(rt.requests) > 0 {
		t.Fatal("unauthorized target opened SSH")
	}
}
func TestTrafficPolicyRealNginxSyntax(t *testing.T) {
	binary, e := exec.LookPath("nginx")
	if e != nil {
		t.Skip("nginx executable unavailable")
	}
	root := t.TempDir()
	snapshot := trafficTestSnapshot(root)
	input := trafficTestInput(snapshot)
	files, _, e := buildTrafficPolicy(snapshot, 1, input)
	if e != nil {
		t.Fatal(e)
	}
	snapshot = trafficApplySnapshot(snapshot, files)
	for _, file := range snapshot.Files {
		if e = os.MkdirAll(filepath.Dir(file.Path), 0755); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(file.Path, []byte(file.Content), 0644); e != nil {
			t.Fatal(e)
		}
	}
	cmd := exec.Command(binary, "-p", root, "-c", snapshot.ConfigPath, "-t")
	if out, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("real nginx rejected traffic configuration: %v %s", e, out)
	}
}

func startTrafficTestNginx(t *testing.T, policy TrafficPolicySpec, backendURL string) string {
	t.Helper()
	binary, e := exec.LookPath("nginx")
	if e != nil {
		t.Skip("nginx executable unavailable")
	}
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	address := listener.Addr().String()
	listener.Close()
	root := t.TempDir()
	snapshot := trafficTestSnapshot(root)
	snapshot.Files[1].Content = "server { listen " + address + "; server_name localhost; location / { proxy_pass " + backendURL + "/original/; } }"
	input := TrafficPolicyInput{Path: snapshot.Files[1].Path, Policy: policy}
	files, _, e := buildTrafficPolicy(snapshot, 1, input)
	if e != nil {
		t.Fatal(e)
	}
	snapshot = trafficApplySnapshot(snapshot, files)
	for _, file := range snapshot.Files {
		if e = os.MkdirAll(filepath.Dir(file.Path), 0755); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(file.Path, []byte(file.Content), 0644); e != nil {
			t.Fatal(e)
		}
	}
	log, e := os.Create(filepath.Join(root, "process.log"))
	if e != nil {
		t.Fatal(e)
	}
	command := exec.Command(binary, "-p", root, "-c", snapshot.ConfigPath, "-g", "daemon off; master_process off;")
	command.Stdout = log
	command.Stderr = log
	if e = command.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		_ = command.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() { command.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			command.Process.Kill()
			<-done
		}
		log.Close()
	})
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, e := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if e == nil {
			conn.Close()
			return "http://" + address
		}
		time.Sleep(20 * time.Millisecond)
	}
	data, _ := os.ReadFile(filepath.Join(root, "process.log"))
	t.Fatalf("isolated nginx not ready: %s", data)
	return ""
}
func TestTrafficPolicyRealNginxDenyAndRequestLimit(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("backend-ok")) }))
	defer backend.Close()
	cases := []struct {
		name     string
		policy   TrafficPolicySpec
		statuses []int
	}{
		{"baseline", TrafficPolicySpec{Enabled: true}, []int{200}},
		{"deny", TrafficPolicySpec{Enabled: true, Deny: []string{"127.0.0.1"}}, []int{403}},
		{"request limit", TrafficPolicySpec{Enabled: true, RequestsPerSecond: 1}, []int{200, 429}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			url := startTrafficTestNginx(t, tc.policy, backend.URL)
			client := http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
			for _, want := range tc.statuses {
				response, e := client.Get(url + "/test")
				if e != nil {
					t.Fatal(e)
				}
				io.Copy(io.Discard, response.Body)
				response.Body.Close()
				if response.StatusCode != want {
					t.Fatalf("got %d want %d", response.StatusCode, want)
				}
			}
		})
	}
}
func TestTrafficPolicyRealNginxConnectionLimit(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	defer close(release)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		w.Write([]byte("ok"))
	}))
	defer backend.CloseClientConnections()
	// Close the blocking request before closing the upstream server.
	t.Cleanup(backend.Close)
	url := startTrafficTestNginx(t, TrafficPolicySpec{Enabled: true, MaxConnections: 1}, backend.URL)
	client := http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	done := make(chan struct{})
	go func() {
		response, _ := client.Get(url + "/hold")
		if response != nil {
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
		}
		close(done)
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream was not reached")
	}
	response, e := client.Get(url + "/other")
	if e != nil {
		t.Fatal(e)
	}
	io.Copy(io.Discard, response.Body)
	response.Body.Close()
	if response.StatusCode != 429 {
		t.Fatalf("connection limit got %d", response.StatusCode)
	}
}

func TestTrafficPolicyConflictsThroughIncludesAndGlobalInheritance(t *testing.T) {
	for _, scope := range []string{"server include", "global inheritance"} {
		t.Run(scope, func(t *testing.T) {
			snapshot := trafficTestSnapshot("/etc/nginx")
			input := trafficTestInput(snapshot)
			if scope == "server include" {
				snapshot.Files[1].Content = strings.Replace(snapshot.Files[1].Content, "server {", "server { include /etc/nginx/security.inc;", 1)
				snapshot.Files = append(snapshot.Files, model.File{Path: "/etc/nginx/security.inc", Content: "# existing controls\nallow 10.0.0.0/8; deny all;"})
			} else {
				snapshot.Files[0].Content = strings.Replace(snapshot.Files[0].Content, "http {", "http { limit_req zone=existing;", 1)
			}
			if _, _, err := buildTrafficPolicy(snapshot, 1, input); err == nil || !strings.Contains(err.Error(), "手工") {
				t.Fatalf("manual inherited controls accepted: %v", err)
			}
		})
	}
}
func TestTrafficPolicyRejectsRelocatedAndOrphanMarkers(t *testing.T) {
	snapshot := trafficTestSnapshot("/etc/nginx")
	input := trafficTestInput(snapshot)
	files, _, err := buildTrafficPolicy(snapshot, 1, input)
	if err != nil {
		t.Fatal(err)
	}
	candidate := trafficApplySnapshot(snapshot, files)
	native, err := newTrafficNativeConfig(candidate)
	if err != nil {
		t.Fatal(err)
	}
	server := native.Servers[input.Path][0]
	metadata, err := readTrafficMetadata(native.Files[input.Path].Content[server.Open+1:server.Close], trafficPolicyKey(1, input.Path, 0))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("server marker moved to location", func(t *testing.T) {
		changed := trafficApplySnapshot(snapshot, files)
		block := trafficServerBlock(*metadata)
		changed.Files[1].Content = strings.Replace(changed.Files[1].Content, block, "", 1)
		changed.Files[1].Content = strings.Replace(changed.Files[1].Content, "location / {", "location / {"+block, 1)
		if _, _, e := buildTrafficPolicy(changed, 1, input); e == nil {
			t.Fatal("moved server controls accepted")
		}
	})
	t.Run("orphan global marker", func(t *testing.T) {
		changed := trafficApplySnapshot(snapshot, files)
		changed.Files[1] = snapshot.Files[1]
		if _, _, e := buildTrafficPolicy(changed, 1, input); e == nil || !strings.Contains(e.Error(), "孤立") {
			t.Fatalf("orphan accepted: %v", e)
		}
	})
	t.Run("route moved", func(t *testing.T) {
		changed := trafficApplySnapshot(snapshot, files)
		block := trafficRouteBlock(*metadata, 0)
		changed.Files[1].Content = strings.Replace(changed.Files[1].Content, block, "return 204;", 1)
		changed.Files[1].Content = strings.Replace(changed.Files[1].Content, "server {", "server { location /other {"+block+"}\n", 1)
		if _, _, e := buildTrafficPolicy(changed, 1, input); e == nil {
			t.Fatal("moved route accepted")
		}
	})
}

func TestTrafficPolicyRejectsDefensePhaseBypass(t *testing.T) {
	for _, directive := range []string{"return 200 ok;", "rewrite ^ /other last;", "error_page 403 =200 /fallback;", "recursive_error_pages on;"} {
		for _, scope := range []string{"server", "location", "nested if", "include", "http"} {
			// return and rewrite are not legal in http, whereas error_page is inherited.
			if scope == "http" && (strings.HasPrefix(directive, "return") || strings.HasPrefix(directive, "rewrite")) {
				continue
			}
			t.Run(scope+"/"+directive, func(t *testing.T) {
				snapshot := trafficTestSnapshot("/etc/nginx")
				switch scope {
				case "server":
					snapshot.Files[1].Content = strings.Replace(snapshot.Files[1].Content, "server {", "server { "+directive, 1)
				case "location":
					snapshot.Files[1].Content = strings.Replace(snapshot.Files[1].Content, "location / {", "location / { "+directive, 1)
				case "nested if":
					snapshot.Files[1].Content = strings.Replace(snapshot.Files[1].Content, "location / {", "location / { if ($request_method = GET) { "+directive+" }", 1)
				case "include":
					snapshot.Files[1].Content = strings.Replace(snapshot.Files[1].Content, "server {", "server { include /etc/nginx/early.inc;", 1)
					snapshot.Files = append(snapshot.Files, model.File{Path: "/etc/nginx/early.inc", Content: directive})
				case "http":
					snapshot.Files[0].Content = strings.Replace(snapshot.Files[0].Content, "http {", "http { "+directive, 1)
				}
				input := TrafficPolicyInput{Path: snapshot.Files[1].Path, Policy: TrafficPolicySpec{Enabled: true, Deny: []string{"127.0.0.1"}}}
				if _, _, err := buildTrafficPolicy(snapshot, 1, input); err == nil || !strings.Contains(err.Error(), "手工") {
					t.Fatalf("bypass accepted: %v", err)
				}
			})
		}
	}
	// Pure observability remains valid for native return-based health endpoints.
	snapshot := trafficTestSnapshot("/etc/nginx")
	snapshot.Files[1].Content = "server { listen 18081; location / { return 200 ok; } }"
	if _, _, err := buildTrafficPolicy(snapshot, 1, TrafficPolicyInput{Path: snapshot.Files[1].Path, Policy: TrafficPolicySpec{Enabled: true, TrafficLog: true}}); err != nil {
		t.Fatalf("logging-only return should remain allowed: %v", err)
	}
}
func TestTrafficPolicyRealNginxReturnCannotPretendToEnforceACL(t *testing.T) {
	// A return handler does answer 200 before Nginx's access handler can deny it.
	// Verify that behavior, then require the policy builder to reject the target.
	binary, e := exec.LookPath("nginx")
	if e != nil {
		t.Skip("nginx executable unavailable")
	}
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	address := listener.Addr().String()
	listener.Close()
	root := t.TempDir()
	snapshot := trafficTestSnapshot(root)
	snapshot.Files[1].Content = "server { listen " + address + "; deny all; location / { return 200 bypass; } }"
	for _, file := range snapshot.Files {
		os.MkdirAll(filepath.Dir(file.Path), 0755)
		if e = os.WriteFile(file.Path, []byte(file.Content), 0644); e != nil {
			t.Fatal(e)
		}
	}
	command := exec.Command(binary, "-p", root, "-c", snapshot.ConfigPath, "-g", "daemon off; master_process off;")
	if e = command.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() {
		command.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() { command.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			command.Process.Kill()
			<-done
		}
	}()
	client := http.Client{Timeout: time.Second}
	var response *http.Response
	for until := time.Now().Add(2 * time.Second); time.Now().Before(until); {
		response, e = client.Get("http://" + address)
		if e == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if e != nil {
		t.Fatal(e)
	}
	io.Copy(io.Discard, response.Body)
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("expected native return bypass demonstration, got %d", response.StatusCode)
	}
	snapshot.Files[1].Content = strings.Replace(snapshot.Files[1].Content, "deny all; ", "", 1)
	for _, policy := range []TrafficPolicySpec{{Enabled: true, Deny: []string{"127.0.0.1"}}, {Enabled: true, RequestsPerSecond: 1}, {Enabled: true, MaxConnections: 1}, {Enabled: true, ResponseBytesPerSecond: 1}} {
		if _, _, err := buildTrafficPolicy(snapshot, 1, TrafficPolicyInput{Path: snapshot.Files[1].Path, Policy: policy}); err == nil || !strings.Contains(err.Error(), "return") {
			t.Fatalf("native return accepted as protected: %+v %v", policy, err)
		}
	}
}
