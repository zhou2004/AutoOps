package kubeasz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"dodevops-api/api/task/model"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	root   string
	mu     sync.Mutex
	cancel map[uint]context.CancelFunc
}

type Check struct {
	Name   string `json:"name"`
	Ready  bool   `json:"ready"`
	Detail string `json:"detail"`
}
type Cluster struct {
	Name      string   `json:"name"`
	Masters   []string `json:"masters"`
	Workers   []string `json:"workers"`
	Etcd      []string `json:"etcd"`
	Snapshots []string `json:"snapshots"`
}
type Catalog struct {
	Version    string            `json:"version"`
	Commit     string            `json:"commit"`
	Kubernetes string            `json:"kubernetes"`
	Root       string            `json:"root"`
	Checks     []Check           `json:"checks"`
	Templates  []Template        `json:"templates"`
	Steps      map[string]string `json:"steps"`
	Addons     map[string]string `json:"addons"`
	Clusters   []Cluster         `json:"clusters"`
}
type Plan struct {
	Request      Request             `json:"request"`
	Commands     []Command           `json:"commands"`
	Command      string              `json:"command"`
	Confirmation string              `json:"confirmation"`
	Hash         string              `json:"hash"`
	Ready        bool                `json:"ready"`
	Blockers     []string            `json:"blockers"`
	Warnings     []string            `json:"warnings"`
	Inventory    string              `json:"inventory,omitempty"`
	Targets      map[string][]string `json:"targets"`
}

func NewService(db *gorm.DB) *Service {
	s := &Service{db: db, root: Root, cancel: map[uint]context.CancelFunc{}}
	// Never retry an interrupted infrastructure action automatically. Only recover
	// when no other process holds the shared kubeasz filesystem lock.
	if lock, err := s.lock(); err == nil {
		now := time.Now()
		db.Model(&model.KubeaszOperation{}).Where("status IN ?", []string{"queued", "running"}).Updates(map[string]interface{}{"status": "interrupted", "error": "执行器已重启；请检查目标节点实际状态后再创建新操作", "finished_at": now})
		unlock(lock)
	}
	return s
}

// path refuses symlinks, including intermediate components. Users cannot supply
// paths; the checks also stop a mounted cluster symlink escaping the system root.
func (s *Service) path(parts ...string) (string, error) {
	root, err := filepath.EvalSymlinks(s.root)
	if err != nil {
		return "", err
	}
	if root != filepath.Clean(s.root) {
		return "", errors.New("kubeasz 系统目录不能是符号链接")
	}
	current := root
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, "/\\\x00") {
			return "", errors.New("无效系统文件路径")
		}
		current = filepath.Join(current, part)
		st, err := os.Lstat(current)
		if err == nil && st.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("系统配置不允许符号链接")
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}
	return current, nil
}
func (s *Service) read(parts ...string) ([]byte, error) {
	p, e := s.path(parts...)
	if e != nil {
		return nil, e
	}
	st, e := os.Stat(p)
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() || st.Size() > 2<<20 {
		return nil, errors.New("系统配置必须为小于 2 MiB 的普通文件")
	}
	return os.ReadFile(p)
}
func (s *Service) file(parts ...string) bool {
	p, e := s.path(parts...)
	if e != nil {
		return false
	}
	st, e := os.Stat(p)
	return e == nil && st.Mode().IsRegular() && st.Size() > 0
}

