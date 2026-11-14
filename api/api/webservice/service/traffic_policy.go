package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type TrafficRoute struct {
	Path     string `json:"path"`
	Modifier string `json:"modifier"`
	Scheme   string `json:"scheme"`
	Upstream string `json:"upstream"`
}
type TrafficPolicySpec struct {
	Enabled                bool           `json:"enabled"`
	Allow                  []string       `json:"allow"`
	Deny                   []string       `json:"deny"`
	RequestsPerSecond      int            `json:"requestsPerSecond"`
	Burst                  int            `json:"burst"`
	Nodelay                bool           `json:"nodelay"`
	MaxConnections         int            `json:"maxConnections"`
	ResponseBytesPerSecond int64          `json:"responseBytesPerSecond"`
	TrafficLog             bool           `json:"trafficLog"`
	Upstream               *UpstreamSpec  `json:"upstream,omitempty"`
	Routes                 []TrafficRoute `json:"routes"`
}
type TrafficPolicyInput struct {
	Name        string            `json:"name"`
	InstanceIDs []uint            `json:"instanceIds"`
	Path        string            `json:"path"`
	ServerIndex int               `json:"serverIndex"`
	Baselines   map[uint]string   `json:"baselines"`
	Policy      TrafficPolicySpec `json:"policy"`
}
type TrafficLocationView struct {
	Path      string `json:"path"`
	Modifier  string `json:"modifier"`
	ProxyPass string `json:"proxyPass"`
}
type TrafficPolicyTarget struct {
	Path        string                `json:"path"`
	ServerIndex int                   `json:"serverIndex"`
	ServerNames []string              `json:"serverNames"`
	Listens     []string              `json:"listens"`
	Locations   []TrafficLocationView `json:"locations"`
	Policy      *TrafficPolicySpec    `json:"policy,omitempty"`
}
type TrafficPolicyView struct {
	ClusterID   uint                  `json:"clusterId"`
	InstanceIDs []uint                `json:"instanceIds"`
	Baselines   map[uint]string       `json:"baselines"`
	Targets     []TrafficPolicyTarget `json:"targets"`
	Warnings    []string              `json:"warnings"`
}
type TrafficPolicyDiff struct {
	Path   string `json:"path"`
	Before string `json:"before"`
	After  string `json:"after"`
}
type TrafficPolicyPreview struct {
	Files        []TrafficPolicyDiff `json:"files"`
	Checks       []ConfigValidation  `json:"checks"`
	Valid        bool                `json:"valid"`
	Baselines    map[uint]string     `json:"baselines"`
	Policy       TrafficPolicySpec   `json:"policy"`
	UpstreamName string              `json:"upstreamName"`
	Warnings     []string            `json:"warnings"`
}
type TrafficPolicyDraft struct {
	Draft   model.Resource       `json:"draft"`
	Preview TrafficPolicyPreview `json:"preview"`
}
type trafficPolicyMetadata struct {
	Version         int               `json:"v"`
	Key             string            `json:"key"`
	Policy          TrafficPolicySpec `json:"policy"`
	OriginalProxies []string          `json:"originalProxies"`
	InheritedLogs   []string          `json:"inheritedLogs"`
	LogPath         string            `json:"logPath"`
}

