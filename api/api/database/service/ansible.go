package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/database/model"
)

func automationRoot() (string, error) {
	if configured := os.Getenv("DATABASE_PLATFORM_ROOT"); configured != "" {
		absolute, e := filepath.Abs(configured)
		if e != nil {
			return "", e
		}
		if st, e := os.Stat(filepath.Join(absolute, "site.yml")); e == nil && !st.IsDir() {
			return absolute, nil
		}
		return "", errors.New("DATABASE_PLATFORM_ROOT 不包含数据库自动化入口")
	}
	cwd, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		for _, relative := range []string{"automation/database-platform", "api/automation/database-platform"} {
			p := filepath.Join(cwd, relative)
			if st, e := os.Stat(filepath.Join(p, "site.yml")); e == nil && !st.IsDir() {
				return p, nil
			}
		}
		cwd = filepath.Dir(cwd)
	}
	for _, p := range []string{os.Getenv("DATABASE_PLATFORM_ROOT"), "automation/database-platform", "api/automation/database-platform", "/app/automation/database-platform"} {
		if p == "" {
			continue
		}
		p, _ = filepath.Abs(p)
		if st, e := os.Stat(filepath.Join(p, "site.yml")); e == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", errors.New("数据库自动化目录不存在")
}

// Secrets are written only to a private temporary inventory. No credential is
// passed as a process argument or returned in an operation plan.
func (s *Service) ansible(ctx context.Context, c *model.Cluster, members []model.Member, play string, extra map[string]any, save func(string)) (map[string]json.RawMessage, error) {
	root, e := automationRoot()
	if e != nil {
		return nil, e
	}
	dir, e := os.MkdirTemp("", "autoops-db-worker-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(dir)
	ids := map[uint]bool{}
	if host := uint(number(extra["restore_host_id"])); host > 0 {
		ids[host] = true
	}
	for _, m := range members {
		if m.HostID == 0 {
			return nil, errors.New("成员未关联可管理的 CMDB 主机")
		}
		ids[m.HostID] = true
	}
	root, guarded, e := s.guardedAutomation(ctx, c, ids, root, dir)
	if e != nil {
		return nil, e
	}
	hostVars, e := cmdbservice.NewDatabaseAutomationService(s.DB).InventoryHosts(sortedKeys(ids), dir)
	if e != nil {
		return nil, e
	}
	sec, e := s.secrets(c)
	if e != nil {
		return nil, e
	}
	redactions := []string{sec.Password, sec.Replication, sec.Control, sec.Sentinel, sec.Topology, sec.Fencing, text(extra["fencing_private_key"])}
	if credentials, ok := extra["restore_credentials"].(map[string]any); ok {
		for _, entry := range credentials {
			redactions = append(redactions, text(object(JSON(entry))["password"]))
		}
	}
	if credentials, ok := extra["repository_credentials"].(map[string]string); ok {
		for _, v := range credentials {
			redactions = append(redactions, v)
		}
	}
	hosts := map[string]any{}
	for _, m := range members {
		vars := map[string]any{}
		for k, v := range hostVars[m.HostID] {
			vars[k] = v
		}
		vars["member"] = m
		var user, password string
		if credentials, ok := extra["restore_credentials"].(map[string]any); ok {
			entry := object(JSON(credentials[fmt.Sprintf("member_%d", m.ID)]))
			user, password = text(entry["username"]), text(entry["password"])
		} else {
			user, password, e = s.memberAccount(c, m)
			if e != nil {
				return nil, e
			}
		}
		vars["db_username"], vars["db_password"] = user, password
		redactions = append(redactions, password, text(vars["ansible_password"]))
		hosts[fmt.Sprintf("member_%d", m.ID)] = vars
	}
	if host := uint(number(extra["restore_host_id"])); host > 0 {
		hosts["restore_target"] = hostVars[host]
		redactions = append(redactions, text(hostVars[host]["ansible_password"]))
	}
	vars := map[string]any{"db_cluster": c, "db_secrets": sec, "result_dir": dir, "ansible_python_interpreter": "/usr/bin/python3"}
	vars["ansible_pipelining"] = true
	if guarded {
		vars["ansible_python_interpreter"] = "/usr/local/sbin/autoops-db-python"
	}
	for k, v := range extra {
		vars[k] = v
	}
	inventory := map[string]any{"all": map[string]any{"hosts": hosts, "vars": vars}}
	path := filepath.Join(dir, "inventory.json")
	if e = os.WriteFile(path, JSON(inventory), 0600); e != nil {
		return nil, e
	}
	program := os.Getenv("DATABASE_ANSIBLE_PLAYBOOK")
	if program == "" {
		program = "ansible-playbook"
	}
	cmd := exec.CommandContext(ctx, program, "-i", path, filepath.Join(root, play))
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "ANSIBLE_HOST_KEY_CHECKING=True", "ANSIBLE_NOCOLOR=1", "ANSIBLE_RETRY_FILES_ENABLED=False", "ANSIBLE_LOG_PATH=/dev/null")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
	cmd.WaitDelay = 5 * time.Second
	log := &safeLog{save: save, secrets: redactions}
	cmd.Stdout, cmd.Stderr = log, log
	e = cmd.Run()
	log.flush()
	if e != nil {
		return nil, errors.New("Ansible 执行未完成，请查看脱敏步骤日志并核对远端状态")
	}
	results := map[string]json.RawMessage{}
	for name := range hosts {
		raw, e := os.ReadFile(filepath.Join(dir, name+".json"))
		if e == nil && json.Valid(raw) {
			results[name] = raw
		}
	}
	return results, nil
}

