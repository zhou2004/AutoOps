package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	cmdb "dodevops-api/api/cmdb/model"
	system "dodevops-api/api/system/model"
	"dodevops-api/api/webservice/model"
	"dodevops-api/common/util"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRuntime struct {
	requests []RemoteRequest
	fail     bool
	ob       Observation
}

func (f *fakeRuntime) Run(ctx context.Context, r RemoteRequest) (json.RawMessage, error) {
	f.requests = append(f.requests, r)
	if f.fail && r.Action == "apply" {
		return nil, errors.New("lost remote response")
	}
	if r.Action == "validate" {
		return JSON(map[string]bool{"validated": true}), nil
	}
	if r.Action == "discover" {
		return JSON(f.ob), nil
	}
	return JSON(Observation{State: "succeeded", Observed: &f.ob}), nil
}
func fixture(t *testing.T) *Service {
	t.Helper()
	t.Setenv("WEB_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	db, e := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "web.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := db.DB()
	raw.SetMaxOpenConns(1)
	t.Cleanup(func() { raw.Close() })
	if e = model.Migrate(db); e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&cmdb.CmdbGroup{}, &cmdb.CmdbAssetPermission{}, &cmdb.CmdbUserGroup{}, &cmdb.CmdbUserGroupMember{}, &system.RolePermission{}); e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{"CREATE TABLE sys_admin (id INTEGER PRIMARY KEY,status INTEGER,username TEXT)", "CREATE TABLE sys_role (id INTEGER PRIMARY KEY,role_key TEXT,status INTEGER)", "CREATE TABLE sys_admin_role (admin_id INTEGER,role_id INTEGER)", "INSERT INTO sys_admin VALUES (1,1,'admin'),(2,1,'reader')", "INSERT INTO sys_role VALUES (1,'admin',1),(2,'reader',1)", "INSERT INTO sys_admin_role VALUES (1,1),(2,2)"} {
		if e = db.Exec(q).Error; e != nil {
			t.Fatal(e)
		}
	}
	db.Create(&system.RolePermission{RoleID: 2, Permissions: `["web:read","web:write","web:execute"]`})
	if e = db.Create(&cmdb.CmdbGroup{ID: 10, Name: "group", CreateTime: util.HTime{Time: time.Now()}}).Error; e != nil {
		t.Fatal(e)
	}
	s := New(db)
	s.Remote = &fakeRuntime{ob: Observation{Digest: "base", Identity: "nginx", Files: []model.File{{Path: "/etc/nginx/nginx.conf", Content: "events {} http {}", Mode: 0644}}}}
	src := model.Resource{ID: 1, Name: "source", GroupID: 10, Revision: 1, Spec: JSON(model.SourceSpec{Runtime: "host", ConfigPath: "/etc/nginx/nginx.conf", Binary: "/usr/sbin/nginx", Prefix: "/etc/nginx", ManagedPaths: []string{"/etc/nginx"}, HealthURL: "http://localhost"})}
	db.Table(model.Tables["sources"]).Create(&src)
	db.Table(model.Tables["clusters"]).Create(&model.Resource{ID: 1, Name: "cluster", GroupID: 10, Revision: 1, Management: "operate"})
	db.Table(model.Tables["instances"]).Create(&model.Resource{ID: 1, Name: "node", GroupID: 10, SourceID: 1, ClusterID: 1, Revision: 1, Spec: src.Spec})
	return s
}
func TestPlanConfirmationIdempotencyAndMutex(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	p, e := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "reload"})
	if e != nil || !p.Executable {
		t.Fatal(p, e)
	}
	if strings.Contains(string(p.Snapshot), "events") {
		t.Fatal("snapshot was not encrypted")
	}
	if _, e = s.Submit(ctx, 1, p.ID, p.IdempotencyKey, false); e == nil {
		t.Fatal("unconfirmed accepted")
	}
	op, e := s.Submit(ctx, 1, p.ID, p.IdempotencyKey, true)
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.Submit(ctx, 1, p.ID, p.IdempotencyKey, true)
	if e != nil || again.ID != op.ID {
		t.Fatal("idempotency", e)
	}
	p2, e := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "reload"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Submit(ctx, 1, p2.ID, p2.IdempotencyKey, true); !errors.Is(e, ErrConflict) {
		t.Fatal("lock did not exclude second operation", e)
	}
}
func TestQueueRechecksRevisionAndPermission(t *testing.T) {
	s := fixture(t)
	p, e := s.Plan(context.Background(), 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "reload"})
	if e != nil {
		t.Fatal(e)
	}
	op, e := s.Submit(context.Background(), 1, p.ID, p.IdempotencyKey, true)
	if e != nil {
		t.Fatal(e)
	}
	s.DB.Table(model.Tables["clusters"]).Where("id = 1").Update("revision", 2)
	s.execute(context.Background(), op)
	s.DB.First(&op, op.ID)
	if op.Status != "failed" {
		t.Fatal(op.Status)
	}
	for _, r := range s.Remote.(*fakeRuntime).requests {
		if r.Action == "reload" {
			t.Fatal("stale plan executed")
		}
	}
}
func TestUncertainResultPreservesLock(t *testing.T) {
	s := fixture(t)
	config := model.Resource{ID: 1, Name: "config", GroupID: 10, ClusterID: 1, Revision: 1, Spec: JSON(model.ConfigSpec{Files: []model.File{{Path: "/etc/nginx/nginx.conf", Content: "events {} http {}"}}})}
	s.DB.Table(model.Tables["configurations"]).Create(&config)
	p, e := s.Plan(context.Background(), 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "publish", Parameters: Parameters{ConfigurationID: 1}})
	if e != nil {
		t.Fatal(e)
	}
	op, e := s.Submit(context.Background(), 1, p.ID, p.IdempotencyKey, true)
	if e != nil {
		t.Fatal(e)
	}
	s.Remote.(*fakeRuntime).fail = true
	s.execute(context.Background(), op)
	s.DB.First(&op, op.ID)
	var count int64
	s.DB.Model(&model.Lock{}).Count(&count)
	if op.Status != "unknown" || count != 1 {
		t.Fatalf("uncertain result released lock: %s %d", op.Status, count)
	}
}
func TestWebScopesAndRedaction(t *testing.T) {
	s := fixture(t)
	s.DB.Create(&cmdb.CmdbAssetPermission{Name: "read source", UserIDs: "[2]", AssetTypes: `["web_source"]`, WebSourceIDs: "[1]", PermissionActions: `["web_view"]`, IsActive: 1})
	if s.Asset(2, "web_source", 1, "web_view") != nil {
		t.Fatal("grant missing")
	}
	if s.Asset(2, "web_cluster", 1, "web_manage") != ErrForbidden {
		t.Fatal("source read escalated")
	}
	r, _ := s.Get("sources", 1)
	r.Observed = JSON(s.Remote.(*fakeRuntime).ob)
	s.Redact(2, "sources", &r)
	if strings.Contains(string(r.Observed), "events") {
		t.Fatal("configuration leaked")
	}
	rows, n, e := s.List(2, "sources", Filter{Page: 1, Size: 1})
	if e != nil || n != 1 || len(rows) != 1 {
		t.Fatal(n, e)
	}
}
func TestCertificateValidationAndSealing(t *testing.T) {
	fixture(t)
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "example.com"}, DNSNames: []string{"example.com"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, e := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	keyDER, _ := x509.MarshalPKCS8PrivateKey(key)
	cert := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	private := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
	if _, e = ParseCertificate(cert, private, []string{"wrong.example"}); e == nil {
		t.Fatal("wrong SAN accepted")
	}
	if _, e = ParseCertificate(cert, private, []string{"example.com"}); e != nil {
		t.Fatal(e)
	}
	encrypted, e := seal(private)
	if e != nil {
		t.Fatal(e)
	}
	plain, e := unseal(encrypted)
	if e != nil || plain != private {
		t.Fatal("secret roundtrip failed")
	}
	raw, _ := base64.StdEncoding.DecodeString(encrypted)
	raw[len(raw)-1] ^= 1
	if _, e = unseal(base64.StdEncoding.EncodeToString(raw)); e == nil {
		t.Fatal("tampered secret accepted")
	}
}
func TestRenderRejectsDirectiveInjection(t *testing.T) {
	p := model.SiteSpec{Path: "/etc/nginx/conf.d/site.conf", Domains: []string{"example.com"}, Ports: []int{80}, Locations: []model.Location{{Path: "/", Kind: "proxy", Target: "http://localhost; include /tmp/evil"}}}
	if _, e := RenderSite(p); e == nil {
		t.Fatal("directive injection accepted")
	}
	p.Locations[0].Target = "http://127.0.0.1:8080"
	p.Locations[0].Websocket = true
	rendered, e := RenderSite(p)
	if e != nil || !strings.Contains(rendered, "proxy_http_version 1.1") {
		t.Fatal(e)
	}
}

