package service

import (
	"context"
	"errors"
	"github.com/miekg/dns"
	"strings"
	"testing"
)

func TestDNSPropagationTCPRequiresEveryAuthorityAndExactChallenge(t *testing.T) {
	for _, tc := range []struct {
		name     string
		bad      string
		accepted bool
	}{{"correct", "", true}, {"wrong-value", "value", false}, {"cached-nonauthoritative", "recursive", false}, {"one-authority-missing", "second", false}, {"delegation", "cname", false}, {"network", "network", false}, {"wrong-record-name", "name", false}} {
		t.Run(tc.name, func(t *testing.T) {
			exchange := func(_ context.Context, msg *dns.Msg, server string) (*dns.Msg, error) {
				r := new(dns.Msg)
				r.SetReply(msg)
				q := msg.Question[0]
				switch q.Qtype {
				case dns.TypeSOA:
					r.Ns = []dns.RR{&dns.SOA{Hdr: dns.RR_Header{Name: "example.org.", Rrtype: dns.TypeSOA}}}
				case dns.TypeNS:
					r.Answer = []dns.RR{&dns.NS{Hdr: dns.RR_Header{Name: "example.org.", Rrtype: dns.TypeNS}, Ns: "a.example.org."}, &dns.NS{Hdr: dns.RR_Header{Name: "example.org.", Rrtype: dns.TypeNS}, Ns: "b.example.org."}}
				case dns.TypeTXT:
					if tc.bad == "network" {
						return nil, errors.New("unreachable")
					}
					r.Authoritative = tc.bad != "recursive"
					v := "expected"
					n := q.Name
					if tc.bad == "value" || (tc.bad == "second" && strings.HasPrefix(server, "b.")) {
						v = "old"
					}
					if tc.bad == "name" {
						n = "_acme-challenge.other.example.org."
					}
					r.Answer = []dns.RR{&dns.TXT{Hdr: dns.RR_Header{Name: n, Rrtype: dns.TypeTXT}, Txt: []string{v}}}
					if tc.bad == "cname" {
						r.Answer = append(r.Answer, &dns.CNAME{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeCNAME}, Target: "elsewhere."})
					}
				}
				return r, nil
			}
			accepted, err := checkDNSPropagationTCP(context.Background(), "_acme-challenge.test.example.org.", "expected", exchange)
			if accepted != tc.accepted || (tc.accepted && err != nil) {
				t.Fatal(accepted, err)
			}
		})
	}
}
