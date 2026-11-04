package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	accounts "dodevops-api/api/configcenter/model"
	"dodevops-api/api/database/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Deleting a registration never removes bucket objects, files or credentials.
// Serialize with task submission and policy changes on the repository row.
func (s *Service) DeleteRepository(ctx context.Context, user, id uint, confirmed bool) error {
	if e := s.auditAdmin(ctx, user); e != nil {
		return e
	}
	if !confirmed {
		return errors.New("请确认删除备份仓库登记")
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var r model.Repository
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&r, id).Error; e != nil {
			return e
		}
		var count int64
		if e := tx.Model(&model.Backup{}).Where("repository_id = ? AND status IN ?", id, []string{"complete", "running"}).Count(&count).Error; e != nil {
			return e
		}
		if count > 0 {
			return errors.New("仓库仍有可用备份或正在生成的备份，请先清理相关备份")
		}
		if e := tx.Model(&model.BackupPolicy{}).Where("repository_id = ?", id).Count(&count).Error; e != nil {
			return e
		}
		if count > 0 {
			return errors.New("仓库仍被备份策略引用，请先将策略的保存位置改为其他仓库或本地暂存")
		}
		var operations []model.Operation
		if e := tx.Where("action = ? AND status NOT IN ?", "backup", []string{"succeeded", "failed", "cancelled"}).Find(&operations).Error; e != nil {
			return e
		}
		for _, op := range operations {
			var p Parameters
			if json.Unmarshal(op.Parameters, &p) != nil {
				return errors.New("存在无法核对仓库引用的备份任务")
			}
			if p.RepositoryID == id {
				return errors.New("仓库仍被排队、执行中或结果待核验的任务使用")
			}
		}
		return tx.Delete(&r).Error
	})
}

func (s *Service) RepositoryCredentials(id uint) (map[string]string, error) {
	var key accounts.KeyManage
	if id == 0 || s.DB.First(&key, id).Error != nil {
		return nil, errors.New("仓库凭据引用不存在")
	}
	a, b, e := key.DecryptKeys()
	if e != nil {
		return nil, errors.New("仓库凭据无法解密")
	}
	return map[string]string{"accessKey": a, "secretKey": b}, nil
}
func (s *Service) SaveRepository(ctx context.Context, user uint, r model.Repository) (*model.Repository, error) {
	if !s.IsAdmin(user) {
		return nil, ErrForbidden
	}
	if !namePattern.MatchString(r.Name) || r.GroupID == 0 || r.HostID == 0 {
		return nil, errors.New("请选择业务分组、执行主机和有效仓库名称")
	}
	if r.Kind != "local" && r.Kind != "s3" {
		return nil, errors.New("仓库类型必须为 local 或 s3")
	}
	if r.Kind == "local" {
		if !strings.HasPrefix(r.Path, "/var/lib/autoops-backup-repository/") || strings.Contains(r.Path, "..") || strings.ContainsAny(r.Path, "\n\r\x00") {
			return nil, errors.New("本地受管仓库目录须位于 /var/lib/autoops-backup-repository/ 下")
		}
	} else {
		u, e := url.Parse(r.Endpoint)
		if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || r.Bucket == "" || strings.ContainsAny(r.Bucket, "/\\\n") {
			return nil, errors.New("S3 地址或桶名无效")
		}
		if _, e = s.RepositoryCredentials(r.KeyID); e != nil {
			return nil, e
		}
	}
	r.ID = 0
	r.Status = "unchecked"
	r.CheckedAt = nil
	if e := s.DB.Create(&r).Error; e != nil {
		return nil, e
	}
	return &r, nil
}

const repositoryCheckScript = `import json,sys,os,tempfile
p=json.load(sys.stdin)
if p['kind']=='local':
 path=p['path'];assert os.path.isdir(path)
 fd,name=tempfile.mkstemp(prefix='.autoops-check-',dir=path)
 try: os.write(fd,b'autoops repository check');os.fsync(fd);os.close(fd)
 finally: os.unlink(name)
 v=os.statvfs(path);result={'healthy':True,'probe':'temporary-file-write-fsync-delete','totalBytes':v.f_blocks*v.f_frsize,'availableBytes':v.f_bavail*v.f_frsize,'freeInodes':v.f_favail}
else:
 import boto3
 s=boto3.client('s3',endpoint_url=p['endpoint'],region_name=p.get('region') or 'us-east-1',aws_access_key_id=p['credentials']['accessKey'],aws_secret_access_key=p['credentials']['secretKey'])
 s.head_bucket(Bucket=p['bucket']);result={'healthy':True,'probe':'HeadBucket','bucketAccessible':True}
print(json.dumps(result))`

func (s *Service) CheckRepository(ctx context.Context, user, id uint) (*model.Repository, error) {
	if !s.IsAdmin(user) {
		return nil, ErrForbidden
	}
	var r model.Repository
	if s.DB.First(&r, id).Error != nil {
		return nil, errors.New("仓库不存在")
	}
	p := object(JSON(r))
	if r.Kind == "s3" {
		cred, e := s.RepositoryCredentials(r.KeyID)
		if e != nil {
			return nil, e
		}
		p["credentials"] = cred
	}
	raw, e := s.remoteJSON(ctx, r.HostID, repositoryCheckScript, p)
	status := "unavailable"
	if e == nil && object(raw)["healthy"] == true {
		status = "healthy"
	}
	now := time.Now().UTC()
	message := "连接检查通过"
	if e != nil {
		message = e.Error()
	}
	if raw == nil {
		raw = JSON(map[string]any{})
	}
	if err := s.DB.Model(&r).Updates(map[string]any{"status": status, "checked_at": now, "check_report": raw, "check_message": message}).Error; err != nil {
		return nil, err
	}
	if e != nil {
		return nil, e
	}
	r.Status = status
	r.CheckedAt, r.CheckReport, r.CheckMessage = &now, raw, message
	return &r, nil
}
