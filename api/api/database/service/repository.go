package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	accounts "dodevops-api/api/configcenter/model"
	"dodevops-api/api/database/model"
)

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
 v=os.statvfs(path);result={'healthy':True,'availableBytes':v.f_bavail*v.f_frsize}
else:
 import boto3
 s=boto3.client('s3',endpoint_url=p['endpoint'],region_name=p.get('region') or 'us-east-1',aws_access_key_id=p['credentials']['accessKey'],aws_secret_access_key=p['credentials']['secretKey'])
 s.head_bucket(Bucket=p['bucket']);result={'healthy':True}
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
	s.DB.Model(&r).Updates(map[string]any{"status": status, "checked_at": time.Now().UTC()})
	if e != nil {
		return nil, e
	}
	r.Status = status
	return &r, nil
}
