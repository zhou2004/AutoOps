package service

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"dodevops-api/api/webservice/model"
	"dodevops-api/pkg/apikey"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/acme"
	acmeapi "github.com/go-acme/lego/v4/acme/api"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/resolver"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/alidns"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/tencentcloud"
	"github.com/go-acme/lego/v4/registration"
	"gorm.io/gorm"
)

type certContent struct {
	PEM         string    `json:"pem"`
	Domains     []string  `json:"domains"`
	Issuer      string    `json:"issuer"`
	Fingerprint string    `json:"fingerprint"`
	NotBefore   time.Time `json:"notBefore"`
	NotAfter    time.Time `json:"notAfter"`
	RenewAt     time.Time `json:"renewAt"`
}

func ParseCertificate(certPEM, keyPEM string, domains []string) (certContent, error) {
	out := certContent{}
	pair, e := certificateKeyPair(certPEM, keyPEM)
	if e != nil {
		return out, errors.New("证书链与私钥无效或不匹配")
	}
	cert, e := x509.ParseCertificate(pair.Certificate[0])
	if e != nil {
		return out, e
	}
	now := time.Now()
	if now.Before(cert.NotBefore) || !now.Before(cert.NotAfter) {
		return out, errors.New("证书尚未生效或已过期")
	}
	if cert.IsCA {
		return out, errors.New("请导入服务器证书")
	}
	for _, d := range domains {
		host := d
		if strings.HasPrefix(d, "*.") {
			found := false
			for _, san := range cert.DNSNames {
				if san == d {
					found = true
				}
			}
			if !found {
				return out, errors.New("证书未覆盖所需泛域名")
			}
			host = "autoops-probe." + strings.TrimPrefix(d, "*.")
		}
		if e = cert.VerifyHostname(host); e != nil {
			return out, errors.New("证书未覆盖域名: " + d)
		}
	}
	if len(cert.ExtKeyUsage)+len(cert.UnknownExtKeyUsage) > 0 {
		server := false
		for _, usage := range cert.ExtKeyUsage {
			server = server || usage == x509.ExtKeyUsageServerAuth || usage == x509.ExtKeyUsageAny
		}
		if !server {
			return out, errors.New("证书不允许服务器认证")
		}
	}
	for i := 0; i+1 < len(pair.Certificate); i++ {
		child, e := x509.ParseCertificate(pair.Certificate[i])
		if e != nil {
			return out, e
		}
		parent, e := x509.ParseCertificate(pair.Certificate[i+1])
		if e != nil {
			return out, e
		}
		if e = child.CheckSignatureFrom(parent); e != nil {
			return out, errors.New("证书链顺序或签名无效")
		}
		if now.Before(parent.NotBefore) || !now.Before(parent.NotAfter) {
			return out, errors.New("证书链包含尚未生效或已过期的证书")
		}
	}
	sans := append([]string(nil), cert.DNSNames...)
	for _, address := range cert.IPAddresses {
		sans = append(sans, address.String())
	}
	if len(sans) == 0 {
		return out, errors.New("证书必须包含 DNS 或 IP 类型的 SAN")
	}
	return certContent{PEM: certPEM, Domains: sans, Issuer: cert.Issuer.String(), Fingerprint: digest(cert.Raw), NotBefore: cert.NotBefore, NotAfter: cert.NotAfter, RenewAt: cert.NotBefore.Add(cert.NotAfter.Sub(cert.NotBefore) * 2 / 3)}, nil
}

// Only PEM certificates belong in the public chain. In particular, reject a
// concatenated private key before this content can become a public API response.
// Unlike import validation, this check deliberately permits expired archives.
func validatePublicCertificatePEM(certPEM string) error {
	if len(certPEM) > 2<<20 {
		return errors.New("证书链超过大小限制")
	}
	remaining := strings.TrimSpace(certPEM)
	count := 0
	for remaining != "" {
		if !strings.HasPrefix(remaining, "-----BEGIN CERTIFICATE-----") {
			return errors.New("证书链仅允许包含 CERTIFICATE PEM 块")
		}
		end := strings.Index(remaining, "-----END CERTIFICATE-----")
		if end < 0 {
			return errors.New("证书链 PEM 格式无效")
		}
		end += len("-----END CERTIFICATE-----")
		segment := remaining[:end]
		// pem.Decode skips malformed leading blocks. Parse exactly one block at
		// a time so skipped text can never be retained in the public PEM value.
		block, rest := pem.Decode([]byte(segment))
		if strings.Count(segment, "-----BEGIN ") != 1 || block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 || strings.TrimSpace(string(rest)) != "" {
			return errors.New("证书链 PEM 格式无效")
		}
		if _, err := x509.ParseCertificate(block.Bytes); err != nil {
			return errors.New("证书链 PEM 内容无效")
		}
		count++
		remaining = strings.TrimSpace(remaining[end:])
	}
	if count == 0 {
		return errors.New("请提供 PEM 证书链")
	}
	return nil
}