func TestRenewalResumesAfterManualDeploymentRecovery(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	cert := model.Resource{ID: 8, Name: "certificate", GroupID: 10, Spec: JSON(model.CertificateSpec{CurrentVersionID: 12})}
	if e := s.DB.Table(model.Tables["certificates"]).Create(&cert).Error; e != nil {
		t.Fatal(e)
	}
	binding := model.Resource{ID: 5, Name: "binding", GroupID: 10, ClusterID: 1, Spec: JSON(model.BindingSpec{CertificateID: 8})}
	if e := s.DB.Table(model.Tables["certificate-bindings"]).Create(&binding).Error; e != nil {
		t.Fatal(e)
	}
	parent := model.Operation{ID: 20, PlanID: 20, Status: "succeeded", IdempotencyKey: "issue-20"}
	child := model.Operation{ID: 21, PlanID: 21, Status: "failed", IdempotencyKey: "deploy-21"}
	s.DB.Create(&parent)
	s.DB.Create(&child)
	p := model.RenewalSpec{Enabled: true, CertificateID: 8, AutoDeploy: true, PendingOperationID: 20, BindingIDs: []uint{5}, DeployOperationIDs: map[uint]uint{5: 21}}
	policy := model.Resource{ID: 4, Name: "renew", GroupID: 10, CreatedBy: 1, Revision: 1, Spec: JSON(p)}
	if e := s.DB.Table(model.Tables["renewal-policies"]).Create(&policy).Error; e != nil {
		t.Fatal(e)
	}
	s.runRenewals(ctx)
	stored, _ := s.Get("renewal-policies", 4)
	stillPending, _ := decode[model.RenewalSpec](stored.Spec)
	if stillPending.PendingOperationID != 20 {
		t.Fatal("failed deployment was incorrectly completed")
	}
	// A manual retry has now verified the issued version on the only cluster member.
	s.DB.Table(model.Tables["certificate-bindings"]).Where("id = ?", 5).Update("observed", JSON(map[string]any{"nodes": map[string]any{"1": map[string]any{"versionId": 12}}}))
	s.DB.Where("`key` = ?", "renewal-scheduler").Delete(&model.Lease{})
	s.runRenewals(ctx)
	stored, _ = s.Get("renewal-policies", 4)
	recovered, _ := decode[model.RenewalSpec](stored.Spec)
	if recovered.PendingOperationID != 0 || recovered.DeployedVersionID != 12 || len(recovered.DeployOperationIDs) != 0 {
		t.Fatalf("manual recovery not recognized: %+v", recovered)
	}
	if s.bindingStatus(1, map[string]any{"1": map[string]any{"versionId": 0}}, 0) == "deployed" {
		t.Fatal("an unissued certificate was treated as deployed")
	}
}

