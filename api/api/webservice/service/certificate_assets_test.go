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
	"dodevops-api/pkg/apikey"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func certificateAssetPEM(t *testing.T, change func(*x509.Certificate)) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(42), Subject: pkix.Name{CommonName: "example.test"},
		DNSNames: []string{"example.test", "*.example.test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(48 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	if change != nil {
		change(template)
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
}

func importCertificateAsset(t *testing.T, s *Service) (CertificateImportResult, string) {
	t.Helper()
	cert, key := certificateAssetPEM(t, nil)
	result, err := s.ImportNewCertificate(context.Background(), 1, CertificateImportInput{Name: "Imported certificate", GroupID: 10,
		Environment: "test", Certificate: cert, PrivateKey: key})
	if err != nil {
		t.Fatal(err)
	}
	return result, key
}

func TestImportNewCertificateAtomicAndMetadata(t *testing.T) {
	s := fixture(t)
	result, key := importCertificateAsset(t, s)
	asset := result.Certificate
	spec, err := decode[model.CertificateSpec](asset.Spec)
	if err != nil || asset.ID == 0 || asset.Revision != 2 || asset.Status != "issued" || spec.Owner != "external" ||
		spec.CurrentVersionID != result.Version.ID || len(spec.Domains) != 2 || result.Version.ResourceID != asset.ID {
		t.Fatalf("import metadata incorrect: asset=%d version=%d error=%v", asset.ID, result.Version.ID, err)
	}
	if result.Version.Secret == key || !strings.Contains(string(asset.Observed), "notAfter") {
		t.Fatal("certificate did not retain encrypted key or public expiry metadata")
	}
	encoded, err := json.Marshal(result)
	if err != nil || strings.Contains(string(encoded), "PRIVATE KEY") || strings.Contains(string(encoded), result.Version.Secret) {
		t.Fatal("import response exposed private material")
	}
	var count int64
	if err := s.DB.Model(&model.Event{}).Where("kind = ?", "certificate-import").Count(&count).Error; err != nil || count != 1 {
		t.Fatal("import audit missing")
	}
}

func TestImportNewCertificateFailureLeavesNoRows(t *testing.T) {
	for _, failure := range []string{"mismatch", "invalid-chain", "expired", "future", "client-only", "ca", "no-san", "private-in-chain", "malformed-prefix", "nested-prefix", "trailing-text", "extra-key", "malformed-key-prefix", "invalid-group", "encryption", "audit"} {
		t.Run(failure, func(t *testing.T) {
			s := fixture(t)
			cert, key := certificateAssetPEM(t, func(c *x509.Certificate) {
				switch failure {
				case "expired":
					c.NotBefore, c.NotAfter = time.Now().Add(-48*time.Hour), time.Now().Add(-time.Minute)
				case "future":
					c.NotBefore = time.Now().Add(time.Hour)
				case "client-only":
					c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
				case "ca":
					c.IsCA, c.BasicConstraintsValid = true, true
				case "no-san":
					c.DNSNames = nil
				}
			})
			group := uint(10)
			switch failure {
			case "mismatch":
				_, key = certificateAssetPEM(t, nil)
			case "invalid-chain":
				other, _ := certificateAssetPEM(t, nil)
				cert += other
			case "private-in-chain":
				cert += key
			case "malformed-prefix":
				cert = "-----BEGIN CERTIFICATE-----\ninvalid\n-----END CERTIFICATE-----\n" + key + cert
			case "nested-prefix":
				cert = "-----BEGIN CERTIFICATE-----\n" + key + cert
			case "trailing-text":
				cert += "not a PEM block"
			case "extra-key":
				key += key
			case "malformed-key-prefix":
				key = "-----BEGIN PRIVATE KEY-----\ninvalid\n-----END PRIVATE KEY-----\n" + key
			case "invalid-group":
				group = 999
			case "encryption":
				t.Setenv("WEB_ENCRYPTION_KEY", "invalid")
			case "audit":
				if err := s.DB.Migrator().DropTable(&model.Event{}); err != nil {
					t.Fatal(err)
				}
			}
			result, err := s.ImportNewCertificate(context.Background(), 1, CertificateImportInput{Name: "bad", GroupID: group,
				Certificate: cert, PrivateKey: key})
			if err == nil || result.Certificate.ID != 0 || result.Version.ID != 0 {
				t.Fatal("invalid import reported success")
			}
			for _, table := range []string{model.Tables["certificates"], (model.Revision{}).TableName()} {
				var count int64
				if err := s.DB.Table(table).Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("failed import left records in %s", table)
				}
			}
		})
	}
}

