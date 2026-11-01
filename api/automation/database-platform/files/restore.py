"""Restore immutable artifacts to a new loopback-only instance; verify and stop."""
import datetime
import glob
import hashlib
import json
import os
import re
import pwd
import shutil
import socket
import subprocess
import sys
import tarfile
import time
from pathlib import Path

def run(args, **kwargs):
    return subprocess.run(args, check=True, timeout=3600, **kwargs)

def digest(path):
    h = hashlib.sha256()
    with open(path,"rb") as source:
        for chunk in iter(lambda:source.read(1024*1024),b""):
            h.update(chunk)
    return h.hexdigest()

def main(path):
    os.umask(0o077)
    spec = json.load(open(path,encoding="utf-8")); root = Path(path).parent
    artifact=spec["artifact"]; archive=root/"backup.tar"
    if digest(archive)!=artifact["sha256"]:
        raise ValueError("backup checksum mismatch")
    if spec.get("targetTime"):
        if artifact["engine"] not in ('postgresql','mysql') or artifact["method"] == "logical" or not artifact.get("archiveCheck"):
            raise ValueError("this artifact has no verified native archive for PITR")
        target = datetime.datetime.fromisoformat(spec["targetTime"].replace('Z','+00:00'))
        end = datetime.datetime.fromisoformat(artifact["archiveThrough"])
        if target.tzinfo is None or target.timestamp() < artifact["backupStop"] or target > end:
            raise ValueError("recovery target is outside the exported archive window")
    with tarfile.open(archive) as tar:
        for item in tar.getmembers():
            if item.issym() or item.islnk() or item.isdev() or not (root/item.name).resolve().is_relative_to(root.resolve()):
                raise ValueError("unsafe backup archive entry")
        seen=set()
        for item in tar.getmembers():
            if item.name in seen or not (item.name=='payload' or item.name.startswith('payload/')):
                raise ValueError("invalid or duplicate archive entry")
            seen.add(item.name); target=root/item.name
            if item.isdir():
                target.mkdir(parents=True,exist_ok=True,mode=0o700)
            elif item.isfile():
                target.parent.mkdir(parents=True,exist_ok=True,mode=0o700)
                with tar.extractfile(item) as source,open(target,'xb') as dest:
                    shutil.copyfileobj(source,dest)
                target.chmod(0o600)
            else:
                raise ValueError("unsupported archive entry")
    payload=root/"payload"; metadata=json.load(open(payload/"metadata.json"))
    for item in metadata["files"]:
        if digest(payload/item["name"])!=item["sha256"]:
            raise ValueError("backup file integrity mismatch")
    user=pwd.getpwnam("autoopsdb")
    for item in [root]+list(root.rglob("*")):
        os.chown(item,user.pw_uid,user.pw_gid)
    port=spec["port"]; engine=artifact["engine"]
    unit=f"autoops-restore-{spec['operationId']}-{artifact['memberId']}"
    data=root/"data"; command=[]; report=dict(verified=False,port=port,path=str(root),hostId=spec["hostId"],service=unit,engine=engine,checksumsVerified=True,databases=[])
    if engine=="mysql":
        binaries=glob.glob("/opt/autoops/mysql/*/bin/mysqld")
        if not binaries:
            raise RuntimeError("matching MySQL binary is not installed")
        binary=binaries[0]
        version=run([binary,"--version"],capture_output=True,text=True).stdout
        if artifact["version"].split('-')[0] not in version:
            raise RuntimeError("restore requires the original MySQL version")
        config=root/"my.cnf"
        config.write_text(f"[mysqld]\nuser=autoopsdb\nport={port}\nbind-address=127.0.0.1\ndatadir={data}\nsocket={root}/mysql.sock\npid-file={root}/mysql.pid\nmysqlx=OFF\nskip-log-bin\ngtid_mode=ON\nenforce_gtid_consistency=ON\nskip-slave-start\npersisted_globals_load=OFF\ninnodb_buffer_pool_size=64M\n")
        os.chown(config,user.pw_uid,user.pw_gid)
        if artifact["method"]!='logical':
            shutil.copytree(payload/"physical",data)
            if artifact.get('prepared') is False:
                xtrabackup='/opt/autoops/xtrabackup/percona-xtrabackup-8.0.26-18-Linux-x86_64.glibc2.17-minimal/bin/xtrabackup'
                increments=metadata.get('incrementals',[])
                args=[xtrabackup,'--no-defaults','--prepare','--target-dir='+str(data)]
                if increments:
                    run(args+['--apply-log-only'],stdout=subprocess.DEVNULL)
                    for name in increments:
                        if not name.startswith('incremental-') or '/' in name:raise ValueError('invalid incremental directory')
                        run(args+['--apply-log-only','--incremental-dir='+str(payload/name)],stdout=subprocess.DEVNULL)
                run(args,stdout=subprocess.DEVNULL)
            for item in [data]+list(data.rglob("*")):
                os.chown(item,user.pw_uid,user.pw_gid)
        else:
            run([binary,"--defaults-file="+str(config),"--initialize-insecure"],stdout=subprocess.DEVNULL)
        command=[binary,"--defaults-file="+str(config)]
    elif engine=="postgresql":
        major=artifact["version"].split('.')[0];bindir=Path("/usr/lib/postgresql")/major/"bin"
        if artifact["method"] == "logical":
            run(["runuser","-u","autoopsdb","--",str(bindir/"initdb"),"-D",str(data),"-U","postgres","--auth-local=trust","--auth-host=trust","--data-checksums"],stdout=subprocess.DEVNULL)
        else:
            conf = root/"pgbackrest.conf"
            for directory in ("log","lock","spool"):
                (root/directory).mkdir();os.chown(root/directory,user.pw_uid,user.pw_gid)
            conf.write_text(f"[global]\nrepo1-path={payload}/repository\nlog-path={root}/log\nlock-path={root}/lock\nspool-path={root}/spool\nprocess-max=1\n[autoops]\npg1-path={data}\n")
            os.chown(conf,user.pw_uid,user.pw_gid)
            args=["runuser","-u","autoopsdb","--","pgbackrest","--config="+str(conf),"--stanza=autoops","--set="+artifact["backupLabel"],"--archive-mode=off","--target-action=promote"]
            if spec.get("targetTime"):
                # PostgreSQL recovery GUCs use SQL timestamp syntax, rather than
                # passing the API's RFC3339 representation through verbatim.
                stamp=datetime.datetime.fromisoformat(spec["targetTime"].replace('Z','+00:00')).astimezone(datetime.timezone.utc)
                args += ["--type=time","--target="+stamp.strftime('%Y-%m-%d %H:%M:%S.%f+00:00')]
            else:args += ["--type=immediate"]
            run(args+["restore"],stdout=subprocess.DEVNULL)
        # Override copied settings before startup. Recovery cannot contact the
        # original leader or archive repository and never advertises a remote listener.
        with open(data/"postgresql.auto.conf","a") as config:
            config.write(f"\nlisten_addresses='127.0.0.1'\nport={port}\nunix_socket_directories='{root}'\nshared_buffers='32MB'\narchive_mode='off'\nprimary_conninfo=''\nprimary_slot_name=''\nsynchronous_standby_names=''\nssl=off\ndata_directory='{data}'\nhba_file='{data}/pg_hba.conf'\nident_file='{data}/pg_ident.conf'\nexternal_pid_file=''\nlogging_collector=off\narchive_command=''\narchive_cleanup_command=''\nrecovery_end_command=''\n")
        (data/"pg_hba.conf").write_text("local all all trust\nhost all all 127.0.0.1/32 trust\n")
        (data/"pg_ident.conf").write_text("")
        for f in (data/"postgresql.auto.conf",data/"pg_hba.conf",data/"pg_ident.conf"):os.chown(f,user.pw_uid,user.pw_gid)
        command=[str(bindir/"postgres"),"-D",str(data)]
    elif engine=="redis":
        data.mkdir();os.chown(data,user.pw_uid,user.pw_gid)
        shutil.copy2(payload/"dump.rdb",data/"dump.rdb");os.chown(data/"dump.rdb",user.pw_uid,user.pw_gid)
        command=["/usr/bin/redis-server","--bind","127.0.0.1","--port",str(port),"--dir",str(data),"--appendonly","no","--save",""]
    else:
        raise ValueError("unknown engine")
    run(["systemd-run","--unit",unit,"--property=User=autoopsdb","--property=Group=autoopsdb","--property=RuntimeMaxSec=3600"]+command)
    try:
        for _ in range(60):
            try:
                with socket.create_connection(("127.0.0.1",port),1):break
            except OSError:time.sleep(1)
        else:raise RuntimeError("isolated instance did not start")
        if engine=="mysql":
            mysql=str(Path(binary).parent/"mysql")
            physical=artifact["method"]!='logical'
            password=spec.get("password","") if physical else ""
            username=spec.get("username","root") if physical else "root"
            if not physical:
                with open(payload/"dump.sql","rb") as dump:
                    run([mysql,"--no-defaults","-h","127.0.0.1","-P",str(port),"-u","root"],stdin=dump,stdout=subprocess.DEVNULL)
            if physical and artifact['method']=='binlog-archive':
                if not spec.get('targetTime'):raise ValueError('binlog restore requires an explicit target time')
                stamp=datetime.datetime.fromisoformat(spec['targetTime'].replace('Z','+00:00')).astimezone(datetime.timezone.utc)
                if stamp.microsecond:raise ValueError('MySQL binlog time targets use whole UTC seconds')
                option=root/'replay.cnf'
                quote=lambda v:'"'+str(v).replace('\\','\\\\').replace('"','\\"').replace('\n','\\n')+'"'
                option.write_text('[client]\n'+'\n'.join(k+'='+quote(v) for k,v in dict(host='127.0.0.1',port=port,user=username,password=password).items()))
                try:
                    binlog=str(Path(binary).parent/'mysqlbinlog')
                    names=metadata['binlogs']
                    # Single native stream preserves transactions spanning files.
                    # Credentials stay in a private option file, never in argv.
                    args=[binlog,'--no-defaults','--verify-binlog-checksum','--start-position='+str(metadata['binlogPosition']),'--stop-datetime='+stamp.strftime('%Y-%m-%d %H:%M:%S')]+[str(payload/'binlogs'/name) for name in names]
                    with open(root/'replay-error.log','wb') as err:
                        producer=subprocess.Popen(args,stdout=subprocess.PIPE,stderr=err,env=dict(os.environ,TZ='UTC'))
                        try:
                            consumer=subprocess.run([mysql,'--defaults-file='+str(option)],stdin=producer.stdout,stdout=subprocess.DEVNULL,stderr=err,timeout=3600)
                            producer.stdout.close()
                            if producer.wait(timeout=60)!=0 or consumer.returncode!=0:raise RuntimeError('native binlog replay failed')
                        finally:
                            if producer.poll() is None:producer.kill();producer.wait()
                finally:option.unlink(missing_ok=True)
                report['targetTime']=spec['targetTime'];report['targetPrecision']='UTC seconds, excludes events at/after target'
            import pymysql
            with pymysql.connect(host="127.0.0.1",port=port,user=username,password=password) as conn:
                with conn.cursor() as cursor:
                    for name in metadata.get("databases",[]):
                        cursor.execute("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=%s",(name,))
                        report["databases"].append(dict(name=name,tables=cursor.fetchone()[0]))
            option=root/"verification.cnf"
            quote=lambda v:'"'+str(v).replace('\\','\\\\').replace('"','\\"').replace('\n','\\n')+'"'
            option.write_text("[client]\n"+"\n".join(k+"="+quote(v) for k,v in dict(host="127.0.0.1",port=port,user=username,password=password).items()))
            try:run([str(Path(binary).parent/"mysqlcheck"),"--defaults-file="+str(option),"--all-databases"],stdout=subprocess.DEVNULL)
            finally:option.unlink(missing_ok=True)
        elif engine=="postgresql":
            env=dict(os.environ,PGHOST="127.0.0.1",PGPORT=str(port),PGUSER="postgres")
            for _ in range(90):
                ready=subprocess.run([str(bindir/"psql"),"-d","postgres","-At","-c","SELECT NOT pg_is_in_recovery()"],env=env,capture_output=True,text=True,timeout=5)
                if ready.returncode==0 and ready.stdout.strip()=='t':break
                time.sleep(1)
            else:raise RuntimeError("native recovery did not reach the requested target")
            for i,name in enumerate(metadata["databases"]):
                if artifact["method"] == "logical":
                    if name!="postgres":run([str(bindir/"createdb"),"--",name],env=env)
                    run([str(bindir/"pg_restore"),"--exit-on-error","--no-owner","--no-privileges","-d",name,str(payload/f"database-{i}.dump")],env=env,stdout=subprocess.DEVNULL)
                count=run([str(bindir/"psql"),"-d",name,"-At","-c","SELECT count(*) FROM pg_class WHERE relkind='r' AND relnamespace NOT IN (SELECT oid FROM pg_namespace WHERE nspname IN ('pg_catalog','information_schema'))"],env=env,capture_output=True,text=True).stdout.strip()
                report["databases"].append(dict(name=name,tables=int(count)))
            report["targetTime"]=spec.get("targetTime")
        else:
            import redis
            check=run(['redis-check-rdb',str(payload/'dump.rdb')],capture_output=True,text=True).stdout
            total=re.search(r'(\d+) keys read',check);expiring=re.search(r'(\d+) expires',check)
            if not total or not expiring:raise RuntimeError('RDB expiry accounting unavailable')
            persistent=0;keys=0
            for db in range(16):
                client=redis.Redis(host='127.0.0.1',port=port,db=db,socket_timeout=10)
                cursor=0
                while True:
                    cursor,batch=client.scan(cursor,count=500)
                    pipeline=client.pipeline(transaction=False)
                    for key in batch:pipeline.pttl(key)
                    for ttl in pipeline.execute():
                        if ttl==-1:persistent+=1
                        if ttl!=-2:keys+=1
                    if cursor==0:break
                client.close()
            expected=int(total.group(1))-int(expiring.group(1))
            if persistent!=expected or keys>int(total.group(1)):
                raise RuntimeError('restored RDB persistent-key count mismatch')
            report.update(keys=keys,snapshotKeys=int(total.group(1)),persistentKeys=persistent,expectedPersistentKeys=expected,expiredKeys=int(total.group(1))-keys)
        report["verified"]=True
    finally:
        run(["systemctl","stop",unit])
    report["state"]="stopped-after-verification"
    (root/"report.json").write_text(json.dumps(report),encoding="utf-8")
    print("Isolated restore verified; instance stopped and files retained")

if __name__=="__main__":main(sys.argv[1])
