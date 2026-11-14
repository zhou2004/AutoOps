package service

import (
	"context"
	configmodel "dodevops-api/api/configcenter/model"
	system "dodevops-api/api/system/model"
	"dodevops-api/api/webservice/model"
	"dodevops-api/pkg/apikey"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const testCloudflareToken = "test-cloudflare-token-only"
const testZoneID = "0123456789abcdef0123456789abcdef"

type dnsRoundTripper func(*http.Request) (*http.Response, error)

func (f dnsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func dnsResponse(code int, data any) *http.Response {
	return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(JSON(data))))}
}
func dnsFixture(t *testing.T) (*Service, model.Resource) {
	t.Helper()
	s := fixture(t)
	r, e := s.Save(context.Background(), 1, "dns-accounts", model.Resource{Name: "Cloudflare", GroupID: 10, Spec: JSON(model.AccountSpec{Provider: "cloudflare"})}, string(JSON(dnsCredential{Token: testCloudflareToken})))
	if e != nil {
		t.Fatal(e)
	}
	return s, r
}
func dnsMock(s *Service, handler func(*http.Request) (*http.Response, error)) {
	s.CloudflareHTTP = &http.Client{Transport: dnsRoundTripper(handler)}
}
func goodCloudflare(t *testing.T, s *Service, hook func(*http.Request)) *int {
	t.Helper()
	calls := new(int)
	dnsMock(s, func(r *http.Request) (*http.Response, error) {
		*calls++
		if r.Method != http.MethodGet || r.URL.Scheme != "https" || r.URL.Host != "api.cloudflare.com" || r.Header.Get("Authorization") != "Bearer "+testCloudflareToken {
			t.Error("unexpected destination, method or credential")
		}
		if hook != nil {
			hook(r)
		}
		if r.URL.Path == "/client/v4/zones" {
			return dnsResponse(200, map[string]any{"success": true, "result": []CloudflareZone{{ID: testZoneID, Name: "example.test", Status: "active"}}, "result_info": map[string]int{"page": 1, "total_pages": 1, "total_count": 1}}), nil
		}
		if r.URL.Path != "/client/v4/zones/"+testZoneID+"/dns_records" {
			t.Error("unexpected API path")
		}
		return dnsResponse(200, map[string]any{"success": true, "result": []any{}}), nil
	})
	return calls
}
func TestDNSAccountSaveValidatesCredentialSourceAndPreservesSecret(t *testing.T) {
	s, r := dnsFixture(t)
	ctx := context.Background()
	old := r.Secret
	if old == "" || strings.Contains(old, testCloudflareToken) {
		t.Fatal("secret not encrypted")
	}
	r.Name = "renamed"
	r, e := s.Save(ctx, 1, "dns-accounts", r, "")
	if e != nil || r.Secret != old {
		t.Fatal("blank update lost credential", e)
	}
	if _, e = s.dnsProvider(r.ID); e != nil {
		t.Fatal("lego could not use dedicated token", e)
	}
	for _, raw := range []string{`{}`, `{"token":""}`, `{"token":"contains space"}`, `{"token":"x\ny"}`, `{"token":"x","secret":"unused"}`, `{"token":"x","unknown":true}`, `{"token":"x"} {}`} {
		if _, e = s.Save(ctx, 1, "dns-accounts", r, raw); e == nil {
			t.Fatal("invalid credential accepted")
		}
	}
	missing := model.Resource{Name: "missing", GroupID: 10, Spec: JSON(model.AccountSpec{Provider: "cloudflare"})}
	t.Setenv("CLOUDFLARE_DNS_API_TOKEN", testCloudflareToken)
	if _, e = s.Save(ctx, 1, "dns-accounts", missing, ""); e == nil {
		t.Fatal("empty credential silently used environment")
	}
	r.Spec = JSON(model.AccountSpec{Provider: "alidns"})
	if _, e = s.Save(ctx, 1, "dns-accounts", r, ""); e == nil {
		t.Fatal("provider change reused old credential")
	}
}
func TestDNSCredentialReferencesAreValidatedAndNeverExposed(t *testing.T) {
	s, r := dnsFixture(t)
	ctx := context.Background()
	if e := s.DB.AutoMigrate(&configmodel.KeyManage{}); e != nil {
		t.Fatal(e)
	}
	k := configmodel.KeyManage{KeyType: 6, KeySecret: testCloudflareToken, Remark: "DNS token label"}
	if e := k.EncryptKeys(); e != nil {
		t.Fatal(e)
	}
	if e := s.DB.Create(&k).Error; e != nil {
		t.Fatal(e)
	}
	r.Spec = JSON(model.AccountSpec{Provider: "cloudflare", CredentialID: k.ID})
	if _, e := s.Save(ctx, 1, "dns-accounts", r, `{"token":"dual-source"}`); e == nil {
		t.Fatal("two credential sources accepted")
	}
	updated, e := s.Save(ctx, 1, "dns-accounts", r, "")
	if e != nil || updated.Secret != "" {
		t.Fatal("reference did not clear unused secret", e)
	}
	if _, e = s.dnsProvider(updated.ID); e != nil {
		t.Fatal(e)
	}
	opts, e := s.DNSCredentials(ctx, 1, "cloudflare")
	if e != nil || len(opts) != 1 || opts[0].Name != k.Remark {
		t.Fatal(opts, e)
	}
	encoded := string(JSON(opts))
	if strings.Contains(encoded, testCloudflareToken) || strings.Contains(encoded, k.KeySecret) || strings.Contains(encoded, "keySecret") {
		t.Fatal("credential options leaked secret")
	}
	if _, e = s.DNSCredentials(ctx, 2, "cloudflare"); !errors.Is(e, ErrForbidden) {
		t.Fatal("non-admin read shared credential metadata")
	}
	updated.Spec = JSON(model.AccountSpec{Provider: "cloudflare"})
	if _, e = s.Save(ctx, 1, "dns-accounts", updated, ""); e == nil {
		t.Fatal("switch to dedicated reused cleared old secret")
	}
	updated.Spec = JSON(model.AccountSpec{Provider: "alidns", CredentialID: k.ID})
	if _, e = s.Save(ctx, 1, "dns-accounts", updated, ""); e == nil {
		t.Fatal("wrong credential type accepted")
	}
	updated.Spec = JSON(model.AccountSpec{Provider: "cloudflare", CredentialID: 999})
	if _, e = s.Save(ctx, 1, "dns-accounts", updated, ""); e == nil {
		t.Fatal("missing credential accepted")
	}
}
func TestDNSAccountCheckReadsOnlyAndPersistsScope(t *testing.T) {
	s, r := dnsFixture(t)
	calls := goodCloudflare(t, s, nil)
	out, e := s.CheckDNSAccount(context.Background(), 1, r.ID, "*.app.example.test")
	if e != nil || !out.Valid || out.Domain != "*.app.example.test" || out.WriteAccess != "unverified" || out.Zone == nil || out.ZonesCount != 1 || *calls != 3 {
		t.Fatal(out, e, *calls)
	}
	current, _ := s.Get("dns-accounts", r.ID)
	if current.Status != "healthy" || current.CollectedAt == nil || current.Revision != r.Revision {
		t.Fatal("check metadata missing")
	}
	encoded := string(current.Observed)
	if strings.Contains(encoded, testCloudflareToken) {
		t.Fatal("diagnostic exposed token")
	}
	if _, e = s.CheckDNSAccount(context.Background(), 1, r.ID, "badexample.test"); e != nil {
		t.Fatal(e)
	}
	current, _ = s.Get("dns-accounts", r.ID)
	check, _ := decode[DNSAccountCheck](current.Observed)
	if current.Status != "unavailable" || check.Valid || check.Domain != "badexample.test" {
		t.Fatal("domain check scope not stored")
	}
	var events int64
	s.DB.Model(&model.Event{}).Where("kind = ?", "dns-account-check").Count(&events)
	if events != 2 {
		t.Fatal("check audit missing")
	}
	updated, e := s.Save(context.Background(), 1, "dns-accounts", current, `{"token":"rotated-token"}`)
	if e != nil || updated.Status != "configured" || updated.CollectedAt != nil || len(updated.Observed) != 0 {
		t.Fatal("rotated token retained old check", e)
	}
	persisted, _ := s.Get("dns-accounts", r.ID)
	if persisted.CollectedAt != nil || len(persisted.Observed) != 0 {
		t.Fatal("old check was not cleared in database")
	}
}
func TestDNSDiagnosticsPermissionsAndAPIKeyScopes(t *testing.T) {
	s, r := dnsFixture(t)
	calls := goodCloudflare(t, s, nil)
	ctx := context.Background()
	if _, e := s.DNSAccountZones(ctx, 2, r.ID); !errors.Is(e, ErrForbidden) {
		t.Fatal("non-admin discovered zones")
	}
	if _, e := s.CheckDNSAccount(ctx, 2, r.ID, ""); !errors.Is(e, ErrForbidden) {
		t.Fatal("non-admin checked shared account")
	}
	if *calls != 0 {
		t.Fatal("unauthorized outbound request")
	}
	if e := s.DB.AutoMigrate(&system.APIKey{}); e != nil {
		t.Fatal(e)
	}
	key := system.APIKey{UserID: 1, Hash: "dns-test-key", GrantMode: apikey.Scoped, Scopes: "web:read", ExpiresAt: time.Now().Add(time.Hour)}
	s.DB.Create(&key)
	ctx = apikey.WithKey(ctx, key.ID)
	if _, e := s.DNSAccountZones(ctx, 1, r.ID); e != nil {
		t.Fatal("reader could not discover zones", e)
	}
	if _, e := s.CheckDNSAccount(ctx, 1, r.ID, ""); !errors.Is(e, ErrForbidden) {
		t.Fatal("reader wrote check metadata")
	}
	s.DB.Model(&key).Update("scopes", "web:read,web:write")
	if _, e := s.CheckDNSAccount(ctx, 1, r.ID, ""); e != nil {
		t.Fatal(e)
	}
	s.DB.Model(&key).Update("revoked_at", time.Now())
	if _, e := s.DNSAccountZones(ctx, 1, r.ID); !errors.Is(e, ErrForbidden) {
		t.Fatal("revoked key read zones")
	}
}
func TestDNSCheckRejectsConcurrentCredentialChangesAndAuditFailure(t *testing.T) {
	for _, mode := range []string{"account", "reference", "audit", "revocation"} {
		t.Run(mode, func(t *testing.T) {
			s, r := dnsFixture(t)
			ctx := context.Background()
			var key system.APIKey
			var credential configmodel.KeyManage
			if mode == "reference" {
				s.DB.AutoMigrate(&configmodel.KeyManage{})
				credential = configmodel.KeyManage{KeyType: 6, KeySecret: testCloudflareToken}
				credential.EncryptKeys()
				s.DB.Create(&credential)
				r.Spec = JSON(model.AccountSpec{Provider: "cloudflare", CredentialID: credential.ID})
				var e error
				r, e = s.Save(ctx, 1, "dns-accounts", r, "")
				if e != nil {
					t.Fatal(e)
				}
			}
			if mode == "revocation" {
				s.DB.AutoMigrate(&system.APIKey{})
				key = system.APIKey{UserID: 1, Hash: "revoked-while-requesting", GrantMode: apikey.Scoped, Scopes: "web:read,web:write", ExpiresAt: time.Now().Add(time.Hour)}
				s.DB.Create(&key)
				ctx = apikey.WithKey(ctx, key.ID)
			}
			done := false
			goodCloudflare(t, s, func(req *http.Request) {
				if done {
					return
				}
				done = true
				switch mode {
				case "account":
					s.DB.Table(model.Tables["dns-accounts"]).Where("id = ?", r.ID).Update("revision", r.Revision+1)
				case "reference":
					credential.KeySecret = "changed-token"
					credential.EncryptKeys()
					s.DB.Model(&credential).Update("key_secret", credential.KeySecret)
				case "audit":
					s.DB.Migrator().DropTable(&model.Event{})
				case "revocation":
					s.DB.Model(&key).Update("revoked_at", time.Now())
				}
			})
			if _, e := s.CheckDNSAccount(ctx, 1, r.ID, ""); e == nil {
				t.Fatal("concurrent change or audit failure accepted")
			}
			current, _ := s.Get("dns-accounts", r.ID)
			if len(current.Observed) > 0 || current.Status != "configured" {
				t.Fatal("failed check persisted stale evidence")
			}
		})
	}
}
func TestCloudflareFailuresDoNotExposeTokensOrFollowRedirects(t *testing.T) {
	for _, mode := range []string{"authentication", "network", "redirect", "oversize", "malformed", "no-zones", "inactive", "no-dns-read"} {
		t.Run(mode, func(t *testing.T) {
			s, r := dnsFixture(t)
			calls := 0
			dnsMock(s, func(req *http.Request) (*http.Response, error) {
				calls++
				switch mode {
				case "authentication":
					return dnsResponse(403, map[string]any{"success": false, "errors": []any{map[string]any{"code": 10000, "message": testCloudflareToken}}}), nil
				case "network":
					return nil, errors.New(testCloudflareToken)
				case "redirect":
					return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://elsewhere.invalid/"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
				case "oversize":
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(strings.Repeat("x", (2<<20)+1)))}, nil
				case "malformed":
					return dnsResponse(200, map[string]any{"success": true, "result": map[string]string{"token": testCloudflareToken}}), nil
				case "no-zones":
					return dnsResponse(200, map[string]any{"success": true, "result": []any{}}), nil
				case "inactive":
					return dnsResponse(200, map[string]any{"success": true, "result": []CloudflareZone{{ID: testZoneID, Name: "example.test", Status: "pending"}}}), nil
				default:
					if strings.HasSuffix(req.URL.Path, "/zones") {
						return dnsResponse(200, map[string]any{"success": true, "result": []CloudflareZone{{ID: testZoneID, Name: "example.test", Status: "active"}}}), nil
					}
					return dnsResponse(403, map[string]any{"success": false}), nil
				}
			})
			out, e := s.CheckDNSAccount(context.Background(), 1, r.ID, "")
			if e != nil || out.Valid {
				t.Fatal("invalid account marked healthy", e)
			}
			if strings.Contains(string(JSON(out)), testCloudflareToken) {
				t.Fatal("token appeared in diagnostic")
			}
			if mode == "redirect" && calls != 1 {
				t.Fatal("redirect followed with credential")
			}
		})
	}
}
func TestCloudflareZonePaginationAndDomainMatching(t *testing.T) {
	s, r := dnsFixture(t)
	calls := 0
	dnsMock(s, func(req *http.Request) (*http.Response, error) {
		calls++
		zones := []CloudflareZone{}
		for i := 0; i < 50; i++ {
			zones = append(zones, CloudflareZone{ID: fmt.Sprintf("%032x", calls*50+i), Name: fmt.Sprintf("z%d.example.test", calls*50+i), Status: "active"})
		}
		return dnsResponse(200, map[string]any{"success": true, "result": zones, "result_info": map[string]int{"page": calls, "total_pages": 11, "total_count": 550}}), nil
	})
	out, e := s.DNSAccountZones(context.Background(), 1, r.ID)
	if e != nil || !out.Truncated || len(out.Zones) != 500 || out.Total != 550 || calls != 10 {
		t.Fatal("pagination was unbounded or incomplete", e)
	}
	zones := []CloudflareZone{{ID: "root", Name: "example.test", Status: "active"}, {ID: "sub", Name: "sub.example.test", Status: "active"}}
	zone, e := MatchCloudflareZone("*.a.SUB.example.test.", zones)
	if e != nil || zone.ID != "sub" {
		t.Fatal("longest zone not selected", e)
	}
	for _, domain := range []string{"badexample.test", "https://example.test", "a..example.test", "foo.*.example.test", "*", "example.test/path"} {
		if _, e := MatchCloudflareZone(domain, zones); e == nil {
			t.Fatal("invalid domain accepted", domain)
		}
	}
}
func TestCloudflareBoundCertificateValidationChecksEveryZoneWithoutMetadata(t *testing.T) {
	s, r := dnsFixture(t)
	calls := goodCloudflare(t, s, nil)
	if e := s.ValidateCloudflareDomains(context.Background(), r.ID, []string{"example.test", "*.example.test", "app.example.test"}); e != nil {
		t.Fatal(e)
	}
	if *calls != 4 {
		t.Fatal("zone DNS reads or challenge names not deduplicated")
	}
	current, _ := s.Get("dns-accounts", r.ID)
	if len(current.Observed) != 0 {
		t.Fatal("certificate check changed account metadata")
	}
	if e := s.ValidateCloudflareDomains(context.Background(), r.ID, []string{"notexample.test"}); e == nil {
		t.Fatal("foreign domain accepted")
	}
}
func TestDNSCredentialsEndpointNeverNeedsSecretDecryption(t *testing.T) {
	s, _ := dnsFixture(t)
	s.DB.AutoMigrate(&configmodel.KeyManage{})
	s.DB.Create(&configmodel.KeyManage{KeyType: 6, KeyID: "not-readable", KeySecret: "invalid-ciphertext", Remark: "token to repair"})
	rows, e := s.DNSCredentials(context.Background(), 1, "cloudflare")
	if e != nil || len(rows) != 1 {
		t.Fatal(rows, e)
	}
	b, _ := json.Marshal(rows)
	if strings.Contains(string(b), "invalid-ciphertext") {
		t.Fatal("encrypted bytes leaked")
	}
}