func trafficPolicyKey(cid uint, p string, index int) string {
	return digest([]byte(fmt.Sprintf("%d:%s:%d", cid, p, index)))[:16]
}
func trafficUpstreamName(key string) string { return "autoops_traffic_" + key }
func normalizeTrafficPolicy(p TrafficPolicySpec, key string) (TrafficPolicySpec, error) {
	if len(p.Allow)+len(p.Deny) > 500 {
		return p, errors.New("IP 规则最多 500 条")
	}
	normalize := func(values []string) ([]string, error) {
		out := []string{}
		seen := map[string]bool{}
		for _, v := range values {
			v = strings.TrimSpace(v)
			if strings.Contains(v, "/") {
				prefix, e := netip.ParsePrefix(v)
				if e != nil {
					return nil, errors.New("请输入有效的 IPv4/IPv6 地址或 CIDR")
				}
				v = prefix.Masked().String()
			} else {
				ip, e := netip.ParseAddr(v)
				if e != nil || ip.Zone() != "" {
					return nil, errors.New("请输入有效的 IPv4/IPv6 地址或 CIDR")
				}
				v = ip.String()
			}
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
		return out, nil
	}
	var err error
	p.Allow, err = normalize(p.Allow)
	if err != nil {
		return p, err
	}
	p.Deny, err = normalize(p.Deny)
	if err != nil {
		return p, err
	}
	if p.RequestsPerSecond < 0 || p.RequestsPerSecond > 100000 || p.Burst < 0 || p.Burst > 100000 || p.MaxConnections < 0 || p.MaxConnections > 100000 || p.ResponseBytesPerSecond < 0 || p.ResponseBytesPerSecond > 1<<40 {
		return p, errors.New("限速或连接参数超过范围")
	}
	if p.RequestsPerSecond == 0 && (p.Burst > 0 || p.Nodelay) {
		return p, errors.New("burst 和 nodelay 需要先启用请求限速")
	}
	if p.Nodelay && p.Burst == 0 {
		return p, errors.New("nodelay 需要 burst 大于 0")
	}
	if len(p.Routes) > 100 {
		return p, errors.New("单个站点最多管理 100 条流向规则")
	}
	if p.Upstream != nil {
		upstream := *p.Upstream
		upstream.Members = append([]UpstreamMember(nil), upstream.Members...)
		upstream.Name = trafficUpstreamName(key)
		upstream.Path = "/autoops/traffic.conf"
		if len(upstream.Members) > 100 {
			return p, errors.New("上游最多 100 个成员")
		}
		available := 0
		primary := 0
		seen := map[string]bool{}
		for _, member := range upstream.Members {
			host, port, e := net.SplitHostPort(member.Address)
			if e != nil || host == "" {
				return p, errors.New("上游地址必须包含主机/IP 和端口")
			}
			number, e := strconv.Atoi(port)
			if e != nil || number < 1 || number > 65535 {
				return p, errors.New("上游端口无效")
			}
			if seen[member.Address] {
				return p, errors.New("上游地址重复")
			}
			seen[member.Address] = true
			if !member.Down {
				available++
			}
			if !member.Backup {
				primary++
			}
		}
		if available == 0 || primary == 0 {
			return p, errors.New("上游至少需要一个主成员及一个未下线成员（可为备用成员）")
		}
		if _, e := RenderUpstream(upstream); e != nil {
			return p, e
		}
		p.Upstream = &upstream
	}
	seenRoutes := map[string]bool{}
	for index, r := range p.Routes {
		if !strings.HasPrefix(r.Path, "/") || strings.ContainsAny(r.Path, " \t\n\r;{}\"\\") || (r.Modifier != "" && r.Modifier != "=" && r.Modifier != "^~") {
			return p, errors.New("流向只能选择已有的普通、精确或前缀 location")
		}
		if r.Scheme == "" {
			r.Scheme = "http"
		}
		if r.Scheme != "http" && r.Scheme != "https" {
			return p, errors.New("上游协议无效")
		}
		if r.Upstream == "" {
			if p.Upstream == nil {
				return p, errors.New("流向规则需要选择或定义上游")
			}
			r.Upstream = trafficUpstreamName(key)
		}
		if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(r.Upstream) {
			return p, errors.New("流向规则只能引用具名上游")
		}
		routeKey := r.Modifier + " " + r.Path
		if seenRoutes[routeKey] {
			return p, errors.New("流向规则重复")
		}
		seenRoutes[routeKey] = true
		p.Routes[index] = r
	}
	return p, nil
}
func trafficMarker(key, part, content string) string {
	return "\n    # AUTOOPS TRAFFIC " + key + " " + part + " BEGIN\n" + content + "    # AUTOOPS TRAFFIC " + key + " " + part + " END\n"
}
func trafficServerBlock(m trafficPolicyMetadata) string {
	var b strings.Builder
	encoded := base64.RawStdEncoding.EncodeToString(JSON(m))
	fmt.Fprintf(&b, "    # metadata %s\n", encoded)
	for _, v := range m.Policy.Deny {
		fmt.Fprintf(&b, "    deny %s;\n", v)
	}
	for _, v := range m.Policy.Allow {
		fmt.Fprintf(&b, "    allow %s;\n", v)
	}
	if len(m.Policy.Allow) > 0 {
		b.WriteString("    deny all;\n")
	}
	if m.Policy.RequestsPerSecond > 0 {
		fmt.Fprintf(&b, "    limit_req zone=autoops_req_%s", m.Key)
		if m.Policy.Burst > 0 {
			fmt.Fprintf(&b, " burst=%d", m.Policy.Burst)
		}
		if m.Policy.Nodelay {
			b.WriteString(" nodelay")
		}
		b.WriteString(";\n    limit_req_status 429;\n")
	}
	if m.Policy.MaxConnections > 0 {
		fmt.Fprintf(&b, "    limit_conn autoops_conn_%s %d;\n    limit_conn_status 429;\n", m.Key, m.Policy.MaxConnections)
	}
	if m.Policy.ResponseBytesPerSecond > 0 {
		fmt.Fprintf(&b, "    limit_rate %d;\n", m.Policy.ResponseBytesPerSecond)
	}
	if m.Policy.TrafficLog {
		for _, line := range m.InheritedLogs {
			fmt.Fprintf(&b, "    %s\n", line)
		}
		fmt.Fprintf(&b, "    access_log \"%s\" autoops_traffic_%s;\n", m.LogPath, m.Key)
	}
	return trafficMarker(m.Key, "SERVER", b.String())
}
func trafficHTTPBlock(m trafficPolicyMetadata) string {
	var b strings.Builder
	if m.Policy.RequestsPerSecond > 0 {
		fmt.Fprintf(&b, "    limit_req_zone $binary_remote_addr zone=autoops_req_%s:10m rate=%dr/s;\n", m.Key, m.Policy.RequestsPerSecond)
	}
	if m.Policy.MaxConnections > 0 {
		fmt.Fprintf(&b, "    limit_conn_zone $binary_remote_addr zone=autoops_conn_%s:10m;\n", m.Key)
	}
	if m.Policy.Upstream != nil {
		text, _ := RenderUpstream(*m.Policy.Upstream)
		b.WriteString(text)
	}
	if m.Policy.TrafficLog {
		fmt.Fprintf(&b, "    log_format autoops_traffic_%s escape=json '{\"time_iso8601\":\"$time_iso8601\",\"remote_addr\":\"$remote_addr\",\"uri\":\"$uri\",\"status\":$status,\"body_bytes_sent\":$body_bytes_sent,\"request_length\":$request_length,\"request_time\":$request_time,\"upstream_addr\":\"$upstream_addr\",\"upstream_response_time\":\"$upstream_response_time\",\"upstream_status\":\"$upstream_status\",\"limit_req_status\":\"$limit_req_status\",\"limit_conn_status\":\"$limit_conn_status\"}';\n", m.Key)
	}
	return trafficMarker(m.Key, "HTTP", b.String())
}
func trafficRouteBlock(m trafficPolicyMetadata, index int) string {
	r := m.Policy.Routes[index]
	nodes, _ := parseNginx(m.OriginalProxies[index])
	suffix, _ := trafficProxySuffix(nodes[0].Args[1])
	return trafficMarker(m.Key, "ROUTE-"+strconv.Itoa(index), fmt.Sprintf("    proxy_pass %s://%s%s;\n", r.Scheme, r.Upstream, suffix))
}

func trafficProxySuffix(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || strings.ContainsAny(value, "$ \t\r\n;{}#\"'\\") {
		return "", errors.New("原 proxy_pass 包含动态变量或复杂 URI，请使用原文配置维护流向")
	}
	return strings.TrimPrefix(value, parsed.Scheme+"://"+parsed.Host), nil
}

func readTrafficMetadata(text, key string) (*trafficPolicyMetadata, error) {
	begin := "\n    # AUTOOPS TRAFFIC " + key + " SERVER BEGIN\n"
	count := strings.Count(text, begin)
	if count == 0 {
		if strings.Contains(text, "# AUTOOPS TRAFFIC ") {
			return nil, errors.New("站点包含无法匹配的策略标记，请先检查原文")
		}
		return nil, nil
	}
	if count != 1 {
		return nil, errors.New("策略标记重复")
	}
	start := strings.Index(text, begin) + len(begin)
	end := strings.IndexByte(text[start:], '\n')
	if end < 0 {
		return nil, errors.New("策略元数据不完整")
	}
	line := text[start : start+end]
	const prefix = "    # metadata "
	if !strings.HasPrefix(line, prefix) {
		return nil, errors.New("策略元数据不完整")
	}
	encoded := strings.TrimPrefix(line, prefix)
	if len(encoded) > 128<<10 {
		return nil, errors.New("策略元数据超过限制")
	}
	raw, e := base64.RawStdEncoding.DecodeString(encoded)
	if e != nil {
		return nil, errors.New("策略元数据编码无效")
	}
	var m trafficPolicyMetadata
	if json.Unmarshal(raw, &m) != nil || m.Version != 1 || m.Key != key || !m.Policy.Enabled {
		return nil, errors.New("策略元数据无效")
	}
	normalized, e := normalizeTrafficPolicy(m.Policy, key)
	if e != nil {
		return nil, e
	}
	if string(JSON(normalized)) != string(JSON(m.Policy)) {
		return nil, errors.New("策略元数据不规范")
	}
	if len(m.OriginalProxies) != len(m.Policy.Routes) {
		return nil, errors.New("策略流向恢复数据不完整")
	}
	for _, original := range m.OriginalProxies {
		nodes, e := parseNginx(original)
		if e != nil || len(nodes) != 1 || len(nodes[0].Args) != 2 || nodes[0].Args[0] != "proxy_pass" || nodes[0].Open >= 0 || nodes[0].Start != 0 || nodes[0].End != len(original) {
			return nil, errors.New("原始 proxy_pass 记录无效")
		}
		if _, e := trafficProxySuffix(nodes[0].Args[1]); e != nil {
			return nil, e
		}
	}
	for _, line := range m.InheritedLogs {
		nodes, e := parseNginx(line)
		if e != nil || len(nodes) != 1 || nodes[0].Args[0] != "access_log" || nodes[0].Open >= 0 {
			return nil, errors.New("原始日志继承记录无效")
		}
	}
	if m.Policy.TrafficLog && (!validPath(m.LogPath) || strings.ContainsAny(m.LogPath, "\"\\")) {
		return nil, errors.New("策略日志路径无效")
	}
	if strings.Count(text, trafficServerBlock(m)) != 1 || !strings.HasPrefix(text, trafficServerBlock(m)) {
		return nil, errors.New("平台策略被手工修改，请先检查漂移，不能自动覆盖")
	}
	return &m, nil
}
func restoreTrafficPolicy(config *trafficNativeConfig, target string, index int, key string) (map[string]string, *trafficPolicyMetadata, error) {
	servers := config.Servers[target]
	if index < 0 || index >= len(servers) {
		return nil, nil, errors.New("目标 HTTP server 不存在或编号已变化")
	}
	server := servers[index]
	file := config.Files[target]
	m, e := readTrafficMetadata(file.Content[server.Open+1:server.Close], key)
	if e != nil {
		return nil, nil, e
	}
	contents := map[string]string{config.Main: config.Files[config.Main].Content, target: file.Content}
	if m == nil {
		if strings.Contains(config.Files[config.Main].Content, "# AUTOOPS TRAFFIC "+key+" HTTP") {
			return nil, nil, errors.New("存在孤立的 HTTP 策略标记，不能自动覆盖")
		}
		return contents, nil, nil
	}
	if !trafficBlockInside(config.Files[config.Main].Content, config.HTTP, trafficHTTPBlock(*m)) {
		return nil, nil, errors.New("HTTP 策略位置已变化，不能自动覆盖")
	}
	for index, route := range m.Policy.Routes {
		matched := 0
		for _, location := range server.Children {
			p, modifier, ok := trafficLocation(location)
			if ok && p == route.Path && modifier == route.Modifier && trafficBlockInside(file.Content, location, trafficRouteBlock(*m, index)) {
				matched++
			}
		}
		if matched != 1 {
			return nil, nil, errors.New("流向策略位置已变化，不能自动覆盖")
		}
	}
	blocks := []struct{ Path, Block, Replacement string }{{target, trafficServerBlock(*m), ""}, {config.Main, trafficHTTPBlock(*m), ""}}
	for idx := range m.Policy.Routes {
		blocks = append(blocks, struct{ Path, Block, Replacement string }{target, trafficRouteBlock(*m, idx), m.OriginalProxies[idx]})
	}
	for _, b := range blocks {
		text := contents[b.Path]
		if strings.Count(text, b.Block) != 1 {
			return nil, nil, errors.New("策略引用或内容已变化，不能自动覆盖或恢复")
		}
		contents[b.Path] = strings.Replace(text, b.Block, b.Replacement, 1)
	}
	return contents, m, nil
}

func trafficBlockInside(text string, parent *nginxNode, block string) bool {
	if strings.Count(text, block) != 1 {
		return false
	}
	start := strings.Index(text, block)
	end := start + len(block)
	if start < parent.Open+1 || end > parent.Close {
		return false
	}
	for _, child := range parent.Children {
		if child.Open >= 0 && child.Start < start && child.End > end {
			return false
		}
	}
	return true
}

func (s *Service) trafficPolicySnapshots(ctx context.Context, u, cid uint, ids []uint, write bool) (model.Resource, []InstanceConfig, error) {
	if err := s.Module(ctx, u, "read"); err != nil {
		return model.Resource{}, nil, err
	}
	cluster, err := s.Get("clusters", cid)
	if err != nil {
		return cluster, nil, err
	}
	action := "web_config_view"
	if write {
		if err = s.Module(ctx, u, "write"); err != nil {
			return cluster, nil, err
		}
		action = "web_config_edit"
	}
	if err = s.Authorize(u, "clusters", cluster, action); err != nil {
		return cluster, nil, err
	}
	members, err := s.Members(u, cid)
	if err != nil {
		return cluster, nil, err
	}
	members, err = selectConfigurationMembers(ids, members)
	if err != nil {
		return cluster, nil, err
	}
	if len(members) == 0 || len(members) > 50 {
		return cluster, nil, errors.New("请选择 1–50 个集群成员")
	}
	for _, m := range members {
		if err = s.Authorize(u, "instances", m, action); err != nil {
			return cluster, nil, err
		}
		if write && m.Management != "operate" {
			return cluster, nil, errors.New("流量策略只允许已接管实例")
		}
	}
	snapshots := []InstanceConfig{}
	for _, m := range members {
		snapshot, e := s.InstanceConfigContext(ctx, u, m.ID)
		if e != nil {
			return cluster, nil, e
		}
		snapshots = append(snapshots, snapshot)
	}
	return cluster, snapshots, nil
}
func (s *Service) TrafficPolicy(ctx context.Context, u, cid, instanceID uint) (TrafficPolicyView, error) {
	ids := []uint{}
	if instanceID > 0 {
		ids = append(ids, instanceID)
	}
	_, snapshots, err := s.trafficPolicySnapshots(ctx, u, cid, ids, false)
	out := TrafficPolicyView{ClusterID: cid, InstanceIDs: []uint{}, Baselines: map[uint]string{}, Targets: []TrafficPolicyTarget{}, Warnings: []string{}}
	if err != nil {
		return out, err
	}
	for _, snapshot := range snapshots {
		out.InstanceIDs = append(out.InstanceIDs, snapshot.InstanceID)
		out.Baselines[snapshot.InstanceID] = snapshot.Digest
	}
	config, err := newTrafficNativeConfig(snapshots[0])
	if err != nil {
		return out, err
	}
	paths := []string{}
	for p := range config.Servers {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if !withinManaged(p, snapshots[0].ManagedPaths) {
			continue
		}
		for index, n := range config.Servers[p] {
			target := TrafficPolicyTarget{Path: p, ServerIndex: index, ServerNames: []string{}, Listens: []string{}, Locations: []TrafficLocationView{}}
			for _, child := range n.Children {
				switch child.Args[0] {
				case "server_name":
					target.ServerNames = append(target.ServerNames, child.Args[1:]...)
				case "listen":
					target.Listens = append(target.Listens, strings.Join(child.Args[1:], " "))
				}
				location, modifier, ok := trafficLocation(child)
				if ok {
					proxy := nginxDirective(child.Children, "proxy_pass")
					if len(proxy) == 1 && len(proxy[0].Args) == 2 {
						target.Locations = append(target.Locations, TrafficLocationView{Path: location, Modifier: modifier, ProxyPass: proxy[0].Args[1]})
					}
				}
			}
			m, e := readTrafficMetadata(config.Files[p].Content[n.Open+1:n.Close], trafficPolicyKey(cid, p, index))
			if e != nil {
				out.Warnings = append(out.Warnings, p+": "+e.Error())
			} else if m != nil {
				target.Policy = &m.Policy
			}
			out.Targets = append(out.Targets, target)
		}
	}
	if len(snapshots) > 1 {
		out.Warnings = append(out.Warnings, "集群策略要求目标文件和主配置内容一致；异构节点请分别选择单个实例")
	}
	return out, nil
}

func trafficWalkDirectives(config *trafficNativeConfig, prefix, file string, nodes []*nginxNode, descend bool, visit func(string, *nginxNode) error) error {
	active := map[string]bool{}
	var walk func(string, []*nginxNode, int) error
	walk = func(file string, nodes []*nginxNode, depth int) error {
		if depth > 32 {
			return errors.New("策略依赖嵌套超过限制")
		}
		for _, n := range nodes {
			if n.Args[0] == "include" {
				if len(n.Args) != 2 || strings.Contains(n.Args[1], "$") {
					return errors.New("动态 include 不能安全合并策略")
				}
				pattern := n.Args[1]
				if !strings.HasPrefix(pattern, "/") {
					pattern = path.Join(prefix, pattern)
				}
				found := false
				for p := range config.Files {
					match, e := path.Match(pattern, p)
					if e != nil {
						return e
					}
					if !match {
						continue
					}
					found = true
					if active[p] {
						return errors.New("配置 include 循环")
					}
					active[p] = true
					if e = walk(p, config.Roots[p], depth+1); e != nil {
						return e
					}
					delete(active, p)
				}
				if !found && !strings.ContainsAny(pattern, "*?[") {
					return errors.New("策略依赖文件不在已发现配置中")
				}
				continue
			}
			if e := visit(file, n); e != nil {
				return e
			}
			if descend && len(n.Children) > 0 {
				if e := walk(file, n.Children, depth+1); e != nil {
					return e
				}
			}
		}
		return nil
	}
	return walk(file, nodes, 0)
}
func buildTrafficPolicy(snapshot InstanceConfig, cid uint, input TrafficPolicyInput) ([]model.File, TrafficPolicyPreview, error) {
	preview := TrafficPolicyPreview{Files: []TrafficPolicyDiff{}, Checks: []ConfigValidation{}, Baselines: input.Baselines, Warnings: []string{}}
	config, err := newTrafficNativeConfig(snapshot)
	if err != nil {
		return nil, preview, err
	}
	if !validPath(input.Path) || !withinManaged(input.Path, snapshot.ManagedPaths) || !withinManaged(config.Main, snapshot.ManagedPaths) {
		return nil, preview, errors.New("策略目标和主配置必须处于受管目录")
	}
	key := trafficPolicyKey(cid, input.Path, input.ServerIndex)
	policy, err := normalizeTrafficPolicy(input.Policy, key)
	if err != nil {
		return nil, preview, err
	}
	preview.Policy = policy
	restored, old, err := restoreTrafficPolicy(config, input.Path, input.ServerIndex, key)
	if err != nil {
		return nil, preview, err
	}
	if !policy.Enabled && old == nil {
		return nil, preview, errors.New("此 server 尚无平台策略可移除")
	}
	base := snapshot
	base.Files = append([]model.File(nil), snapshot.Files...)
	for index := range base.Files {
		if text, ok := restored[base.Files[index].Path]; ok {
			base.Files[index].Content = text
		}
	}
	clean, err := newTrafficNativeConfig(base)
	if err != nil {
		return nil, preview, err
	}
	servers := clean.Servers[input.Path]
	if input.ServerIndex < 0 || input.ServerIndex >= len(servers) {
		return nil, preview, errors.New("server 编号已变化")
	}
	server := servers[input.ServerIndex]
	after := restored
	if policy.Enabled {
		conflicts := map[string]bool{}
		defending := len(policy.Allow)+len(policy.Deny) > 0 || policy.RequestsPerSecond > 0 || policy.MaxConnections > 0 || policy.ResponseBytesPerSecond > 0
		if defending {
			// Rewrite-module returns run before access/preaccess handlers. An
			// error_page can turn a rejected request into an internal upstream
			// redirect. Do not advertise protection for these native flows.
			for _, name := range []string{"return", "rewrite", "error_page", "recursive_error_pages"} {
				conflicts[name] = true
			}
		}
		if len(policy.Allow)+len(policy.Deny) > 0 {
			for _, name := range []string{"allow", "deny", "satisfy"} {
				conflicts[name] = true
			}
		}
		if policy.RequestsPerSecond > 0 {
			for _, name := range []string{"limit_req", "limit_req_status", "limit_req_dry_run"} {
				conflicts[name] = true
			}
		}
		if policy.MaxConnections > 0 {
			for _, name := range []string{"limit_conn", "limit_conn_status", "limit_conn_dry_run"} {
				conflicts[name] = true
			}
		}
		if policy.ResponseBytesPerSecond > 0 {
			conflicts["limit_rate"] = true
			conflicts["limit_rate_after"] = true
		}
		check := func(file string, n *nginxNode) error {
			if conflicts[n.Args[0]] {
				if defending && (n.Args[0] == "return" || n.Args[0] == "rewrite" || n.Args[0] == "error_page" || n.Args[0] == "recursive_error_pages") {
					return fmt.Errorf("%s 已有手工 %s 指令，可能在防御前返回或改变拒绝结果，请先整理原文配置", file, n.Args[0])
				}
				return fmt.Errorf("%s 已有手工 %s 指令，须先在配置管理中整理，不能覆盖其继承规则", file, n.Args[0])
			}
			return nil
		}
		if err = trafficWalkDirectives(clean, snapshot.Prefix, clean.Main, clean.HTTP.Children, false, check); err != nil {
			return nil, preview, err
		}
		if err = trafficWalkDirectives(clean, snapshot.Prefix, input.Path, server.Children, true, check); err != nil {
			return nil, preview, err
		}
		metadata := trafficPolicyMetadata{Version: 1, Key: key, Policy: policy, OriginalProxies: []string{}, InheritedLogs: []string{}}
		if policy.TrafficLog {
			access := []*nginxNode{}
			if err = trafficWalkDirectives(clean, snapshot.Prefix, input.Path, server.Children, false, func(_ string, n *nginxNode) error {
				if n.Args[0] == "access_log" {
					access = append(access, n)
				}
				return nil
			}); err != nil {
				return nil, preview, err
			}
			if len(access) == 0 {
				if len(clean.HTTPLogs) == 0 {
					return nil, preview, errors.New("现有访问日志使用隐式默认路径，启用流量日志前请在 http/server 显式配置 access_log")
				}
				for _, line := range clean.HTTPLogs {
					nodes, _ := parseNginx(line)
					if len(nodes) != 1 || len(nodes[0].Args) < 2 || nodes[0].Args[1] == "off" {
						return nil, preview, errors.New("现有访问日志已关闭，需先明确日志保留策略")
					}
					metadata.InheritedLogs = append(metadata.InheritedLogs, line)
				}
			} else {
				for _, n := range access {
					if len(n.Args) < 2 || n.Args[1] == "off" {
						return nil, preview, errors.New("站点访问日志已关闭，不能隐式重新开启")
					}
				}
			}
			// A location access_log overrides its parent's logs; refuse rather than
			// claim the entire selected server is covered by the dedicated stream.
			for _, n := range server.Children {
				if n.Open >= 0 {
					if err = trafficWalkDirectives(clean, snapshot.Prefix, input.Path, n.Children, true, func(file string, child *nginxNode) error {
						if child.Args[0] == "access_log" {
							return fmt.Errorf("%s 的 location 有独立 access_log，需先整理日志继承后启用站点流量日志", file)
						}
						return nil
					}); err != nil {
						return nil, preview, err
					}
				}
			}
			// Locate the new log beside an already configured file. Its directory
			// exists, and it is not a versioned configuration symlink target.
			logs := append([]string(nil), metadata.InheritedLogs...)
			if len(logs) == 0 {
				for _, n := range access {
					if len(n.Args) > 1 {
						logs = append(logs, "access_log \""+n.Args[1]+"\";")
					}
				}
			}
			for _, line := range logs {
				nodes, e := parseNginx(line)
				if e == nil && len(nodes) == 1 && len(nodes[0].Args) > 1 {
					candidate := nodes[0].Args[1]
					if strings.HasPrefix(candidate, "/") && !strings.ContainsAny(candidate, "$\"\\") {
						metadata.LogPath = path.Join(path.Dir(candidate), "autoops-traffic-"+key+".jsonl")
						break
					}
				}
			}
			if metadata.LogPath == "" {
				return nil, preview, errors.New("流量日志需要现有绝对路径的文件 access_log，不能使用动态路径或 syslog")
			}
			preview.Warnings = append(preview.Warnings, "专用 JSON 日志仅覆盖所选 server；保留原有访问日志。请为新增日志配置轮转和磁盘保留策略")
		}
		upstreamNames := map[string]bool{}
		if err = trafficWalkDirectives(clean, snapshot.Prefix, clean.Main, clean.HTTP.Children, false, func(_ string, n *nginxNode) error {
			if n.Args[0] == "upstream" && len(n.Args) == 2 {
				upstreamNames[n.Args[1]] = true
			}
			return nil
		}); err != nil {
			return nil, preview, err
		}
		if policy.Upstream != nil {
			if upstreamNames[policy.Upstream.Name] {
				return nil, preview, errors.New("平台上游名称已被其他配置占用")
			}
			upstreamNames[policy.Upstream.Name] = true
			preview.UpstreamName = policy.Upstream.Name
		}
		edits := map[string][]nginxEdit{}
		for index, route := range policy.Routes {
			if !upstreamNames[route.Upstream] {
				return nil, preview, errors.New("流向引用的具名上游不在 HTTP 配置中")
			}
			matched := []*nginxNode{}
			for _, n := range server.Children {
				p, modifier, ok := trafficLocation(n)
				if ok && p == route.Path && modifier == route.Modifier {
					matched = append(matched, n)
				}
			}
			if len(matched) != 1 {
				return nil, preview, errors.New("流向的 location 不存在或不唯一")
			}
			location := matched[0]
			for _, child := range location.Children {
				if child.Open >= 0 || child.Args[0] == "include" || child.Args[0] == "rewrite" || child.Args[0] == "return" || child.Args[0] == "try_files" {
					return nil, preview, errors.New("该 location 包含嵌套、include 或其他转发控制，请使用原文配置维护流向")
				}
			}
			proxies := nginxDirective(location.Children, "proxy_pass")
			if len(proxies) != 1 || len(proxies[0].Args) != 2 || proxies[0].Open >= 0 {
				return nil, preview, errors.New("流向需要 location 中唯一的直接 proxy_pass")
			}
			original := clean.Files[input.Path].Content[proxies[0].Start:proxies[0].End]
			if _, e := trafficProxySuffix(proxies[0].Args[1]); e != nil {
				return nil, preview, e
			}
			metadata.OriginalProxies = append(metadata.OriginalProxies, original)
			edits[input.Path] = append(edits[input.Path], nginxEdit{Start: proxies[0].Start, End: proxies[0].End, Text: trafficRouteBlock(metadata, index)})
		}
		edits[input.Path] = append(edits[input.Path], nginxEdit{Start: server.Open + 1, End: server.Open + 1, Text: trafficServerBlock(metadata)})
		edits[clean.Main] = append(edits[clean.Main], nginxEdit{Start: clean.HTTP.Open + 1, End: clean.HTTP.Open + 1, Text: trafficHTTPBlock(metadata)})
		for file, changes := range edits {
			text, e := applyNginxEdits(clean.Files[file].Content, changes)
			if e != nil {
				return nil, preview, e
			}
			after[file] = text
		}
	}
	paths := []string{}
	for p := range after {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	files := []model.File{}
	for _, p := range paths {
		before := config.Files[p]
		if before.Content == after[p] {
			continue
		}
		files = append(files, model.File{Path: p, Content: after[p], Mode: before.Mode, Group: before.Group})
		preview.Files = append(preview.Files, TrafficPolicyDiff{Path: p, Before: before.Content, After: after[p]})
	}
	if len(files) == 0 {
		return nil, preview, errors.New("策略配置没有变化")
	}
	preview.Warnings = append(preview.Warnings, "规则按 Nginx 看到的 remote_addr 生效；处于代理之后时需先配置可信 real_ip 来源。此功能不是完整 WAF 或上游网络清洗")
	return files, preview, nil
}

func (s *Service) prepareTrafficPolicy(ctx context.Context, u, cid uint, input TrafficPolicyInput) (model.Resource, model.ConfigSpec, TrafficPolicyPreview, error) {
	cluster, snapshots, err := s.trafficPolicySnapshots(ctx, u, cid, input.InstanceIDs, true)
	if err != nil {
		return cluster, model.ConfigSpec{}, TrafficPolicyPreview{}, err
	}
	ids := []uint{}
	for _, snapshot := range snapshots {
		ids = append(ids, snapshot.InstanceID)
		if input.Baselines[snapshot.InstanceID] == "" || input.Baselines[snapshot.InstanceID] != snapshot.Digest {
			return cluster, model.ConfigSpec{}, TrafficPolicyPreview{}, ErrConflict
		}
	}
	files, preview, err := buildTrafficPolicy(snapshots[0], cid, input)
	if err != nil {
		return cluster, model.ConfigSpec{}, preview, err
	}
	for _, snapshot := range snapshots[1:] {
		if snapshot.ConfigPath != snapshots[0].ConfigPath || snapshot.Prefix != snapshots[0].Prefix {
			return cluster, model.ConfigSpec{}, preview, errors.New("节点配置布局不同，请选择单个实例维护策略")
		}
		otherFiles, _, e := buildTrafficPolicy(snapshot, cid, input)
		if e != nil {
			return cluster, model.ConfigSpec{}, preview, e
		}
		if string(JSON(otherFiles)) != string(JSON(files)) {
			return cluster, model.ConfigSpec{}, preview, errors.New("节点配置内容不同，请选择单个实例维护策略")
		}
	}
	candidate := model.ConfigSpec{Files: files, Baselines: input.Baselines, InstanceIDs: ids}
	return cluster, candidate, preview, nil
}
func (s *Service) PreviewTrafficPolicy(ctx context.Context, u, cid uint, input TrafficPolicyInput) (TrafficPolicyPreview, error) {
	_, candidate, preview, err := s.prepareTrafficPolicy(ctx, u, cid, input)
	if err != nil {
		return preview, err
	}
	preview.Checks, err = s.ValidateClusterConfig(ctx, u, cid, candidate.Files, candidate)
	if err != nil {
		return preview, err
	}
	preview.Valid = len(preview.Checks) > 0
	for _, c := range preview.Checks {
		if !c.Valid {
			preview.Valid = false
		}
	}
	return preview, nil
}
func (s *Service) DraftTrafficPolicy(ctx context.Context, u, cid uint, input TrafficPolicyInput) (TrafficPolicyDraft, error) {
	cluster, candidate, preview, err := s.prepareTrafficPolicy(ctx, u, cid, input)
	if err != nil {
		return TrafficPolicyDraft{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = "流量策略 · " + path.Base(input.Path) + " #" + strconv.Itoa(input.ServerIndex+1)
	}
	draft, err := s.Save(ctx, u, "configurations", model.Resource{Name: name, GroupID: cluster.GroupID, ClusterID: cid, Engine: cluster.Engine, Environment: cluster.Environment, Spec: JSON(candidate)}, "")
	if err != nil {
		return TrafficPolicyDraft{}, err
	}
	preview.Valid = true
	return TrafficPolicyDraft{Draft: draft, Preview: preview}, nil
}
