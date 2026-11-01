package service

import (
	"context"
	"dodevops-api/api/database/model"
	"fmt"
	"testing"
	"time"
)

// A four MiB test-only tmpfs, never the host root volume or a database mount.
const labFullRepository = `import json,sys,pathlib,subprocess,os,errno
p=json.load(sys.stdin);path=pathlib.Path(p['path']);assert str(path).startswith('/var/lib/autoops-backup-repository/full-lab-')
unit='autoops-db-lab-full-'+path.name;script=pathlib.Path('/run')/(unit+'.py')
if p['action']=='create':
 assert not path.exists();path.mkdir(mode=0o700)
 script.write_text("import subprocess,pathlib\np="+repr(str(path))+"\nif subprocess.run(['mountpoint','-q',p]).returncode==0: subprocess.run(['umount',p],check=True)\npathlib.Path(p).rmdir()\n");script.chmod(0o700)
 subprocess.run(['systemd-run','--quiet','--unit',unit,'--on-active=180s','/usr/bin/python3',str(script)],check=True)
 subprocess.run(['mount','-t','tmpfs','-o','size=4m,nr_inodes=128,mode=0700','autoops-lab',str(path)],check=True)
elif p['action']=='fill':
 assert subprocess.check_output(['findmnt','-n','-o','FSTYPE','--target',str(path)],text=True).strip()=='tmpfs'
 try:
  with (path/'fixture-fill').open('wb',buffering=0) as f:
   for _ in range(2048):f.write(b'x'*4096)
 except OSError as e:assert e.errno==errno.ENOSPC
 assert os.statvfs(path).f_bavail==0
else:
 assert script.exists();subprocess.run(['python3',str(script)],check=True)
 subprocess.run(['systemctl','stop',unit+'.timer'],check=True);script.unlink()
print(json.dumps({'done':True}))`

func exerciseStorageFailure(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	if c.ID != 1 || c.Topology != "redis-sentinel" {
		t.Fatal("requires authorized isolated Sentinel")
	}
	root, e := primary(c)
	if e != nil {
		t.Fatal(e)
	}
	host := uint(530)
	if root.HostID == host {
		host = 528
	}
	path := fmt.Sprintf("/var/lib/autoops-backup-repository/full-lab-%d", time.Now().Unix())
	run := func(action string) error {
		_, err := s.remoteJSON(ctx, host, labFullRepository, map[string]any{"action": action, "path": path})
		return err
	}
	if e = run("create"); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if err := run("cleanup"); err != nil {
			t.Error("bounded volume cleanup", err)
		}
	}()
	repo, e := s.SaveRepository(ctx, user, model.Repository{Name: fmt.Sprintf("full-lab-%d", time.Now().Unix()), GroupID: c.GroupID, HostID: host, Kind: "local", Path: path})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CheckRepository(ctx, user, repo.ID); e != nil {
		t.Fatal(e)
	}
	plan, e := s.Plan(ctx, c.ID, user, "backup", Parameters{Method: "rdb", RepositoryID: repo.ID})
	if e != nil || !plan.Executable {
		t.Fatal("backup plan", e)
	}
	if e = run("fill"); e != nil {
		t.Fatal(e)
	}
	op, e := s.Submit(ctx, c.ID, user, plan.ID, fmt.Sprintf("bounded-volume-%d", time.Now().UnixNano()), c.Name)
	if e != nil {
		t.Fatal(e)
	}
	op.WorkerID = "integration-storage-failure"
	s.DB.Model(op).Updates(map[string]any{"worker_id": op.WorkerID, "status": "preflight", "heartbeat_at": time.Now()})
	s.run(ctx, *op)
	s.DB.First(op, op.ID)
	var b model.Backup
	if e = s.DB.Where("operation_id = ?", op.ID).First(&b).Error; e != nil {
		t.Fatal(e)
	}
	if b.Status == "complete" || flag(object(b.Manifest)["remoteComplete"]) || op.Status == "succeeded" {
		t.Fatal("full repository falsely reported success")
	}
	if _, e = s.CheckRepository(ctx, user, repo.ID); e == nil {
		t.Fatal("full repository check falsely healthy")
	}
	if op.Status == "unknown" {
		runLiveOperation(t, ctx, s, c, user, "reconcile", Parameters{OperationID: op.ID})
	}
	client, e := s.redis(c, *root)
	if e != nil {
		t.Fatal(e)
	}
	defer client.Close()
	if e = client.Set(ctx, fmt.Sprintf("autoops-storage-proof-%d", op.ID), "preserved", 0).Err(); e != nil {
		t.Fatal(e)
	}
	s.event(c.ID, user, op.ID, "integration-report", "info", "4 MiB 独立 tmpfs 写满测试：远端备份失败且未标记成功，源数据库继续可写；测试挂载已安排独立定时清理")
	t.Logf("bounded full repository rejected upload: backup=%d task=%d; database remains writable", b.ID, op.ID)
}