func certificateKeyPair(certPEM, keyPEM string) (tls.Certificate, error) {
	if err := validatePublicCertificatePEM(certPEM); err != nil {
		return tls.Certificate{}, err
	}
	if len(keyPEM) > 64<<10 {
		return tls.Certificate{}, errors.New("私钥超过大小限制")
	}
	keyText := strings.TrimSpace(keyPEM)
	if !strings.HasPrefix(keyText, "-----BEGIN ") {
		return tls.Certificate{}, errors.New("请提供 PEM 证书链与私钥")
	}
	key, rest := pem.Decode([]byte(keyText))
	if strings.Count(keyText, "-----BEGIN ") != 1 || key == nil || len(key.Headers) != 0 || strings.TrimSpace(string(rest)) != "" ||
		(key.Type != "PRIVATE KEY" && key.Type != "RSA PRIVATE KEY" && key.Type != "EC PRIVATE KEY") {
		return tls.Certificate{}, errors.New("请提供单个未加密的 PEM 私钥")
	}
	return tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
}

func (s *Service) ImportCertificate(ctx context.Context, u, id uint, cert, key string) (model.Revision, error) {
	r, e := s.Get("certificates", id)
	if e != nil {
		return model.Revision{}, e
	}
	if e = s.Module(ctx, u, "write"); e != nil {
		return model.Revision{}, e
	}
	if e = s.Asset(u, "web_certificate", id, "web_certificate_manage"); e != nil {
		return model.Revision{}, e
	}
	var version model.Revision
	e = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		version, err = New(tx).storeCertificate(u, r, cert, key)
		if err != nil {
			return err
		}
		return tx.Create(&model.Event{UserID: u, Kind: "certificate-import", Message: fmt.Sprintf("certificate=%d version=%d", id, version.ID)}).Error
	})
	return version, e
}
func (s *Service) storeCertificate(u uint, r model.Resource, cert, key string) (model.Revision, error) {
	p, e := decode[model.CertificateSpec](r.Spec)
	if e != nil {
		return model.Revision{}, e
	}
	meta, e := ParseCertificate(cert, key, p.Domains)
	if e != nil {
		return model.Revision{}, e
	}
	secret, e := seal(key)
	if e != nil {
		return model.Revision{}, e
	}
	v := model.Revision{ResourceID: r.ID, Kind: "certificate", Name: r.Name, Content: JSON(meta), Secret: secret, Digest: meta.Fingerprint, CreatedBy: u}
	e = s.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(&v).Error; e != nil {
			return e
		}
		p.CurrentVersionID = v.ID
		res := tx.Table(model.Tables["certificates"]).Where("id = ? AND revision = ?", r.ID, r.Revision).Updates(map[string]any{"spec": JSON(p), "revision": gorm.Expr("revision + 1"), "status": "issued"})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrConflict
		}
		return nil
	})
	return v, e
}
func (s *Service) certificateFiles(u, cid, bid uint) ([]model.File, error) {
	return s.certificateFilesAt(u, cid, bid, 0)
}
func (s *Service) certificateFilesAt(u, cid, bid, versionOverride uint) ([]model.File, error) {
	binding, e := s.Get("certificate-bindings", bid)
	if e != nil {
		return nil, e
	}
	if binding.ClusterID != cid {
		return nil, ErrForbidden
	}
	p, e := decode[model.BindingSpec](binding.Spec)
	if e != nil {
		return nil, e
	}
	if e = s.Asset(u, "web_certificate", p.CertificateID, "web_certificate_use"); e != nil {
		return nil, e
	}
	cert, e := s.Get("certificates", p.CertificateID)
	if e != nil {
		return nil, e
	}
	cs, _ := decode[model.CertificateSpec](cert.Spec)
	version := p.VersionID
	if versionOverride > 0 {
		version = versionOverride
	}
	if version == 0 {
		version = cs.CurrentVersionID
	}
	var rev model.Revision
	if e = s.DB.Where("id = ? AND kind = ? AND resource_id = ?", version, "certificate", cert.ID).First(&rev).Error; e != nil {
		return nil, e
	}
	content, e := decode[certContent](rev.Content)
	if e != nil {
		return nil, e
	}
	key, e := unseal(rev.Secret)
	if e != nil {
		return nil, e
	}
	if _, e = ParseCertificate(content.PEM, key, cs.Domains); e != nil {
		return nil, e
	}
	return []model.File{{Path: p.CertificatePath, Content: content.PEM, Mode: 0644}, {Path: p.KeyPath, Content: key, Mode: 0600}}, nil
}
func (s *Service) validateIssuance(u uint, r model.Resource) error {
	p, e := decode[model.CertificateSpec](r.Spec)
	if e != nil {
		return e
	}
	if p.Owner != "platform" {
		return errors.New("仅平台托管证书支持自动签发")
	}
	if _, e = canonicalIssuanceDomains(p.Domains); e != nil {
		return e
	}
	account, e := s.Get("acme-accounts", p.ACMEAccountID)
	if e != nil {
		return errors.New("请选择 ACME 账户")
	}
	a, e := decode[model.AccountSpec](account.Spec)
	if e != nil {
		return e
	}
	if e = validateACMEAccount(a); e != nil {
		return e
	}
	if p.Challenge == "dns-01" {
		dns, err := s.Get("dns-accounts", p.DNSAccountID)
		if err != nil {
			return errors.New("请选择有效的 DNS 账户")
		}
		_, _, _, e = s.dnsAccountCredentials(dns)
		return e
	}
	for _, domain := range p.Domains {
		if strings.HasPrefix(strings.TrimSpace(domain), "*.") {
			return errors.New("泛域名证书需要 DNS 验证")
		}
	}
	if p.Challenge != "http-01" || !validPath(p.ChallengeRoot) || len(p.InstanceIDs) == 0 {
		return errors.New("HTTP 验证需要挑战根目录及全部入口实例")
	}
	for _, id := range p.InstanceIDs {
		i, e := s.Get("instances", id)
		if e != nil {
			return e
		}
		if e = s.Authorize(u, "instances", i, "web_manage"); e != nil {
			return e
		}
	}
	return nil
}

