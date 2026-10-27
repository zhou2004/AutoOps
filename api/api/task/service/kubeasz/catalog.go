// Package kubeasz implements the fixed system Ansible catalog. It deliberately
// does not expose shell commands, Git locations, or filesystem paths as inputs.
package kubeasz

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"
)

const Root = "/etc/kubeasz"
const AnsibleBin = "/opt/kubeasz-venv/bin"
const Version = "3.6.8"
const Commit = "0e9cee984c211e9e9f9e64910206baab9fbd3d8e"
const EzctlSHA256 = "e5d7837102d2090f76775e277f629c3fa128f0295b0ce6c5963b2c0f1d96b76b"

var safeName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
var safeUser = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
var safeSnapshot = regexp.MustCompile(`^snapshot(?:_[0-9]{12,14})?\.db$`)

type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Risk        string `json:"risk"`
	Node        bool   `json:"node"`
	Step        bool   `json:"step"`
	Addon       bool   `json:"addon"`
}

var Templates = []Template{
	{ID: "new", Name: "创建集群配置", Description: "在系统目录生成 hosts 和 config.yml；随后执行安装模板。", Risk: "local"},
	{ID: "setup", Name: "安装集群 / 分步安装", Description: "准备、etcd、运行时、控制面、工作节点、网络、插件或全部步骤。", Risk: "high", Step: true},
	{ID: "addon", Name: "安装服务插件", Description: "使用固定官方 role 安装指定插件。", Risk: "high", Addon: true},
	{ID: "add-node", Name: "扩容工作节点", Description: "更新 inventory 并安装工作节点。失败可能留下部分部署，需要检查实际状态。", Risk: "high", Node: true},
	{ID: "del-node", Name: "删除工作节点", Description: "清理节点 Kubernetes 服务和数据；先迁移业务并检查 PDB / 本地卷。", Risk: "destructive", Node: true},
	{ID: "add-master", Name: "扩容控制面", Description: "安装控制面并重新配置集群负载均衡。", Risk: "high", Node: true},
	{ID: "del-master", Name: "删除控制面", Description: "删除控制面并更新负载均衡；禁止删除最后一个控制面。", Risk: "destructive", Node: true},
	{ID: "add-etcd", Name: "扩容 etcd", Description: "增加成员并重新配置 etcd 和 apiserver。", Risk: "high", Node: true},
	{ID: "del-etcd", Name: "删除 etcd 成员", Description: "必须保留至少 3 个成员；先备份并确认 etcd 健康。", Risk: "destructive", Node: true},
	{ID: "backup", Name: "备份 etcd", Description: "备份 Kubernetes 对象到集群 backup 目录；不包含业务持久卷数据。", Risk: "high"},
	{ID: "restore", Name: "恢复 etcd", Description: "停止集群服务并使用所选快照替换 etcd 数据；需要维护窗口。", Risk: "destructive"},
	{ID: "start", Name: "启动集群服务", Description: "启动 kubeasz 管理的集群服务。", Risk: "high"},
	{ID: "stop", Name: "停止集群服务", Description: "停止整个集群的 Kubernetes 服务；需要维护窗口。", Risk: "destructive"},
	{ID: "upgrade", Name: "升级集群组件", Description: "使用部署端预先审核并放入 bin 的二进制升级，先备份并核对版本跨度。", Risk: "high"},
}

var Steps = map[string]string{"all": "全部安装", "01": "准备证书和系统", "02": "etcd", "03": "containerd", "04": "控制面", "05": "工作节点", "06": "网络", "07": "集群插件"}
var Addons = map[string]string{"metrics-server": "Metrics Server", "coredns": "CoreDNS", "local-path": "Local Path Provisioner", "prometheus": "Prometheus", "ingress-nginx": "Ingress NGINX"}
var addonTasks = map[string]string{"metrics-server": "metrics-server.yml", "coredns": "coredns.yml", "local-path": "local-storage.yml", "prometheus": "prometheus.yml", "ingress-nginx": "ingress-nginx.yml"}
var addonVars = map[string]string{"metrics-server": "metricsserver_install", "coredns": "dns_install", "local-path": "local_path_provisioner_install", "prometheus": "prom_install", "ingress-nginx": "ingress_nginx_install"}

type ClusterConfig struct {
	Masters     []string `json:"masters"`
	Workers     []string `json:"workers"`
	Etcd        []string `json:"etcd"`
	SSHUser     string   `json:"ssh_user"`
	SSHPort     int      `json:"ssh_port"`
	PodCIDR     string   `json:"pod_cidr"`
	ServiceCIDR string   `json:"service_cidr"`
	Network     string   `json:"network"`
}

type Request struct {
	Cluster      string         `json:"cluster"`
	Action       string         `json:"action"`
	Node         string         `json:"node,omitempty"`
	Step         string         `json:"step,omitempty"`
	Addon        string         `json:"addon,omitempty"`
	Snapshot     string         `json:"snapshot,omitempty"`
	Config       *ClusterConfig `json:"config,omitempty"`
	PlanHash     string         `json:"plan_hash,omitempty"`
	Confirmation string         `json:"confirmation,omitempty"`
}