func TestRecoveryRestoresExpectedDigestAndCertificateState(t *testing.T) {
	s := fixture(t)
	binding := model.Resource{ID: 5, Name: "binding", GroupID: 10, ClusterID: 1, Observed: JSON(map[string]any{"nodes": map[string]any{"1": map[string]any{"versionId": 12}}})}
	s.DB.Table(model.Tables["certificate-bindings"]).Create(&binding)
	target := Target{Instance: model.Resource{ID: 1}, PreviousCertificates: map[uint]json.RawMessage{5: JSON(map[string]any{"versionId": 11})}}
	if e := s.recordRestoredState(target, &Observation{Digest: "restored-baseline"}); e != nil {
		t.Fatal(e)
	}
	i, _ := s.Get("instances", 1)
	ob, _ := decode[Observation](i.Observed)
	if ob.ExpectedDigest != "restored-baseline" || i.Status != "healthy" {
		t.Fatal("restored node retained the candidate state")
	}
	b, _ := s.Get("certificate-bindings", 5)
	if strings.Contains(string(b.Observed), `"versionId":12`) || !strings.Contains(string(b.Observed), `"versionId":11`) {
		t.Fatal("restored certificate version not recorded")
	}
	if e := s.recordRestoredState(target, nil); e == nil {
		t.Fatal("recovery accepted missing observation")
	}
}

