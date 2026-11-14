package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	cmdb "dodevops-api/api/cmdb/model"
	cmdbservice "dodevops-api/api/cmdb/service"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/api/webservice/model"
	"dodevops-api/common/config"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

// This test requires explicit host authorization. Production CMDB is read-only;
// Web records and copied connection credentials are confined to t.TempDir().
func TestLiveSSHClusterLifecycle(t *testing.T) {
	if os.Getenv("AUTOOPS_WEB_SSH_TEST") != "1" {
		t.Skip("requires explicitly authorized SSH hosts")
	}
	hosts := strings.Split(os.Getenv("AUTOOPS_WEB_SSH_HOSTS"), ",")
	if len(hosts) < 2 || hosts[0] == "" {
		t.Fatal("provide at least two authorized CMDB addresses")
	}
	if e := config.LoadConfig(os.Getenv("AUTOOPS_WEB_CONFIG")); e != nil {
		t.Fatal(e)
	}
	d := config.Config.Db
	production, e := gorm.Open(mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=True&timeout=5s", d.Username, d.Password, d.Host, d.Port, d.Db)), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal("CMDB metadata unavailable")
	}
	connection, _ := production.DB()
	defer connection.Close()
	s := fixture(t)
	ctx := context.Background()
	s.Remote = &SSHRuntime{DB: s.DB}
	for _, kind := range []string{"instances", "sources", "clusters"} {
		if e := s.DB.Exec("DELETE FROM " + model.Tables[kind]).Error; e != nil {
			t.Fatal(e)
		}
	}
	if e = s.DB.AutoMigrate(&cmdb.CmdbHost{}, &cmdb.CmdbAssetConnection{}, &configmodel.EcsAuth{}); e != nil {
		t.Fatal(e)
	}
	cluster, e := s.Save(ctx, 1, "clusters", model.Resource{Name: "isolated-live-cluster", GroupID: 10, Spec: JSON(model.ClusterSpec{})}, "")
	if e != nil {
		t.Fatal(e)
	}
	root := fmt.Sprintf("/tmp/autoops-web-live-%d", time.Now().UnixNano())
	version := os.Getenv("AUTOOPS_WEB_NGINX_PACKAGE")
	if version == "" {
		t.Fatal("pin AUTOOPS_WEB_NGINX_PACKAGE")
	}
	fixtureScript, e := os.ReadFile("../../../automation/web-platform/ssh_fixture.py")
	if e != nil {
		t.Fatal(e)
	}
	var instances, sources []model.Resource
	for _, address := range hosts {
		var host cmdb.CmdbHost
		if e = production.Where("private_ip = ?", address).First(&host).Error; e != nil {
			t.Fatal(address, e)
		}
		var profile cmdb.CmdbAssetConnection
		if e = production.Where("asset_type = ? AND asset_id = ? AND protocol = ? AND enabled = ?", "host", host.ID, "ssh", true).First(&profile).Error; e != nil {
			t.Fatal(address, e)
		}
		var credential configmodel.EcsAuth
		if e = production.First(&credential, profile.CredentialID).Error; e != nil {
			t.Fatal("connection credential unavailable")
		}
		if s.DB.Where("id = ?", credential.ID).First(&configmodel.EcsAuth{}).Error != nil {
			if e = s.DB.Create(&credential).Error; e != nil {
				t.Fatal(e)
			}
		}
		if e = s.DB.Omit("Group").Create(&host).Error; e != nil {
			t.Fatal(e)
		}
		if e = s.DB.Create(&profile).Error; e != nil {
			t.Fatal(e)
		}
		remote := func(script, command string) ([]byte, error) {
			client, err := cmdbservice.ResolveAssetSSH(s.DB, "host", host.ID, profile.ID)
			if err != nil {
				return nil, err
			}
			defer client.Close()
			session, err := client.NewSession()
			if err != nil {
				return nil, err
			}
			defer session.Close()
			session.Stdin = strings.NewReader(script)
			return session.CombinedOutput(command)
		}
		t.Cleanup(func() {
			out, err := remote(string(fixtureScript), "python3 - "+quote(root)+fmt.Sprintf(" cleanup %d", host.ID))
			if err != nil {
				t.Errorf("%s cleanup: %s %v", address, out, err)
			} else {
				t.Logf("%s isolated fixture cleaned", address)
			}
		})
		output, err := remote(string(fixtureScript), "python3 - "+quote(root)+" setup "+quote(version))
		if err != nil {
			t.Fatalf("%s setup: %s %v", address, output, err)
		}
		spec := model.SourceSpec{HostID: host.ID, ConnectionID: profile.ID, Runtime: "host", Binary: root + "/payload/usr/sbin/nginx", ConfigPath: root + "/nginx.conf", Prefix: root, ManagedPaths: []string{root}, HealthURL: "http://127.0.0.1:28080/", ExpectedStatus: 200}
		source, err := s.Save(ctx, 1, "sources", model.Resource{Name: address, GroupID: 10, Spec: JSON(spec)}, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.Discover(ctx, 1, source.ID); err != nil {
			t.Fatal(err)
		}
		var instance model.Resource
		if err = s.DB.Table(model.Tables["instances"]).Where("source_id = ?", source.ID).First(&instance).Error; err != nil {
			t.Fatal(err)
		}
		if instance.Management != "observe" {
			t.Fatal("discovery adopted service without approval")
		}
		if err = s.BindMember(ctx, 1, cluster.ID, instance.ID, false); err != nil {
			t.Fatal(err)
		}
		sources = append(sources, source)
		instances = append(instances, instance)
		client, err := cmdbservice.ResolveAssetSSH(s.DB, "host", host.ID, profile.ID)
		if err != nil {
			t.Fatal(err)
		}
		session, _ := client.NewSession()
		if err = session.RequestPty("xterm", 24, 80, ssh.TerminalModes{}); err != nil {
			t.Fatal(err)
		}
		output, err = session.CombinedOutput("printf autoops-pty-ok")
		session.Close()
		client.Close()
		if err != nil || !bytes.Contains(output, []byte("autoops-pty-ok")) {
			t.Fatal("SSH PTY failed", err)
		}
		t.Logf("%s: instance %d via CMDB connection %d; SSH PTY verified", address, instance.ID, profile.ID)
	}
	operate := func(action string, params Parameters, expected string) {
		t.Helper()
		p, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: cluster.ID, Action: action, Parameters: params})
		if err != nil || !p.Executable {
			t.Fatalf("%s preflight: %v %s", action, err, p.Checks)
		}
		op, err := s.Submit(ctx, 1, p.ID, p.IdempotencyKey, true)
		if err != nil {
			t.Fatal(err)
		}
		s.DB.Model(&op).Update("status", "preflight")
		s.execute(ctx, op)
		s.DB.First(&op, op.ID)
		if op.Status != expected {
			t.Fatalf("%s: %s %s", action, op.Status, op.Message)
		}
		t.Logf("operation %d %s -> %s", op.ID, action, op.Status)
	}
	operate("adopt", Parameters{}, "succeeded")
	siteSpec := model.SiteSpec{Domains: []string{"web-test.invalid"}, Ports: []int{28082}, Mode: "structured", Path: root + "/sites/app.conf", HealthURL: "http://127.0.0.1:28082/", ExpectedStatus: 200, Locations: []model.Location{{Path: "/", Kind: "proxy", Target: "http://127.0.0.1:28081", Timeout: 10}}}
	site, e := s.Save(ctx, 1, "sites", model.Resource{Name: "reverse-proxy", GroupID: 10, ClusterID: cluster.ID, Spec: JSON(siteSpec)}, "")
	if e != nil {
		t.Fatal(e)
	}
	rendered, e := RenderSite(siteSpec)
	if e != nil {
		t.Fatal(e)
	}
	draft, e := s.Save(ctx, 1, "configurations", model.Resource{Name: "reverse-proxy", GroupID: 10, ClusterID: cluster.ID, Spec: JSON(model.ConfigSpec{Files: []model.File{{Path: siteSpec.Path, Content: rendered, Mode: 0644}}})}, "")
	if e != nil {
		t.Fatal(e)
	}
	operate("publish", Parameters{ConfigurationID: draft.ID}, "succeeded")
	certificate, e := s.Save(ctx, 1, "certificates", model.Resource{Name: "isolated-cert", GroupID: 10, Spec: JSON(model.CertificateSpec{Owner: "external", Domains: []string{"web-test.invalid"}})}, "")
	if e != nil {
		t.Fatal(e)
	}
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	now := time.Now()
	template := &x509.Certificate{SerialNumber: big.NewInt(now.UnixNano()), Subject: pkix.Name{CommonName: "web-test.invalid"}, DNSNames: []string{"web-test.invalid"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, _ := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	keyDER, _ := x509.MarshalPKCS8PrivateKey(key)
	if _, e = s.ImportCertificate(ctx, 1, certificate.ID, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))); e != nil {
		t.Fatal(e)
	}
	binding, e := s.Save(ctx, 1, "certificate-bindings", model.Resource{Name: "isolated-binding", GroupID: 10, ClusterID: cluster.ID, Spec: JSON(model.BindingSpec{CertificateID: certificate.ID, CertificatePath: root + "/certs/fullchain.pem", KeyPath: root + "/certs/key.pem", VerifyHost: "127.0.0.1", VerifyPort: 28443, ServerName: "web-test.invalid"})}, "")
	if e != nil {
		t.Fatal(e)
	}
	siteSpec.TLS = true
	siteSpec.CertificateID = certificate.ID
	siteSpec.Ports = []int{28443}
	siteSpec.HealthURL = ""
	site.Spec = JSON(siteSpec)
	site, e = s.Save(ctx, 1, "sites", site, "")
	if e != nil {
		t.Fatal(e)
	}
	siteSpec, _ = decode[model.SiteSpec](site.Spec)
	rendered, _ = RenderSite(siteSpec)
	draft, _ = s.Get("configurations", draft.ID)
	draft.Spec = JSON(model.ConfigSpec{Files: []model.File{{Path: siteSpec.Path, Content: rendered, Mode: 0644}}})
	draft, e = s.Save(ctx, 1, "configurations", draft, "")
	if e != nil {
		t.Fatal(e)
	}
	operate("publish", Parameters{ConfigurationID: draft.ID}, "succeeded")
	binding, _ = s.Get("certificate-bindings", binding.ID)
	if binding.Status != "deployed" {
		t.Fatal("TLS was not verified on every node", binding.Status)
	}
	for _, source := range sources {
		spec, _ := decode[model.SourceSpec](source.Spec)
		ob, err := (Nginx{}).Discover(ctx, s.Remote, spec)
		if err != nil {
			t.Fatal(err)
		}
		if len(ob.Logs) == 0 {
			t.Fatal("no remote logs discovered")
		}
		log, err := s.Remote.Run(ctx, RemoteRequest{Action: "logs", Spec: spec, LogID: ob.Logs[0]["id"], Logs: ob.Logs, Lines: 100})
		if err != nil || len(log) == 0 {
			t.Fatal("remote log query failed", err)
		}
	}
	first, _ := s.Get("instances", instances[0].ID)
	observed, _ := decode[Observation](first.Observed)
	var main string
	for _, f := range observed.Files {
		if f.Path == root+"/nginx.conf" {
			main = f.Content
		}
	}
	failing := strings.Replace(main, "return 200", "return ${HEALTH_STATUS}", 1)
	variables := map[uint]map[string]string{}
	for idx, i := range instances {
		status := "200"
		if idx == 1 {
			status = "503"
		}
		variables[i.ID] = map[string]string{"HEALTH_STATUS": status}
	}
	draft, _ = s.Get("configurations", draft.ID)
	draft.Spec = JSON(model.ConfigSpec{Files: []model.File{{Path: root + "/nginx.conf", Content: failing, Mode: 0644}}, Variables: variables})
	draft, e = s.Save(ctx, 1, "configurations", draft, "")
	if e != nil {
		t.Fatal(e)
	}
	operate("publish", Parameters{ConfigurationID: draft.ID}, "rolled-back")
	for _, source := range sources {
		spec, _ := decode[model.SourceSpec](source.Spec)
		if _, e = s.Remote.Run(ctx, RemoteRequest{Action: "health", Spec: spec}); e != nil {
			t.Fatal("rollback health check failed", e)
		}
	}
	var audits int64
	s.DB.Model(&model.Step{}).Count(&audits)
	if audits < 8 {
		t.Fatal("missing audit evidence")
	}
	report := map[string]any{"hosts": hosts, "nodes": len(instances), "checks": []string{"CMDB pinned SSH", "observe/adopt", "three-node reverse proxy", "PEM import", "TLS fingerprints", "remote logs", "SSH PTY", "second-node health failure and rollback"}, "auditSteps": audits, "testedAt": time.Now().UTC()}
	if output := os.Getenv("AUTOOPS_WEB_TEST_REPORT"); output != "" {
		b, _ := json.MarshalIndent(report, "", "  ")
		os.WriteFile(output, b, 0600)
	}
}