func TestCertificateImportAssetPermissionsAndSAN(t *testing.T) {
	s := fixture(t)
	cert, key := certificateAssetPEM(t, func(c *x509.Certificate) {
		c.DNSNames = nil
		c.IPAddresses = []net.IP{net.ParseIP("192.0.2.10"), net.ParseIP("2001:db8::1")}
	})
	input := CertificateImportInput{Name: "IP certificate", GroupID: 10, Certificate: cert, PrivateKey: key}
	if _, err := s.ImportNewCertificate(context.Background(), 2, input); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ungranted group accepted: %v", err)
	}
	grant := cmdb.CmdbAssetPermission{Name: "certificate creators", UserIDs: "[2]", AssetTypes: `["web_certificate"]`,
		HostGroupIDs: "[10]", PermissionActions: `["web_view","web_certificate_manage"]`, IsActive: 1}
	if err := s.DB.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
	result, err := s.ImportNewCertificate(context.Background(), 2, input)
	if err != nil {
		t.Fatal(err)
	}
	spec, _ := decode[model.CertificateSpec](result.Certificate.Spec)
	if len(spec.Domains) != 2 || spec.Domains[1] != "2001:db8::1" {
		t.Fatal("IP SANs not imported")
	}
	if _, err := s.ImportCertificate(context.Background(), 2, result.Certificate.ID, cert, key); err != nil {
		t.Fatal("certificate manager could not import an updated version")
	}
	if _, err := s.ExportCertificate(context.Background(), 2, result.Certificate.ID, 0, true); !errors.Is(err, ErrForbidden) {
		t.Fatal("nonadministrator certificate manager exported a private key")
	}
}

func TestCertificateImportFailurePreservesCurrentVersion(t *testing.T) {
	s := fixture(t)
	initial, _ := importCertificateAsset(t, s)
	cert, key := certificateAssetPEM(t, func(c *x509.Certificate) { c.DNSNames = []string{"another.test"} })
	if _, err := s.ImportCertificate(context.Background(), 1, initial.Certificate.ID, cert, key); err == nil {
		t.Fatal("certificate with different SAN scope accepted")
	}
	current, err := s.Get("certificates", initial.Certificate.ID)
	if err != nil || current.Revision != initial.Certificate.Revision || string(current.Spec) != string(initial.Certificate.Spec) {
		t.Fatal("invalid import changed the active version")
	}
	validCert, validKey := certificateAssetPEM(t, nil)
	if _, err = s.ImportCertificate(context.Background(), 1, current.ID, validCert, validKey); err != nil {
		t.Fatal(err)
	}
	if _, err = s.storeCertificate(1, current, validCert, validKey); !errors.Is(err, ErrConflict) {
		t.Fatal("concurrent stale import overwrote the active version")
	}
	var count int64
	s.DB.Model(&model.Revision{}).Where("kind = ?", "certificate").Count(&count)
	if count != 2 {
		t.Fatal("failed or conflicting import left an immutable version")
	}
}

func TestCertificateExportPermissionsAndVersionOwnership(t *testing.T) {
	s := fixture(t)
	first, key := importCertificateAsset(t, s)
	second, _ := importCertificateAsset(t, s)
	if _, err := s.ExportCertificate(context.Background(), 1, first.Certificate.ID, second.Version.ID, false); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("version from another certificate was exported")
	}
	if _, err := s.ExportCertificate(context.Background(), 2, first.Certificate.ID, 0, false); !errors.Is(err, ErrForbidden) {
		t.Fatal("public certificate exported without asset permission")
	}
	if err := s.DB.Create(&cmdb.CmdbAssetPermission{Name: "certificate readers", UserIDs: "[2]", AssetTypes: `["web_certificate"]`,
		HostGroupIDs: "[10]", PermissionActions: `["web_view"]`, IsActive: 1}).Error; err != nil {
		t.Fatal(err)
	}
	public, err := s.ExportCertificate(context.Background(), 2, first.Certificate.ID, 0, false)
	if err != nil || public["certificate"] == "" || public["privateKey"] != "" {
		t.Fatal("read-only public export failed or exposed a private key")
	}
	private, err := s.ExportCertificate(context.Background(), 1, first.Certificate.ID, first.Version.ID, true)
	if err != nil || private["privateKey"] != key {
		t.Fatal("administrator could not export the original key")
	}
	if err := s.DB.Migrator().DropTable(&model.Event{}); err != nil {
		t.Fatal(err)
	}
	for _, includeKey := range []bool{false, true} {
		out, err := s.ExportCertificate(context.Background(), 1, first.Certificate.ID, 0, includeKey)
		if err == nil || out != nil {
			t.Fatal("export returned material despite audit failure")
		}
	}
}

func TestCertificateExportRespectsAPIKeyScopeAndRevocation(t *testing.T) {
	s := fixture(t)
	asset, _ := importCertificateAsset(t, s)
	if err := s.DB.AutoMigrate(&system.APIKey{}); err != nil {
		t.Fatal(err)
	}
	key := system.APIKey{UserID: 1, Hash: "unit-test", GrantMode: apikey.Scoped, Scopes: "web:read", ExpiresAt: time.Now().Add(time.Hour)}
	if err := s.DB.Create(&key).Error; err != nil {
		t.Fatal(err)
	}
	ctx := apikey.WithKey(context.Background(), key.ID)
	if _, err := s.ExportCertificate(ctx, 1, asset.Certificate.ID, 0, false); err != nil {
		t.Fatal("scoped reader could not export public certificate")
	}
	if _, err := s.ExportCertificate(ctx, 1, asset.Certificate.ID, 0, true); !errors.Is(err, ErrForbidden) {
		t.Fatal("read-only API Key exported private material")
	}
	s.DB.Model(&key).Update("scopes", "web:read,web:write")
	if _, err := s.ExportCertificate(ctx, 1, asset.Certificate.ID, 0, true); err != nil {
		t.Fatal("scoped administrator export failed")
	}
	s.DB.Model(&key).Update("scopes", "web:execute")
	if _, err := s.ExportCertificate(ctx, 1, asset.Certificate.ID, 0, false); !errors.Is(err, ErrForbidden) {
		t.Fatal("API Key without read scope exported certificate")
	}
	s.DB.Model(&key).Updates(map[string]any{"scopes": "web:read,web:write", "revoked_at": time.Now()})
	if _, err := s.ExportCertificate(ctx, 1, asset.Certificate.ID, 0, true); !errors.Is(err, ErrForbidden) {
		t.Fatal("revoked API Key exported certificate")
	}
}

