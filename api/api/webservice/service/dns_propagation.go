package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/miekg/dns"
	"net"
	"strings"
	"time"
)

type dnsExchange func(context.Context, *dns.Msg, string) (*dns.Msg, error)

// Some networks redirect UDP/53 and return recursive cache entries even when
// addressed to an authoritative server. TCP is a fallback, never a skipped check.
func dnsPropagationFallback(ctx context.Context) dns01.ChallengeOption {
	return dns01.WrapPreCheck(func(domain, fqdn, value string, check dns01.PreCheckFunc) (bool, error) {
		ok, err := check(fqdn, value)
		if ok && err == nil {
			return true, nil
		}
		exchange := func(ctx context.Context, msg *dns.Msg, address string) (*dns.Msg, error) {
			client := &dns.Client{Net: "tcp", Timeout: 4 * time.Second}
			res, _, e := client.ExchangeContext(ctx, msg, address)
			return res, e
		}
		fallback, e := checkDNSPropagationTCP(ctx, fqdn, value, exchange)
		if e != nil {
			return false, fmt.Errorf("DNS 传播检查尚未通过（UDP 与 TCP）：%w", e)
		}
		return fallback, nil
	})
}
func checkDNSPropagationTCP(ctx context.Context, fqdn, value string, exchange dnsExchange) (bool, error) {
	fqdn = dns.Fqdn(strings.ToLower(fqdn))
	if !strings.HasPrefix(fqdn, "_acme-challenge.") || value == "" {
		return false, errors.New("无效的 DNS 挑战")
	}
	resolvers := []string{"1.1.1.1:53", "8.8.8.8:53"}
	query := func(name string, kind uint16, server string, recursive bool) (*dns.Msg, error) {
		m := new(dns.Msg)
		m.SetQuestion(name, kind)
		m.RecursionDesired = recursive
		reply, e := exchange(ctx, m, server)
		if e != nil {
			return nil, e
		}
		if reply == nil || reply.Truncated {
			return nil, errors.New("DNS 返回不完整")
		}
		return reply, nil
	}
	var zone string
	for _, resolver := range resolvers {
		reply, e := query(fqdn, dns.TypeSOA, resolver, true)
		if e != nil {
			continue
		}
		for _, record := range append(reply.Answer, reply.Ns...) {
			if soa, ok := record.(*dns.SOA); ok && dns.IsSubDomain(soa.Hdr.Name, fqdn) {
				zone = soa.Hdr.Name
				break
			}
		}
		if zone != "" {
			break
		}
	}
	if zone == "" {
		return false, errors.New("TCP DNS 未找到权威区域")
	}
	var names []string
	for _, resolver := range resolvers {
		reply, e := query(zone, dns.TypeNS, resolver, true)
		if e != nil || reply.Rcode != dns.RcodeSuccess {
			continue
		}
		for _, record := range reply.Answer {
			if ns, ok := record.(*dns.NS); ok && strings.EqualFold(ns.Hdr.Name, zone) {
				names = append(names, ns.Ns)
			}
		}
		if len(names) > 0 {
			break
		}
	}
	if len(names) == 0 || len(names) > 16 {
		return false, errors.New("TCP DNS 权威服务器列表无效")
	}
	for _, name := range names {
		reply, e := query(fqdn, dns.TypeTXT, net.JoinHostPort(name, "53"), false)
		if e != nil {
			return false, fmt.Errorf("权威 DNS %s TCP 查询失败: %w", name, e)
		}
		if !reply.Authoritative || reply.Rcode != dns.RcodeSuccess {
			return false, fmt.Errorf("权威 DNS %s 尚未提供有效响应", name)
		}
		found := false
		for _, record := range reply.Answer {
			if _, ok := record.(*dns.CNAME); ok {
				return false, errors.New("验证记录存在 CNAME 委派，请重新检查域名配置")
			}
			if txt, ok := record.(*dns.TXT); ok && strings.EqualFold(txt.Hdr.Name, fqdn) && strings.Join(txt.Txt, "") == value {
				found = true
			}
		}
		if !found {
			return false, fmt.Errorf("权威 DNS %s 尚未返回本次挑战记录", name)
		}
	}
	return true, nil
}