func (s *Service) checks() []Check {
	stamp, e := s.read(".autoops-version")
	stampOK := e == nil && strings.TrimSpace(string(stamp)) == Version+" "+Commit
	ez, e := s.read("ezctl")
	sum := sha256.Sum256(ez)
	sourceOK := e == nil && hex.EncodeToString(sum[:]) == EzctlSHA256
	tools := []string{}
	for _, name := range []string{"bash", AnsibleBin + "/ansible", AnsibleBin + "/ansible-playbook", "ssh"} {
		if _, e := exec.LookPath(name); e != nil {
			tools = append(tools, name)
		}
	}
	binaries := []string{}
	for _, name := range []string{"kubectl", "kube-apiserver", "kube-controller-manager", "kube-scheduler", "kube-proxy", "kubelet", "etcd", "etcdctl", "cfssl", "cfssljson", "crictl", "helm"} {
		if !s.file("bin", name) {
			binaries = append(binaries, name)
		}
	}
	if !s.file("bin", "containerd-bin", "containerd") {
		binaries = append(binaries, "containerd-bin/containerd")
	}
	artifacts := false
	if p, e := s.path("down"); e == nil {
		entries, _ := os.ReadDir(p)
		for _, item := range entries {
			if !item.IsDir() && strings.HasSuffix(item.Name(), ".tar") {
				artifacts = true
				break
			}
		}
	}
	home, _ := os.UserHomeDir()
	known, ke := os.Stat(filepath.Join(home, ".ssh", "known_hosts"))
	knownOK := ke == nil && known.Size() > 0
	keyOK := os.Getenv("SSH_AUTH_SOCK") != ""
	for _, name := range []string{"id_ed25519", "id_rsa"} {
		if st, e := os.Stat(filepath.Join(home, ".ssh", name)); e == nil && st.Mode().IsRegular() && st.Size() > 0 {
			keyOK = true
		}
	}
	return []Check{
		{"source", stampOK && sourceOK, "固定 kubeasz " + Version + " / " + Commit + "；版本标记及 ezctl 摘要必须匹配"},
		{"tools", len(tools) == 0, "执行工具缺失: " + strings.Join(tools, ", ")},
		{"binaries", len(binaries) == 0, "离线二进制缺失: " + strings.Join(binaries, ", ")},
		{"images", artifacts, "down 目录须包含离线镜像 tar；这里只检查资源存在，版本 / 架构 / 镜像完整性需部署人员核验"},
		{"ssh_key", keyOK, "挂载执行用户 ~/.ssh/id_ed25519 或 id_rsa，或配置 SSH_AUTH_SOCK"},
		{"known_hosts", knownOK, "挂载已核验的 ~/.ssh/known_hosts；执行器强制启用主机密钥校验"},
	}
}

func (s *Service) Catalog() Catalog {
	c := Catalog{Version: Version, Commit: Commit, Kubernetes: "v1.34.1（默认资源包）", Root: s.root, Checks: s.checks(), Templates: Templates, Steps: Steps, Addons: Addons, Clusters: []Cluster{}}
	p, e := s.path("clusters")
	if e != nil {
		return c
	}
	entries, _ := os.ReadDir(p)
	for _, entry := range entries {
		if !entry.IsDir() || !safeName.MatchString(entry.Name()) {
			continue
		}
		hosts, e := s.read("clusters", entry.Name(), "hosts")
		if e != nil {
			continue
		}
		g := parseGroups(hosts)
		cl := Cluster{Name: entry.Name(), Masters: g["kube_master"], Workers: g["kube_node"], Etcd: g["etcd"], Snapshots: []string{}}
		if p, e := s.path("clusters", entry.Name(), "backup"); e == nil {
			backups, _ := os.ReadDir(p)
			for _, b := range backups {
				if !b.IsDir() && safeSnapshot.MatchString(b.Name()) && s.file("clusters", entry.Name(), "backup", b.Name()) {
					cl.Snapshots = append(cl.Snapshots, b.Name())
				}
			}
		}
		sort.Sort(sort.Reverse(sort.StringSlice(cl.Snapshots)))
		c.Clusters = append(c.Clusters, cl)
	}
	return c
}

