"""Version-checked, isolated database backups. Arguments never contain passwords."""
import configparser
import datetime
import glob
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tarfile
from pathlib import Path

def run(args, **kwargs):
    if not kwargs.get('capture_output') and 'stderr' not in kwargs:
        kwargs['stderr'] = subprocess.PIPE
    return subprocess.run(args, check=True, timeout=3600, **kwargs)

def sha(path):
    h = hashlib.sha256()
    with open(path, "rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()

XTRABACKUP = '/opt/autoops/xtrabackup/percona-xtrabackup-8.0.26-18-Linux-x86_64.glibc2.17-minimal/bin/xtrabackup'

def parent_payload(artifact, root, identity):
    path=Path(artifact.get('file',''))
    if not re.fullmatch(r'/var/lib/autoops-database-backups/\d+/\d+/\d+/backup.tar',str(path)) or sha(path)!=artifact.get('sha256'):
        raise ValueError('physical parent path or checksum invalid')
    if artifact.get('nativeId')!=identity:
        raise ValueError('physical parent belongs to a different instance')
    target=root/'parent';target.mkdir()
    with tarfile.open(path) as tar:
        seen=set()
        for item in tar.getmembers():
            dest=target/item.name
            if item.name in seen or not (item.name=='payload' or item.name.startswith('payload/')) or not dest.resolve().is_relative_to(target.resolve()):
                raise ValueError('invalid parent entry')
            seen.add(item.name)
            if item.isdir():dest.mkdir(parents=True,exist_ok=True)
            elif item.isfile():
                dest.parent.mkdir(parents=True,exist_ok=True)
                with tar.extractfile(item) as src,open(dest,'xb') as out:shutil.copyfileobj(src,out)
            else:raise ValueError('parent links and special files are forbidden')
    metadata=json.load(open(target/'payload/metadata.json'))
    for item in metadata['files']:
        if sha(target/'payload'/item['name'])!=item['sha256']:raise ValueError('physical parent file corrupt')
    return target/'payload',metadata

def checkpoint(path):
    out={}
    for line in (path/'xtrabackup_checkpoints').read_text().splitlines():
        if '=' in line:
            k,v=line.split('=',1);out[k.strip()]=v.strip()
    return out

def main(spec_path):
    os.umask(0o077)
    spec = json.load(open(spec_path, encoding="utf-8"))
    root = Path(spec_path).parent
    data = root / "payload"
    data.mkdir(exist_ok=False)
    m, engine, method = spec["member"], spec["engine"], spec["method"]
    manifest = dict(engine=engine, method=method, memberId=m["id"], nativeId=m["nativeId"], version=m["version"], complete=False)
    env = os.environ.copy()
    if engine == "redis":
        if method != "rdb":
            raise ValueError("Redis supports RDB backup")
        env["REDISCLI_AUTH"] = spec["password"]
        run(["redis-cli", "-h", m["address"], "-p", str(m["port"]), "--user", spec["username"], "--rdb", str(data / "dump.rdb")], env=env, stdout=subprocess.DEVNULL)
        result = run(["redis-check-rdb", str(data / "dump.rdb")], capture_output=True, text=True)
        match = re.search(r"(\d+) keys read", result.stdout)
        if not match:
            raise ValueError("RDB validator did not report a complete key count")
        manifest["keys"] = int(match.group(1))
        expires = re.search(r"(\d+) expires", result.stdout)
        if not expires:
            raise ValueError("RDB validator did not report expiry counts")
        manifest["expiringKeys"] = int(expires.group(1))
    elif engine == "mysql":
        import pymysql
        with pymysql.connect(host=m["address"], port=m["port"], user=spec["username"], password=spec["password"], connect_timeout=5) as conn:
            with conn.cursor() as cursor:
                cursor.execute("SELECT @@version, @@server_uuid, @@gtid_executed")
                version, identity, gtid = cursor.fetchone()
                manifest.update(version=version, nativeId=identity, gtid=gtid)
                cursor.execute("SHOW DATABASES")
                databases = [r[0] for r in cursor.fetchall() if r[0] not in ("mysql", "sys", "information_schema", "performance_schema")]
        manifest["databases"] = databases
        option = root / "client.cnf"
        # Quote and escape option-file values; never put credentials on argv.
        quote = lambda value: '"' + str(value).replace('\\','\\\\').replace('"','\\"').replace('\n','\\n') + '"'
        option.write_text("[client]\n" + "\n".join(k+"="+quote(v) for k,v in dict(user=spec["username"],password=spec["password"],host=m["address"],port=m["port"]).items()), encoding="utf-8")
        try:
            if method == "logical":
                binaries = glob.glob("/opt/autoops/mysql/*/bin/mysqldump")
                binary = binaries[0] if binaries else shutil.which("mysqldump")
                if not binary:
                    raise RuntimeError("matching mysqldump is not installed")
                if databases:
                    with open(data / "dump.sql", "wb") as out:
                        run([binary, "--defaults-file="+str(option), "--single-transaction", "--routines", "--events", "--triggers", "--hex-blob", "--set-gtid-purged=OFF", "--order-by-primary", "--databases"]+databases, stdout=out)
                else:
                    (data / "dump.sql").write_text("-- No application databases\n")
                manifest["databases"] = databases
                manifest["scope"] = "application databases; server accounts excluded"
            elif method in ('physical','incremental','binlog-archive'):
                binary = XTRABACKUP
                tool = run([binary, "--no-defaults", "--version"], capture_output=True, text=True)
                if not re.match(r'^8\.0\.(24|25|26)(-|$)',version) or 'version 8.0.26-18 ' not in tool.stdout+tool.stderr:
                    raise RuntimeError("XtraBackup version does not match MySQL")
                incrementals=[]
                args=[binary,'--defaults-file='+str(option),'--backup','--datadir='+m['dataDir']]
                if method!='physical':
                    parent,previous=parent_payload(spec['parentArtifact'],root,identity)
                    shutil.copytree(parent/'physical',data/'physical')
                    incrementals=previous.get('incrementals',[])
                    for name in incrementals:
                        if not re.fullmatch(r'incremental-\d+',name):raise ValueError('invalid incremental path')
                        shutil.copytree(parent/name,data/name)
                    manifest.update({k:previous[k] for k in ('binlogFile','binlogPosition','backupStop')})
                if method=='physical':
                    dest=data/'physical'
                elif method=='incremental':
                    base=data/(incrementals[-1] if incrementals else 'physical')
                    name='incremental-'+str(len(incrementals)+1);dest=data/name
                    args+=['--incremental-basedir='+str(base)]
                if method!='binlog-archive':
                    run(args+['--target-dir='+str(dest)],stdout=subprocess.DEVNULL)
                    point=checkpoint(dest)
                    if method=='incremental':
                        if point['from_lsn']!=checkpoint(base)['to_lsn']:raise ValueError('incremental LSN chain is discontinuous')
                        incrementals.append(name)
                    coords=(dest/'xtrabackup_binlog_info').read_text().strip().split()
                    if len(coords)<2:raise ValueError('physical backup has no binlog coordinate')
                    manifest.update(binlogFile=coords[0],binlogPosition=int(coords[1]),backupStop=datetime.datetime.now(datetime.timezone.utc).timestamp(),checkpoint=point,gtid=coords[2] if len(coords)>2 else '')
                else:
                    with pymysql.connect(host=m['address'],port=m['port'],user=spec['username'],password=spec['password'],connect_timeout=5) as conn:
                        with conn.cursor() as cursor:
                            cursor.execute('SELECT @@global.log_bin_basename');base=Path(cursor.fetchone()[0])
                            through=datetime.datetime.now(datetime.timezone.utc).isoformat()
                            cursor.execute('FLUSH BINARY LOGS')
                            cursor.execute('SHOW BINARY LOGS');logs=[r[0] for r in cursor.fetchall()]
                    first=manifest['binlogFile']
                    if first not in logs:raise ValueError('required binlog was purged; archive chain is broken')
                    names=logs[logs.index(first):-1]
                    if not names:raise ValueError('no closed binlog available')
                    numbers=[int(n.rsplit('.',1)[1]) for n in names]
                    if numbers!=list(range(numbers[0],numbers[-1]+1)):raise ValueError('binlog sequence gap')
                    logs_dir=data/'binlogs';logs_dir.mkdir()
                    mysqlbinlog=str(Path(glob.glob('/opt/autoops/mysql/*/bin/mysqlbinlog')[0]))
                    for name in names:
                        if Path(name).name!=name or not name.startswith(base.name+'.'):raise ValueError('invalid native binlog name')
                        shutil.copy2(base.parent/name,logs_dir/name)
                        run([mysqlbinlog,'--no-defaults','--verify-binlog-checksum',str(logs_dir/name)],stdout=subprocess.DEVNULL)
                    manifest.update(binlogs=names,archiveThrough=through,archiveCheck=True,archiveScope='closed-binlog snapshot; schedule exports within the retention window')
                manifest.update(incrementals=incrementals,xtrabackupVersion='8.0.26-18',prepared=False)
            else:
                raise ValueError("incremental backup requires an explicitly validated physical parent")
        finally:
            option.unlink(missing_ok=True)
    elif engine == "postgresql":
        env.update(PGPASSWORD=spec["password"], PGHOST=m["address"], PGPORT=str(m["port"]), PGUSER=spec["username"], PGCONNECT_TIMEOUT="5")
        if method == "logical":
            query = "SELECT datname FROM pg_database WHERE NOT datistemplate ORDER BY datname"
            databases = run(["psql","-d","postgres","-At","-c",query],env=env,capture_output=True,text=True).stdout.splitlines()
            manifest["databases"] = databases
            manifest["scope"] = "all non-template databases; ownership restored to management role"
            for index,name in enumerate(databases):
                run(["pg_dump","-Fc","--no-owner","--no-privileges","-d",name,"-f",str(data / f"database-{index}.dump")],env=env,stdout=subprocess.DEVNULL)
        elif method in ("physical","incremental","differential","wal-archive"):
            conf = Path(m["dataDir"]).parent / "pgbackrest.conf"
            if not conf.exists():
                raise RuntimeError("pgBackRest repository and WAL archive must be configured first")
            options = configparser.ConfigParser();options.read(conf)
            repo = Path(options["global"]["repo1-path"])
            if not repo.resolve().is_relative_to(Path(m["dataDir"]).parent.resolve()):
                raise RuntimeError("archive export must use this instance's owned repository")
            native = ["runuser","-u","autoopsdb","--","pgbackrest","--config="+str(conf),"--stanza=autoops"]
            before = json.loads(run(native+["--output=json","info"],capture_output=True,text=True).stdout)
            previous = before[0].get("backup",[]) if before else []
            if method != "physical" and not previous:
                raise RuntimeError("a complete physical base backup is required in this repository")
            if method in ("incremental","differential"):
                parent = spec.get("parentLabel")
                if not parent or parent != previous[-1]["label"]:
                    raise RuntimeError("explicit parent does not match native repository history")
            if method != "wal-archive":
                native_type = {"physical":"full","incremental":"incr","differential":"diff"}[method]
                run(native+["--type="+native_type,"backup"],stdout=subprocess.DEVNULL)
            # check forces a WAL switch and confirms that its archive reached the repository.
            through = datetime.datetime.now(datetime.timezone.utc).isoformat()
            run(native+["check"],stdout=subprocess.DEVNULL)
            info = json.loads(run(native+["--output=json","info"],capture_output=True,text=True).stdout)
            latest = info[0]["backup"][-1]
            manifest.update(pgBackRest=info,backupLabel=latest["label"],archiveThrough=through,
                            backupStop=latest["timestamp"]["stop"],repositoryScope="node-local",archiveCheck=True)
            query="SELECT datname FROM pg_database WHERE NOT datistemplate ORDER BY datname"
            manifest["databases"]=run(["psql","-d","postgres","-At","-c",query],env=env,capture_output=True,text=True).stdout.splitlines()
            shutil.copytree(repo,data/"repository",ignore=lambda directory,names:[n for n in names if (Path(directory)/n).is_symlink() or n.endswith((".tmp",".partial"))])
        else:
            raise ValueError("unsupported PostgreSQL backup method")
    else:
        raise ValueError("unknown engine")
    files = [{"name":str(p.relative_to(data)),"size":p.stat().st_size,"sha256":sha(p)} for p in sorted(data.rglob("*")) if p.is_file()]
    manifest["files"] = files
    (data / "metadata.json").write_text(json.dumps(manifest),encoding="utf-8")
    archive = root / "backup.tar"
    with tarfile.open(archive,"w") as tar:
        tar.add(data,arcname="payload")
    manifest.update(file=str(archive),sha256=sha(archive),size=archive.stat().st_size,complete=True)
    (root / "manifest.json").write_text(json.dumps(manifest),encoding="utf-8")
    print("Backup files and checksum recorded")

if __name__ == "__main__":
    try:
        main(sys.argv[1])
    except Exception as error:
        detail = str(error)
        if isinstance(error, subprocess.CalledProcessError):
            detail += '\n' + (error.stderr.decode(errors='replace') if isinstance(error.stderr, bytes) else error.stderr or '')
        spec = json.load(open(sys.argv[1]))
        for secret in (spec.get('password'), spec.get('username')):
            if secret:
                detail = detail.replace(secret, '[redacted]')
        (Path(sys.argv[1]).parent/'failure.json').write_text(json.dumps({'error':detail[-12000:]}))
        raise SystemExit('Backup failed; sanitized diagnostic recorded in failure.json')
