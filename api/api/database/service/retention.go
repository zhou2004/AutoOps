package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"dodevops-api/api/database/model"
)

func (s *Service) retentionDays(cluster uint) int {
	var days int
	s.DB.Model(&model.BackupPolicy{}).Where("cluster_id = ?", cluster).Select("COALESCE(MAX(retention_days),7)").Scan(&days)
	if days < 1 {
		days = 7
	}
	return days
}

func (s *Service) expirableBackup(cluster, id uint) (*model.Backup, error) {
	var b model.Backup
	if id == 0 || s.DB.Where("id = ? AND cluster_id = ? AND status = ?", id, cluster, "complete").First(&b).Error != nil {
		return nil, errors.New("只能清理已完成且过期的受管备份")
	}
	if b.CreatedAt.After(time.Now().AddDate(0, 0, -s.retentionDays(cluster))) {
		return nil, errors.New("备份仍在恢复保留窗口内")
	}
	var children int64
	s.DB.Model(&model.Backup{}).Where("parent_id = ? AND status <> ?", b.ID, "expired").Count(&children)
	if children > 0 {
		return nil, errors.New("仍有增量或日志归档引用此备份，必须保护整个恢复依赖链")
	}
	var newer int64
	s.DB.Model(&model.Backup{}).Where("cluster_id = ? AND id > ? AND status = ? AND verified_at IS NOT NULL", cluster, b.ID, "complete").Count(&newer)
	if newer == 0 {
		return nil, errors.New("必须保留一个更新且实际恢复验证通过的备份")
	}
	return &b, nil
}

const expireLocalBackup = `import hashlib,json,pathlib,shutil,sys
p=json.load(sys.stdin);root=pathlib.Path('/var/lib/autoops-database-backups')/str(p['cluster'])/str(p['operation'])/str(p['member'])
assert str(root/'backup.tar')==p['file']
if root.exists():
 assert not root.is_symlink() and root.resolve()==root
 metadata=json.load(open(root/'manifest.json'));assert metadata['sha256']==p['sha256']
 h=hashlib.sha256()
 with open(root/'backup.tar','rb') as f:
  for block in iter(lambda:f.read(1048576),b''):h.update(block)
 assert h.hexdigest()==p['sha256'],'backup changed since retention plan'
 shutil.rmtree(root)
print(json.dumps({'expired':True}))`

func (s *Service) expireBackup(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	b, e := s.expirableBackup(c.ID, p.BackupID)
	if e != nil {
		return e
	}
	var manifest struct {
		Members          []model.Member
		Artifacts        map[string]json.RawMessage
		RepositoryReport map[string]json.RawMessage
	}
	if json.Unmarshal(b.Manifest, &manifest) != nil || len(manifest.Members) == 0 {
		return errors.New("缺少受管备份清单")
	}
	return s.step(op, "核对恢复依赖链并清理指定过期备份", func(log func(string)) error {
		// Keep archive/repository deletion explicit and scoped to exact recorded
		// objects. Never run bucket lifecycle deletion or recursive prefix cleanup.
		if b.RepositoryID > 0 {
			if len(manifest.RepositoryReport) != len(manifest.Members) {
				return errors.New("独立仓库清单不完整，不能清理")
			}
			for _, m := range manifest.Members {
				key := fmt.Sprintf("member_%d", m.ID)
				if object(manifest.RepositoryReport[key])["verified"] != true || text(object(manifest.RepositoryReport[key])["sha256"]) != text(object(manifest.Artifacts[key])["sha256"]) {
					return errors.New("仓库副本与备份清单不一致，不能清理")
				}
			}
			var repo model.Repository
			if s.DB.First(&repo, b.RepositoryID).Error != nil {
				return errors.New("仓库记录不存在，不能清理")
			}
			credentials := map[string]string{}
			if repo.Kind == "s3" {
				credentials, e = s.RepositoryCredentials(repo.KeyID)
				if e != nil {
					return e
				}
			}
			for _, raw := range manifest.RepositoryReport {
				_, e = s.remoteMutation(ctx, repo.HostID, expireRepositoryObject, map[string]any{"repository": repo, "report": object(raw), "credentials": credentials, "cluster": c.ID, "backupOperation": b.OperationID})
				if e != nil {
					return e
				}
			}
		}
		for _, m := range manifest.Members {
			a := object(manifest.Artifacts[fmt.Sprintf("member_%d", m.ID)])
			if _, e = s.remoteMutation(ctx, m.HostID, expireLocalBackup, map[string]any{"cluster": c.ID, "operation": b.OperationID, "member": m.ID, "file": a["file"], "sha256": a["sha256"]}); e != nil {
				return e
			}
		}
		log("保留窗口、后继恢复验证与所有子依赖均通过；历史备份记录保留")
		return s.DB.Model(b).Update("status", "expired").Error
	})
}

const expireRepositoryObject = `import json,sys,pathlib,hashlib
p=json.load(sys.stdin);r=p['repository'];a=p['report'];assert a.get('verified') is True
if r['kind']=='s3':
 import boto3
 key=a['key'];assert key.startswith(str(p['cluster'])+'/'+str(p['backupOperation'])+'/') and '..' not in key
 s=boto3.client('s3',endpoint_url=r['endpoint'],region_name=r.get('region') or 'us-east-1',aws_access_key_id=p['credentials']['accessKey'],aws_secret_access_key=p['credentials']['secretKey'])
 args={'Bucket':r['bucket'],'Key':key}
 if a.get('versionId'):args['VersionId']=a['versionId']
 from botocore.exceptions import ClientError
 try:obj=s.get_object(**args)
 except ClientError as error:
  if error.response['Error']['Code'] not in ('NoSuchKey','NoSuchVersion'):raise
 else:
  h=hashlib.sha256()
  for chunk in iter(lambda:obj['Body'].read(1048576),b''):h.update(chunk)
  obj['Body'].close();assert h.hexdigest()==a['sha256']
  s.delete_object(**args)
else:
 f=pathlib.Path(a['path']);base=pathlib.Path(r['path']);assert f.resolve().is_relative_to(base.resolve()) and not f.is_symlink()
 if f.exists():
  h=hashlib.sha256()
  with f.open('rb') as stream:
   for chunk in iter(lambda:stream.read(1048576),b''):h.update(chunk)
  assert h.hexdigest()==a['sha256'];f.unlink()
print(json.dumps({'expired':True}))`