func TestRenewalSubmissionCommitsPolicyAndTaskTogether(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	p := model.RenewalSpec{Enabled: true, CertificateID: 8}
	policy := model.Resource{ID: 4, Name: "renew", GroupID: 10, CreatedBy: 1, Revision: 1, Spec: JSON(p)}
	s.DB.Table(model.Tables["renewal-policies"]).Create(&policy)
	plan, e := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "reload"})
	if e != nil {
		t.Fatal(e)
	}
	// Editing a policy while preflight runs must roll back both task and mutex.
	s.DB.Table(model.Tables["renewal-policies"]).Where("id = 4").Update("revision", 2)
	if e = s.submitRenewal(ctx, &policy, &p, plan, 5); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	var tasks, locks int64
	s.DB.Model(&model.Operation{}).Count(&tasks)
	s.DB.Model(&model.Lock{}).Count(&locks)
	if tasks != 0 || locks != 0 {
		t.Fatal("orphan renewal task or mutex after policy conflict")
	}
	policy.Revision = 2
	if e = s.submitRenewal(ctx, &policy, &p, plan, 5); e != nil {
		t.Fatal(e)
	}
	stored, _ := s.Get("renewal-policies", 4)
	saved, _ := decode[model.RenewalSpec](stored.Spec)
	var op model.Operation
	if saved.DeployOperationIDs[5] == 0 || s.DB.First(&op, saved.DeployOperationIDs[5]).Error != nil {
		t.Fatal("durable child reference missing")
	}
}

type rollingRuntime struct {
	nodes    map[uint]Observation
	backups  map[uint]Observation
	calls    []RemoteRequest
	failNode uint
	drift    bool
}