func (s *Service) Preview(r Request) (*Plan, error) {
	commands, e := Commands(r)
	if e != nil {
		return nil, e
	}
	r.Confirmation = ""
	r.PlanHash = ""
	p := &Plan{Request: r, Commands: commands, Confirmation: Confirmation(r), Ready: true, Blockers: []string{}, Warnings: []string{}, Targets: map[string][]string{}}
	for _, cmd := range commands {
		if p.Command != "" {
			p.Command += "\n"
		}
		p.Command += cmd.String()
	}
	var hosts, config, backup []byte
	if r.Action == "new" {
		dir, e := s.path("clusters", r.Cluster)
		if e != nil {
			return nil, fmt.Errorf("系统目录未准备: %w", e)
		}
		if _, e = os.Lstat(dir); e == nil {
			return nil, errors.New("集群配置已经存在；不会覆盖现有配置")
		}
		p.Inventory = Inventory(r.Cluster, *r.Config)
		p.Targets = parseGroups([]byte(p.Inventory))
	} else {
		hosts, e = s.read("clusters", r.Cluster, "hosts")
		if e != nil {
			return nil, errors.New("集群 inventory 不存在或不是安全的普通文件")
		}
		config, e = s.read("clusters", r.Cluster, "config.yml")
		if e != nil {
			return nil, errors.New("集群 config.yml 不存在或不是安全的普通文件")
		}
		if e = validateInventory(hosts); e != nil {
			return nil, e
		}
		var configValues map[string]interface{}
		if e = yaml.Unmarshal(config, &configValues); e != nil {
			return nil, errors.New("config.yml YAML 格式错误")
		}
		for _, key := range []string{"ansible_host", "ansible_ssh_host", "ansible_connection"} {
			if _, ok := configValues[key]; ok {
				return nil, errors.New("config.yml 不允许覆盖目标 SSH 地址或连接类型")
			}
		}
		if e = checkMembership(r, hosts); e != nil {
			return nil, e
		}
		p.Targets = parseGroups(hosts)
		if r.Action == "restore" {
			file, e := s.path("clusters", r.Cluster, "backup", r.Snapshot)
			if e != nil {
				return nil, e
			}
			st, e := os.Stat(file)
			if e != nil || !st.Mode().IsRegular() || st.Size() == 0 {
				return nil, errors.New("所选快照不存在或为空")
			}
			// Bind restore confirmation to this backup's identity without reading an
			// arbitrarily large snapshot into memory.
			backup = []byte(fmt.Sprintf("%s:%d:%d", r.Snapshot, st.Size(), st.ModTime().UnixNano()))
		}
	}
	if r.Action == "restore" {
		p.Command = "# 本地快照完整恢复校验，临时目录执行时分配；不连接集群\n" + s.root + "/bin/etcdctl snapshot restore clusters/" + r.Cluster + "/backup/" + r.Snapshot + " --data-dir=<系统临时目录>/data --name=autoops-preflight --initial-cluster=autoops-preflight=http://127.0.0.1:2380 --initial-advertise-peer-urls=http://127.0.0.1:2380\n" + p.Command
	}
	p.Hash = planDigest(r, hosts, config, backup)
	for _, check := range s.checks() {
		required := check.Name == "source" || check.Name == "tools"
		if r.Action != "new" {
			required = true
		}
		// Operations on existing services do not consume image archives.
		if check.Name == "images" && (r.Action == "backup" || r.Action == "restore" || r.Action == "start" || r.Action == "stop" || strings.HasPrefix(r.Action, "del-")) {
			required = false
		}
		if required && !check.Ready {
			p.Blockers = append(p.Blockers, check.Detail)
		}
	}
	if r.Action != "new" && r.Action != "setup" && !s.file("clusters", r.Cluster, "kubectl.kubeconfig") {
		p.Blockers = append(p.Blockers, "集群缺少 kubectl.kubeconfig；先完成安装或导入完整 kubeasz 集群目录")
	}
	if r.Action == "addon" && (r.Addon == "prometheus" || r.Addon == "ingress-nginx") && !s.file("bin", "helm") {
		p.Blockers = append(p.Blockers, "所选插件需要 bin/helm，请准备固定版本工具包")
	}
	if r.Action == "addon" && !s.file("autoops-playbooks", r.Addon+".yml") {
		p.Blockers = append(p.Blockers, "系统插件 playbook 缺失；重新运行固定版本安装脚本")
	}
	if r.Action == "new" {
		p.Warnings = append(p.Warnings, "仅创建系统配置，不连接目标主机。密钥由部署端提供；非 root 用户须配置免密 sudo。")
	} else {
		p.Warnings = append(p.Warnings, "该操作会修改目标基础设施。Ansible 失败或取消不会自动回滚；请检查节点、inventory 和集群健康后重试。")
	}
	if r.Action == "backup" {
		p.Warnings = append(p.Warnings, "仅备份 etcd 中的 Kubernetes 对象，业务 PV 数据需通过 Ceph / CSI / 应用备份维护。")
	}
	if r.Action == "restore" || strings.HasPrefix(r.Action, "del-") || r.Action == "stop" {
		p.Warnings = append(p.Warnings, "可能中断业务或删除节点数据；先完成备份、业务迁移和维护窗口确认。")
	}
	if r.Action == "upgrade" {
		p.Warnings = append(p.Warnings, "预览不下载版本；部署人员必须预先替换并核验 bin 二进制和 config.yml 的 K8S_VER，遵循 Kubernetes 逐小版本升级要求。")
	}
	p.Ready = len(p.Blockers) == 0
	return p, nil
}

