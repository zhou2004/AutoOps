package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/api/webservice/model"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type issuanceTransport func(*http.Request) (*http.Response, error)

func (f issuanceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func issuanceFixture(t *testing.T) (*Service, model.Resource, *int) {
	t.Helper()
	s := fixture(t)
	if err := s.DB.AutoMigrate(&configmodel.KeyManage{}); err != nil {
		t.Fatal(err)
	}
	secret, err := seal(`{"token":"isolated-test-token"}`)
	if err != nil {
		t.Fatal(err)
	}
	account := model.Resource{ID: 1, Name: "test CA", GroupID: 10, Revision: 1,
		Spec: JSON(model.AccountSpec{Email: "ops@example.test", DirectoryURL: "https://ca.example.test/directory"})}
	dns := model.Resource{ID: 1, Name: "test DNS", GroupID: 10, Revision: 1, Secret: secret,
		Spec: JSON(model.AccountSpec{Provider: "cloudflare"})}
	certificate := model.Resource{ID: 1, Name: "test certificate", GroupID: 10, Revision: 1,
		Spec: JSON(model.CertificateSpec{Owner: "platform", Domains: []string{"example.test", "*.example.test"}, Challenge: "dns-01", ACMEAccountID: 1, DNSAccountID: 1})}
	for kind, resource := range map[string]model.Resource{"acme-accounts": account, "dns-accounts": dns, "certificates": certificate} {
		if err := s.DB.Table(model.Tables[kind]).Create(&resource).Error; err != nil {
			t.Fatal(err)
		}
	}
	calls := new(int)
	s.CloudflareHTTP = &http.Client{Transport: issuanceTransport(func(request *http.Request) (*http.Response, error) {
		*calls++
		if request.Method != http.MethodGet || request.URL.Host != "api.cloudflare.com" {
			t.Fatal("preflight attempted a non-readonly Cloudflare request")
		}
		body := `{"success":true,"result":[]}`
		if request.URL.Path == "/client/v4/zones" {
			body = `{"success":true,"result":[{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","name":"example.test","status":"active"}],"result_info":{"page":1,"total_pages":1,"total_count":1}}`
		} else if !strings.HasSuffix(request.URL.Path, "/dns_records") {
			t.Fatal("unexpected Cloudflare preflight endpoint", request.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	return s, certificate, calls
}

func certificatePlan(t *testing.T, s *Service, user uint) model.Plan {
	t.Helper()
	plan, err := s.Plan(context.Background(), user, PlanInput{TargetType: "certificates", TargetID: 1, Action: "issue"})
	if err != nil || !plan.Executable {
		t.Fatal("certificate plan failed", err, string(plan.Checks))
	}
	return plan
}

func TestIssuancePlanFreezesReferencedAccountsAndCredentialMaterial(t *testing.T) {
	for _, test := range []string{"acme-spec", "acme-key", "dns-spec", "dns-key", "referenced-credential"} {
		t.Run(test, func(t *testing.T) {
			s, _, _ := issuanceFixture(t)
			if test == "referenced-credential" {
				credential := configmodel.KeyManage{ID: 8, KeyType: 6, KeySecret: "referenced-test-token"}
				if err := credential.EncryptKeys(); err != nil {
					t.Fatal(err)
				}
				if err := s.DB.Create(&credential).Error; err != nil {
					t.Fatal(err)
				}
				s.DB.Table(model.Tables["dns-accounts"]).Where("id = 1").Updates(map[string]any{"spec": JSON(model.AccountSpec{Provider: "cloudflare", CredentialID: 8}), "secret": ""})
			}
			plan := certificatePlan(t, s, 1)
			if strings.Contains(string(plan.Snapshot), "test-token") || strings.Contains(string(plan.Snapshot), "ops@example.test") {
				t.Fatal("signing material escaped the digest")
			}
			switch test {
			case "acme-spec":
				s.DB.Table(model.Tables["acme-accounts"]).Where("id = 1").Update("spec", JSON(model.AccountSpec{Email: "changed@example.test", DirectoryURL: "https://ca.example.test/directory"}))
			case "acme-key":
				s.DB.Table(model.Tables["acme-accounts"]).Where("id = 1").Update("secret", "changed-encrypted-account-key")
			case "dns-spec":
				s.DB.Table(model.Tables["dns-accounts"]).Where("id = 1").Update("spec", JSON(model.AccountSpec{Provider: "alidns"}))
			case "dns-key":
				secret, _ := seal(`{"token":"changed-token"}`)
				s.DB.Table(model.Tables["dns-accounts"]).Where("id = 1").Update("secret", secret)
			case "referenced-credential":
				secret, _ := seal("changed-token")
				s.DB.Model(&configmodel.KeyManage{}).Where("id = 8").Update("key_secret", secret)
			}
			if _, err := s.Submit(context.Background(), 1, plan.ID, plan.IdempotencyKey, true); !errors.Is(err, ErrConflict) {
				t.Fatal("changed signing dependency was accepted", err)
			}
			var count int64
			s.DB.Model(&model.Operation{}).Count(&count)
			if count != 0 {
				t.Fatal("a stale signing operation was queued")
			}
		})
	}
}

func TestIssuanceWorkerRechecksSigningDependenciesBeforeExternalRequests(t *testing.T) {
	s, _, calls := issuanceFixture(t)
	plan := certificatePlan(t, s, 1)
	op, err := s.Submit(context.Background(), 1, plan.ID, plan.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	*calls = 0
	s.DB.Table(model.Tables["dns-accounts"]).Where("id = 1").Update("revision", 2)
	s.execute(context.Background(), op)
	s.DB.First(&op, op.ID)
	if op.Status != "failed" || *calls != 0 {
		t.Fatalf("worker used changed account: status=%s calls=%d", op.Status, *calls)
	}
	var orders int64
	s.DB.Model(&model.Order{}).Count(&orders)
	if orders != 0 {
		t.Fatal("a stale signing job created a CA order")
	}
}

func TestIssuancePreflightDetectsCredentialEditDuringCloudflareRead(t *testing.T) {
	s, _, _ := issuanceFixture(t)
	original := s.CloudflareHTTP.Transport
	s.CloudflareHTTP.Transport = issuanceTransport(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/client/v4/zones" {
			s.DB.Table(model.Tables["dns-accounts"]).Where("id = 1").Update("revision", 2)
		}
		return original.RoundTrip(request)
	})
	plan, err := s.Plan(context.Background(), 1, PlanInput{TargetType: "certificates", TargetID: 1, Action: "issue"})
	if err != nil || plan.Executable || !strings.Contains(string(plan.Checks), ErrConflict.Error()) {
		t.Fatal("concurrent account update passed preflight", err, string(plan.Checks))
	}
}

func TestCertificateManagerCanUseAdministratorBoundSigningAccount(t *testing.T) {
	s, _, _ := issuanceFixture(t)
	grant := cmdb.CmdbAssetPermission{Name: "certificate operator", UserIDs: "[2]", AssetTypes: `["web_certificate"]`,
		HostGroupIDs: "[10]", PermissionActions: `["web_view","web_certificate_manage"]`, IsActive: 1}
	if err := s.DB.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
	plan := certificatePlan(t, s, 2)
	if _, err := s.Submit(context.Background(), 2, plan.ID, plan.IdempotencyKey, true); err != nil {
		t.Fatal("bound account could not be used by a certificate manager", err)
	}
	if _, err := s.DNSAccountZones(context.Background(), 2, 1); !errors.Is(err, ErrForbidden) {
		t.Fatal("certificate manager gained DNS account management")
	}
}

func TestIssuancePreflightRejectsMalformedIdentityAndDomainBeforeNetwork(t *testing.T) {
	for _, name := range []string{"empty-domains", "invalid-domain", "duplicate-domain", "bad-email", "missing-token", "outside-zone"} {
		t.Run(name, func(t *testing.T) {
			s, cert, calls := issuanceFixture(t)
			spec, _ := decode[model.CertificateSpec](cert.Spec)
			switch name {
			case "empty-domains":
				spec.Domains = nil
			case "invalid-domain":
				spec.Domains = []string{"bad..example.test"}
			case "duplicate-domain":
				spec.Domains = []string{"example.test", "EXAMPLE.TEST"}
			case "bad-email":
				s.DB.Table(model.Tables["acme-accounts"]).Where("id = 1").Update("spec", JSON(model.AccountSpec{Email: "not an email", DirectoryURL: "https://ca.example.test/directory"}))
			case "missing-token":
				s.DB.Table(model.Tables["dns-accounts"]).Where("id = 1").Update("secret", "")
			case "outside-zone":
				spec.Domains = []string{"notexample.test"}
			}
			s.DB.Table(model.Tables["certificates"]).Where("id = 1").Update("spec", JSON(spec))
			plan, err := s.Plan(context.Background(), 1, PlanInput{TargetType: "certificates", TargetID: 1, Action: "issue"})
			if err != nil || plan.Executable {
				t.Fatal("invalid signing identity passed preflight", err)
			}
			if name != "outside-zone" && *calls != 0 {
				t.Fatal("invalid local inputs contacted Cloudflare")
			}
		})
	}
}

func TestOrderIdentityBindsCAAccountKeyAndDomainSet(t *testing.T) {
	s, _, _ := issuanceFixture(t)
	account, _ := s.Get("acme-accounts", 1)
	account.Secret = "encrypted-account-key"
	identity, err := issuanceOrderIdentity(account, []string{"example.test", "*.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	order := model.Order{IdentityDigest: identity}
	canonical, err := issuanceOrderIdentity(account, []string{"*.EXAMPLE.TEST", "EXAMPLE.TEST"})
	if err != nil || checkOrderIdentity(order, canonical) != nil {
		t.Fatal("domain ordering and case changed the order identity")
	}
	for _, field := range []string{"id", "directory", "email", "key", "domains"} {
		changed := account
		domains := []string{"example.test", "*.example.test"}
		a, _ := decode[model.AccountSpec](changed.Spec)
		switch field {
		case "id":
			changed.ID = 2
		case "directory":
			a.DirectoryURL = "https://other-ca.test/directory"
		case "email":
			a.Email = "other@example.test"
		case "key":
			changed.Secret = "another-encrypted-account-key"
		case "domains":
			domains = []string{"other.example.test"}
		}
		changed.Spec = JSON(a)
		digest, err := issuanceOrderIdentity(changed, domains)
		if err != nil || checkOrderIdentity(order, digest) == nil {
			t.Fatal("changed order identity accepted", field, err)
		}
	}
	// Replacing a DNS token does not change the ACME order identity.
	s.DB.Table(model.Tables["dns-accounts"]).Where("id = 1").Update("secret", "replacement DNS material")
	unchanged, _ := issuanceOrderIdentity(account, []string{"example.test", "*.example.test"})
	if checkOrderIdentity(order, unchanged) != nil {
		t.Fatal("DNS repair prevented recovery of the same CA order")
	}
	if checkOrderIdentity(model.Order{}, identity) == nil {
		t.Fatal("legacy unbound order accepted")
	}
}

func TestLegacyUnboundOrderIsRejectedBeforeAccountRegistration(t *testing.T) {
	s, certificate, _ := issuanceFixture(t)
	plan := certificatePlan(t, s, 1)
	operation, err := s.Submit(context.Background(), 1, plan.ID, plan.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	order := model.Order{CertificateID: 1, Status: "pending", URL: "https://previous-ca.test/order/1"}
	if err = s.DB.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	err = s.issue(context.Background(), operation, certificate)
	if err == nil || !strings.Contains(err.Error(), "缺少签发身份记录") {
		t.Fatal("legacy order reached registration or resumed", err)
	}
	account, _ := s.Get("acme-accounts", 1)
	if account.Secret != "" {
		t.Fatal("legacy order failure changed the ACME account key")
	}
}

func TestIssuedCertificateAndOrderCompletionAreAtomic(t *testing.T) {
	s, certificate, _ := issuanceFixture(t)
	chain, key := certificateAssetPEM(t, nil)
	order := model.Order{CertificateID: certificate.ID, Status: "pending", IdentityDigest: "bound-order-identity"}
	if err := s.DB.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.storeIssuedCertificate(1, certificate, order, chain, key); !errors.Is(err, ErrConflict) {
		t.Fatal("non-final order stored a certificate", err)
	}
	var count int64
	s.DB.Model(&model.Revision{}).Where("kind = ?", "certificate").Count(&count)
	if count != 0 {
		t.Fatal("failed order completion left an orphan certificate version")
	}
	s.DB.Model(&order).Update("status", "valid")
	version, err := s.storeIssuedCertificate(1, certificate, order, chain, key)
	if err != nil {
		t.Fatal(err)
	}
	s.DB.First(&order, order.ID)
	updated, _ := s.Get("certificates", certificate.ID)
	spec, _ := decode[model.CertificateSpec](updated.Spec)
	if order.Status != "stored" || spec.CurrentVersionID != version.ID {
		t.Fatal("order and issued version did not commit together")
	}
}
