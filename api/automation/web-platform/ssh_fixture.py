"""Opt-in SSH test fixture: unpack packages without installing them or changing systemd."""
import hashlib,json,os,pathlib,re,shutil,socket,subprocess,sys,time
root=sys.argv[1];action=sys.argv[2]
if not re.fullmatch(r'/tmp/autoops-web-live-[a-z0-9-]+',root):raise ValueError('invalid test root')
p=pathlib.Path(root)
if action=='cleanup':
 for file in ['nginx.pid','backend.pid']:
  try:
   pid=int((p/file).read_text());cmd=pathlib.Path('/proc/'+str(pid)+'/cmdline').read_bytes().decode(errors='ignore')
   if root in cmd:os.kill(pid,3 if file=='nginx.pid' else 15)
  except (FileNotFoundError,ProcessLookupError):pass
 time.sleep(.5);shutil.rmtree(root,ignore_errors=True)
 if len(sys.argv)>3:
  host_id=str(int(sys.argv[3]));key=hashlib.sha256((host_id+':host::'+root+'/nginx.conf').encode()).hexdigest()
  shutil.rmtree('/var/tmp/autoops-web/'+key,ignore_errors=True)
 print(json.dumps({'cleaned':True}));sys.exit()
os.mkdir(root,0o755)
for port in (28080,28081,28082,28443):
 with socket.socket() as sock:sock.bind(('127.0.0.1',port))
result=subprocess.run(['apt-get','download','nginx='+sys.argv[3]],cwd=root,capture_output=True,text=True,timeout=90)
if result.returncode:raise ValueError(result.stderr[-2000:])
archives=list(p.glob('nginx_*.deb'))
if len(archives)!=1:raise ValueError('expected one pinned nginx package')
subprocess.run(['dpkg-deb','-x',str(archives[0]),str(p/'payload')],check=True)
binary=str(p/'payload/usr/sbin/nginx');subprocess.run([binary,'-v'],check=True)
(p/'sites').mkdir();(p/'sites/placeholder.conf').write_text('# Integration fixture\n');(p/'www').mkdir();(p/'www/index.html').write_text('autoops-backend')
temporary=''.join(kind+'_temp_path '+root+'/'+kind+';' for kind in ['client_body','proxy','fastcgi','uwsgi','scgi'])
(p/'nginx.conf').write_text('pid '+root+'/nginx.pid;\nerror_log '+root+'/error.log;\nevents {}\nhttp {'+temporary+'access_log '+root+'/access.log;include '+root+'/sites/*.conf;server {listen 127.0.0.1:28080;location / {return 200 "baseline";}}}\n')
backend=subprocess.Popen(['python3','-m','http.server','28081','--bind','127.0.0.1','--directory',str(p/'www')],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)
(p/'backend.pid').write_text(str(backend.pid))
subprocess.run([binary,'-p',root,'-c',str(p/'nginx.conf')],check=True)
print(json.dumps({'root':root,'binary':binary,'port':28080}))