type safeLog struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	secrets []string
	save    func(string)
	last    time.Time
}

func (l *safeLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf.Write(p)
	if l.buf.Len() > 2<<20 {
		b := append([]byte(nil), l.buf.Bytes()[l.buf.Len()-(2<<20):]...)
		l.buf.Reset()
		l.buf.Write(b)
	}
	if time.Since(l.last) > time.Second {
		l.publish()
	}
	return len(p), nil
}
func (l *safeLog) publish() {
	v := l.buf.String()
	for _, secret := range l.secrets {
		if secret == "" {
			continue
		}
		v = strings.ReplaceAll(v, secret, "[REDACTED]")
		for i := len(secret) - 1; i > 0; i-- {
			if strings.HasSuffix(v, secret[:i]) {
				v = v[:len(v)-i] + "[REDACTED]"
				break
			}
		}
	}
	if l.save != nil {
		l.save(v)
	}
	l.last = time.Now()
}
func (l *safeLog) flush() { l.mu.Lock(); defer l.mu.Unlock(); l.publish() }

func (s *Service) remoteJSON(ctx context.Context, host uint, script string, input any) (json.RawMessage, error) {
	return s.remoteJSONCommand(ctx, host, "python3", script, input)
}

func (s *Service) remoteMutation(ctx context.Context, host uint, script string, input any) (json.RawMessage, error) {
	if e := s.executionOwned(ctx); e != nil {
		return nil, e
	}
	op, ok := ctx.Value(operationContextKey{}).(model.Operation)
	if !ok {
		return nil, errors.New("远端变更要求持久化任务身份")
	}
	if _, e := s.remoteJSON(ctx, host, prepareGuardScript, map[string]any{"cluster": op.ClusterID, "epoch": op.ID, "program": remotePythonGuard}); e != nil {
		return nil, e
	}
	if e := s.executionOwned(ctx); e != nil {
		return nil, e
	}
	command := fmt.Sprintf("AUTOOPS_DB_CLUSTER_ID=%d AUTOOPS_DB_OPERATION_ID=%d /usr/local/sbin/autoops-db-python", op.ClusterID, op.ID)
	return s.remoteJSONCommand(ctx, host, command, script, input)
}

