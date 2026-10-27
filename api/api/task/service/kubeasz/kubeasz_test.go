package kubeasz

import (
	"bytes"
	"context"
	"dodevops-api/api/task/model"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func validConfig() *ClusterConfig {
	return &ClusterConfig{Masters: []string{"192.168.10.11"}, Workers: []string{"192.168.10.21"}, Etcd: []string{"192.168.10.11"}, SSHUser: "root", SSHPort: 22, PodCIDR: "172.20.0.0/16", ServiceCIDR: "10.68.0.0/16", Network: "calico"}
}
func TestFixedCommandCatalog(t *testing.T) {
	cases := []Request{{Action: "new", Cluster: "prod", Config: validConfig()}, {Action: "setup", Cluster: "prod", Step: "all"}, {Action: "add-node", Cluster: "prod", Node: "192.168.10.22"}, {Action: "del-master", Cluster: "prod", Node: "192.168.10.11"}, {Action: "addon", Cluster: "prod", Addon: "metrics-server"}, {Action: "restore", Cluster: "prod", Snapshot: "snapshot_202610011230.db"}}
	for _, req := range cases {
		t.Run(req.Action, func(t *testing.T) {
			commands, e := Commands(req)
			if e != nil {
				t.Fatal(e)
			}
			if len(commands) != 1 {
				t.Fatal(commands)
			}
			cmd := commands[0]
			if cmd.Program != AnsibleBin+"/ansible-playbook" && cmd.Program != Root+"/ezctl" {
				t.Fatal("uncontrolled executable", cmd)
			}
			if strings.Contains(cmd.String(), " -c ") {
				t.Fatal("shell execution")
			}
		})
	}
	restore, _ := Commands(cases[5])
	if restore[0].Args[len(restore[0].Args)-1] != "playbooks/95.restore.yml" {
		t.Fatal(restore)
	}
}
func TestRejectUnsafeOrIrrelevantInputs(t *testing.T) {
	cases := []Request{{Action: "backup", Cluster: "../../tmp"}, {Action: "backup", Cluster: "prod;touch-x"}, {Action: "backup", Cluster: "$(id)"}, {Action: "backup", Cluster: "-prod"}, {Action: "destroy", Cluster: "prod"}, {Action: "add-node", Cluster: "prod", Node: "192.168.0.1;id"}, {Action: "add-node", Cluster: "prod", Node: "192.168.0.1 ansible_password=secret"}, {Action: "add-node", Cluster: "prod", Node: "127.0.0.1"}, {Action: "add-node", Cluster: "prod", Node: "::1"}, {Action: "setup", Cluster: "prod", Step: "all -e @/etc/passwd"}, {Action: "addon", Cluster: "prod", Addon: "../metrics-server"}, {Action: "restore", Cluster: "prod", Snapshot: "../../etc/passwd"}, {Action: "backup", Cluster: "prod", Node: "192.168.0.1"}, {Action: "backup", Cluster: "prod", Config: validConfig()}}
	for _, req := range cases {
		if _, e := Commands(req); e == nil {
			t.Fatalf("accepted unsafe request: %+v", req)
		}
	}
	config := validConfig()
	config.SSHUser = "root\nansible_connection=local"
	if Validate(Request{Cluster: "prod", Action: "new", Config: config}) == nil {
		t.Fatal("accepted inventory injection")
	}
	config = validConfig()
	config.PodCIDR = "10.68.0.0/16"
	if Validate(Request{Cluster: "prod", Action: "new", Config: config}) == nil {
		t.Fatal("accepted overlapping CIDRs")
	}
	config = validConfig()
	config.PodCIDR = "192.168.0.0/16"
	if Validate(Request{Cluster: "prod", Action: "new", Config: config}) == nil {
		t.Fatal("accepted node CIDR overlap")
	}
}
func TestInventoryAndDeletionSafeguards(t *testing.T) {
	inventory := []byte(Inventory("prod", *validConfig()))
	if !strings.Contains(string(inventory), `cluster_dir="{{ base_dir }}/clusters/prod"`) {
		t.Fatal("cluster directory missing")
	}
	for _, req := range []Request{{Action: "del-master", Cluster: "prod", Node: "192.168.10.11"}, {Action: "del-etcd", Cluster: "prod", Node: "192.168.10.11"}, {Action: "del-node", Cluster: "prod", Node: "192.168.10.99"}, {Action: "del-node", Cluster: "prod", Node: "192.168.10.11"}, {Action: "add-node", Cluster: "prod", Node: "192.168.10.11"}} {
		if checkMembership(req, inventory) == nil {
			t.Fatalf("unsafe membership accepted: %+v", req)
		}
	}
	if e := checkMembership(Request{Action: "del-node", Cluster: "prod", Node: "192.168.10.21"}, inventory); e != nil {
		t.Fatal(e)
	}
}
func TestPlanBindsInputsAndConfiguration(t *testing.T) {
	req := Request{Cluster: "prod", Action: "add-node", Node: "192.168.10.22"}
	original := planDigest(req, []byte("inventory"), []byte("config"))
	req.Node = "192.168.10.23"
	if planDigest(req, []byte("inventory"), []byte("config")) == original {
		t.Fatal("target not bound")
	}
	req.Node = "192.168.10.22"
	req.Confirmation = "anything"
	req.PlanHash = "anything"
	if planDigest(req, []byte("inventory"), []byte("config")) != original {
		t.Fatal("confirmation affected digest")
	}
	if planDigest(req, []byte("changed inventory"), []byte("config")) == original {
		t.Fatal("inventory not bound")
	}
	if Confirmation(req) != "add-node prod 192.168.10.22" {
		t.Fatal("confirmation missing target")
	}
}
func testService(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	db, e := gorm.Open(sqlite.Open(filepath.Join(root, "test.db")), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&model.KubeaszOperation{}); e != nil {
		t.Fatal(e)
	}
	return &Service{db: db, root: root, cancel: map[uint]context.CancelFunc{}}
}
func TestNoSymlinkTraversal(t *testing.T) {
	s := testService(t)
	if e := os.Symlink(t.TempDir(), filepath.Join(s.root, "clusters")); e != nil {
		t.Fatal(e)
	}
	if _, e := s.path("clusters", "prod", "hosts"); e == nil {
		t.Fatal("accepted symlink")
	}
	if _, e := s.path("..", "hosts"); e == nil {
		t.Fatal("accepted traversal")
	}
}
func TestGlobalOperationLock(t *testing.T) {
	s := testService(t)
	lock, e := s.lock()
	if e != nil {
		t.Fatal(e)
	}
	other := &Service{root: s.root}
	if second, e := other.lock(); e == nil {
		unlock(second)
		t.Fatal("concurrent operation acquired lock")
	}
	unlock(lock)
	next, e := other.lock()
	if e != nil {
		t.Fatal(e)
	}
	unlock(next)
}
func TestLogRedactsAcrossOutputChunks(t *testing.T) {
	var output bytes.Buffer
	w := newLogWriter(&output)
	w.Write([]byte("TASK [prepare]\npass"))
	w.Write([]byte("word=supersecret\n-----BEGIN RSA PRIVATE KEY-----\nbase64-secret\n"))
	w.Write([]byte("-----END RSA PRIVATE KEY-----\nok\n"))
	w.Close()
	if strings.Contains(output.String(), "supersecret") || strings.Contains(output.String(), "base64-secret") {
		t.Fatal("secret leaked", output.String())
	}
	if !strings.Contains(output.String(), "TASK [prepare]") || !strings.Contains(output.String(), "ok") {
		t.Fatal("ordinary output removed")
	}
}
func TestExecutionStatusAndCancel(t *testing.T) {
	for _, tc := range []struct {
		name    string
		script  string
		timeout time.Duration
		status  string
	}{{"success", "echo completed", time.Second, "succeeded"}, {"failure", "echo failed; exit 7", time.Second, "failed"}, {"cancel-process-group", "sleep 30 & wait", 150 * time.Millisecond, "canceled"}} {
		t.Run(tc.name, func(t *testing.T) {
			s := testService(t)
			lock, e := s.lock()
			if e != nil {
				t.Fatal(e)
			}
			op := model.KubeaszOperation{Cluster: "prod", Action: "backup", Status: "queued", Command: "test fixture"}
			if e = s.db.Create(&op).Error; e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			s.cancel[op.ID] = cancel
			before := time.Now()
			s.run(ctx, cancel, lock, op, &Plan{Commands: []Command{{Program: "/bin/sh", Args: []string{"-c", tc.script}}}})
			if time.Since(before) > 5*time.Second {
				t.Fatal("subprocess group did not stop promptly")
			}
			detail, log, e := s.Detail(op.ID)
			if e != nil {
				t.Fatal(e)
			}
			if detail.Status != tc.status || detail.FinishedAt == nil || detail.StartedAt == nil {
				t.Fatalf("unexpected state: %+v", detail)
			}
			if log == "" {
				t.Fatal("missing persistent log")
			}
			if e = s.Cancel(op.ID); e == nil {
				t.Fatal("completed operation still cancelable")
			}
			lock, e = s.lock()
			if e != nil {
				t.Fatal("lock not released", e)
			}
			unlock(lock)
		})
	}
}
func TestAnsibleEnvironmentEnforcesCredentialProtection(t *testing.T) {
	t.Setenv("ANSIBLE_HOST_KEY_CHECKING", "False")
	t.Setenv("ANSIBLE_LOG_PATH", "/tmp/unredacted.log")
	t.Setenv("KUBECONFIG", "/wrong-cluster")
	s := testService(t)
	env := strings.Join(s.environment("prod"), "\n")
	if strings.Contains(env, "ANSIBLE_LOG_PATH") || strings.Contains(env, "HOST_KEY_CHECKING=False") || strings.Contains(env, "/wrong-cluster") {
		t.Fatal("unsafe inherited environment", env)
	}
	for _, required := range []string{"ANSIBLE_HOST_KEY_CHECKING=True", "ANSIBLE_NO_LOG=True", "StrictHostKeyChecking=yes", "BatchMode=yes", "/clusters/prod/kubectl.kubeconfig"} {
		if !strings.Contains(env, required) {
			t.Fatal("missing environment", required)
		}
	}
}
func TestConcurrentLogWrites(t *testing.T) {
	var output bytes.Buffer
	w := newLogWriter(&output)
	var group sync.WaitGroup
	for i := 0; i < 4; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for i := 0; i < 10; i++ {
				w.Write([]byte("ordinary task output\n"))
			}
		}()
	}
	group.Wait()
	w.Close()
	if strings.Count(output.String(), "ordinary task output") != 40 {
		t.Fatal("lost log output")
	}
}

