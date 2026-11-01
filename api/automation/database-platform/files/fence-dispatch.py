#!/usr/bin/python3
"""Root forced-command SSH dispatcher; persistent start guard prevents old-primary return."""
import json
import os
import re
import shlex
import subprocess
import sys
from pathlib import Path

cluster=sys.argv[1] if len(sys.argv)==2 else ''
if not cluster.isdecimal():sys.exit(1)
args=shlex.split(os.environ.get('SSH_ORIGINAL_COMMAND',''))
if len(args)!=2 or args[0] not in ('check','fence','persist-promotion'):sys.exit(1)
action,unit=args
allowed=json.load(open('/etc/autoops-fencing/'+cluster+'.json'))
if unit not in allowed or not re.fullmatch(r'autoops-[A-Za-z0-9_.@-]{1,100}',unit):sys.exit(1)
if action=='check':
    result=subprocess.run(['systemctl','show',unit,'-p','LoadState','--value'],check=True,capture_output=True,text=True,timeout=5)
    if result.stdout.strip()!='loaded':sys.exit(1)
    print('READY:'+unit);sys.exit(0)
if action=='persist-promotion':
    # Only persist a role that the native manager has already established.
    # This command cannot turn a protected replica into a writable primary.
    import pymysql
    with pymysql.connect(**allowed[unit],connect_timeout=3,read_timeout=10,write_timeout=10) as conn:
        with conn.cursor() as cursor:
            cursor.execute('SHOW SLAVE STATUS')
            if cursor.fetchall():sys.exit(1)
            cursor.execute('SELECT @@global.read_only,@@global.super_read_only')
            if cursor.fetchone()!=(0,0):sys.exit(1)
            cursor.execute('SET PERSIST super_read_only=OFF')
            cursor.execute('SET PERSIST read_only=OFF')
    print('PERSISTED:'+unit);sys.exit(0)
# A reachable old primary is fenced by persistent super_read_only, verified
# locally through the independent SSH channel. This also supports Orchestrator's
# graceful flow, which invokes ordinary pre-failover hooks before demotion.
# If SQL cannot positively confirm write protection, stop the service and retain
# a persistent boot guard. A connection timeout never counts as successful fencing.
try:
    import pymysql
    with pymysql.connect(**allowed[unit],connect_timeout=3,read_timeout=10,write_timeout=10) as conn:
        with conn.cursor() as cursor:
            cursor.execute('SET PERSIST super_read_only=ON')
            cursor.execute('SET PERSIST read_only=ON')
            cursor.execute('SELECT @@global.read_only,@@global.super_read_only')
            if cursor.fetchone()!=(1,1):raise RuntimeError('write protection not confirmed')
    print('FENCED:'+unit);sys.exit(0)
except Exception:
    pass
root=Path('/var/lib/autoops-fenced')/cluster;root.mkdir(parents=True,exist_ok=True,mode=0o700)
marker=root/unit;marker.write_text('Fenced by independent recovery hook\n')
dropin=Path('/etc/systemd/system')/(unit+'.service.d');dropin.mkdir(parents=True,exist_ok=True,mode=0o755)
(dropin/'90-autoops-fenced.conf').write_text('[Unit]\nConditionPathExists=!'+str(marker)+'\n')
subprocess.run(['systemctl','daemon-reload'],check=True,timeout=15)
subprocess.run(['systemctl','stop',unit],check=True,timeout=60)
state=subprocess.run(['systemctl','show',unit,'-p','ActiveState','--value'],check=True,capture_output=True,text=True,timeout=5)
pid=subprocess.run(['systemctl','show',unit,'-p','MainPID','--value'],check=True,capture_output=True,text=True,timeout=5)
if state.stdout.strip() not in ('inactive','failed') or pid.stdout.strip()!='0':sys.exit(1)
print('FENCED:'+unit)
