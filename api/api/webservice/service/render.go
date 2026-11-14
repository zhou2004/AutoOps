package service

import (
	"dodevops-api/api/webservice/model"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var domainPattern = regexp.MustCompile(`^(\*\.)?[a-zA-Z0-9](?:[a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)
var safeValue = regexp.MustCompile(`^[a-zA-Z0-9_./:@?=&%+,$~ -]+$`)

func nginxString(v string) (string, error) {
	if strings.ContainsAny(v, "\r\n\x00\"\\;{}") {
		return "", errors.New("配置值包含非法字符")
	}
	return "\"" + v + "\"", nil
}

type UpstreamMember struct {
	Address     string `json:"address"`
	Weight      int    `json:"weight"`
	Backup      bool   `json:"backup"`
	Down        bool   `json:"down"`
	MaxFails    int    `json:"maxFails"`
	FailTimeout int    `json:"failTimeout"`
}
type UpstreamSpec struct {
	Name     string           `json:"name"`
	Strategy string           `json:"strategy"`
	Members  []UpstreamMember `json:"members"`
	Path     string           `json:"path"`
}

func RenderUpstream(p UpstreamSpec) (string, error) {
	if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(p.Name) || !validPath(p.Path) || len(p.Members) == 0 {
		return "", errors.New("上游名称、路径或成员无效")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "upstream %s {\n", p.Name)
	switch p.Strategy {
	case "", "round_robin":
	case "least_conn", "ip_hash":
		fmt.Fprintf(&b, "    %s;\n", p.Strategy)
	default:
		return "", errors.New("上游策略无效")
	}
	for _, m := range p.Members {
		if !regexp.MustCompile(`^[A-Za-z0-9.\[\]:_-]+$`).MatchString(m.Address) || m.Weight < 1 || m.Weight > 1000 {
			return "", errors.New("上游地址或权重无效")
		}
		if m.MaxFails < 0 || m.MaxFails > 1000 || m.FailTimeout < 0 || m.FailTimeout > 3600 {
			return "", errors.New("上游失败次数或失败等待时间无效")
		}
		if m.Backup && p.Strategy == "ip_hash" {
			return "", errors.New("IP 哈希策略不支持 backup 成员")
		}
		fmt.Fprintf(&b, "    server %s weight=%d", m.Address, m.Weight)
		if m.MaxFails > 0 {
			fmt.Fprintf(&b, " max_fails=%d", m.MaxFails)
		}
		if m.FailTimeout > 0 {
			fmt.Fprintf(&b, " fail_timeout=%ds", m.FailTimeout)
		}
		if m.Backup {
			b.WriteString(" backup")
		}
		if m.Down {
			b.WriteString(" down")
		}
		b.WriteString(";\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}
func RenderSite(p model.SiteSpec) (string, error) {
	if !validPath(p.Path) {
		return "", errors.New("站点配置路径无效")
	}
	if p.Mode == "raw" {
		if len(p.Raw) == 0 {
			return "", errors.New("原文配置为空")
		}
		return p.Raw, nil
	}
	if len(p.Domains) == 0 || len(p.Ports) == 0 {
		return "", errors.New("请输入站点域名和监听端口")
	}
	for _, d := range p.Domains {
		if !domainPattern.MatchString(d) {
			return "", errors.New("站点域名无效")
		}
	}
	var b strings.Builder
	b.WriteString("# Managed by AutoOps\nserver {\n")
	for _, port := range p.Ports {
		if port < 1 || port > 65535 {
			return "", errors.New("监听端口无效")
		}
		tls := ""
		if p.TLS {
			tls = " ssl"
		}
		fmt.Fprintf(&b, "    listen %d%s;\n", port, tls)
	}
	fmt.Fprintf(&b, "    server_name %s;\n", strings.Join(p.Domains, " "))
	if p.TLS {
		if p.CertificateID == 0 || !validPath(p.CertificatePath) || !validPath(p.KeyPath) {
			return "", errors.New("HTTPS 站点需要已签发的证书版本")
		}
		certPath, err := nginxString(p.CertificatePath)
		if err != nil {
			return "", err
		}
		keyPath, err := nginxString(p.KeyPath)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "    ssl_certificate %s;\n    ssl_certificate_key %s;\n", certPath, keyPath)
	}
	if p.MaxBodySize != "" {
		if !regexp.MustCompile(`^[0-9]+[kKmMgG]?$`).MatchString(p.MaxBodySize) {
			return "", errors.New("请求体大小无效")
		}
		fmt.Fprintf(&b, "    client_max_body_size %s;\n", p.MaxBodySize)
	}
	if p.AccessLog != "" {
		if !validPath(p.AccessLog) {
			return "", errors.New("日志路径无效")
		}
		q, _ := nginxString(p.AccessLog)
		fmt.Fprintf(&b, "    access_log %s;\n", q)
	}
	if p.ErrorLog != "" {
		if !validPath(p.ErrorLog) {
			return "", errors.New("错误日志路径无效")
		}
		q, e := nginxString(p.ErrorLog)
		if e != nil {
			return "", e
		}
		fmt.Fprintf(&b, "    error_log %s;\n", q)
	}
	for _, l := range p.Locations {
		if !strings.HasPrefix(l.Path, "/") || strings.ContainsAny(l.Path, " \t;{}\r\n\"\\") {
			return "", errors.New("路径规则无效")
		}
		fmt.Fprintf(&b, "    location %s {\n", l.Path)
		target, e := nginxString(l.Target)
		if e != nil {
			return "", e
		}
		switch l.Kind {
		case "static":
			if !validPath(l.Target) {
				return "", errors.New("静态目录无效")
			}
			fmt.Fprintf(&b, "        root %s;\n", target)
		case "proxy":
			v, e := url.Parse(l.Target)
			if e != nil || v.Host == "" || (v.Scheme != "http" && v.Scheme != "https") {
				return "", errors.New("反向代理目标无效")
			}
			fmt.Fprintf(&b, "        proxy_pass %s;\n        proxy_http_version 1.1;\n", target)
			if v.Scheme == "https" {
				b.WriteString("        proxy_ssl_server_name on;\n")
			}
			if l.Websocket {
				b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n        proxy_set_header Connection \"upgrade\";\n")
			}
			if l.Timeout > 0 {
				if l.Timeout > 3600 {
					return "", errors.New("代理超时超过限制")
				}
				fmt.Fprintf(&b, "        proxy_read_timeout %ds;\n", l.Timeout)
			}
		case "redirect":
			fmt.Fprintf(&b, "        return 301 %s;\n", target)
		default:
			return "", errors.New("不支持的站点类型")
		}
		keys := []string{}
		for key := range l.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(k) {
				return "", errors.New("请求头名称无效")
			}
			v, e := nginxString(l.Headers[k])
			if e != nil {
				return "", e
			}
			fmt.Fprintf(&b, "        proxy_set_header %s %s;\n", k, v)
		}
		b.WriteString("    }\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}