func (s *Service) remoteJSONCommand(ctx context.Context, host uint, interpreter, script string, input any) (json.RawMessage, error) {
	client, e := cmdbservice.ResolveAssetSSH(s.DB, "host", host, 0)
	if e != nil {
		return nil, errors.New("CMDB SSH 连接或指纹校验失败")
	}
	defer client.Close()
	session, e := client.NewSession()
	if e != nil {
		return nil, e
	}
	defer session.Close()
	// The script is embedded application code; user input travels on stdin as JSON.
	session.Stdin = bytes.NewReader(JSON(input))
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			session.Close()
		case <-done:
		}
	}()
	b, e := session.Output(interpreter + " -c '" + strings.ReplaceAll(script, "'", "'\"'\"'") + "'")
	if e != nil || !json.Valid(b) {
		return nil, errors.New("主机检查未完成或返回格式无效")
	}
	return b, nil
}

const preflightScript = `import sys,json,socket,os,shutil,subprocess
p=json.load(sys.stdin); results=[]
def add(name,ok,msg): results.append(dict(name=name,passed=bool(ok),message=msg))
mem={k:int(v.split()[0]) for k,v in (line.split(':',1) for line in open('/proc/meminfo'))}
add('可用内存',mem.get('MemAvailable',0)>p['memoryMB']*1024,'需要满足显式资源配额')
add('磁盘余量',shutil.disk_usage('/var/lib').free>2*1024**3,'至少需要 2 GiB 空闲空间')
add('systemd',os.path.isdir('/run/systemd/system'),'部署要求 systemd')
ntp=subprocess.run(['timedatectl','show','-p','NTPSynchronized','--value'],capture_output=True,text=True)
add('时间同步',ntp.returncode==0 and ntp.stdout.strip()=='yes','请配置并验证系统时间同步')
for m in p['members']:
 ports=[m['port']]
 if p['topology']=='redis-cluster' and m['kind']=='data': ports.append(m['port']+10000)
 if p['engine']=='mysql' and m['kind']=='data': ports.append(m['port']+20000)
 if m['kind'] in ('etcd','proxy'): ports.append(m['port']+1)
 if m['kind']=='orchestrator': ports.append(m['port']+2000)
 for port in ports:
  busy=False
  for host in ('127.0.0.1',m['address']):
   sock=socket.socket();sock.settimeout(1)
   try: busy=busy or sock.connect_ex((host,port))==0
   finally: sock.close()
  add('端口 '+str(port),not busy and port<=65535,'端口必须空闲，不能覆盖已有实例')
 add('目录 '+m['dataDir'],not os.path.lexists(m['dataDir']),'新建实例不允许覆盖已存在的目录')
 add('服务 '+m['service'],not os.path.lexists('/etc/systemd/system/'+m['service']+'.service'),'服务名必须独立')
for peer in p['peers']:
 sock=socket.socket();sock.settimeout(2)
 try: sock.connect((peer,22));ok=True
 except OSError: ok=False
 finally: sock.close()
 add('节点连通性 '+peer,ok,'节点间管理网络需要可达')
print(json.dumps(results))`

func (s *Service) preflightHosts(ctx context.Context, c *model.Cluster) []Check {
	groups := map[uint][]model.Member{}
	peers := []string{}
	for _, m := range c.Members {
		groups[m.HostID] = append(groups[m.HostID], m)
		if m.Kind == "data" {
			peers = append(peers, m.Address)
		}
	}
	out := []Check{}
	for host, members := range groups {
		memory := number(object(c.Policy)["memoryMB"])
		dataNodes := 0
		for _, m := range members {
			if m.Kind == "data" {
				dataNodes++
			}
		}
		memory = memory * float64(dataNodes) * 2
		if memory < 128 {
			memory = 128
		}
		sub, cancel := context.WithTimeout(ctx, 20*time.Second)
		raw, e := s.remoteJSON(sub, host, preflightScript, map[string]any{"members": members, "memoryMB": memory, "topology": c.Topology, "engine": c.Engine, "peers": peers})
		cancel()
		if e != nil {
			out = append(out, Check{fmt.Sprintf("主机 %d", host), false, e.Error()})
			continue
		}
		var checks []Check
		if json.Unmarshal(raw, &checks) == nil {
			for _, c := range checks {
				c.Name = fmt.Sprintf("主机 %d · %s", host, c.Name)
				out = append(out, c)
			}
		}
	}
	return out
}