func (s *Service) lock() (*os.File, error) {
	dir, e := s.path(".autoops")
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	path, e := s.path(".autoops", "operation.lock")
	if e != nil {
		return nil, e
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, errors.New("已有 kubeasz 系统操作执行中，请等待完成")
	}
	return f, nil
}
func unlock(f *os.File) { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }

func (s *Service) Start(r Request, operatorID uint, operatorName string) (*model.KubeaszOperation, error) {
	lock, e := s.lock()
	if e != nil {
		return nil, e
	}
	if e = s.requireReconciled(); e != nil {
		unlock(lock)
		return nil, e
	}
	p, e := s.Preview(r)
	if e != nil {
		unlock(lock)
		return nil, e
	}
	if r.PlanHash != p.Hash || r.Confirmation != p.Confirmation {
		unlock(lock)
		return nil, errors.New("预览已变化或确认词不匹配，请重新预览并确认")
	}
	if !p.Ready {
		unlock(lock)
		return nil, errors.New(strings.Join(p.Blockers, "；"))
	}
	data, _ := json.Marshal(p.Request)
	op := &model.KubeaszOperation{Cluster: r.Cluster, Action: r.Action, Parameters: string(data), Command: p.Command, Version: Version, Status: "queued", OperatorID: operatorID, OperatorName: operatorName}
	if e = s.db.Create(op).Error; e != nil {
		unlock(lock)
		return nil, e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	s.mu.Lock()
	s.cancel[op.ID] = cancel
	s.mu.Unlock()
	go s.run(ctx, cancel, lock, *op, p)
	return op, nil
}

func (s *Service) run(ctx context.Context, cancel context.CancelFunc, lock *os.File, op model.KubeaszOperation, p *Plan) {
	defer unlock(lock)
	defer cancel()
	defer func() { s.mu.Lock(); delete(s.cancel, op.ID); s.mu.Unlock() }()
	start := time.Now()
	status := "succeeded"
	message := ""
	defer func() {
		if recovered := recover(); recovered != nil {
			status = "failed"
			message = "执行器内部错误；请检查目标实际状态"
		}
		s.db.Model(&op).Updates(map[string]interface{}{"status": status, "error": message, "finished_at": time.Now()})
	}()
	if err := s.db.Model(&op).Updates(map[string]interface{}{"status": "running", "started_at": start}).Error; err != nil {
		status = "failed"
		message = "写入执行状态失败，未启动命令"
		return
	}
	logPath, e := s.path(".autoops", fmt.Sprintf("operation-%d.log", op.ID))
	if e != nil {
		status = "failed"
		message = e.Error()
		return
	}
	f, e := os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		status = "failed"
		message = "无法创建操作日志"
		return
	}
	defer f.Close()
	log := newLogWriter(f)
	defer log.Close()
	fmt.Fprintf(log, "kubeasz %s / %s\n目标集群: %s\n命令: %s\n", Version, Commit, op.Cluster, op.Command)
	if op.Action == "restore" {
		fmt.Fprintln(log, "先在本地临时目录验证 etcd 快照可恢复性；验证通过前不会执行远端变更")
		e = s.verifySnapshot(ctx, lock, p.Request, log)
	}
	if e == nil {
		for _, command := range p.Commands {
			if e = s.runCommand(ctx, lock, op.Cluster, command, log); e != nil {
				break
			}
		}
	}
	if e == nil && op.Action == "new" {
		e = s.configure(p.Request)
	}
	if e != nil {
		if op.Action == "new" {
			// A partially generated upstream sample inventory must never become
			// selectable for a later remote setup operation.
			if hosts, pathErr := s.path("clusters", op.Cluster, "hosts"); pathErr == nil {
				_ = os.Rename(hosts, filepath.Join(filepath.Dir(hosts), "hosts.incomplete"))
			}
		}
		status = "failed"
		message = "执行失败: " + e.Error()
		if ctx.Err() != nil {
			status = "canceled"
			message = "执行已取消或超过 4 小时；远端已完成的修改不会回滚"
		}
		fmt.Fprintln(log, message)
	} else {
		fmt.Fprintln(log, "命令已完成，请核查集群及业务状态")
		if strings.HasPrefix(op.Action, "del-") {
			message = "官方删除脚本部分清理步骤会忽略错误；命令退出成功不能代替节点服务、磁盘和 inventory 的实际核查"
			fmt.Fprintln(log, message)
		}
	}
}