func TestExpiredCertificateHistoryRemainsExportable(t *testing.T) {
	s := fixture(t)
	asset, _ := importCertificateAsset(t, s)
	cert, key := certificateAssetPEM(t, func(c *x509.Certificate) {
		c.NotBefore, c.NotAfter = time.Now().Add(-48*time.Hour), time.Now().Add(-time.Minute)
	})
	block, _ := pem.Decode([]byte(cert))
	parsed, _ := x509.ParseCertificate(block.Bytes)
	sealed, err := seal(key)
	if err != nil {
		t.Fatal(err)
	}
	meta := certContent{PEM: cert, Domains: parsed.DNSNames, Fingerprint: digest(parsed.Raw), NotBefore: parsed.NotBefore, NotAfter: parsed.NotAfter}
	if err := s.DB.Model(&model.Revision{}).Where("id = ?", asset.Version.ID).Updates(map[string]any{"content": JSON(meta), "secret": sealed}).Error; err != nil {
		t.Fatal(err)
	}
	versions, _, err := s.CertificateVersions(1, asset.Certificate.ID)
	if err != nil || len(versions) != 1 || versions[0].Days != -1 {
		t.Fatal("recently expired certificate not identified as expired")
	}
	s.Redact(1, "certificates", &asset.Certificate)
	var expiry struct {
		Days int `json:"days"`
	}
	json.Unmarshal(asset.Certificate.Observed, &expiry)
	if expiry.Days != -1 {
		t.Fatal("certificate list did not show expired state")
	}
	if out, err := s.ExportCertificate(context.Background(), 1, asset.Certificate.ID, asset.Version.ID, true); err != nil || out["privateKey"] != key {
		t.Fatal("expired stored version could not be backed up")
	}
}

func TestPublicCertificateExportRejectsContaminatedStoredPEM(t *testing.T) {
	s := fixture(t)
	asset, key := importCertificateAsset(t, s)
	var content map[string]any
	if err := json.Unmarshal(asset.Version.Content, &content); err != nil {
		t.Fatal(err)
	}
	content["pem"] = content["pem"].(string) + key
	if err := s.DB.Model(&model.Revision{}).Where("id = ?", asset.Version.ID).Update("content", JSON(content)).Error; err != nil {
		t.Fatal(err)
	}
	for _, includeKey := range []bool{false, true} {
		out, err := s.ExportCertificate(context.Background(), 1, asset.Certificate.ID, 0, includeKey)
		if err == nil || out != nil {
			t.Fatal("export returned a stored chain containing private material")
		}
	}
}

func TestPublicCertificateExportDoesNotRequirePrivateKeyDecryption(t *testing.T) {
	s := fixture(t)
	asset, _ := importCertificateAsset(t, s)
	t.Setenv("WEB_ENCRYPTION_KEY", "unavailable")
	out, err := s.ExportCertificate(context.Background(), 1, asset.Certificate.ID, 0, false)
	if err != nil || out["certificate"] == "" || out["privateKey"] != "" {
		t.Fatal("public export depended on private key decryption")
	}
	if out, err = s.ExportCertificate(context.Background(), 1, asset.Certificate.ID, 0, true); err == nil || out != nil {
		t.Fatal("private export succeeded without the encryption key")
	}
}

func TestCertificateVersionImportAuditFailureRollsBack(t *testing.T) {
	s := fixture(t)
	asset, _ := importCertificateAsset(t, s)
	if err := s.DB.Migrator().DropTable(&model.Event{}); err != nil {
		t.Fatal(err)
	}
	cert, key := certificateAssetPEM(t, nil)
	if _, err := s.ImportCertificate(context.Background(), 1, asset.Certificate.ID, cert, key); err == nil {
		t.Fatal("import without an audit record succeeded")
	}
	current, err := s.Get("certificates", asset.Certificate.ID)
	if err != nil || current.Revision != asset.Certificate.Revision || string(current.Spec) != string(asset.Certificate.Spec) {
		t.Fatal("audit failure changed the current certificate")
	}
	var count int64
	if err := s.DB.Model(&model.Revision{}).Where("kind = ? AND resource_id = ?", "certificate", asset.Certificate.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("audit failure left a certificate version")
	}
}
