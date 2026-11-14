package service

import (
	"context"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/api/webservice/model"
	"errors"
	"net"
	"net/mail"
	"net/url"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// Issuance plans freeze the account material separately from the certificate
// revision. Account and credential edits must require a new reviewed plan.
func (s *Service) issuanceDependencyDigest(certificate model.Resource) (string, error) {
	p, err := decode[model.CertificateSpec](certificate.Spec)
	if err != nil {
		return "", err
	}
	account, err := s.Get("acme-accounts", p.ACMEAccountID)
	if err != nil {
		return "", errors.New("请选择有效的 ACME 账户")
	}
	type accountIdentity struct {
		ID       uint
		Revision uint64
		Spec     string
		Secret   string
	}
	identity := struct {
		ACME       accountIdentity
		DNS        accountIdentity
		Credential struct {
			ID        uint
			Type      int
			KeyID     string
			KeySecret string
		}
	}{ACME: accountIdentity{account.ID, account.Revision, string(account.Spec), account.Secret}}
	if p.Challenge == "dns-01" {
		dns, err := s.Get("dns-accounts", p.DNSAccountID)
		if err != nil {
			return "", errors.New("请选择有效的 DNS 账户")
		}
		identity.DNS = accountIdentity{dns.ID, dns.Revision, string(dns.Spec), dns.Secret}
		spec, err := decode[model.AccountSpec](dns.Spec)
		if err != nil {
			return "", err
		}
		if spec.CredentialID > 0 {
			var credential configmodel.KeyManage
			if err := s.DB.First(&credential, spec.CredentialID).Error; err != nil {
				return "", errors.New("DNS 引用凭据不存在，请重新选择")
			}
			// Do not marshal KeyManage: its public JSON intentionally redacts the
			// token. Hash the stored material in memory without exposing it.
			identity.Credential.ID = credential.ID
			identity.Credential.Type = credential.KeyType
			identity.Credential.KeyID = credential.KeyID
			identity.Credential.KeySecret = credential.KeySecret
		}
	}
	return digest(JSON(identity)), nil
}

func (s *Service) checkIssuanceDependencies(certificate model.Resource, plan model.Plan) error {
	if plan.TargetType != "certificates" {
		return nil
	}
	snapshot, err := decode[map[string]string](plan.Snapshot)
	if err != nil || snapshot["issuanceDigest"] == "" {
		return errors.New("签发计划缺少账户身份记录，请重新预检")
	}
	current, err := s.issuanceDependencyDigest(certificate)
	if err != nil {
		return err
	}
	if current != snapshot["issuanceDigest"] {
		return ErrConflict
	}
	return nil
}

// canonicalIssuanceDomains returns a stable identity for the order. DNS names
// are case-insensitive; SAN order must not accidentally cause order replay.
func canonicalIssuanceDomains(domains []string) ([]string, error) {
	if len(domains) == 0 || len(domains) > 100 {
		return nil, errors.New("证书申请需要 1 至 100 个域名")
	}
	result := make([]string, 0, len(domains))
	seen := make(map[string]bool)
	for _, raw := range domains {
		domain := strings.ToLower(strings.TrimSpace(raw))
		plain := strings.TrimPrefix(domain, "*.")
		if len(plain) == 0 || len(plain) > 253 || net.ParseIP(plain) != nil || strings.ContainsAny(plain, "* /\\\t\r\n") {
			return nil, errors.New("ACME 申请需要有效的 DNS 域名")
		}
		labels := strings.Split(plain, ".")
		if len(labels) < 2 {
			return nil, errors.New("ACME 申请需要完整域名")
		}
		for _, label := range labels {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return nil, errors.New("ACME 域名格式无效")
			}
			for _, c := range label {
				if c != '-' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
					return nil, errors.New("ACME 域名格式无效，请将国际化域名转换为 Punycode")
				}
			}
		}
		if seen[domain] {
			return nil, errors.New("证书申请域名重复")
		}
		seen[domain] = true
		result = append(result, domain)
	}
	sort.Strings(result)
	return result, nil
}

func validateACMEAccount(spec model.AccountSpec) error {
	directory, err := url.Parse(spec.DirectoryURL)
	if err != nil || directory.Scheme != "https" || directory.Hostname() == "" || directory.User != nil || directory.Fragment != "" {
		return errors.New("ACME 账户需要有效的 HTTPS 服务地址")
	}
	email, err := mail.ParseAddress(strings.TrimSpace(spec.Email))
	if err != nil || email.Address != strings.TrimSpace(spec.Email) {
		return errors.New("ACME 账户需要有效的联系邮箱")
	}
	return nil
}

// Account registration/key creation occurs before this digest is recorded.
// DNS credentials are intentionally excluded: fixing a DNS token is allowed
// when retrying the same CA order for the same ACME identity and domain set.
func issuanceOrderIdentity(account model.Resource, domains []string) (string, error) {
	a, err := decode[model.AccountSpec](account.Spec)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalIssuanceDomains(domains)
	if err != nil {
		return "", err
	}
	return digest(JSON(struct {
		AccountID uint
		Directory string
		Email     string
		KeyDigest string
		Domains   []string
	}{account.ID, a.DirectoryURL, a.Email, digest([]byte(account.Secret)), canonical})), nil
}

func checkOrderIdentity(order model.Order, identity string) error {
	if order.IdentityDigest == "" {
		return errors.New("已有 ACME 订单缺少签发身份记录，请先核验该订单，不能自动恢复")
	}
	if identity == "" || order.IdentityDigest != identity {
		return errors.New("已有 ACME 订单的账户或域名已变化，请核验原订单或创建新的证书申请")
	}
	return nil
}

func (s *Service) checkIssuanceDNS(ctx context.Context, certificate model.Resource) error {
	spec, err := decode[model.CertificateSpec](certificate.Spec)
	if err != nil || spec.Challenge != "dns-01" {
		return err
	}
	return s.ValidateCloudflareDomains(ctx, spec.DNSAccountID, spec.Domains)
}

func (s *Service) preflightIssuance(ctx context.Context, user uint, certificate model.Resource) (string, error) {
	if err := s.validateIssuance(user, certificate); err != nil {
		return "", err
	}
	before, err := s.issuanceDependencyDigest(certificate)
	if err != nil {
		return "", err
	}
	if err = s.checkIssuanceDNS(ctx, certificate); err != nil {
		return "", err
	}
	after, err := s.issuanceDependencyDigest(certificate)
	if err != nil {
		return "", err
	}
	if before != after {
		return "", ErrConflict
	}
	return before, nil
}

// A successfully issued version and its completed order are one durable result.
// Failure to finish the order must not leave a version that a retry duplicates.
func (s *Service) storeIssuedCertificate(user uint, certificate model.Resource, order model.Order, chain, key string) (model.Revision, error) {
	var version model.Revision
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		local := *s
		local.DB = tx
		var err error
		version, err = local.storeCertificate(user, certificate, chain, key)
		if err != nil {
			return err
		}
		if order.IdentityDigest == "" {
			return ErrConflict
		}
		updated := tx.Model(&model.Order{}).Where("id = ? AND certificate_id = ? AND identity_digest = ? AND status = ?", order.ID, certificate.ID, order.IdentityDigest, "valid").Update("status", "stored")
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrConflict
		}
		return nil
	})
	if err != nil {
		return model.Revision{}, err
	}
	return version, nil
}
