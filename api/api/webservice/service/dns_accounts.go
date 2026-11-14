package service

import (
	"context"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/api/webservice/model"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

var dnsCredentialTypes = map[string]int{"alidns": 1, "tencentcloud": 2, "cloudflare": 6}

type DNSCredentialOption struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// DNSCredentials exposes names, never credential identifiers or secret values.
func (s *Service) DNSCredentials(ctx context.Context, u uint, provider string) ([]DNSCredentialOption, error) {
	if e := s.Module(ctx, u, "read"); e != nil || !s.Admin(u) {
		return nil, ErrForbidden
	}
	typ, ok := dnsCredentialTypes[provider]
	if !ok {
		return nil, errors.New("DNS 服务商无效")
	}
	var rows []configmodel.KeyManage
	if err := s.DB.WithContext(ctx).Select("id", "remark").Where("key_type = ?", typ).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]DNSCredentialOption, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.Remark)
		if name == "" {
			name = fmt.Sprintf("%s #%d", provider, row.ID)
		}
		out = append(out, DNSCredentialOption{ID: row.ID, Name: name, Provider: provider})
	}
	return out, nil
}

type dnsCredential struct {
	Key    string `json:"key,omitempty"`
	Secret string `json:"secret,omitempty"`
	Token  string `json:"token,omitempty"`
}

func parseDNSCredential(provider, raw string) (dnsCredential, error) {
	var credential dnsCredential
	if len(raw) > 16384 {
		return credential, errors.New("DNS 凭据内容超过大小限制")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&credential); err != nil {
		return credential, errors.New("DNS 凭据格式无效")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return credential, errors.New("DNS 凭据格式无效")
	}
	credential.Key = strings.TrimSpace(credential.Key)
	credential.Secret = strings.TrimSpace(credential.Secret)
	credential.Token = strings.TrimSpace(credential.Token)
	if provider == "cloudflare" {
		if credential.Key != "" || credential.Secret != "" || !validDNSToken(credential.Token) {
			return credential, errors.New("请填写有效的 Cloudflare API Token")
		}
	} else if credential.Token != "" || !validDNSSecret(credential.Key) || !validDNSSecret(credential.Secret) {
		return credential, errors.New("请填写完整的 DNS 密钥 ID 和 Secret")
	}
	return credential, nil
}

func validDNSSecret(value string) bool {
	return value != "" && len(value) <= 4096 && !strings.ContainsAny(value, "\x00\r\n\t")
}

var dnsTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func validDNSToken(value string) bool {
	return validDNSSecret(value) && dnsTokenPattern.MatchString(value)
}

// dnsAccountCredentials is shared by connection diagnostics and lego. There is
// no fallback to process environment credentials when an account is incomplete.
func (s *Service) dnsAccountCredentials(r model.Resource) (model.AccountSpec, string, string, error) {
	p, err := decode[model.AccountSpec](r.Spec)
	if err != nil {
		return p, "", "", errors.New("DNS 账户配置无效")
	}
	typ, ok := dnsCredentialTypes[p.Provider]
	if !ok {
		return p, "", "", errors.New("DNS 服务商无效")
	}
	if p.CredentialID > 0 {
		var credential configmodel.KeyManage
		if err := s.DB.First(&credential, p.CredentialID).Error; err != nil {
			return p, "", "", errors.New("引用的配置中心 DNS 凭据不存在")
		}
		if credential.KeyType != typ {
			return p, "", "", errors.New("云凭据类型与 DNS 服务商不匹配")
		}
		key, secret, err := credential.DecryptKeys()
		if err != nil {
			return p, "", "", errors.New("DNS 凭据解密失败，请检查部署密钥和凭据配置")
		}
		key, secret = strings.TrimSpace(key), strings.TrimSpace(secret)
		if p.Provider == "cloudflare" {
			if !validDNSToken(secret) {
				return p, "", "", errors.New("引用的 Cloudflare API Token 无效")
			}
		} else if !validDNSSecret(key) || !validDNSSecret(secret) {
			return p, "", "", errors.New("引用的 DNS 凭据不完整")
		}
		return p, key, secret, nil
	}
	if r.Secret == "" {
		return p, "", "", errors.New("请配置 DNS 凭据：选择配置中心凭据或填写专用凭据")
	}
	raw, err := unseal(r.Secret)
	if err != nil {
		return p, "", "", errors.New("DNS 凭据解密失败，请检查部署密钥和凭据配置")
	}
	credential, err := parseDNSCredential(p.Provider, raw)
	if err != nil {
		return p, "", "", err
	}
	if p.Provider == "cloudflare" {
		return p, "", credential.Token, nil
	}
	return p, credential.Key, credential.Secret, nil
}