type acmeUser struct {
	email        string
	key          crypto.PrivateKey
	registration *registration.Resource
}

func (u *acmeUser) GetEmail() string                        { return u.email }
func (u *acmeUser) GetPrivateKey() crypto.PrivateKey        { return u.key }
func (u *acmeUser) GetRegistration() *registration.Resource { return u.registration }
func newPrivateKey() (crypto.PrivateKey, string, error) {
	k, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		return nil, "", e
	}
	b, e := x509.MarshalPKCS8PrivateKey(k)
	return k, string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b})), e
}
func parsePrivateKey(raw string) (crypto.PrivateKey, error) {
	b, _ := pem.Decode([]byte(raw))
	if b == nil {
		return nil, errors.New("invalid account key")
	}
	return x509.ParsePKCS8PrivateKey(b.Bytes)
}

type cancelTransport struct {
	ctx   context.Context
	inner http.RoundTripper
}

func (t cancelTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return t.inner.RoundTrip(r.WithContext(t.ctx))
}
func (s *Service) dnsProvider(id uint) (challenge.Provider, error) {
	r, e := s.Get("dns-accounts", id)
	if e != nil {
		return nil, e
	}
	p, key, secret, e := s.dnsAccountCredentials(r)
	if e != nil {
		return nil, e
	}
	switch p.Provider {
	case "alidns":
		c := alidns.NewDefaultConfig()
		c.APIKey = key
		c.SecretKey = secret
		c.RegionID = "cn-hangzhou"
		return alidns.NewDNSProviderConfig(c)
	case "tencentcloud":
		c := tencentcloud.NewDefaultConfig()
		c.SecretID = key
		c.SecretKey = secret
		return tencentcloud.NewDNSProviderConfig(c)
	case "cloudflare":
		c := cloudflare.NewDefaultConfig()
		c.AuthToken = secret
		return cloudflare.NewDNSProviderConfig(c)
	}
	return nil, errors.New("不支持的 DNS 服务商")
}
func (s *Service) issue(ctx context.Context, op model.Operation, r model.Resource) error {
	p, e := decode[model.CertificateSpec](r.Spec)
	if e != nil {
		return e
	}
	if e = s.validateIssuance(op.UserID, r); e != nil {
		return e
	}
	if e = s.checkIssuanceDNS(ctx, r); e != nil {
		return e
	}
	// Construct the provider before checking the frozen dependency digest so
	// this operation keeps using the reviewed credential even if it is edited
	// after execution begins.
	var dns challenge.Provider
	if p.Challenge == "dns-01" {
		dns, e = s.dnsProvider(p.DNSAccountID)
		if e != nil {
			return e
		}
	}
	account, e := s.Get("acme-accounts", p.ACMEAccountID)
	if e != nil {
		return e
	}
	a, e := decode[model.AccountSpec](account.Spec)
	if e != nil {
		return e
	}
	// Account registration is serialized independently of certificate orders.
	leaseKey := fmt.Sprintf("acme-account:%d", account.ID)
	if !s.lease(leaseKey, fmt.Sprint(op.ID), 10*time.Minute) {
		return errors.New("ACME 账户正在使用，请稍后重试")
	}
	defer s.DB.Model(&model.Lease{}).Where("`key` = ?", leaseKey).Update("expires_at", time.Now().UTC())
	var plan model.Plan
	if e = s.DB.First(&plan, op.PlanID).Error; e != nil {
		return e
	}
	if e = s.checkIssuanceDependencies(r, plan); e != nil {
		return e
	}
	var order model.Order
	found := s.DB.Where("certificate_id = ? AND status IN ?", r.ID, []string{"creating", "pending", "ready", "processing", "valid"}).Order("id DESC").First(&order)
	if found.Error == nil {
		identity, err := issuanceOrderIdentity(account, p.Domains)
		if err != nil {
			return err
		}
		if err = checkOrderIdentity(order, identity); err != nil {
			return err
		}
		if order.URL == "" {
			return errors.New("签发订单创建结果待核验，保留订单以避免重复签发")
		}
	} else if !errors.Is(found.Error, gorm.ErrRecordNotFound) {
		return found.Error
	}
	var key crypto.PrivateKey
	if account.Secret == "" {
		k, raw, e := newPrivateKey()
		if e != nil {
			return e
		}
		key = k
		sealed, e := seal(raw)
		if e != nil {
			return e
		}
		updated := s.DB.Table(model.Tables["acme-accounts"]).Where("id = ? AND revision = ? AND secret = ?", account.ID, account.Revision, account.Secret).Update("secret", sealed)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrConflict
		}
		account.Secret = sealed
	} else {
		raw, e := unseal(account.Secret)
		if e != nil {
			return e
		}
		key, e = parsePrivateKey(raw)
		if e != nil {
			return e
		}
	}
	user := &acmeUser{email: a.Email, key: key}
	if len(a.Registration) > 0 && string(a.Registration) != "null" {
		user.registration = &registration.Resource{}
		if e = json.Unmarshal(a.Registration, user.registration); e != nil {
			return e
		}
	}
	config := lego.NewConfig(user)
	config.CADirURL = a.DirectoryURL
	config.HTTPClient = &http.Client{Timeout: 30 * time.Second, Transport: cancelTransport{ctx, http.DefaultTransport}}
	client, e := lego.NewClient(config)
	if e != nil {
		return e
	}
	if user.registration == nil {
		reg, e := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if e != nil {
			return e
		}
		user.registration = reg
		a.Registration = JSON(reg)
		updated := s.DB.Table(model.Tables["acme-accounts"]).Where("id = ? AND revision = ? AND secret = ?", account.ID, account.Revision, account.Secret).Update("spec", JSON(a))
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrConflict
		}
		account.Spec = JSON(a)
	}
	core, e := acmeapi.New(config.HTTPClient, "AutoOps Web", a.DirectoryURL, user.registration.URI, key)
	if e != nil {
		return e
	}
	solvers := resolver.NewSolversManager(core)
	if p.Challenge == "dns-01" {
		if e = solvers.SetDNS01Provider(dns, dnsPropagationFallback(ctx)); e != nil {
			return e
		}
	} else {
		if e = solvers.SetHTTP01Provider(&httpChallenge{s: s, ctx: ctx, user: op.UserID, spec: p}); e != nil {
			return e
		}
	}
	var private crypto.PrivateKey
	var privatePEM string
	var remote acme.ExtendedOrder
	if found.Error == nil {
		privatePEM, e = unseal(order.Secret)
		if e != nil {
			return e
		}
		private, e = parsePrivateKey(privatePEM)
		if e != nil {
			return e
		}
		if order.URL == "" {
			return errors.New("签发订单创建结果待核验，保留订单以避免重复签发")
		}
		remote, e = core.Orders.Get(order.URL)
		if e != nil {
			return e
		}
	} else if errors.Is(found.Error, gorm.ErrRecordNotFound) {
		identity, err := issuanceOrderIdentity(account, p.Domains)
		if err != nil {
			return err
		}
		private, privatePEM, e = newPrivateKey()
		if e != nil {
			return e
		}
		sealed, e := seal(privatePEM)
		if e != nil {
			return e
		}
		order = model.Order{CertificateID: r.ID, Status: "creating", Secret: sealed, IdentityDigest: identity}
		if e = s.DB.Create(&order).Error; e != nil {
			return e
		}
		remote, e = core.Orders.New(p.Domains)
		if e != nil {
			return e
		}
		order.URL = remote.Location
		order.Status = remote.Status
		if e = s.DB.Save(&order).Error; e != nil {
			return e
		}
	} else {
		return found.Error
	}
	if remote.Status == acme.StatusInvalid {
		order.Status = "invalid"
		if e = s.DB.Save(&order).Error; e != nil {
			return e
		}
		return errors.New("ACME 订单已失效，可重新签发")
	}
	if remote.Status == acme.StatusPending {
		var auths []acme.Authorization
		for _, uri := range remote.Authorizations {
			v, e := core.Authorizations.Get(uri)
			if e != nil {
				return e
			}
			auths = append(auths, v)
		}
		if e = resolver.NewProber(solvers).Solve(auths); e != nil {
			return e
		}
		remote, e = core.Orders.Get(order.URL)
		if e != nil {
			return e
		}
	}
	if remote.Status == acme.StatusReady {
		csr, e := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{DNSNames: p.Domains}, private)
		if e != nil {
			return e
		}
		if _, e = core.Orders.UpdateForCSR(remote.Finalize, csr); e != nil {
			return e
		}
	}
	deadline := time.Now().Add(3 * time.Minute)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		remote, e = core.Orders.Get(order.URL)
		if e != nil {
			return e
		}
		order.Status = remote.Status
		if e = s.DB.Save(&order).Error; e != nil {
			return e
		}
		if remote.Status == acme.StatusValid {
			break
		}
		if remote.Status == acme.StatusInvalid || time.Now().After(deadline) {
			return errors.New("ACME 签发未完成，订单已保留")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	cert, _, e := core.Certificates.Get(remote.Certificate, true)
	if e != nil {
		return e
	}
	rev, e := s.storeIssuedCertificate(op.UserID, r, order, string(cert), privatePEM)
	if e != nil {
		return e
	}
	meta, _ := decode[certContent](rev.Content)
	block, _ := pem.Decode(cert)
	if block != nil {
		leaf, _ := x509.ParseCertificate(block.Bytes)
		if leaf != nil {
			if ari, err := client.Certificate.GetRenewalInfo(certificate.RenewalInfoRequest{Cert: leaf}); err == nil {
				meta.RenewAt = ari.SuggestedWindow.Start.Add(ari.SuggestedWindow.End.Sub(ari.SuggestedWindow.Start) / 2)
				s.DB.Model(&rev).Update("content", JSON(meta))
			}
		}
	}
	return nil
}