func (s *Service) environment(cluster string) []string {
	env := []string{}
	// Explicitly override inherited Ansible settings used by ordinary tasks.
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "ANSIBLE_") && !strings.HasPrefix(value, "KUBECONFIG=") && !strings.HasPrefix(value, "PATH=") {
			env = append(env, value)
		}
	}
	return append(env, "PATH="+AnsibleBin+":"+os.Getenv("PATH"), "ETCDCTL_API=3", "ANSIBLE_CONFIG="+s.root+"/ansible.cfg", "ANSIBLE_ROLES_PATH="+s.root+"/roles", "ANSIBLE_HOST_KEY_CHECKING=True", "ANSIBLE_SSH_ARGS=-C -o ControlMaster=auto -o ControlPersist=60s -o StrictHostKeyChecking=yes -o BatchMode=yes", "ANSIBLE_NO_LOG=True", "ANSIBLE_DISPLAY_ARGS_TO_STDOUT=False", "ANSIBLE_NOCOLOR=1", "ANSIBLE_FORCE_COLOR=False", "KUBECONFIG="+s.root+"/clusters/"+cluster+"/kubectl.kubeconfig")
}

func (s *Service) configure(r Request) error {
	dir, e := s.path("clusters", r.Cluster)
	if e != nil {
		return e
	}
	if e = os.Chmod(dir, 0700); e != nil {
		return e
	}
	// ezctl new creates only local templates. Replace sample inventory before any
	// remote operation and change only the known cluster-name setting.
	if e = s.writeCluster(r.Cluster, "hosts", []byte(Inventory(r.Cluster, *r.Config))); e != nil {
		return e
	}
	raw, e := s.read("clusters", r.Cluster, "config.yml")
	if e != nil {
		return e
	}
	var cfg map[string]interface{}
	if e = yaml.Unmarshal(raw, &cfg); e != nil {
		return e
	}
	cfg["CLUSTER_NAME"] = r.Cluster
	// The API container does not host ezdown's Docker registry. Operators may
	// configure their own trusted mirror in this file before installation.
	cfg["ENABLE_MIRROR_REGISTRY"] = false
	raw, e = yaml.Marshal(cfg)
	if e != nil {
		return e
	}
	return s.writeCluster(r.Cluster, "config.yml", raw)
}
func (s *Service) writeCluster(cluster, name string, data []byte) error {
	p, e := s.path("clusters", cluster, name)
	if e != nil {
		return e
	}
	// The operation lock serializes all API writers. Atomic rename avoids a torn
	// inventory if the API process exits while creating local configuration.
	tmp, e := os.CreateTemp(filepath.Dir(p), ".autoops-config-")
	if e != nil {
		return e
	}
	defer os.Remove(tmp.Name())
	if _, e = tmp.Write(data); e != nil {
		tmp.Close()
		return e
	}
	if e = tmp.Close(); e != nil {
		return e
	}
	return os.Rename(tmp.Name(), p)
}

func (s *Service) Cancel(id uint) error {
	s.mu.Lock()
	cancel, ok := s.cancel[id]
	s.mu.Unlock()
	if !ok {
		return errors.New("此操作未在当前执行器运行或已经结束")
	}
	cancel()
	return nil
}
func (s *Service) History(cluster string, page, size int) ([]model.KubeaszOperation, int64, error) {
	if lock, err := s.lock(); err == nil {
		_ = s.requireReconciled()
		unlock(lock)
	}
	if cluster != "" && !safeName.MatchString(cluster) {
		return nil, 0, errors.New("无效集群名")
	}
	q := s.db.Model(&model.KubeaszOperation{})
	if cluster != "" {
		q = q.Where("cluster = ?", cluster)
	}
	var total int64
	if e := q.Count(&total).Error; e != nil {
		return nil, 0, e
	}
	ops := []model.KubeaszOperation{}
	e := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&ops).Error
	return ops, total, e
}
func (s *Service) Detail(id uint) (*model.KubeaszOperation, string, error) {
	var op model.KubeaszOperation
	if e := s.db.First(&op, id).Error; e != nil {
		return nil, "", e
	}
	p, e := s.path(".autoops", fmt.Sprintf("operation-%d.log", id))
	if e != nil {
		return &op, "", nil
	}
	f, e := os.Open(p)
	if os.IsNotExist(e) {
		return &op, "", nil
	}
	if e != nil {
		return nil, "", e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return nil, "", e
	}
	const limit = 256 << 10
	prefix := ""
	if st.Size() > limit {
		f.Seek(-limit, io.SeekEnd)
		prefix = "[仅显示最后 256 KiB 日志]\n"
	}
	content, e := io.ReadAll(io.LimitReader(f, limit))
	return &op, prefix + string(content), e
}