func (s *Service) prepareDNSAccount(r *model.Resource, old model.Resource, secretInput string) (bool, error) {
	p, err := decode[model.AccountSpec](r.Spec)
	if err != nil {
		return false, errors.New("DNS 账户配置无效")
	}
	previous, _ := decode[model.AccountSpec](old.Spec)
	changed := old.ID == 0 || p.Provider != previous.Provider || p.CredentialID != previous.CredentialID || secretInput != ""
	if p.CredentialID > 0 {
		if secretInput != "" {
			return false, errors.New("配置中心凭据与专用 DNS 凭据只能选择一种")
		}
		r.Secret = ""
	} else if secretInput != "" {
		credential, err := parseDNSCredential(p.Provider, secretInput)
		if err != nil {
			return false, err
		}
		r.Secret, err = seal(string(JSON(credential)))
		if err != nil {
			return false, err
		}
	} else if old.ID == 0 || previous.CredentialID > 0 || previous.Provider != p.Provider {
		return false, errors.New("请填写专用 DNS 凭据；切换来源或服务商后不能沿用旧凭据")
	}
	_, _, _, err = s.dnsAccountCredentials(*r)
	return changed, err
}

type CloudflareZone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type DNSZonesResult struct {
	Zones     []CloudflareZone `json:"zones"`
	Total     int              `json:"total"`
	Truncated bool             `json:"truncated"`
	CheckedAt time.Time        `json:"checkedAt"`
}

type DNSAccountCheck struct {
	Valid       bool            `json:"valid"`
	Domain      string          `json:"domain,omitempty"`
	CheckedAt   time.Time       `json:"checkedAt"`
	Checks      []Check         `json:"checks"`
	Zone        *CloudflareZone `json:"zone,omitempty"`
	ZonesCount  int             `json:"zonesCount"`
	WriteAccess string          `json:"writeAccess"`
}

type cloudflareEnvelope struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Errors  []struct {
		Code int `json:"code"`
	} `json:"errors"`
	ResultInfo struct {
		Page       int `json:"page"`
		TotalPages int `json:"total_pages"`
		TotalCount int `json:"total_count"`
	} `json:"result_info"`
}

// Only a fixed HTTPS API origin receives the bearer credential. Redirects are
// rejected even when tests supply a custom transport; error bodies are not echoed.
func (s *Service) cloudflareGET(ctx context.Context, token, path string, query url.Values) (cloudflareEnvelope, error) {
	var envelope cloudflareEnvelope
	u := url.URL{Scheme: "https", Host: "api.cloudflare.com", Path: "/client/v4" + path, RawQuery: query.Encode()}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return envelope, errors.New("Cloudflare 请求参数无效")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	if s.CloudflareHTTP != nil {
		*client = *s.CloudflareHTTP
		if client.Timeout <= 0 || client.Timeout > 15*time.Second {
			client.Timeout = 15 * time.Second
		}
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("redirect rejected") }
	response, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return envelope, ctx.Err()
		}
		return envelope, errors.New("无法连接 Cloudflare API，请检查网络、代理或稍后重试")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, (2<<20)+1))
	if err != nil || len(data) > 2<<20 {
		return envelope, errors.New("Cloudflare 响应读取失败或超过大小限制")
	}
	if err = json.Unmarshal(data, &envelope); err != nil {
		return envelope, errors.New("Cloudflare API 返回了无效响应")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || !envelope.Success {
		code := 0
		if len(envelope.Errors) > 0 {
			code = envelope.Errors[0].Code
		}
		message := "Cloudflare 请求失败，请检查账户配置"
		if response.StatusCode == 401 || response.StatusCode == 403 || code == 10000 || code == 9109 || code == 6003 {
			message = "Cloudflare 拒绝访问，请检查 API Token、Zone:Read 权限和授权域名范围"
		} else if response.StatusCode == 429 {
			message = "Cloudflare 请求频率受限，请稍后重试"
		}
		return envelope, fmt.Errorf("%s（HTTP %d，代码 %d）", message, response.StatusCode, code)
	}
	return envelope, nil
}