type httpChallenge struct {
	s    *Service
	ctx  context.Context
	user uint
	spec model.CertificateSpec
}

func (h *httpChallenge) Present(domain, token, keyAuth string) error {
	return h.update(domain, token, keyAuth, false)
}
func (h *httpChallenge) CleanUp(domain, token, keyAuth string) error {
	return h.update(domain, token, keyAuth, true)
}
func (h *httpChallenge) update(domain, token, keyAuth string, remove bool) error {
	for _, id := range h.spec.InstanceIDs {
		i, e := h.s.Get("instances", id)
		if e != nil {
			return e
		}
		if e = h.s.Authorize(h.user, "instances", i, "web_manage"); e != nil {
			return e
		}
		src, e := h.s.Get("sources", i.SourceID)
		if e != nil {
			return e
		}
		spec, e := decode[model.SourceSpec](src.Spec)
		if e != nil {
			return e
		}
		action := "challenge-present"
		if remove {
			action = "challenge-cleanup"
		}
		_, e = h.s.Remote.Run(h.ctx, RemoteRequest{Action: action, Spec: spec, Files: []model.File{{Path: strings.TrimRight(h.spec.ChallengeRoot, "/") + "/.well-known/acme-challenge/" + token, Content: keyAuth}}})
		if e != nil {
			return e
		}
	}
	if !remove {
		req, e := http.NewRequestWithContext(h.ctx, "GET", "http://"+domain+"/.well-known/acme-challenge/"+token, nil)
		if e != nil {
			return e
		}
		client := &http.Client{Timeout: 15 * time.Second}
		resp, e := client.Do(req)
		if e != nil {
			return e
		}
		defer resp.Body.Close()
		b, e := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if e != nil || resp.StatusCode != 200 || string(b) != keyAuth {
			return errors.New("HTTP 挑战公网入口内容不一致")
		}
	}
	return nil
}

