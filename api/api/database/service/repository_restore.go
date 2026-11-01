package service

import (
	"context"
	"dodevops-api/api/database/model"
	"errors"
	"fmt"
)

// Download by immutable backup identity and checksum, independent of the old
// database host. Only transport HostID changes; the manifest's instance identity
// and restore credentials remain those captured with the backup.
func (s *Service) retrieveRepository(ctx context.Context, c *model.Cluster, b *model.Backup, op model.Operation, members []model.Member) ([]model.Member, map[string]any, error) {
	manifest := object(b.Manifest)
	reports := object(JSON(manifest["repositoryReport"]))
	artifacts := object(JSON(manifest["artifacts"]))
	var repo model.Repository
	if s.DB.First(&repo, b.RepositoryID).Error != nil || repo.HostID == 0 {
		return nil, nil, errors.New("备份仓库记录或执行主机不存在")
	}
	credentials := map[string]string{}
	if repo.Kind == "s3" {
		var e error
		credentials, e = s.RepositoryCredentials(repo.KeyID)
		if e != nil {
			return nil, nil, e
		}
	}
	if len(reports) != len(members) {
		return nil, nil, errors.New("独立仓库备份清单不完整")
	}
	for i, m := range members {
		key := fmt.Sprintf("member_%d", m.ID)
		a := object(JSON(artifacts[key]))
		r := object(JSON(reports[key]))
		if r["verified"] != true || text(r["sha256"]) == "" || r["sha256"] != a["sha256"] {
			return nil, nil, errors.New("独立副本校验记录与备份不一致")
		}
		raw, e := s.remoteMutation(ctx, repo.HostID, repositoryRetrieveScript, map[string]any{"repository": repo, "report": r, "credentials": credentials, "cluster": c.ID, "operation": op.ID, "backupOperation": b.OperationID, "member": m.ID})
		if e != nil {
			return nil, nil, e
		}
		a["file"] = text(object(raw)["file"])
		artifacts[key] = a
		members[i].HostID = repo.HostID
	}
	manifest["artifacts"] = artifacts
	return members, manifest, nil
}

const repositoryRetrieveScript = `import json,sys,pathlib,hashlib,os
p=json.load(sys.stdin);r=p['repository'];a=p['report'];os.umask(0o077)
root=pathlib.Path('/var/lib/autoops-database-backups')/str(p['cluster'])/str(p['operation'])/str(p['member'])
root.mkdir(parents=True,exist_ok=True);assert root.resolve()==root
path=root/'backup.tar';tmp=root/'backup.partial';assert not path.exists() and not tmp.exists()
if r['kind']=='s3':
 import boto3
 key=a['key'];assert key==str(p['cluster'])+'/'+str(p['backupOperation'])+'/member_'+str(p['member'])+'.tar'
 client=boto3.client('s3',endpoint_url=r['endpoint'],region_name=r.get('region') or 'us-east-1',aws_access_key_id=p['credentials']['accessKey'],aws_secret_access_key=p['credentials']['secretKey'])
 args={'Bucket':r['bucket'],'Key':key}
 if a.get('versionId'):args['VersionId']=a['versionId']
 source=client.get_object(**args)['Body']
else:
 source_path=pathlib.Path(a['path']);base=pathlib.Path(r['path'])
 expected=base/str(p['cluster'])/str(p['backupOperation'])/('member_'+str(p['member'])+'.tar')
 assert source_path==expected and not source_path.is_symlink() and source_path.resolve().is_relative_to(base.resolve())
 source=source_path.open('rb')
try:
 h=hashlib.sha256()
 with tmp.open('xb') as out:
  for block in iter(lambda:source.read(1048576),b''):h.update(block);out.write(block)
  out.flush();os.fsync(out.fileno())
 assert h.hexdigest()==a['sha256'],'repository backup checksum mismatch'
 os.rename(tmp,path)
finally:source.close();tmp.unlink(missing_ok=True)
print(json.dumps({'file':str(path),'verified':True}))`