func TestCloudflareDNSChallengeCNAMEDelegationIsExplicitlyRejected(t *testing.T) {
	s, r := dnsFixture(t)
	dnsMock(s, func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/zones") {
			return dnsResponse(200, map[string]any{"success": true, "result": []CloudflareZone{{ID: testZoneID, Name: "example.test", Status: "active"}}}), nil
		}
		if req.URL.Query().Get("type") == "CNAME" {
			if req.URL.Query().Get("name") != "_acme-challenge.example.test" {
				t.Fatal("unexpected challenge record name")
			}
			return dnsResponse(200, map[string]any{"success": true, "result": []any{map[string]string{"type": "CNAME", "name": "_acme-challenge.example.test", "content": "delegated.other.test"}}}), nil
		}
		return dnsResponse(200, map[string]any{"success": true, "result": []any{}}), nil
	})
	checked, e := s.CheckDNSAccount(context.Background(), 1, r.ID, "*.example.test")
	if e != nil || checked.Valid || !strings.Contains(string(JSON(checked)), "CNAME") {
		t.Fatal("delegation check was not reported", e)
	}
	if e = s.ValidateCloudflareDomains(context.Background(), r.ID, []string{"example.test"}); e == nil || !strings.Contains(e.Error(), "CNAME") {
		t.Fatal("delegated signing preflight was accepted", e)
	}
}