// Keep the scheduler lease while remote preflight runs; a slow cluster must not
// allow another worker to enqueue the same renewal.
func (s *Service) renewalLease(parent context.Context) (context.Context, func(), bool) {
	owner := fmt.Sprint(time.Now().UnixNano())
	const key = "renewal-scheduler"
	if !s.lease(key, owner, 2*time.Minute) {
		return parent, func() {}, false
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r := s.DB.Model(&model.Lease{}).Where("`key` = ? AND owner = ?", key, owner).Update("expires_at", time.Now().UTC().Add(2*time.Minute))
				if r.Error != nil || r.RowsAffected != 1 {
					cancel()
					return
				}
			}
		}
	}()
	return ctx, func() { cancel(); <-done; s.DB.Where("`key` = ? AND owner = ?", key, owner).Delete(&model.Lease{}) }, true
}

// Queue and remember the operation in one commit, including its mutex. A crash
// cannot leave a submitted renewal without its policy's durable task reference.
func (s *Service) submitRenewal(ctx context.Context, policy *model.Resource, p *model.RenewalSpec, plan model.Plan, bindingID uint) error {
	next := *p
	next.DeployOperationIDs = map[uint]uint{}
	for id, op := range p.DeployOperationIDs {
		next.DeployOperationIDs[id] = op
	}
	e := s.DB.Transaction(func(tx *gorm.DB) error {
		local := *s
		local.DB = tx
		op, err := local.Submit(ctx, policy.CreatedBy, plan.ID, plan.IdempotencyKey, true)
		if err != nil {
			return err
		}
		if bindingID == 0 {
			next.PendingOperationID = op.ID
		} else {
			next.DeployOperationIDs[bindingID] = op.ID
		}
		r := tx.Table(model.Tables["renewal-policies"]).Where("id = ? AND revision = ?", policy.ID, policy.Revision).Updates(map[string]any{"spec": JSON(next), "revision": gorm.Expr("revision + 1")})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected != 1 {
			return ErrConflict
		}
		return nil
	})
	if e == nil {
		*p = next
		policy.Revision++
	}
	return e
}

