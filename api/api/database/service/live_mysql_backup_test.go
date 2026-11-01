package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"dodevops-api/api/database/model"
)

func exerciseMySQLArchive(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	root, e := primary(c)
	if e != nil {
		t.Fatal(e)
	}
	db, e := s.sql(ctx, c, *root, "autoops_database_lab")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS autoops_pitr_probe(id varchar(191) PRIMARY KEY)"); e != nil {
		t.Fatal(e)
	}
	prefix := fmt.Sprintf("mysql-pitr-%d-", time.Now().UnixNano())
	write := func(suffix string) {
		t.Helper()
		if _, e := db.ExecContext(ctx, "INSERT INTO autoops_pitr_probe VALUES (?)", prefix+suffix); e != nil {
			t.Fatal(e)
		}
	}
	write("base")
	full := runLiveOperation(t, ctx, s, c, user, "backup", Parameters{Method: "physical"})
	var base model.Backup
	s.DB.Where("operation_id = ?", full.ID).First(&base)
	runLiveOperation(t, ctx, s, c, user, "verify-backup", Parameters{BackupID: base.ID, HostID: 530, Port: 14308})
	write("incremental")
	incOp := runLiveOperation(t, ctx, s, c, user, "backup", Parameters{Method: "incremental", ParentBackupID: base.ID})
	var inc model.Backup
	s.DB.Where("operation_id = ?", incOp.ID).First(&inc)
	runLiveOperation(t, ctx, s, c, user, "verify-backup", Parameters{BackupID: inc.ID, HostID: 530, Port: 14309})
	write("before")
	var epoch int64
	if e = db.QueryRowContext(ctx, "SELECT UNIX_TIMESTAMP()").Scan(&epoch); e != nil {
		t.Fatal(e)
	}
	target := time.Unix(epoch+2, 0).UTC()
	if e = waitUntil(ctx, 10*time.Second, func() bool {
		var current int64
		err := db.QueryRowContext(ctx, "SELECT UNIX_TIMESTAMP()").Scan(&current)
		return err == nil && current >= target.Unix()+1
	}); e != nil {
		t.Fatal(e)
	}
	write("after")
	archiveOp := runLiveOperation(t, ctx, s, c, user, "backup", Parameters{Method: "binlog-archive", ParentBackupID: inc.ID})
	var archive model.Backup
	s.DB.Where("operation_id = ?", archiveOp.ID).First(&archive)
	restore := runLiveOperation(t, ctx, s, c, user, "restore", Parameters{BackupID: archive.ID, HostID: 530, Port: 14310, TargetTime: target.Format(time.RFC3339)})
	username, password, e := s.memberAccount(c, *root)
	if e != nil {
		t.Fatal(e)
	}
	dir := fmt.Sprintf("/var/lib/autoops-database-restores/%d/%d/%d", c.ID, restore.ID, root.ID)
	raw, e := s.remoteJSON(ctx, 530, `import json,sys,subprocess,time,pymysql,glob
p=json.load(sys.stdin);unit='autoops-mysql-pitr-proof-'+str(p['operation']);binary=glob.glob('/opt/autoops/mysql/*/bin/mysqld')[0]
subprocess.run(['systemd-run','--unit',unit,'--property=User=autoopsdb','--property=RuntimeMaxSec=120',binary,'--defaults-file='+p['dir']+'/my.cnf'],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 for attempt in range(30):
  try:conn=pymysql.connect(host='127.0.0.1',port=14310,user=p['username'],password=p['password'],database='autoops_database_lab');break
  except Exception:time.sleep(1)
 else:raise RuntimeError('isolated restore did not restart')
 with conn.cursor() as cursor:
  out={}
  for name in ('base','incremental','before','after'):
   cursor.execute('SELECT COUNT(*) FROM autoops_pitr_probe WHERE id=%s',(p['prefix']+name,));out[name]=cursor.fetchone()[0]
  assert out=={'base':1,'incremental':1,'before':1,'after':0},'PITR boundary mismatch'
 print(json.dumps(out))
finally:subprocess.run(['systemctl','stop',unit],check=True)`, map[string]any{"operation": restore.ID, "dir": dir, "prefix": prefix, "username": username, "password": password})
	if e != nil {
		t.Fatal("MySQL PITR data verification failed", e)
	}
	t.Logf("MySQL full %d, incremental %d, binlog %d and isolated PITR %s verified: %s", base.ID, inc.ID, archive.ID, target.Format(time.RFC3339), raw)
	s.event(c.ID, user, restore.ID, "integration-report", "info", fmt.Sprintf("MySQL XtraBackup 全量 #%d、增量 #%d 与 binlog #%d 已隔离恢复，目标 %s 前后业务记录核验通过", base.ID, inc.ID, archive.ID, target.Format(time.RFC3339)))
}
