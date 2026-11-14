package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// configRuntime answers the synchronous validate action used by the page and
// records every candidate file set it receives.
type configRuntime struct {
	requests []RemoteRequest
	err      error
	ob       Observation
}

func (r *configRuntime) Run(_ context.Context, req RemoteRequest) (json.RawMessage, error) {
	r.requests = append(r.requests, req)
	if req.Action == "discover" {
		return JSON(r.ob), nil
	}
	if req.Action != "validate" {
		return nil, errors.New("unexpected action")
	}
	if r.err != nil {
		return nil, r.err
	}
	return JSON(map[string]bool{"validated": true}), nil
}

func configFixture(t *testing.T) (*Service, *configRuntime) {
	s := fixture(t)
	rt := &configRuntime{}
	s.Remote = rt
	observed := Observation{Digest: "base", Identity: "nginx", Files: []model.File{
		{Path: "/etc/nginx/nginx.conf", Content: "events {} http { include conf.d/*.conf; }", Mode: 0644},
		{Path: "/etc/nginx/conf.d/site.conf", Content: "server { listen 80; }", Mode: 0644},
	}}
	rt.ob = observed
	if e := s.DB.Table(model.Tables["instances"]).Where("id = 1").Update("observed", JSON(observed)).Error; e != nil {
		t.Fatal(e)
	}
	return s, rt
}

func TestValidateInstanceConfigOnlySendsPathsInsideManagedScope(t *testing.T) {
	s, rt := configFixture(t)
	ctx := context.Background()
	good, e := s.ValidateInstanceConfig(ctx, 1, 1, []model.File{{Path: "/etc/nginx/conf.d/new.conf", Content: "server { listen 8080; }"}})
	if e != nil || !good.Valid || good.Checked != 1 {
		t.Fatalf("managed path rejected: %+v %v", good, e)
	}
	if len(rt.requests) != 2 || rt.requests[1].Action != "validate" || rt.requests[1].Identity != "nginx" {
		t.Fatalf("validation request missing identity: %+v", rt.requests)
	}
	outside, e := s.ValidateInstanceConfig(ctx, 1, 1, []model.File{{Path: "/etc/passwd", Content: "root"}})
	if e != nil {
		t.Fatal(e)
	}
	if outside.Valid || !strings.Contains(outside.Message, "受管目录") {
		t.Fatalf("path outside managed scope was accepted: %+v", outside)
	}
	if len(rt.requests) != 2 {
		t.Fatal("invalid path must not reach the remote host")
	}
}

func TestValidateInstanceConfigSurfacesNginxSyntaxError(t *testing.T) {
	s, rt := configFixture(t)
	injected := errors.New("nginx: [emerg] unexpected \"}\" in /tmp/autoops-web-12-34/etc/nginx/conf.d/site.conf:4")
	rt.err = injected
	got, e := s.ValidateInstanceConfig(context.Background(), 1, 1, []model.File{{Path: "/etc/nginx/conf.d/site.conf", Content: "server { } }"}})
	if e != nil {
		t.Fatal(e)
	}
	if got.Valid {
		t.Fatal("syntax error reported as valid")
	}
	if !strings.Contains(got.Message, "[emerg]") {
		t.Fatalf("nginx output lost: %q", got.Message)
	}
	if strings.Contains(got.Message, "/tmp/autoops-web") {
		t.Fatalf("staging path leaked into user message: %q", got.Message)
	}
}

func TestValidateInstanceConfigRejectsPrivateKeyAndMissingDeleteTarget(t *testing.T) {
	s, _ := configFixture(t)
	ctx := context.Background()
	key, e := s.ValidateInstanceConfig(ctx, 1, 1, []model.File{{Path: "/etc/nginx/conf.d/key.conf", Content: "-----BEGIN PRIVATE KEY-----"}})
	if e != nil || key.Valid || !strings.Contains(key.Message, "私钥") {
		t.Fatalf("private key accepted: %+v %v", key, e)
	}
	missing, e := s.ValidateInstanceConfig(ctx, 1, 1, []model.File{{Path: "/etc/nginx/conf.d/gone.conf", Deleted: true}})
	if e != nil || missing.Valid || !strings.Contains(missing.Message, "不存在") {
		t.Fatalf("deleting an unknown file was accepted: %+v %v", missing, e)
	}
}

