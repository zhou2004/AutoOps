package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"dodevops-api/api/database/model"
	"gopkg.in/yaml.v3"
)

type operationContextKey struct{}

func (s *Service) executionOwned(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	op, ok := ctx.Value(operationContextKey{}).(model.Operation)
	if !ok {
		return nil
	}
	var count int64
	err := s.DB.Model(&model.Operation{}).Where("id = ? AND worker_id = ? AND status IN ?", op.ID, op.WorkerID, []string{"preflight", "running", "verifying"}).Count(&count).Error
	if err != nil || count != 1 {
		return errors.New("任务已失去执行权，禁止继续发出命令")
	}
	return nil
}

// A remote shared flock spans the entire Ansible module. A newer operation
// obtains the exclusive lock before advancing the epoch; an old worker cannot
// launch another module afterward. Expired API leases alone never release it.
const remotePythonGuard = `#!/usr/bin/python3
import fcntl,json,os,sys
cluster=os.environ.get('AUTOOPS_DB_CLUSTER_ID','');epoch=os.environ.get('AUTOOPS_DB_OPERATION_ID','')
if not cluster.isdecimal() or not epoch.isdecimal():sys.exit('database execution identity missing')
root='/var/lib/autoops-db-execution/'
fd=os.open(root+cluster+'.lock',os.O_RDONLY)
fcntl.flock(fd,fcntl.LOCK_SH)
if json.load(open(root+cluster+'.json'))['epoch']!=int(epoch):sys.exit('stale database operation blocked')
os.set_inheritable(fd,True)
os.execv('/usr/bin/python3',['/usr/bin/python3']+sys.argv[1:])
`

const prepareGuardScript = `import fcntl,json,os,pathlib,sys,time
p=json.load(sys.stdin);assert int(p['epoch'])>0 and int(p['cluster'])>0
root=pathlib.Path('/var/lib/autoops-db-execution');root.mkdir(mode=0o755,exist_ok=True);root.chmod(0o755)
assert root.stat().st_uid==0
lock=open(root/(str(p['cluster'])+'.lock'),'a');os.chmod(lock.name,0o644);deadline=time.monotonic()+8
while True:
 try: fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB);break
 except BlockingIOError:
  if time.monotonic()>deadline:raise RuntimeError('previous remote module is still executing')
  time.sleep(.2)
state=root/(str(p['cluster'])+'.json')
if state.exists():assert json.load(open(state))['epoch']<=p['epoch'],'stale operation cannot take execution rights'
tmp=state.with_suffix('.new');tmp.write_text(json.dumps({'epoch':p['epoch']}));tmp.chmod(0o644);os.replace(tmp,state)
program=pathlib.Path('/usr/local/sbin/autoops-db-python');temporary=program.with_name('.autoops-python-'+str(p['cluster'])+'-'+str(p['epoch']));temporary.write_text(p['program']);temporary.chmod(0o755);os.replace(temporary,program)
print(json.dumps({'epoch':p['epoch'],'previousModulesFinished':True}))`

func (s *Service) guardedAutomation(ctx context.Context, c *model.Cluster, hosts map[uint]bool, root, dir string) (string, bool, error) {
	op, ok := ctx.Value(operationContextKey{}).(model.Operation)
	if !ok {
		return root, false, nil
	} // Read-only probes and explicit lab fixtures.
	var live model.Operation
	if s.DB.Where("id = ? AND worker_id = ? AND status IN ?", op.ID, op.WorkerID, []string{"running", "verifying"}).First(&live).Error != nil || live.ClusterID != c.ID {
		return "", false, errors.New("执行权已变化，不能再发起远端变更")
	}
	for _, host := range sortedKeys(hosts) {
		if _, e := s.remoteJSON(ctx, host, prepareGuardScript, map[string]any{"cluster": c.ID, "epoch": op.ID, "program": remotePythonGuard}); e != nil {
			return "", false, errors.New("远端执行权未取得；旧命令可能仍在运行，保留集群锁")
		}
	}
	copyRoot := filepath.Join(dir, "automation")
	if e := os.CopyFS(copyRoot, os.DirFS(root)); e != nil {
		return "", false, e
	}
	defaults, e := os.ReadFile(filepath.Join(root, "..", "database", "defaults.yml"))
	if e != nil {
		return "", false, e
	}
	if e = os.Mkdir(filepath.Join(dir, "database"), 0700); e != nil {
		return "", false, e
	}
	if e = os.WriteFile(filepath.Join(dir, "database", "defaults.yml"), defaults, 0600); e != nil {
		return "", false, e
	}
	files, e := filepath.Glob(filepath.Join(copyRoot, "*.yml"))
	if e != nil {
		return "", false, e
	}
	for _, path := range files {
		data, e := os.ReadFile(path)
		if e != nil {
			return "", false, e
		}
		var entries []map[string]any
		if e = yaml.Unmarshal(data, &entries); e != nil {
			return "", false, e
		}
		changed := false
		for _, play := range entries {
			if play["hosts"] == nil {
				continue
			}
			environment, ok := play["environment"].(map[string]any)
			if !ok {
				environment = map[string]any{}
			}
			environment["AUTOOPS_DB_CLUSTER_ID"] = c.ID
			environment["AUTOOPS_DB_OPERATION_ID"] = op.ID
			play["environment"] = environment
			changed = true
		}
		if changed {
			data, e = yaml.Marshal(entries)
			if e != nil {
				return "", false, e
			}
			if e = os.WriteFile(path, data, 0600); e != nil {
				return "", false, e
			}
		}
	}
	return copyRoot, true, nil
}
