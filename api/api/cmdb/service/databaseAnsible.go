package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"dodevops-api/api/cmdb/model"
	"golang.org/x/crypto/ssh"
)

// inventory uses JSON-compatible YAML, never interpolated INI/shell fragments.
func (s *DatabaseAutomationService) inventory(d model.DatabaseDeployment, dir string) (map[string]any, error) {
	children := map[string]any{}
	keys := []string{}
	for group, ids := range map[string][]uint{"mysql_nodes": decodeIDs(d.Nodes), "orc_manager": decodeIDs(d.Managers)} {
		hosts := map[string]any{}
		for _, id := range ids {
			p, err := s.hostConnection(id)
			if err != nil {
				return nil, err
			}
			cred, err := connectionCredential(s.db, p)
			if err != nil {
				return nil, err
			}
			user := p.Username
			if user == "" {
				user = cred.Username
			}
			address := net.JoinHostPort(p.Address, fmt.Sprint(p.Port))
			var key ssh.PublicKey
			conn, err := net.DialTimeout("tcp", address, 8*time.Second)
			if err != nil {
				return nil, fmt.Errorf("主机 %d SSH 不可达", id)
			}
			_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
			_, _, _, _ = ssh.NewClientConn(conn, address, &ssh.ClientConfig{User: user, HostKeyCallback: func(_ string, _ net.Addr, k ssh.PublicKey) error {
				if ssh.FingerprintSHA256(k) == p.HostKey {
					key = k
				}
				return errors.New("host key captured")
			}})
			_ = conn.Close()
			if key == nil {
				return nil, fmt.Errorf("主机 %d SSH 指纹不匹配", id)
			}
			keys = append(keys, fmt.Sprintf("[%s]:%d %s", p.Address, p.Port, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))))
			if p.Port == 22 {
				keys = append(keys, p.Address+" "+strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))
			}
			vars := map[string]any{"ansible_host": p.Address, "ansible_port": p.Port, "ansible_user": user, "database_address": p.Address, "database_server_id": id + 1000, "database_primary": id == decodeIDs(d.Nodes)[0], "ansible_ssh_common_args": "-o StrictHostKeyChecking=yes -o UserKnownHostsFile=" + filepath.Join(dir, "known_hosts")}
			switch cred.Type {
			case 1:
				vars["ansible_password"] = cred.Password
			case 2:
				path := filepath.Join(dir, fmt.Sprintf("key-%d", id))
				if err := os.WriteFile(path, []byte(strings.TrimSpace(strings.ReplaceAll(cred.PublicKey, "\r\n", "\n"))+"\n"), 0600); err != nil {
					return nil, err
				}
				vars["ansible_ssh_private_key_file"] = path
			case 3: // uses the controller's established default key
			default:
				return nil, errors.New("不支持的 SSH 认证方式")
			}
			hosts[fmt.Sprintf("node_%d", id)] = vars
		}
		children[group] = map[string]any{"hosts": hosts}
	}
	if err := os.WriteFile(filepath.Join(dir, "known_hosts"), []byte(strings.Join(keys, "\n")+"\n"), 0600); err != nil {
		return nil, err
	}
	return map[string]any{"all": map[string]any{"children": children}}, nil
}

type databaseRunLog struct {
	mu      sync.Mutex
	text    string
	secrets []string
	save    func(string)
	last    time.Time
}