func TestDNSAccountEarlierCheckCannotOverwriteNewerObservation(t *testing.T) {
	s, r := dnsFixture(t)
	startedNewer := false
	var newer DNSAccountCheck
	goodCloudflare(t, s, func(req *http.Request) {
		if startedNewer {
			return
		}
		startedNewer = true
		// The older request is awaiting its first response. A later request
		// completes and persists a different domain scope before it can continue.
		time.Sleep(2 * time.Millisecond)
		var err error
		newer, err = s.CheckDNSAccount(context.Background(), 1, r.ID, "app.example.test")
		if err != nil || !newer.Valid {
			t.Fatal("newer diagnostic did not finish", err)
		}
	})
	older, err := s.CheckDNSAccount(context.Background(), 1, r.ID, "example.test")
	if !errors.Is(err, ErrConflict) || !older.CheckedAt.Before(newer.CheckedAt) {
		t.Fatal("late old response was not rejected", err)
	}
	current, err := s.Get("dns-accounts", r.ID)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := decode[DNSAccountCheck](current.Observed)
	if err != nil || saved.Domain != "app.example.test" || !saved.CheckedAt.Equal(newer.CheckedAt) || current.Status != "healthy" {
		t.Fatal("newer observation was overwritten", err)
	}
	var events int64
	s.DB.Model(&model.Event{}).Where("kind = ?", "dns-account-check").Count(&events)
	if events != 1 {
		t.Fatal("superseded check added an audit success")
	}
}