func TestConfigGroupIsUniqueAndFiltersConfigurationList(t *testing.T) {
	s, _ := configFixture(t)
	ctx := context.Background()
	group := model.Resource{Name: "站点配置", GroupID: 10, ClusterID: 1, Engine: "nginx", Revision: 0,
		Spec: JSON(model.ConfigGroupSpec{Description: "server blocks", Color: "#409EFF"})}
	saved, e := s.Save(ctx, 1, "config-groups", group, "")
	if e != nil {
		t.Fatal(e)
	}
	duplicate := model.Resource{Name: "站点配置", GroupID: 10, ClusterID: 1, Engine: "nginx", Revision: 0,
		Spec: JSON(model.ConfigGroupSpec{})}
	if _, e = s.Save(ctx, 1, "config-groups", duplicate, ""); e == nil {
		t.Fatal("duplicate config group name was accepted")
	}
	badColor := model.Resource{Name: "上游配置", GroupID: 10, ClusterID: 1, Engine: "nginx", Revision: 0,
		Spec: JSON(model.ConfigGroupSpec{Color: "blue"})}
	if _, e = s.Save(ctx, 1, "config-groups", badColor, ""); e == nil {
		t.Fatal("invalid color was accepted")
	}
	draft := model.Resource{Name: "conf.d", GroupID: 10, ClusterID: 1, Engine: "nginx", Revision: 0,
		Spec: JSON(model.ConfigSpec{ConfigGroupID: saved.ID, Files: []model.File{
			{Path: "/etc/nginx/conf.d/site.conf", Content: "server { listen 80; }", Mode: 0644, Group: "站点"},
			{Path: "/etc/nginx/conf.d/up.conf", Content: "upstream a { server 127.0.0.1:8080; }", Mode: 0644, Group: "上游"},
		}})}
	if _, e = s.Save(ctx, 1, "configurations", draft, ""); e != nil {
		t.Fatal(e)
	}
	rows, total, e := s.List(1, "configurations", Filter{ConfigGroupID: saved.ID})
	if e != nil || total != 1 || len(rows) != 1 {
		t.Fatalf("config group filter: %v %d %v", rows, total, e)
	}
	if _, total, _ = s.List(1, "configurations", Filter{FileGroup: "上游"}); total != 1 {
		t.Fatal("file group filter missed the upstream draft")
	}
	if _, total, _ = s.List(1, "configurations", Filter{FileGroup: "证书"}); total != 0 {
		t.Fatal("file group filter matched an absent group")
	}
	if _, total, _ = s.List(1, "configurations", Filter{ConfigGroupID: saved.ID + 99}); total != 0 {
		t.Fatal("unknown config group matched drafts")
	}
}

func TestConfigurationRejectsGroupFromAnotherCluster(t *testing.T) {
	s, _ := configFixture(t)
	ctx := context.Background()
	s.DB.Table(model.Tables["clusters"]).Create(&model.Resource{ID: 2, Name: "other", GroupID: 10, Revision: 1})
	foreign, e := s.Save(ctx, 1, "config-groups", model.Resource{Name: "别的集群", GroupID: 10, ClusterID: 2, Engine: "nginx",
		Spec: JSON(model.ConfigGroupSpec{})}, "")
	if e != nil {
		t.Fatal(e)
	}
	draft := model.Resource{Name: "conf.d", GroupID: 10, ClusterID: 1, Engine: "nginx",
		Spec: JSON(model.ConfigSpec{ConfigGroupID: foreign.ID, Files: []model.File{{Path: "/etc/nginx/conf.d/a.conf", Content: "server {}", Mode: 0644}}})}
	if _, e = s.Save(ctx, 1, "configurations", draft, ""); e == nil || !strings.Contains(e.Error(), "同一集群") {
		t.Fatalf("foreign config group accepted: %v", e)
	}
}