// Renewal policies keep the submitting identity and child task IDs across restarts.
func (s *Service) runRenewals(ctx context.Context) {
	ctx, release, acquired := s.renewalLease(ctx)
	if !acquired {
		return
	}
	defer release()
	var policies []model.Resource
	if s.DB.Table(model.Tables["renewal-policies"]).Find(&policies).Error != nil {
		return
	}
	for _, policy := range policies {
		if ctx.Err() != nil {
			return
		}
		p, e := decode[model.RenewalSpec](policy.Spec)
		if e != nil || !p.Enabled {
			continue
		}
		authctx := ctx
		if p.APIKeyID > 0 {
			authctx = apikey.WithKey(ctx, p.APIKeyID)
		}
		if s.Module(authctx, policy.CreatedBy, "execute") != nil || s.Asset(policy.CreatedBy, "web_certificate", p.CertificateID, "web_certificate_manage") != nil {
			continue
		}
		r, e := s.Get("certificates", p.CertificateID)
		if e != nil {
			continue
		}
		cs, _ := decode[model.CertificateSpec](r.Spec)
		save := func() {
			s.DB.Table(model.Tables["renewal-policies"]).Where("id = ? AND revision = ?", policy.ID, policy.Revision).Updates(map[string]any{"spec": JSON(p), "revision": gorm.Expr("revision + 1")})
		}
		if p.PendingOperationID > 0 {
			var parent model.Operation
			if s.DB.First(&parent, p.PendingOperationID).Error != nil {
				continue
			}
			if parent.Status == "succeeded" {
				complete := true
				if p.AutoDeploy {
					if p.DeployOperationIDs == nil {
						p.DeployOperationIDs = map[uint]uint{}
					}
					for _, bid := range p.BindingIDs {
						binding, err := s.Get("certificate-bindings", bid)
						if err != nil || s.Authorize(policy.CreatedBy, "certificate-bindings", binding, "web_publish") != nil {
							complete = false
							continue
						}
						bs, err := decode[model.BindingSpec](binding.Spec)
						if err != nil || bs.CertificateID != p.CertificateID || bs.VersionID != 0 {
							complete = false
							continue
						}
						// A manual retry can finish a previously failed child. Inspect the
						// actual node versions instead of remaining stuck on that child's ID.
						var observed struct {
							Nodes map[string]any `json:"nodes"`
						}
						json.Unmarshal(binding.Observed, &observed)
						if s.bindingStatus(binding.ClusterID, observed.Nodes, cs.CurrentVersionID) == "deployed" {
							continue
						}
						if oid := p.DeployOperationIDs[bid]; oid > 0 {
							// Failed or uncertain deployment remains visible for an explicit
							// retry; never replay an unknown remote change automatically.
							complete = false
							continue
						}
						plan, err := s.Plan(authctx, policy.CreatedBy, PlanInput{TargetType: "clusters", TargetID: binding.ClusterID, Action: "deploy-certificate", Parameters: Parameters{BindingID: bid}})
						if err == nil && plan.Executable {
							_ = s.submitRenewal(authctx, &policy, &p, plan, bid)
						}
						complete = false
					}
				}
				if complete {
					p.DeployedVersionID = cs.CurrentVersionID
					p.PendingOperationID = 0
					p.DeployOperationIDs = nil
				}
			} else if parent.Status == "failed" || parent.Status == "rolled-back" {
				p.PendingOperationID = 0
				p.NextRunAt = time.Now().UTC().Add(time.Hour)
			}
			save()
			continue
		}
		if time.Now().Before(p.NextRunAt) {
			continue
		}
		var rev model.Revision
		if cs.CurrentVersionID > 0 && s.DB.First(&rev, cs.CurrentVersionID).Error == nil {
			meta, _ := decode[certContent](rev.Content)
			if time.Now().Before(meta.RenewAt) {
				p.NextRunAt = meta.RenewAt
				save()
				continue
			}
		}
		plan, e := s.Plan(authctx, policy.CreatedBy, PlanInput{TargetType: "certificates", TargetID: r.ID, Action: "renew"})
		if e == nil && plan.Executable {
			e = s.submitRenewal(authctx, &policy, &p, plan, 0)
		}
		p.NextRunAt = time.Now().UTC().Add(time.Hour)
		save()
		if e != nil {
			s.DB.Create(&model.Event{UserID: policy.CreatedBy, Kind: "renewal-failed", Message: e.Error()})
		}
	}
}