func (l *databaseRunLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Redaction occurs on the whole accumulated buffer, including secrets split
	// between writes. Only Ansible no_log tasks handle database/SSH credentials.
	l.text += string(p)
	if len(l.text) > 2<<20 {
		l.text = l.text[len(l.text)-(2<<20):]
	}
	if time.Since(l.last) > time.Second {
		l.save(l.redacted())
		l.last = time.Now()
	}
	return len(p), nil
}
func (l *databaseRunLog) redacted() string {
	value := l.text
	for _, s := range l.secrets {
		if s != "" {
			value = strings.ReplaceAll(value, s, "[REDACTED]")
			// A subprocess write may end halfway through a password. Do not
			// persist that partial prefix while waiting for the next chunk.
			for n := len(s) - 1; n > 0; n-- {
				if strings.HasSuffix(value, s[:n]) {
					value = value[:len(value)-n] + "[REDACTED]"
					break
				}
			}
		}
	}
	return value
}
func (s *DatabaseAutomationService) execute(d model.DatabaseDeployment, run model.DatabaseAutomationRun) {
	status, artifact := "failed", ""
	log := &databaseRunLog{save: func(value string) {
		s.db.Model(&model.DatabaseAutomationRun{}).Where("id = ?", run.ID).Update("log", value)
	}}
	defer func() {
		if recover() != nil {
			log.text += "\n执行器异常，检查目标主机状态后重试。"
		}
		s.finish(run, status, log.redacted(), artifact)
	}()
	sec, err := s.secrets(&d)
	if err != nil {
		log.text = err.Error()
		return
	}
	log.secrets = []string{sec.Password, sec.Replication, sec.Topology, sec.HTTP}
	s.db.Model(&run).Update("status", "running")
	install := run.Action == "install" || run.Action == "retry-install"
	if install {
		s.db.Model(&d).Update("status", "installing")
		defer func() {
			if status != "success" {
				s.db.Model(&d).Where("status <> ?", "registration_failed").Update("status", "failed")
			}
		}()
	}
	root, err := databaseAutomationRoot()
	if err != nil {
		log.text = err.Error()
		return
	}
	dir, err := os.MkdirTemp("", "autoops-db-")
	if err != nil {
		log.text = "无法建立临时执行目录"
		return
	}
	defer os.RemoveAll(dir)
	inventory, err := s.inventory(d, dir)
	if err != nil {
		log.text = err.Error()
		return
	}
	encoded, _ := json.Marshal(inventory)
	if err = os.WriteFile(filepath.Join(dir, "inventory.json"), encoded, 0600); err != nil {
		log.text = "无法写入主机清单"
		return
	}
	nodes := decodeIDs(d.Nodes)
	first, err := s.hostConnection(nodes[0])
	if err != nil {
		log.text = err.Error()
		return
	}
	managers := []string{}
	for _, id := range decodeIDs(d.Managers) {
		p, e := s.hostConnection(id)
		if e != nil {
			log.text = e.Error()
			return
		}
		managers = append(managers, p.Address)
	}
	vars := map[string]any{
		"database_engine": d.Engine, "database_mode": d.Mode, "database_port": d.Port, "database_password": sec.Password,
		"database_name": d.Name, "database_run_id": run.ID, "database_primary_host": first.Address,
		"mysql_root_password": sec.Password, "repl_password": sec.Replication, "orc_topology_pass": sec.Topology, "orc_http_password": sec.HTTP,
		"orc_nodes": managers, "orc_auto_recovery": sec.AutoRecovery,
		"mysql_version": d.Version, "mysql_port": d.Port,
	}
	encoded, _ = json.Marshal(vars)
	if err = os.WriteFile(filepath.Join(dir, "vars.json"), encoded, 0600); err != nil {
		log.text = "无法写入执行参数"
		return
	}
	play := "backup.yml"
	if install {
		play = "install.yml"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ansible-playbook", "-i", filepath.Join(dir, "inventory.json"), filepath.Join(root, "playbooks", play), "--extra-vars", "@"+filepath.Join(dir, "vars.json"))
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "ANSIBLE_HOST_KEY_CHECKING=True", "ANSIBLE_NOCOLOR=1", "ANSIBLE_DISPLAY_ARGS_TO_STDOUT=False", "ANSIBLE_ROLES_PATH="+filepath.Join(root, "roles"), "ANSIBLE_LOG_PATH="+filepath.Join(dir, "ansible.log"))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 5 * time.Second
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Run(); err != nil {
		log.text += "\nAnsible 执行失败；任务未标记成功。"
		return
	}
	if install {
		if err := s.registerAssets(&d, sec); err != nil {
			log.text += "\n安装完成，但资产登记失败；重试安装可补齐登记。"
			s.db.Model(&d).Update("status", "registration_failed")
			return
		}
		if len(managers) > 0 {
			client := s.orchestrator(&d, sec)
			recoveryAction := "disable-global-recoveries"
			if sec.AutoRecovery {
				recoveryAction = "enable-global-recoveries"
			}
			if _, e := client.call(ctx, recoveryAction, true); e != nil {
				log.text += "\norchestrator 自动恢复状态设置失败。"
				return
			}
			for _, id := range nodes {
				p, e := s.hostConnection(id)
				if e != nil {
					log.text = e.Error()
					return
				}
				path, _ := databaseInstancePath(p.Address, d.Port)
				if _, e := client.call(ctx, "discover/"+path, false); e != nil {
					log.text += "\norchestrator 发现失败，请检查管理端连通性后重试。"
					return
				}
			}
		}
		s.db.Model(&d).Update("status", "ready")
	} else {
		artifact = fmt.Sprintf("/var/backups/autoops/%s/%d", d.Name, run.ID)
	}
	status = "success"
}