func TestCertificateExportProtectsPrivateKey(t *testing.T) {
	s, _ := configFixture(t)
	certPEM, keyPEM := certificateAssetPEM(t, nil)
	s.DB.Table(model.Tables["certificates"]).Create(&model.Resource{ID: 1, Name: "cert", GroupID: 10, Engine: "nginx", Revision: 1,
		Status: "issued", Spec: JSON(model.CertificateSpec{Domains: []string{"a.example.com"}, CurrentVersionID: 1})})
	content := certContent{PEM: certPEM,
		Domains: []string{"a.example.com"}, Fingerprint: "fp", NotAfter: time.Now().Add(30 * 24 * time.Hour)}
	sealed, e := seal(keyPEM)
	if e != nil {
		t.Fatal(e)
	}
	s.DB.Create(&model.Revision{ID: 1, ClusterID: 0, ResourceID: 1, Kind: "certificate", Name: "cert", Digest: "fp", Content: JSON(content), Secret: sealed})
	public, e := s.ExportCertificate(context.Background(), 1, 1, 0, false)
	if e != nil {
		t.Fatal(e)
	}
	if public["certificate"] == "" || public["privateKey"] != "" {
		t.Fatalf("public export leaked the key: %+v", public)
	}
	var events int64
	s.DB.Model(&model.Event{}).Where("kind = ?", "certificate-export").Count(&events)
	if events != 1 {
		t.Fatal("certificate export was not audited")
	}
	if _, e = s.ExportCertificate(context.Background(), 2, 1, 0, true); !errors.Is(e, ErrForbidden) {
		t.Fatalf("private key export was not restricted: %v", e)
	}
	versions, current, e := s.CertificateVersions(1, 1)
	if e != nil || current != 1 || len(versions) != 1 || !versions[0].Current || versions[0].Fingerprint != "fp" {
		t.Fatalf("certificate versions: %+v %d %v", versions, current, e)
	}
}

func TestConfigGroupInUseCannotBeDeleted(t *testing.T) {
	s, _ := configFixture(t)
	ctx := context.Background()
	group, e := s.Save(ctx, 1, "config-groups", model.Resource{Name: "在用分组", GroupID: 10, ClusterID: 1, Engine: "nginx",
		Spec: JSON(model.ConfigGroupSpec{})}, "")
	if e != nil {
		t.Fatal(e)
	}
	draft, e := s.Save(ctx, 1, "configurations", model.Resource{Name: "conf.d", GroupID: 10, ClusterID: 1, Engine: "nginx",
		Spec: JSON(model.ConfigSpec{ConfigGroupID: group.ID, Files: []model.File{{Path: "/etc/nginx/conf.d/a.conf", Content: "server {}", Mode: 0644}}})}, "")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Delete(ctx, 1, "config-groups", group.ID); e == nil || !strings.Contains(e.Error(), "引用") {
		t.Fatalf("in-use config group was deleted: %v", e)
	}
	if e = s.Delete(ctx, 1, "configurations", draft.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.Delete(ctx, 1, "config-groups", group.ID); e != nil {
		t.Fatal(e)
	}
}

func TestSavingConfigurationRequiresRemoteSyntaxCheckBeforePersistence(t *testing.T) {
	s, rt := configFixture(t)
	ctx := context.Background()
	draft := model.Resource{Name: "native", GroupID: 10, ClusterID: 1, Engine: "nginx",
		Spec: JSON(model.ConfigSpec{Files: []model.File{{Path: "/etc/nginx/conf.d/site.conf", Content: "server { listen 8080; }"}}})}
	rt.err = errors.New("nginx: unknown directive")
	if _, err := s.Save(ctx, 1, "configurations", draft, ""); err == nil || !strings.Contains(err.Error(), "unknown directive") {
		t.Fatalf("failed syntax saved: %v", err)
	}
	var count int64
	s.DB.Table(model.Tables["configurations"]).Count(&count)
	if count != 0 {
		t.Fatal("invalid syntax created a draft")
	}
	rt.err = nil
	saved, err := s.Save(ctx, 1, "configurations", draft, "")
	if err != nil {
		t.Fatal(err)
	}
	spec, _ := decode[model.ConfigSpec](saved.Spec)
	if spec.Baselines[1] != "base" {
		t.Fatal("save did not freeze checked baseline")
	}
	original := string(saved.Spec)
	spec.Files[0].Content = "invalid on;"
	saved.Spec = JSON(spec)
	rt.err = errors.New("nginx: syntax error")
	if _, err = s.Save(ctx, 1, "configurations", saved, ""); err == nil {
		t.Fatal("invalid edit accepted")
	}
	live, err := s.Get("configurations", saved.ID)
	if err != nil || string(live.Spec) != original || live.Revision != saved.Revision {
		t.Fatal("failed syntax changed existing draft")
	}
}

func TestValidationBindsFreshIdentityBaselineVariablesAndSelectedNode(t *testing.T) {
	s, rt := configFixture(t)
	var source model.Resource
	s.DB.Table(model.Tables["sources"]).First(&source, 1)
	s.DB.Table(model.Tables["instances"]).Create(&model.Resource{ID: 2, Name: "second", GroupID: 10, SourceID: source.ID, ClusterID: 1, Revision: 1, Spec: source.Spec, Observed: JSON(rt.ob)})
	files := []model.File{{Path: "/etc/nginx/conf.d/site.conf", Content: "server { listen ${PORT}; }"}}
	candidate := model.ConfigSpec{InstanceIDs: []uint{2}, Baselines: map[uint]string{2: "base"}, Variables: map[uint]map[string]string{2: {"PORT": "8088"}}}
	checks, err := s.ValidateClusterConfig(context.Background(), 1, 1, files, candidate)
	if err != nil || len(checks) != 1 || checks[0].InstanceID != 2 || !checks[0].Valid {
		t.Fatalf("subset validation: %+v %v", checks, err)
	}
	if len(rt.requests) != 2 || rt.requests[1].Files[0].Content != "server { listen 8088; }" || rt.requests[1].Baseline != "base" {
		t.Fatalf("request was not frozen or rendered: %+v", rt.requests)
	}
	candidate.Baselines[2] = "stale"
	if _, err = s.ValidateClusterConfig(context.Background(), 1, 1, files, candidate); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale baseline accepted: %v", err)
	}
	rt.ob.Identity = "replaced"
	got, err := s.InstanceConfig(1, 1)
	if err == nil || got.InstanceID != 0 || !strings.Contains(err.Error(), "身份") {
		t.Fatalf("replacement read accepted: %+v %v", got, err)
	}
}