// requireReconciled is called while holding the global filesystem lock. A stale
// running record can occur if startup happened before an orphan child exited.
func (s *Service) requireReconciled() error {
	if err := s.db.Model(&model.KubeaszOperation{}).Where("status IN ?", []string{"queued", "running"}).Updates(map[string]interface{}{"status": "interrupted", "finished_at": time.Now(), "error": "执行器中断；核查遗留进程和节点实际状态后才能解除运行限制"}).Error; err != nil {
		return err
	}
	var count int64
	if err := s.db.Model(&model.KubeaszOperation{}).Where("status = ? AND reconciled_at IS NULL", "interrupted").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("存在未核查的中断操作；请在历史中检查遗留进程和节点状态，并完成核查确认")
	}
	return nil
}

func (s *Service) Reconcile(id uint, confirmation, note, operator string) error {
	if len(strings.TrimSpace(note)) < 5 || len(note) > 1000 {
		return errors.New("请填写 5–1000 字节的核查记录，说明遗留进程、节点状态及处理结果")
	}
	lock, err := s.lock()
	if err != nil {
		return err
	}
	defer unlock(lock)
	var op model.KubeaszOperation
	if err = s.db.First(&op, id).Error; err != nil {
		return err
	}
	if op.Status != "interrupted" || op.ReconciledAt != nil {
		return errors.New("仅能确认尚未核查的中断操作")
	}
	if confirmation != fmt.Sprintf("reconcile %d %s", id, op.Cluster) {
		return errors.New("核查确认词不匹配")
	}
	return s.db.Model(&op).Updates(map[string]interface{}{"reconciled_at": time.Now(), "reconciled_by": operator, "reconcile_note": strings.TrimSpace(note)}).Error
}

func (s *Service) runCommand(ctx context.Context, lock *os.File, cluster string, command Command, log io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, command.Program, command.Args...)
	cmd.Dir = s.root
	cmd.Env = s.environment(cluster)
	cmd.Stdout = log
	cmd.Stderr = log
	// Keep the flock reference alive in the fixed ezctl/Ansible process tree if
	// the API dies. Recovery still requires a persisted human acknowledgement.
	cmd.ExtraFiles = []*os.File{lock}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
	cmd.WaitDelay = 5 * time.Second
	return cmd.Run()
}

func (s *Service) verifySnapshot(ctx context.Context, lock *os.File, r Request, log io.Writer) error {
	if !safeSnapshot.MatchString(r.Snapshot) {
		return errors.New("恢复快照名称无效")
	}
	snapshot, err := s.path("clusters", r.Cluster, "backup", r.Snapshot)
	if err != nil {
		return err
	}
	before, err := os.Stat(snapshot)
	if err != nil || !before.Mode().IsRegular() || before.Size() == 0 {
		return errors.New("恢复快照不存在或无效")
	}
	scratchRoot, err := s.path(".autoops")
	if err != nil {
		return err
	}
	scratch, err := os.MkdirTemp(scratchRoot, "restore-check-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(scratch)
	binary, err := s.path("bin", "etcdctl")
	if err != nil {
		return err
	}
	command := Command{Program: binary, Args: []string{"snapshot", "restore", snapshot, "--data-dir=" + filepath.Join(scratch, "data"), "--name=autoops-preflight", "--initial-cluster=autoops-preflight=http://127.0.0.1:2380", "--initial-advertise-peer-urls=http://127.0.0.1:2380"}}
	fmt.Fprintln(log, "本地校验命令: "+command.String())
	// A real local restore checks the snapshot hash and database format. It
	// starts no etcd server and has no network connection to cluster members.
	if err = s.runCommand(ctx, lock, r.Cluster, command, log); err != nil {
		return fmt.Errorf("快照本地恢复验证失败，未启动远端恢复: %w", err)
	}
	after, err := os.Stat(snapshot)
	if err != nil || before.Size() != after.Size() || before.ModTime() != after.ModTime() {
		return errors.New("验证期间快照发生变化，未启动远端恢复")
	}
	fmt.Fprintln(log, "快照本地验证通过，开始执行已确认的集群恢复")
	return nil
}