func (s *Service) cloudflareZones(ctx context.Context, token string) (DNSZonesResult, error) {
	out := DNSZonesResult{Zones: []CloudflareZone{}, CheckedAt: time.Now().UTC()}
	seen := map[string]bool{}
	for page := 1; page <= 10; page++ {
		response, err := s.cloudflareGET(ctx, token, "/zones", url.Values{"page": {strconv.Itoa(page)}, "per_page": {"50"}, "order": {"name"}, "direction": {"asc"}})
		if err != nil {
			return out, err
		}
		var zones []CloudflareZone
		if err := json.Unmarshal(response.Result, &zones); err != nil || len(zones) > 50 {
			return out, errors.New("Cloudflare 域名列表格式无效")
		}
		for _, zone := range zones {
			_, domainErr := normalizeCloudflareDomain(zone.Name)
			if !cloudflareID.MatchString(zone.ID) || domainErr != nil {
				return out, errors.New("Cloudflare 域名列表格式无效")
			}
			if !seen[zone.ID] {
				out.Zones = append(out.Zones, zone)
				seen[zone.ID] = true
			}
		}
		out.Total = response.ResultInfo.TotalCount
		if out.Total < len(out.Zones) {
			out.Total = len(out.Zones)
		}
		if response.ResultInfo.TotalPages <= page || len(zones) == 0 {
			break
		}
		out.Truncated = page == 10
	}
	return out, nil
}

var cloudflareID = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)

func normalizeCloudflareDomain(domain string) (string, error) {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	domain = strings.TrimPrefix(domain, "*.")
	if domain == "" || len(domain) > 253 || strings.Contains(domain, "*") {
		return "", errors.New("请输入有效的 DNS 域名")
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return "", errors.New("请输入有效的 DNS 域名")
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", errors.New("请输入有效的 DNS 域名")
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return "", errors.New("请输入有效的 DNS 域名，国际化域名请使用 Punycode")
			}
		}
	}
	return domain, nil
}

// MatchCloudflareZone picks the most specific visible zone, with label-boundary
// matching; a zone named example.com must not authorize notexample.com.
func MatchCloudflareZone(domain string, zones []CloudflareZone) (*CloudflareZone, error) {
	normalized, err := normalizeCloudflareDomain(domain)
	if err != nil {
		return nil, err
	}
	var matched *CloudflareZone
	for _, zone := range zones {
		name := strings.ToLower(strings.TrimSuffix(zone.Name, "."))
		if normalized == name || strings.HasSuffix(normalized, "."+name) {
			if matched == nil || len(name) > len(matched.Name) {
				copy := zone
				matched = &copy
			}
		}
	}
	if matched == nil {
		return nil, errors.New("该域名不在此 Cloudflare Token 可访问的 Zone 范围内")
	}
	return matched, nil
}