func TestInstanceConfigurationReadsRemoteAndIncludesInactiveFiles(t *testing.T) {
	s, rt := configFixture(t)
	rt.ob.Digest = "fresh"
	rt.ob.Files = append(rt.ob.Files, model.File{Path: "/etc/nginx/conf.d/inactive.conf", Content: "# current contents"})
	got, err := s.InstanceConfig(1, 1)
	if err != nil || got.Digest != "fresh" || len(got.Files) != 3 {
		t.Fatalf("stale cached data returned: %+v %v", got, err)
	}
	if len(rt.requests) != 1 || rt.requests[0].Action != "discover" {
		t.Fatal("config did not read the current target")
	}
}

func TestConfigGroupsOwnNativePathsAndSavingDraftAssignsMembership(t *testing.T) {
	s, _ := configFixture(t)
	ctx := context.Background()
	group, err := s.Save(ctx, 1, "config-groups", model.Resource{Name: "native", GroupID: 10, ClusterID: 1, Engine: "nginx", Spec: JSON(model.ConfigGroupSpec{})}, "")
	if err != nil {
		t.Fatal(err)
	}
	path := "/etc/nginx/conf.d/site.conf"
	_, err = s.Save(ctx, 1, "configurations", model.Resource{Name: "native edit", GroupID: 10, ClusterID: 1, Engine: "nginx", Spec: JSON(model.ConfigSpec{ConfigGroupID: group.ID, Files: []model.File{{Path: path, Content: "server {}"}}})}, "")
	if err != nil {
		t.Fatal(err)
	}
	group, _ = s.Get("config-groups", group.ID)
	assigned, _ := decode[model.ConfigGroupSpec](group.Spec)
	if len(assigned.Paths) != 1 || assigned.Paths[0] != path {
		t.Fatal("saved draft did not assign the native file")
	}
	duplicate := model.Resource{Name: "another", GroupID: 10, ClusterID: 1, Engine: "nginx", Spec: JSON(model.ConfigGroupSpec{Paths: []string{path}})}
	if _, err = s.Save(ctx, 1, "config-groups", duplicate, ""); err == nil || !strings.Contains(err.Error(), "其他分组") {
		t.Fatalf("duplicate file ownership accepted: %v", err)
	}
	duplicate.Spec = JSON(model.ConfigGroupSpec{Paths: []string{"/etc/passwd"}})
	if _, err = s.Save(ctx, 1, "config-groups", duplicate, ""); err == nil || !strings.Contains(err.Error(), "受管目录") {
		t.Fatalf("out-of-scope membership accepted: %v", err)
	}
}
