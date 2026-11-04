package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/database/model"
	"dodevops-api/common/config"
	"fmt"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"strconv"
	"testing"
	"time"
)

// Explicitly opt into the authorized lab only. Creates its own key, backup and
// isolated restore, and cleans only its second, freshly-created backup.
func TestLiveMinIOBackupRestoreAndManualCleanup(t *testing.T) {
	if os.Getenv("AUTOOPS_DB_LIVE_MINIO") != "1" {
		t.Skip("requires authorized CMDB .51/.52/.53 lab and provisioned MinIO")
	}
	if e := config.LoadConfig("../../../config.yaml"); e != nil {
		t.Fatal(e)
	}
	cfg := config.Config.Db
	dsn := driver.NewConfig()
	dsn.User = cfg.Username
	dsn.Passwd = cfg.Password
	dsn.Net = "tcp"
	dsn.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	dsn.DBName = cfg.Db
	dsn.ParseTime = true
	dsn.Loc = time.Local
	db, e := gorm.Open(mysql.Open(dsn.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal("metadata unavailable")
	}
	raw, _ := db.DB()
	defer raw.Close()
	s := New(db)
	var user uint
	db.Table("sys_admin u").Select("u.id").Joins("JOIN sys_admin_role ar ON ar.admin_id = u.id JOIN sys_role r ON r.id = ar.role_id").Where("u.status = 1 AND r.role_key = ? AND r.status = 1", "admin").Order("u.id").Limit(1).Scan(&user)
	if !s.IsAdmin(user) {
		t.Fatal("admin identity unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	var repo model.Repository
	if e = db.Where("name = ? AND kind = ?", "minio-53-database-backups", "s3").First(&repo).Error; e != nil {
		t.Fatal(e)
	}
	if repo.Endpoint != "http://192.168.80.53:9000" || repo.Bucket != "autoops-database-backups" || repo.HostID != 531 {
		t.Fatal("unexpected repository")
	}
	var host cmdb.CmdbHost
	if e = db.First(&host, 530).Error; e != nil || host.SSHIP != "192.168.80.51" {
		t.Fatal("unexpected restore host")
	}
	var cluster model.Cluster
	if e = db.Where("name = ? AND topology = ?", "db-lab-redis-sentinel", "redis-sentinel").First(&cluster).Error; e != nil {
		t.Fatal(e)
	}
	if requested := os.Getenv("AUTOOPS_DB_LIVE_MINIO_RECONCILE"); requested != "" {
		id, err := strconv.ParseUint(requested, 10, 32)
		if err != nil {
			t.Fatal(err)
		}
		var previous model.Operation
		if err = db.Where("id = ? AND cluster_id = ? AND action IN ? AND status = ?", uint(id), cluster.ID, []string{"backup", "verify-backup"}, "unknown").First(&previous).Error; err != nil {
			t.Fatal("not an interrupted lab backup", err)
		}
		params := object(previous.Parameters)
		previousRepository := uint(number(params["repositoryId"]))
		if previous.Action == "verify-backup" {
			var backup model.Backup
			if e := db.First(&backup, uint(number(params["backupId"]))).Error; e != nil {
				t.Fatal(e)
			}
			previousRepository = backup.RepositoryID
		}
		if previousRepository != repo.ID {
			t.Fatal("task uses another repository")
		}
		c, err := s.Cluster(cluster.ID)
		if err != nil {
			t.Fatal(err)
		}
		runLiveOperation(t, ctx, s, c, user, "reconcile", Parameters{OperationID: uint(id)})
	}
	c, e := s.Refresh(ctx, cluster.ID)
	if e != nil {
		t.Fatal(e)
	}
	root, e := primary(c)
	if e != nil {
		t.Fatal(e)
	}
	if root.Port != 17379 {
		t.Fatal("non-lab source port")
	}
	client, e := s.redis(c, *root)
	if e != nil {
		t.Fatal(e)
	}
	defer client.Close()
	key := fmt.Sprintf("autoops-minio-proof-%d", time.Now().UnixNano())
	value := "S3 isolated restore proof"
	if e = client.Set(ctx, key, value, 0).Err(); e != nil {
		t.Fatal(e)
	}
	defer client.Del(context.Background(), key)
	op := runLiveOperation(t, ctx, s, c, user, "backup", Parameters{Method: "rdb", RepositoryID: repo.ID})
	var b model.Backup
	if e = db.Where("operation_id = ?", op.ID).First(&b).Error; e != nil {
		t.Fatal(e)
	}
	if object(b.Manifest)["remoteComplete"] != true {
		t.Fatal("S3 upload incomplete")
	}
	// Restore always retrieves the remote replica for a repository-backed backup.
	restore := runLiveOperation(t, ctx, s, c, user, "verify-backup", Parameters{BackupID: b.ID, HostID: 530, Port: 19395})
	path := fmt.Sprintf("/var/lib/autoops-database-restores/%d/%d/%d/data", c.ID, restore.ID, root.ID)
	proof, e := s.remoteJSON(ctx, 530, `import json,sys,subprocess,time,redis
p=json.load(sys.stdin);unit='autoops-minio-proof-'+str(p['operation'])
subprocess.run(['systemd-run','--unit',unit,'--property=User=autoopsdb','--property=RuntimeMaxSec=120','redis-server','--bind','127.0.0.1','--port','19395','--dir',p['path'],'--appendonly','no','--save',''],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 client=redis.Redis(port=19395,socket_timeout=2)
 for i in range(20):
  try:
   if client.ping():break
  except Exception:time.sleep(1)
 assert client.get(p['key'])==p['value'].encode(),'restored fixture mismatch'
 print(json.dumps({'verified':True,'keyMatched':True}))
finally:subprocess.run(['systemctl','stop',unit],check=True)`, map[string]any{"operation": restore.ID, "path": path, "key": key, "value": value})
	if e != nil {
		t.Fatal("restored key verification", e)
	}
	db.First(&b, b.ID)
	if b.VerifiedAt == nil {
		t.Fatal("restore verification missing")
	}
	freshOp := runLiveOperation(t, ctx, s, c, user, "backup", Parameters{Method: "rdb", RepositoryID: repo.ID})
	var fresh model.Backup
	db.Where("operation_id = ?", freshOp.ID).First(&fresh)
	cleanup := runLiveOperation(t, ctx, s, c, user, "expire-backup", Parameters{BackupID: fresh.ID})
	db.First(&fresh, fresh.ID)
	if fresh.Status != "expired" {
		t.Fatal("fresh backup cleanup failed")
	}
	creds, e := s.RepositoryCredentials(repo.KeyID)
	if e != nil {
		t.Fatal(e)
	}
	report, e := s.remoteJSON(ctx, repo.HostID, `import json,sys,boto3,pathlib
p=json.load(sys.stdin);s=boto3.client('s3',endpoint_url=p['endpoint'],aws_access_key_id=p['credentials']['accessKey'],aws_secret_access_key=p['credentials']['secretKey'])
items=s.list_objects_v2(Bucket=p['bucket'],Prefix=p['prefix']).get('Contents',[])
assert not items,'cleaned backup objects remain'
print(json.dumps({'cleanedObjectsAbsent':True}))`, map[string]any{"endpoint": repo.Endpoint, "bucket": repo.Bucket, "credentials": creds, "prefix": fmt.Sprintf("%d/%d/", c.ID, freshOp.ID)})
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("MinIO repository=%d verified backup=%d restore=%d proof=%s; fresh backup=%d cleanup=%d report=%s", repo.ID, b.ID, restore.ID, proof, fresh.ID, cleanup.ID, report)
	s.event(c.ID, user, restore.ID, "integration-report", "info", fmt.Sprintf("MinIO .53 仓库备份 #%d 已从 S3 取回到 .51 隔离恢复，实际键值核验通过；新建备份 #%d 未过期清理通过", b.ID, fresh.ID))
}