func (s *Service) cloudflareChallengeRecord(ctx context.Context, token string, zone CloudflareZone, domain string) error {
	base, err := normalizeCloudflareDomain(domain)
	if err != nil {
		return err
	}
	response, err := s.cloudflareGET(ctx, token, "/zones/"+zone.ID+"/dns_records", url.Values{"per_page": {"1"}, "type": {"CNAME"}, "name": {"_acme-challenge." + base}})
	if err != nil {
		return err
	}
	var records []json.RawMessage
	if err := json.Unmarshal(response.Result, &records); err != nil {
		return errors.New("Cloudflare DNS 记录响应格式无效")
	}
	if len(records) > 0 {
		return errors.New("此申请入口暂不支持 ACME CNAME 委派，请使用直接管理的验证域名")
	}
	return nil
}

// ValidateCloudflareDomains is used after the caller authorizes a certificate
// and its administrator-bound signing identities. It deliberately has no
// administrator gate of its own, so an authorized certificate operator can
// use a previously bound DNS account without gaining credential management.
func (s *Service) ValidateCloudflareDomains(ctx context.Context, id uint, domains []string) error {
	r, err := s.Get("dns-accounts", id)
	if err != nil {
		return errors.New("请选择有效的 DNS 账户")
	}
	p, _, token, err := s.dnsAccountCredentials(r)
	if err != nil {
		return err
	}
	if p.Provider != "cloudflare" {
		return nil
	}
	if len(domains) == 0 || len(domains) > 100 {
		return errors.New("DNS 验证需要 1 到 100 个域名")
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	zones, err := s.cloudflareZones(ctx, token)
	if err != nil {
		return err
	}
	checked := map[string]bool{}
	challengeChecked := map[string]bool{}
	for _, domain := range domains {
		zone, err := MatchCloudflareZone(domain, zones.Zones)
		if err != nil {
			if zones.Truncated {
				return errors.New("域名列表超过 500 个，当前未找到目标域名；请缩小 Token 的授权范围")
			}
			return fmt.Errorf("%s: %w", domain, err)
		}
		if zone.Status != "active" {
			return fmt.Errorf("%s: Cloudflare Zone 尚未激活", domain)
		}
		if !checked[zone.ID] {
			if _, err := s.cloudflareGET(ctx, token, "/zones/"+zone.ID+"/dns_records", url.Values{"per_page": {"1"}, "type": {"TXT"}}); err != nil {
				return err
			}
			checked[zone.ID] = true
		}
		base, _ := normalizeCloudflareDomain(domain)
		if !challengeChecked[base] {
			if err := s.cloudflareChallengeRecord(ctx, token, *zone, domain); err != nil {
				return err
			}
			challengeChecked[base] = true
		}
	}
	return nil
}

func (s *Service) cloudflareAccount(ctx context.Context, u, id uint, action string) (model.Resource, string, error) {
	if e := s.Module(ctx, u, action); e != nil || !s.Admin(u) {
		return model.Resource{}, "", ErrForbidden
	}
	r, err := s.Get("dns-accounts", id)
	if err != nil {
		return r, "", err
	}
	p, _, token, err := s.dnsAccountCredentials(r)
	if err != nil {
		return r, "", err
	}
	if p.Provider != "cloudflare" {
		return r, "", errors.New("当前只支持 Cloudflare 的连接检查与域名发现")
	}
	return r, token, nil
}

func (s *Service) DNSAccountZones(ctx context.Context, u, id uint) (DNSZonesResult, error) {
	_, token, err := s.cloudflareAccount(ctx, u, id, "read")
	if err != nil {
		return DNSZonesResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	return s.cloudflareZones(ctx, token)
}

func (s *Service) CheckDNSAccount(ctx context.Context, u, id uint, domain string) (DNSAccountCheck, error) {
	out := DNSAccountCheck{CheckedAt: time.Now().UTC(), Checks: []Check{}, WriteAccess: "unverified", Domain: strings.TrimSpace(domain)}
	if domain != "" {
		if _, err := normalizeCloudflareDomain(domain); err != nil {
			return out, err
		}
	}
	r, token, err := s.cloudflareAccount(ctx, u, id, "write")
	if err != nil {
		return out, err
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	out.Checks = append(out.Checks, Check{Name: "凭据配置", Passed: true, Message: "已加载账户凭据，不返回 Token"})
	zones, err := s.cloudflareZones(ctx, token)
	if err != nil {
		out.Checks = append(out.Checks, Check{Name: "Cloudflare 域名读取", Message: err.Error()})
	} else {
		out.ZonesCount = zones.Total
		out.Checks = append(out.Checks, Check{Name: "Cloudflare 域名读取", Passed: true, Message: fmt.Sprintf("可访问 %d 个 Zone", zones.Total)})
		if domain != "" {
			out.Zone, err = MatchCloudflareZone(domain, zones.Zones)
			message := "已找到域名所属 Zone"
			if err != nil {
				message = err.Error()
				if zones.Truncated {
					message = "域名列表超过 500 个，当前未找到目标域名；请缩小 Token 的授权范围"
				}
			}
			out.Checks = append(out.Checks, Check{Name: "域名授权范围", Passed: err == nil, Message: message})
		} else if len(zones.Zones) > 0 {
			// An account check samples one active zone, a domain-specific check
			// always checks that domain's zone instead.
			sort.SliceStable(zones.Zones, func(i, j int) bool { return zones.Zones[i].Status == "active" && zones.Zones[j].Status != "active" })
			out.Zone = &zones.Zones[0]
		} else {
			out.Checks = append(out.Checks, Check{Name: "域名授权范围", Message: "Token 没有可访问的 Zone，请检查授权范围"})
		}
		if out.Zone != nil {
			active := out.Zone.Status == "active"
			message := "Zone 已激活"
			if !active {
				message = "Zone 尚未激活，请在 Cloudflare 检查域名状态与权威 DNS"
			}
			out.Checks = append(out.Checks, Check{Name: "Zone 状态", Passed: active, Message: message})
			_, readErr := s.cloudflareGET(ctx, token, "/zones/"+out.Zone.ID+"/dns_records", url.Values{"per_page": {"1"}, "type": {"TXT"}})
			message = "DNS 记录可读取；此只读检查不验证 DNS:Edit 写权限"
			if readErr != nil {
				message = readErr.Error()
			}
			out.Checks = append(out.Checks, Check{Name: "DNS 记录读取", Passed: readErr == nil, Message: message})
			if domain != "" && active && readErr == nil {
				recordErr := s.cloudflareChallengeRecord(ctx, token, *out.Zone, domain)
				message := "未发现 ACME 验证记录的 CNAME 委派"
				if recordErr != nil {
					message = recordErr.Error()
				}
				out.Checks = append(out.Checks, Check{Name: "ACME 验证记录", Passed: recordErr == nil, Message: message})
			}
		}
	}
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	out.Valid = true
	for _, check := range out.Checks {
		out.Valid = out.Valid && check.Passed
	}
	// Recheck revocation after network I/O, and do not attach old results to an
	// account whose provider/token was changed while this request was running.
	if err := s.Module(ctx, u, "write"); err != nil || !s.Admin(u) {
		return out, ErrForbidden
	}
	status := "healthy"
	if !out.Valid {
		status = "unavailable"
	}
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Referenced config-center credentials have their own lifecycle and may
		// rotate without changing this account's revision.
		local := *s
		local.DB = tx
		_, _, currentToken, credentialErr := local.dnsAccountCredentials(r)
		if credentialErr != nil || currentToken != token {
			return ErrConflict
		}
		result := tx.Table(model.Tables["dns-accounts"]).Where("id = ? AND revision = ?", r.ID, r.Revision).
			Where("collected_at IS NULL OR collected_at <= ?", out.CheckedAt).
			Updates(map[string]any{"observed": JSON(out), "collected_at": out.CheckedAt, "status": status})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrConflict
		}
		return tx.Create(&model.Event{UserID: u, Kind: "dns-account-check", Message: fmt.Sprintf("dns_account=%d valid=%t", r.ID, out.Valid)}).Error
	})
	return out, err
}