func (f *rollingRuntime) Run(ctx context.Context, r RemoteRequest) (json.RawMessage, error) {
	f.calls = append(f.calls, r)
	node := f.nodes[r.Spec.HostID]
	switch r.Action {
	case "discover":
		return JSON(node), nil
	case "validate":
		return JSON(map[string]bool{"validated": true}), nil
	case "apply":
		if r.Baseline != node.Digest {
			return nil, ErrConflict
		}
		if r.Spec.HostID == f.failNode {
			if f.drift {
				n := f.nodes[1]
				n.Digest = "human-change"
				f.nodes[1] = n
			}
			return JSON(Observation{State: "rolled-back", Observed: &node}), nil
		}
		f.backups[r.Spec.HostID] = node
		node.Digest = "candidate"
		node.Files = r.Files
		f.nodes[r.Spec.HostID] = node
		return JSON(Observation{State: "succeeded", Observed: &node}), nil
	case "restore":
		if r.Baseline != node.Digest {
			return nil, ErrConflict
		}
		node = f.backups[r.Spec.HostID]
		f.nodes[r.Spec.HostID] = node
		return JSON(Observation{State: "rolled-back", Observed: &node}), nil
	}
	return JSON(Observation{State: "succeeded", Observed: &node}), nil
}
func rollingFixture(t *testing.T) (*Service, *rollingRuntime) {
	s := fixture(t)
	f := &rollingRuntime{nodes: map[uint]Observation{}, backups: map[uint]Observation{}, failNode: 2}
	s.Remote = f
	src, _ := s.Get("sources", 1)
	spec, _ := decode[model.SourceSpec](src.Spec)
	spec.HostID = 1
	s.DB.Table(model.Tables["sources"]).Where("id = 1").Update("spec", JSON(spec))
	spec.HostID = 2
	src.ID = 2
	src.Name = "node2"
	src.Spec = JSON(spec)
	s.DB.Table(model.Tables["sources"]).Create(&src)
	s.DB.Table(model.Tables["instances"]).Create(&model.Resource{ID: 2, Name: "node2", SourceID: 2, GroupID: 10, ClusterID: 1, Revision: 1})
	for id := uint(1); id <= 2; id++ {
		f.nodes[id] = Observation{Identity: fmt.Sprint(id), Digest: fmt.Sprint("base-", id), Files: []model.File{{Path: "/etc/nginx/nginx.conf", Content: fmt.Sprint("# node ", id), Mode: 0644}}}
	}
	s.DB.Table(model.Tables["configurations"]).Create(&model.Resource{ID: 1, Name: "candidate", GroupID: 10, ClusterID: 1, Revision: 1, Spec: JSON(model.ConfigSpec{Files: []model.File{{Path: "/etc/nginx/nginx.conf", Content: "events {} http {}", Mode: 0644}}})})
	return s, f
}
func TestRollingFailureRestoresPriorNodeWithoutAcceptingDrift(t *testing.T) {
	for _, drift := range []bool{false, true} {
		t.Run(fmt.Sprint(drift), func(t *testing.T) {
			s, f := rollingFixture(t)
			f.drift = drift
			p, e := s.Plan(context.Background(), 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "publish", Parameters: Parameters{ConfigurationID: 1}})
			if e != nil || !p.Executable {
				t.Fatal(e, p.Checks)
			}
			op, e := s.Submit(context.Background(), 1, p.ID, p.IdempotencyKey, true)
			if e != nil {
				t.Fatal(e)
			}
			s.execute(context.Background(), op)
			s.DB.First(&op, op.ID)
			expected := "rolled-back"
			if drift {
				expected = "unknown"
			}
			if op.Status != expected {
				t.Fatal(op.Status, op.Message)
			}
			for _, call := range f.calls {
				if call.Action == "restore" && call.Baseline != "candidate" {
					t.Fatal("recovery accepted a new baseline")
				}
			}
			var locked int64
			s.DB.Model(&model.Lock{}).Count(&locked)
			if (locked > 0) != drift {
				t.Fatal("incorrect recovery lock state")
			}
			var revisions []model.Revision
			s.DB.Where("kind = ?", "configuration").Find(&revisions)
			if len(revisions) == 0 {
				t.Fatal("missing baseline revision")
			}
			bundle, e := decode[model.ConfigSpec](revisions[0].Content)
			if e != nil || len(bundle.Nodes) != 2 || bundle.Nodes[1][0].Content == bundle.Nodes[2][0].Content {
				t.Fatal("node versions collapsed")
			}
		})
	}
}
func TestQueuedOperationRechecksDisabledAccount(t *testing.T) {
	s := fixture(t)
	p, e := s.Plan(context.Background(), 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "reload"})
	if e != nil {
		t.Fatal(e)
	}
	op, e := s.Submit(context.Background(), 1, p.ID, p.IdempotencyKey, true)
	if e != nil {
		t.Fatal(e)
	}
	s.DB.Exec("UPDATE sys_admin SET status = 0 WHERE id = 1")
	s.execute(context.Background(), op)
	s.DB.First(&op, op.ID)
	if op.Status != "failed" {
		t.Fatal(op.Status)
	}
	for _, r := range s.Remote.(*fakeRuntime).requests {
		if r.Action == "reload" {
			t.Fatal("disabled user executed remote operation")
		}
	}
}