func TestRejectAmbiguousImportedInventory(t *testing.T) {
	valid := Inventory("prod", *validConfig())
	if err := validateInventory([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{strings.Replace(valid, "192.168.10.11", "master-a ansible_host=192.168.10.11", 1), strings.Replace(valid, "[kube_node]", "[kube_node:children]", 1), strings.Replace(valid, "192.168.10.21", "192.168.10.21 ansible_host=192.168.10.99", 1), valid + "\nansible_connection=local\n"} {
		if validateInventory([]byte(content)) == nil {
			t.Fatal("accepted ambiguous inventory")
		}
	}
}
func TestInterruptedOperationRequiresRecordedReview(t *testing.T) {
	s := testService(t)
	op := model.KubeaszOperation{Cluster: "prod", Action: "backup", Status: "running"}
	if err := s.db.Create(&op).Error; err != nil {
		t.Fatal(err)
	}
	if s.requireReconciled() == nil {
		t.Fatal("stale operation did not block")
	}
	if err := s.Reconcile(op.ID, "wrong", "Checked processes and cluster", "admin"); err == nil {
		t.Fatal("bad confirmation accepted")
	}
	if err := s.Reconcile(op.ID, fmt.Sprintf("reconcile %d prod", op.ID), "Checked processes and cluster", "admin"); err != nil {
		t.Fatal(err)
	}
	if err := s.requireReconciled(); err != nil {
		t.Fatal(err)
	}
	var stored model.KubeaszOperation
	s.db.First(&stored, op.ID)
	if stored.Status != "interrupted" || stored.ReconciledAt == nil || stored.ReconciledBy != "admin" {
		t.Fatalf("missing persisted audit: %+v", stored)
	}
}
func TestBadSnapshotNeverStartsRemoteRestore(t *testing.T) {
	s := testService(t)
	for _, dir := range []string{"bin", "clusters/prod/backup"} {
		if err := os.MkdirAll(filepath.Join(s.root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(s.root, "bin/etcdctl"), []byte("#!/bin/sh\necho invalid-snapshot\nexit 9\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.root, "clusters/prod/backup/snapshot.db"), []byte("corrupt snapshot"), 0600); err != nil {
		t.Fatal(err)
	}
	lock, err := s.lock()
	if err != nil {
		t.Fatal(err)
	}
	op := model.KubeaszOperation{Cluster: "prod", Action: "restore", Status: "queued"}
	if err = s.db.Create(&op).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	marker := filepath.Join(s.root, "remote-was-called")
	s.run(ctx, cancel, lock, op, &Plan{Request: Request{Action: "restore", Cluster: "prod", Snapshot: "snapshot.db"}, Commands: []Command{{Program: "/usr/bin/touch", Args: []string{marker}}}})
	if _, err = os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("remote restore ran after invalid snapshot")
	}
	detail, _, err := s.Detail(op.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Status != "failed" || !strings.Contains(detail.Error, "未启动远端恢复") {
		t.Fatalf("missing restore failure: %+v", detail)
	}
	entries, err := os.ReadDir(filepath.Join(s.root, ".autoops"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "restore-check-") {
			t.Fatal("restore scratch directory leaked")
		}
	}
}