type Command struct {
	Program string   `json:"program"`
	Args    []string `json:"args"`
}

func (c Command) String() string {
	parts := []string{c.Program}
	for _, arg := range c.Args {
		// Display quoting only; commands always execute via exec.Command arguments.
		if strings.ContainsAny(arg, " \t\n'\"{}") {
			arg = "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}

func templateFor(action string) (Template, error) {
	for _, item := range Templates {
		if item.ID == action {
			return item, nil
		}
	}
	return Template{}, errors.New("未知系统模板")
}

func ipv4(value string) bool {
	ip := net.ParseIP(value)
	return ip != nil && ip.To4() != nil && ip.String() == value && !ip.IsUnspecified() && !ip.IsMulticast() && !ip.IsLoopback()
}

func Validate(r Request) error {
	if !safeName.MatchString(r.Cluster) {
		return errors.New("集群名须以小写字母开头，只能包含小写字母、数字和连字符，最多 63 字符")
	}
	t, err := templateFor(r.Action)
	if err != nil {
		return err
	}
	if t.Node {
		if !ipv4(r.Node) {
			return errors.New("节点必须为有效的非回环 IPv4 地址")
		}
	} else if r.Node != "" {
		return errors.New("此操作不接受节点参数")
	}
	if t.Step {
		if _, ok := Steps[r.Step]; !ok {
			return errors.New("请选择有效安装步骤")
		}
	} else if r.Step != "" {
		return errors.New("此操作不接受安装步骤")
	}
	if t.Addon {
		if _, ok := Addons[r.Addon]; !ok {
			return errors.New("插件不在系统模板目录中")
		}
	} else if r.Addon != "" {
		return errors.New("此操作不接受插件参数")
	}
	if r.Action == "restore" {
		if !safeSnapshot.MatchString(r.Snapshot) {
			return errors.New("必须选择有效的 etcd 快照文件")
		}
	} else if r.Snapshot != "" {
		return errors.New("此操作不接受快照参数")
	}
	if r.Action != "new" {
		if r.Config != nil {
			return errors.New("只有新建集群可提交集群配置")
		}
		return nil
	}
	if r.Config == nil {
		return errors.New("缺少集群配置")
	}
	c := r.Config
	if len(c.Masters) < 1 || len(c.Masters) > 15 || len(c.Workers) > 500 || len(c.Etcd) < 1 || len(c.Etcd) > 7 || len(c.Etcd)%2 == 0 {
		return errors.New("控制面 1–15 个，工作节点最多 500 个，etcd 必须为 1、3、5、7 个")
	}
	for _, group := range [][]string{c.Masters, c.Workers, c.Etcd} {
		seen := map[string]bool{}
		for _, ip := range group {
			if !ipv4(ip) || seen[ip] {
				return errors.New("节点地址须为有效且组内不重复的 IPv4")
			}
			seen[ip] = true
		}
	}
	for _, ip := range c.Workers {
		for _, m := range c.Masters {
			if m == ip {
				return errors.New("控制面和工作节点不能重复")
			}
		}
	}
	if !safeUser.MatchString(c.SSHUser) || c.SSHPort < 1 || c.SSHPort > 65535 {
		return errors.New("SSH 用户或端口无效")
	}
	if c.Network != "calico" && c.Network != "flannel" && c.Network != "cilium" {
		return errors.New("支持 calico、flannel 或 cilium 网络")
	}
	pod, pn, pe := net.ParseCIDR(c.PodCIDR)
	svc, sn, se := net.ParseCIDR(c.ServiceCIDR)
	if pe != nil || se != nil || pod.To4() == nil || svc.To4() == nil || pn.Contains(svc) || sn.Contains(pod) {
		return errors.New("Pod / Service CIDR 须为不重叠的 IPv4 网段")
	}
	po, _ := pn.Mask.Size()
	so, _ := sn.Mask.Size()
	if po > 24 || po < 8 || so > 24 || so < 8 {
		return errors.New("CIDR 前缀长度需为 8–24")
	}
	for _, group := range [][]string{c.Masters, c.Workers, c.Etcd} {
		for _, host := range group {
			if pn.Contains(net.ParseIP(host)) || sn.Contains(net.ParseIP(host)) {
				return errors.New("Pod / Service CIDR 不能包含节点地址")
			}
		}
	}
	return nil
}

func Commands(r Request) ([]Command, error) {
	if err := Validate(r); err != nil {
		return nil, err
	}
	ez := Command{Root + "/ezctl", []string{r.Action, r.Cluster}}
	switch r.Action {
	case "setup":
		ez.Args = append(ez.Args, r.Step)
	case "add-node", "del-node", "add-master", "del-master", "add-etcd", "del-etcd":
		ez.Args = append(ez.Args, r.Node)
	case "addon":
		return []Command{{AnsibleBin + "/ansible-playbook", []string{"-i", "clusters/" + r.Cluster + "/hosts", "-e", "@clusters/" + r.Cluster + "/config.yml", "-e", addonVars[r.Addon] + "=yes", "autoops-playbooks/" + r.Addon + ".yml"}}}, nil
	case "restore":
		vars, _ := json.Marshal(map[string]string{"db_to_restore": r.Snapshot})
		return []Command{{AnsibleBin + "/ansible-playbook", []string{"-i", "clusters/" + r.Cluster + "/hosts", "-e", "@clusters/" + r.Cluster + "/config.yml", "-e", string(vars), "playbooks/95.restore.yml"}}}, nil
	}
	return []Command{ez}, nil
}

func Confirmation(r Request) string {
	parts := []string{r.Action, r.Cluster}
	for _, v := range []string{r.Node, r.Step, r.Addon, r.Snapshot} {
		if v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " ")
}

func planDigest(r Request, files ...[]byte) string {
	r.Confirmation = ""
	r.PlanHash = ""
	data, _ := json.Marshal(r)
	hash := sha256.New()
	hash.Write([]byte(Commit))
	hash.Write(data)
	for _, content := range files {
		hash.Write([]byte{0})
		hash.Write(content)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func Inventory(name string, c ClusterConfig) string {
	var b strings.Builder
	for _, group := range []struct {
		name  string
		hosts []string
	}{{"etcd", c.Etcd}, {"kube_master", c.Masters}, {"kube_node", c.Workers}, {"harbor", nil}, {"ex_lb", nil}, {"chrony", nil}} {
		fmt.Fprintf(&b, "[%s]\n", group.name)
		for _, host := range group.hosts {
			fmt.Fprintln(&b, host)
		}
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, `[all:vars]
ansible_user=%s
ansible_port=%d
ansible_become=true
ansible_python_interpreter=/usr/bin/python3
SECURE_PORT="6443"
CONTAINER_RUNTIME="containerd"
CLUSTER_NETWORK="%s"
PROXY_MODE="ipvs"
SERVICE_CIDR="%s"
CLUSTER_CIDR="%s"
NODE_PORT_RANGE="30000-32767"
CLUSTER_DNS_DOMAIN="cluster.local"
bin_dir="/opt/kube/bin"
base_dir="/etc/kubeasz"
cluster_dir="{{ base_dir }}/clusters/%s"
ca_dir="/etc/kubernetes/ssl"
k8s_nodename=''
`, c.SSHUser, c.SSHPort, c.Network, c.ServiceCIDR, c.PodCIDR, name)
	return b.String()
}

func parseGroups(data []byte) map[string][]string {
	groups := map[string][]string{}
	group := ""
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			group = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && ipv4(fields[0]) {
			groups[group] = append(groups[group], fields[0])
		}
	}
	for _, hosts := range groups {
		sort.Strings(hosts)
	}
	return groups
}

func checkMembership(r Request, hosts []byte) error {
	if !strings.HasPrefix(r.Action, "add-") && !strings.HasPrefix(r.Action, "del-") {
		return nil
	}
	groups := parseGroups(hosts)
	group := map[string]string{"node": "kube_node", "master": "kube_master", "etcd": "etcd"}[strings.Split(r.Action, "-")[1]]
	contains := func(values []string) bool {
		for _, v := range values {
			if v == r.Node {
				return true
			}
		}
		return false
	}
	if strings.HasPrefix(r.Action, "add-") {
		if contains(groups[group]) || (group == "kube_node" && contains(groups["kube_master"])) || (group == "kube_master" && contains(groups["kube_node"])) {
			return errors.New("目标节点已在对应角色或冲突角色的 inventory 中")
		}
	} else {
		if !contains(groups[group]) {
			return errors.New("目标节点不在所选角色的 inventory 中")
		}
		if group == "kube_master" && len(groups[group]) <= 1 {
			return errors.New("不能删除最后一个控制面节点")
		}
		if group == "etcd" && len(groups[group]) <= 3 {
			return errors.New("删除 etcd 成员后须保留至少 3 个成员")
		}
	}
	return nil
}

// Upstream ezctl mutates INI with sed/grep. Restrict imported inventory to the
// same direct IPv4 format so preview targets cannot differ from SSH targets.
func validateInventory(data []byte) error {
	group := ""
	allowed := map[string]bool{"etcd": true, "kube_master": true, "kube_node": true, "harbor": true, "ex_lb": true, "chrony": true, "all:vars": true}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			group = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if !allowed[group] {
				return errors.New("仅支持直接 IPv4 的 kubeasz inventory；不支持 children、范围或自定义主机组")
			}
			continue
		}
		if strings.Contains(line, "ansible_host") || strings.Contains(line, "ansible_ssh_host") || strings.Contains(line, "ansible_connection") {
			return errors.New("系统 inventory 不允许 ansible_host 重映射或覆盖连接类型")
		}
		if group == "all:vars" {
			continue
		}
		fields := strings.Fields(line)
		if !allowed[group] || len(fields) == 0 || !ipv4(fields[0]) {
			return errors.New("系统 inventory 主机行必须以直接 IPv4 开头；不支持别名、范围或动态 inventory")
		}
	}
	groups := parseGroups(data)
	if len(groups["etcd"]) == 0 || len(groups["kube_master"]) == 0 {
		return errors.New("inventory 缺少 etcd 或控制面节点")
	}
	return nil
}