// Freeze public certificate metadata and private material into the encrypted plan.
func (s *Service) prepareCertificates(u, cid uint, files []model.File, bindingID uint, versions map[uint]uint) ([]model.File, []TLSProbe, error) {
	var bindings []model.Resource
	if e := s.DB.Table(model.Tables["certificate-bindings"]).Where("cluster_id = ?", cid).Find(&bindings).Error; e != nil {
		return nil, nil, e
	}
	var probes []TLSProbe
	text := ""
	for _, f := range files {
		text += "\n" + f.Content
	}
	found := bindingID == 0
	for _, binding := range bindings {
		b, e := decode[model.BindingSpec](binding.Spec)
		if e != nil {
			return nil, nil, e
		}
		if binding.ID != bindingID && !strings.Contains(text, b.CertificatePath) {
			continue
		}
		if binding.ID == bindingID {
			found = true
		}
		additions, e := s.certificateFilesAt(u, cid, binding.ID, versions[binding.ID])
		if e != nil {
			return nil, nil, e
		}
		cert, e := s.Get("certificates", b.CertificateID)
		if e != nil {
			return nil, nil, e
		}
		cs, _ := decode[model.CertificateSpec](cert.Spec)
		vid := b.VersionID
		if versions[binding.ID] > 0 {
			vid = versions[binding.ID]
		}
		if vid == 0 {
			vid = cs.CurrentVersionID
		}
		var rev model.Revision
		if e = s.DB.First(&rev, vid).Error; e != nil {
			return nil, nil, e
		}
		content, e := decode[certContent](rev.Content)
		if e != nil {
			return nil, nil, e
		}
		block, _ := pem.Decode([]byte(content.PEM))
		if block == nil {
			return nil, nil, errors.New("证书内容无效")
		}
		leaf, e := x509.ParseCertificate(block.Bytes)
		if e != nil {
			return nil, nil, e
		}
		if e = leaf.VerifyHostname(b.ServerName); e != nil {
			return nil, nil, errors.New("证书未覆盖 TLS 核验域名")
		}
		probes = append(probes, TLSProbe{BindingID: binding.ID, VersionID: vid, Host: b.VerifyHost, Port: b.VerifyPort, ServerName: b.ServerName, Fingerprint: rev.Digest})
		for _, f := range additions {
			exists := false
			for _, old := range files {
				if old.Path == f.Path {
					if old.Content != f.Content {
						return nil, nil, errors.New("证书文件与绑定版本冲突")
					}
					exists = true
				}
			}
			if !exists {
				files = append(files, f)
			}
		}
	}
	if !found {
		return nil, nil, errors.New("证书绑定不属于目标集群")
	}
	return files, probes, nil
}

