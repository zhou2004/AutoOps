package service

import (
	"context"
	"dodevops-api/api/database/model"
	"fmt"
	"testing"
	"time"
)

func exerciseRepository(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	root, e := primary(c)
	if e != nil {
		t.Fatal(e)
	}
	host := uint(530)
	if root.HostID == host {
		host = 528
	}
	path := "/var/lib/autoops-backup-repository/database-lab"
	_, e = s.remoteJSON(ctx, host, `import pathlib,json,sys
p=json.load(sys.stdin);path=pathlib.Path(p['path']);path.mkdir(parents=True,exist_ok=True,mode=0o700);assert path.resolve()==path
print(json.dumps({'ready':True}))`, map[string]any{"path": path})
	if e != nil {
		t.Fatal(e)
	}
	repo, e := s.SaveRepository(ctx, user, model.Repository{Name: fmt.Sprintf("lab-repository-%d", time.Now().Unix()), GroupID: c.GroupID, HostID: host, Kind: "local", Path: path})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CheckRepository(ctx, user, repo.ID); e != nil {
		t.Fatal(e)
	}
	client, e := s.redis(c, *root)
	if e != nil {
		t.Fatal(e)
	}
	defer client.Close()
	key := fmt.Sprintf("autoops-lab-repository-%d", time.Now().UnixNano())
	value := "isolated restore proof"
	if e = client.Set(ctx, key, value, 0).Err(); e != nil {
		t.Fatal(e)
	}
	backupOp := runLiveOperation(t, ctx, s, c, user, "backup", Parameters{Method: "rdb", RepositoryID: repo.ID})
	var b model.Backup
	s.DB.Where("operation_id = ?", backupOp.ID).First(&b)
	a := object(JSON(object(JSON(object(b.Manifest)["artifacts"]))[fmt.Sprintf("member_%d", root.ID)]))
	offline := `import json,sys,pathlib
p=json.load(sys.stdin);file=pathlib.Path(p['file']);assert str(file).startswith('/var/lib/autoops-database-backups/') and file.name=='backup.tar'
saved=file.with_name('backup.offline-test')
if p['restore']:saved.rename(file)
else:assert file.exists() and not saved.exists();file.rename(saved)
print(json.dumps({'done':True}))`
	if _, e = s.remoteJSON(ctx, root.HostID, offline, map[string]any{"file": a["file"], "restore": false}); e != nil {
		t.Fatal(e)
	}
	defer func() {
		_, err := s.remoteJSON(context.Background(), root.HostID, offline, map[string]any{"file": a["file"], "restore": true})
		if err != nil {
			t.Error("restore local fixture file", err)
		}
	}()
	restore := runLiveOperation(t, ctx, s, c, user, "verify-backup", Parameters{BackupID: b.ID, HostID: 531, Port: 19389})
	path = fmt.Sprintf("/var/lib/autoops-database-restores/%d/%d/%d/data", c.ID, restore.ID, root.ID)
	raw, e := s.remoteJSON(ctx, 531, `import json,sys,subprocess,time,redis
p=json.load(sys.stdin);unit='autoops-repository-proof-'+str(p['operation'])
subprocess.run(['systemd-run','--unit',unit,'--property=User=autoopsdb','--property=RuntimeMaxSec=120','redis-server','--bind','127.0.0.1','--port','19389','--dir',p['path'],'--appendonly','no','--save',''],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 client=redis.Redis(port=19389,socket_timeout=2)
 for i in range(20):
  try:
   if client.ping():break
  except Exception:time.sleep(1)
 assert client.get(p['key'])==p['value'].encode(),'restored fixture mismatch'
 print(json.dumps({'verified':True,'sourceArtifactUnavailable':True}))
finally:subprocess.run(['systemctl','stop',unit],check=True)`, map[string]any{"path": path, "operation": restore.ID, "key": key, "value": value})
	if e != nil {
		t.Fatal("repository restored data mismatch", e)
	}
	t.Logf("Independent local repository #%d backup #%d restored while source artifact unavailable: %s", repo.ID, b.ID, raw)
	s.event(c.ID, user, restore.ID, "integration-report", "info", fmt.Sprintf("独立受管仓库 #%d 的备份 #%d 在原暂存文件不可用时完成隔离恢复，实际键值核验通过", repo.ID, b.ID))
}