// CertificateVersion is public metadata for one stored certificate version.
// Private material is only returned by an explicit, audited export.
type CertificateVersion struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Domains     []string  `json:"domains"`
	Issuer      string    `json:"issuer"`
	Fingerprint string    `json:"fingerprint"`
	NotBefore   time.Time `json:"notBefore"`
	NotAfter    time.Time `json:"notAfter"`
	RenewAt     time.Time `json:"renewAt"`
	Days        int       `json:"days"`
	Current     bool      `json:"current"`
	CreatedBy   uint      `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (s *Service) CertificateVersions(u, id uint) ([]CertificateVersion, uint, error) {
	r, e := s.Get("certificates", id)
	if e != nil {
		return nil, 0, e
	}
	if e = s.Authorize(u, "certificates", r, "web_view"); e != nil {
		return nil, 0, e
	}
	cs, _ := decode[model.CertificateSpec](r.Spec)
	var rows []model.Revision
	if e = s.DB.Where("kind = ? AND resource_id = ?", "certificate", id).Order("id DESC").Limit(100).Find(&rows).Error; e != nil {
		return nil, 0, e
	}
	out := make([]CertificateVersion, 0, len(rows))
	now := time.Now()
	for _, rev := range rows {
		meta, err := decode[certContent](rev.Content)
		if err != nil {
			continue
		}
		out = append(out, CertificateVersion{ID: rev.ID, Name: rev.Name, Domains: meta.Domains, Issuer: meta.Issuer,
			Fingerprint: meta.Fingerprint, NotBefore: meta.NotBefore, NotAfter: meta.NotAfter, RenewAt: meta.RenewAt,
			Days: certificateDaysLeft(meta.NotAfter, now), Current: rev.ID == cs.CurrentVersionID,
			CreatedBy: rev.CreatedBy, CreatedAt: rev.CreatedAt})
	}
	return out, cs.CurrentVersionID, nil
}

func certificateDaysLeft(notAfter, now time.Time) int {
	return int(math.Floor(notAfter.Sub(now).Hours() / 24))
}

// ExportCertificate returns the public chain, and the private key only for an
// administrator with certificate management rights. Reading a key is audited
// because the platform otherwise never exposes stored private material.
func (s *Service) ExportCertificate(ctx context.Context, u, id, versionID uint, includeKey bool) (map[string]string, error) {
	if e := s.Module(ctx, u, "read"); e != nil {
		return nil, e
	}
	r, e := s.Get("certificates", id)
	if e != nil {
		return nil, e
	}
	if e = s.Authorize(u, "certificates", r, "web_view"); e != nil {
		return nil, e
	}
	if includeKey {
		if e = s.Module(ctx, u, "write"); e != nil {
			return nil, e
		}
		if !s.Admin(u) {
			return nil, ErrForbidden
		}
		if e = s.Asset(u, "web_certificate", id, "web_certificate_manage"); e != nil {
			return nil, e
		}
	}
	cs, _ := decode[model.CertificateSpec](r.Spec)
	version := versionID
	if version == 0 {
		version = cs.CurrentVersionID
	}
	if version == 0 {
		return nil, errors.New("证书尚未导入或签发")
	}
	var rev model.Revision
	if e = s.DB.Where("id = ? AND kind = ? AND resource_id = ?", version, "certificate", id).First(&rev).Error; e != nil {
		return nil, e
	}
	meta, e := decode[certContent](rev.Content)
	if e != nil {
		return nil, e
	}
	// Validate stored public material too: older imports may predate strict PEM
	// validation. A read-only export must never return a concatenated private key.
	if e = validatePublicCertificatePEM(meta.PEM); e != nil {
		return nil, e
	}
	out := map[string]string{"certificate": meta.PEM, "fingerprint": meta.Fingerprint,
		"domains": strings.Join(meta.Domains, ","), "name": r.Name, "versionId": fmt.Sprint(rev.ID)}
	if includeKey {
		key, err := unseal(rev.Secret)
		if err != nil {
			return nil, err
		}
		if _, err = certificateKeyPair(meta.PEM, key); err != nil {
			return nil, err
		}
		out["privateKey"] = key
	}
	if e = s.DB.WithContext(ctx).Create(&model.Event{UserID: u, Kind: "certificate-export",
		Message: fmt.Sprintf("certificate=%d version=%d includesPrivateKey=%t", id, rev.ID, includeKey)}).Error; e != nil {
		return nil, errors.New("证书导出审计写入失败，请稍后重试")
	}
	return out, nil
}
