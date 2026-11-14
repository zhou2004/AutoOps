"""One-shot Nginx adapter. JSON on stdin/stdout; no credentials in argv.
Runs on the CMDB host. Docker containers only require their ordinary POSIX tools.
"""
import shlex, datetime, base64, contextlib, fcntl, glob, hashlib, json, os, pathlib, re, shutil, subprocess, sys, tempfile, time, urllib.request, urllib.error, socket, ssl, selectors

LIMIT = 8 * 1024 * 1024

def sha(value):
    return hashlib.sha256(value if isinstance(value, bytes) else value.encode()).hexdigest()

def run(args, data=None, timeout=45, check=True):
    # Drain both streams with a hard bound before allocating the full output.
    with tempfile.TemporaryFile() as stdin:
        if data: stdin.write(data)
        stdin.seek(0)
        p = subprocess.Popen(args, stdin=stdin, stdout=subprocess.PIPE, stderr=subprocess.PIPE, start_new_session=True)
        streams = {p.stdout:bytearray(), p.stderr:bytearray()}
        sel=selectors.DefaultSelector()
        for stream in streams: sel.register(stream,selectors.EVENT_READ)
        deadline=time.monotonic()+timeout
        try:
            while sel.get_map():
                if time.monotonic()>deadline: raise TimeoutError('command timed out')
                for key,_ in sel.select(min(0.2,max(0,deadline-time.monotonic()))):
                    chunk=os.read(key.fileobj.fileno(),65536)
                    if not chunk:sel.unregister(key.fileobj);continue
                    streams[key.fileobj].extend(chunk)
                    if len(streams[key.fileobj])>LIMIT:raise ValueError('command output exceeded limit')
            p.wait(timeout=max(0.1,deadline-time.monotonic()))
        except BaseException:
            import signal
            try:os.killpg(p.pid,signal.SIGKILL)
            except ProcessLookupError:pass
            p.wait();p.stdout.close();p.stderr.close();raise
        finally:sel.close()
        out,err=(bytes(streams[x]).decode(errors='replace') for x in (p.stdout,p.stderr))
        p.stdout.close();p.stderr.close()
        if check and p.returncode:raise ValueError(err[-3000:] or 'command failed')
        return out,err,p.returncode

def verify_tls(probes):
    result=[]
    for p in probes:
        # Pin the imported leaf; private CAs and staging certificates are supported.
        context=ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT);context.check_hostname=False;context.verify_mode=ssl.CERT_NONE
        with socket.create_connection((p['host'],int(p['port'])),timeout=10) as raw:
            with context.wrap_socket(raw,server_hostname=p['serverName']) as conn:
                fingerprint=sha(conn.getpeercert(binary_form=True))
        if p.get('fingerprint') and fingerprint!=p['fingerprint']:raise ValueError('live TLS certificate fingerprint mismatch for '+p['serverName'])
        result.append({'bindingId':p['bindingId'],'versionId':p['versionId'],'fingerprint':fingerprint})
    return result

def save_journal(path,record):
    fd,temporary=tempfile.mkstemp(prefix='.journal.',dir=os.path.dirname(path))
    try:
        with os.fdopen(fd,'w') as f:json.dump(record,f);f.flush();os.fsync(f.fileno())
        os.replace(temporary,path)
        directory=os.open(os.path.dirname(path),os.O_DIRECTORY)
        try:os.fsync(directory)
        finally:os.close(directory)
    finally:
        if os.path.exists(temporary):os.unlink(temporary)

class Runtime:
    def __init__(self, spec):
        self.s = spec
        self.container = spec.get('container') if spec['runtime'] == 'docker' else None
        self.identity = '';self.running=True
        if self.container:
            raw = json.loads(run(['docker', 'inspect', self.container])[0])[0]
            self.identity = raw['Id'];self.running=raw['State']['Running']
            self.container = self.identity  # never resolve a mutable name again in this operation
    def cmd(self, args, data=None, **kw):
        prefix = ['docker', 'exec', '-i', self.container] if self.container else []
        return run(prefix + args, data=data, **kw)
    def read(self, path):
        if self.container:
            return self.cmd(['cat', '--', path])[0]
        if os.path.getsize(path) > LIMIT: raise ValueError('file exceeds limit')
        return pathlib.Path(path).read_text()
    def exists(self, path):
        return self.cmd(['test', '-e', path], check=False)[2] == 0
    def write(self, path, data, mode=0o644):
        parent = os.path.dirname(path)
        self.cmd(['mkdir', '-p', '--', parent])
        if self.container:
            # Positional parameters prevent path/content shell interpolation.
            self.cmd(['sh', '-c', 'umask 077; t=$(mktemp "$1/.autoops.XXXXXX") || exit; cat > "$t" && chmod "$3" "$t" && mv -f -- "$t" "$2"', 'write', parent, path, oct(mode)[2:]], data.encode())
        else:
            fd, temporary = tempfile.mkstemp(prefix='.autoops.', dir=parent)
            try:
                with os.fdopen(fd, 'w') as f:
                    f.write(data); f.flush(); os.fsync(f.fileno())
                if os.path.exists(path):
                    st=os.stat(path); os.chown(temporary,st.st_uid,st.st_gid)
                os.chmod(temporary, mode); os.replace(temporary, path)
            finally:
                if os.path.exists(temporary): os.unlink(temporary)
    def realpath(self, path):
        if not self.container:return os.path.realpath(path)
        suffix=[];current=path
        while not self.exists(current):
            parent,name=os.path.split(current)
            if parent==current:raise ValueError('cannot resolve path')
            suffix.insert(0,name);current=parent
        resolved=self.cmd(['readlink','-f','--',current])[0].strip()
        if not resolved.startswith('/'):raise ValueError('cannot resolve path')
        return os.path.join(resolved,*suffix)
    def mode(self, path):
        if not self.exists(path): return 0o644
        return int(self.cmd(['stat', '-c', '%a', '--', path])[0].strip(), 8)
    def nginx(self, *args, config=None):
        return self.cmd([self.s['binary'], '-p', self.s['prefix'], '-c', config or self.s['configPath']] + list(args))
    def discover(self):
        out, err, _ = self.nginx('-T')
        _, version, _ = self.cmd([self.s['binary'], '-V'])
        paths = re.findall(r'^# configuration file (/.+):$', out, re.M)
        if self.s['configPath'] not in paths: paths.insert(0, self.s['configPath'])
        included = set(paths)
        # List every managed native .conf file, including conf.d files absent
        # from nginx -T, so changes to inactive files also participate in drift.
        # The resolved scope check prevents symlinked directories escaping the
        # adopted roots; these additional files are never implicitly activated.
        for root in self.s.get('managedPaths', []):
            if not self.exists(root): continue
            found = self.cmd(['find', '-L', root, '-type', 'f', '-name', '*.conf', '-print'])[0]
            for path in found.splitlines():
                try: self.guard_path(path)
                except ValueError: continue
                if path not in included: paths.append(path)
        paths = list(dict.fromkeys(paths))
        if len(paths)>200: raise ValueError('too many configuration files')
        files = []
        for p in paths:
            try: self.guard_path(p); editable = True
            except ValueError: editable = False
            content=self.read(p)
            if 'PRIVATE KEY-----' in content:raise ValueError('private key material cannot be read as Nginx configuration: '+p)
            files.append({'path':p,'content':content,'mode':self.mode(p),'included':p in included,'editable':editable})
        for f in files: f['hash'] = sha(f['content'])
        if sum(len(f['content']) for f in files)>4*1024*1024: raise ValueError('configuration too large')
        # Log and TLS metadata describe the live include tree, not inactive drafts.
        alltext='\n'.join(f['content'] for f in files if f['included'])
        logs=[]
        for directive in directives(alltext):
            if len(directive)<2 or directive[0][0] not in ('access_log','error_log'):continue
            p=directive[1][0].strip('\"\'')
            if p.startswith('/') and '$' not in p and not p.startswith('/dev/'):
                if self.cmd(['test','-f',p],check=False)[2]==0:logs.append({'id':sha(p), 'path':p,'realPath':self.realpath(p),'kind':'access' if directive[0][0]=='access_log' else 'error','format':directive[2][0].strip("\"'") if directive[0][0]=='access_log' and len(directive)>2 else 'combined'})
        logs=list({item['id']:item for item in reversed(logs)}.values())
        if self.container: logs.append({'id':'stdout','path':'stdout/stderr','kind':'container'})
        identity = self.identity
        if not identity:
            entry=self.s['configPath'] if re.fullmatch(r'/etc/autoops-web/[A-Za-z0-9_-]+/current/nginx.conf',self.s['configPath']) else self.realpath(self.s['configPath'])
            identity = sha(self.realpath(self.s['binary']) + ':' + entry)
        dependencies=[];certificates=[]
        for directive in directives(alltext):
            if len(directive)<2 or directive[0][0] not in ('ssl_certificate','ssl_certificate_key'):continue
            dependency=directive[1][0].strip('"\'');dependency=dependency if dependency.startswith('/') else os.path.join(self.s['prefix'],dependency)
            if '$' in dependency:continue
            dependencies.append((dependency,sha(self.read(dependency))))
            if directive[0][0]=='ssl_certificate':
                public=self.read(dependency);first=public[:public.find('-----END CERTIFICATE-----')+len('-----END CERTIFICATE-----')]
                metadata={'path':dependency,'fingerprint':sha(ssl.PEM_cert_to_DER_cert(first))}
                if shutil.which('openssl'):
                    metadata['details']=run(['openssl','x509','-noout','-dates','-issuer','-subject','-ext','subjectAltName'],data=public.encode())[0].strip()
                certificates.append(metadata)
        return {'files':files,'certificates':certificates, 'digest':sha(json.dumps([(f['path'],f['hash']) for f in sorted(files,key=lambda f:f['path'])]+sorted(set(dependencies)))), 'identity':identity, 'version':version.strip(), 'logs':logs, 'configCheck':True}
    def process(self,required=True):
        out,_,_=self.nginx('-T');_,version,_=self.cmd([self.s['binary'],'-V'])
        pidpath=None
        for directive in directives(out):
            if len(directive)>1 and directive[0][0]=='pid':pidpath=directive[1][0].strip('"\'');break
        if not pidpath:
            match=re.search(r'--pid-path=([^\s]+)',version);pidpath=match[1] if match else os.path.join(self.s['prefix'],'logs/nginx.pid')
        if not pidpath.startswith('/'):pidpath=os.path.join(self.s['prefix'],pidpath)
        if not self.exists(pidpath):
            if required:raise ValueError('nginx master PID file is missing')
            return {'running':False,'pidPath':pidpath}
        pid=self.read(pidpath).strip()
        if not pid.isdigit() or int(pid)<1:raise ValueError('invalid nginx master PID')
        proc='/proc/'+pid+'/cmdline'
        if not self.exists(proc):
            if required:raise ValueError('nginx master is not running')
            return {'running':False,'pidPath':pidpath}
        command=self.read(proc).replace('\x00',' ').strip()
        if not command.startswith('nginx: master process '):raise ValueError('PID does not identify an nginx master')
        tokens=shlex.split(command[len('nginx: master process '):])
        if '-c' in tokens:entry=tokens[tokens.index('-c')+1]
        else:
            match=re.search(r'--conf-path=([^\s]+)',version);entry=match[1] if match else os.path.join(self.s['prefix'],'conf/nginx.conf')
        if not entry.startswith('/'):entry=os.path.join(self.s['prefix'],entry)
        if self.realpath(entry)!=self.realpath(self.s['configPath']):raise ValueError('PID belongs to a different nginx configuration')
        return {'running':True,'pid':pid,'pidPath':pidpath}
    def control_guard(self,action):
        self.process(required=action not in ('start',))
        if not self.container and self.s.get('service') and action in ('start','restart'):
            command=run(['systemctl','show','--property=ExecStart','--value',self.s['service']])[0]
            if self.s['configPath']!='/etc/nginx/nginx.conf' and self.s['configPath'] not in command:raise ValueError('systemd unit does not target this configuration')
    def health(self):
        self.nginx('-t');self.process()
        url=self.s.get('healthUrl')
        if not url: raise ValueError('configure a health URL before service adoption')
        headers={}
        if self.s.get('healthHost'): headers['Host']=self.s['healthHost']
        # Probe from the managed host, with normal TLS verification.
        req=urllib.request.Request(url,headers=headers)
        with urllib.request.build_opener(urllib.request.ProxyHandler({})).open(req,timeout=10) as res:
            code=res.status
        if code != self.s.get('expectedStatus',200): raise ValueError('health status mismatch: '+str(code))
        return {'status':code}
    def guard_path(self,p):
        if not p.startswith('/') or os.path.normpath(p)!=p or '\x00' in p: raise ValueError('invalid absolute path')
        allowed=self.s.get('managedPaths',[])
        real=self.realpath(p)
        if not any(real==self.realpath(a) or real.startswith(self.realpath(a).rstrip('/')+'/') for a in allowed): raise ValueError('path outside managed scope: '+p)
        return real

# Token spans preserve comments and unknown directives. Only file dependency tokens change.
TOKEN=re.compile(r'\s+|\#[^\n]*|"(?:\\.|[^"\\])*"|\'(?:\\.|[^\'\\])*\'|[{};]|[^\s{};]+')
def directives(text):
    tokens=[]
    for m in TOKEN.finditer(text):
        v=m.group()
        if v.isspace() or v.startswith('#'): continue
        if v in '{};':
            if tokens: yield tokens
            tokens=[]
        else: tokens.append((v,m.start(),m.end()))

def staging(rt,files,changed=None):
    base='/tmp/autoops-web-'+str(os.getpid())+'-'+str(time.time_ns())
    rt.cmd(['mkdir','-m','700','--',base])
    paths={f['path']:base+f['path'] for f in files}
    dependencies={p:set() for p in paths}; staged={}
    try:
        for f in files:
            text=f['content']; changes=[]
            for directive in directives(text):
                if len(directive)<2: continue
                key=directive[0][0]
                if key not in ('include','ssl_certificate','ssl_certificate_key','pid','error_log'): continue
                token,start,end=directive[1]; value=token.strip('"\'')
                if '$' in value and key=='include': raise ValueError('dynamic include cannot be staged')
                if key=='include':
                    absolute=value if value.startswith('/') else os.path.join(rt.s['prefix'],value)
                    matches=[p for p in paths if pathlib.PurePosixPath(p).match(absolute)]
                    dependencies[f['path']].update(matches)
                    if not matches:
                        if not any(c in value for c in '*?['):raise ValueError('unresolved include: '+value)
                        rt.cmd(['mkdir','-p',base+'/empty']);replacement='"'+base+'/empty/*.conf"'
                    else:replacement='; include '.join('"'+paths[p]+'"' for p in sorted(matches))
                    # Expand known files to prevent tests silently reading the live configuration.
                elif key in ('ssl_certificate','ssl_certificate_key'):
                    absolute=value if value.startswith('/') else os.path.join(rt.s['prefix'],value)
                    if absolute not in paths:continue
                    replacement='"'+paths[absolute]+'"'
                elif key=='pid': replacement='"'+base+'/nginx.pid"'
                elif key=='error_log': replacement='"'+base+'/error.log"'
                else: continue
                changes.append((start,end,replacement))
            for start,end,value in reversed(changes): text=text[:start]+value+text[end:]
            staged[f['path']]=text
            rt.write(paths[f['path']],text,f.get('mode') or 0o644)
        entry=rt.s['configPath']
        if entry not in paths: raise ValueError('cannot remove the Nginx configuration entry')
        rt.nginx('-t',config=paths[entry])
        reachable=set();pending=[entry]
        while pending:
            current=pending.pop()
            if current in reachable: continue
            reachable.add(current);pending.extend(dependencies[current])
        # A successful main-config test says nothing about a file outside its
        # include tree. Check changed inactive .conf files in the candidate HTTP
        # context as well, without adding includes to the published configuration.
        changes=changed if changed is not None else files
        certificate_paths=set()
        for current in reachable:
            for directive in directives(staged[current]):
                if len(directive)>1 and directive[0][0] in ('ssl_certificate','ssl_certificate_key'):
                    certificate_paths.add(directive[1][0].strip('"\''))
        orphans=[f['path'] for f in changes if not f.get('deleted') and f['path'] not in reachable and paths[f['path']] not in certificate_paths]
        for orphan in orphans:
            if not orphan.endswith('.conf'):
                raise ValueError('inactive configuration must use .conf or be included explicitly: '+orphan)
            # Main configuration can itself include a file containing http {}.
            position=None;owner=None
            for current in sorted(reachable):
                text=staged[current];tokens=[];depth=0
                for match in TOKEN.finditer(text):
                    token=match.group()
                    if token.isspace() or token.startswith('#'):continue
                    if token=='{':
                        if depth==0 and tokens==['http']:position=match.end();owner=current;break
                        depth+=1;tokens=[]
                    elif token=='}':depth-=1;tokens=[]
                    elif token==';':tokens=[]
                    else:tokens.append(token)
                if position is not None:break
            if position is None: raise ValueError('inactive configuration has no HTTP validation context; include it explicitly: '+orphan)
            text=staged[owner]
            candidate=text[:position]+'\ninclude "'+paths[orphan]+'";\n'+text[position:]
            rt.write(paths[owner],candidate,0o600)
            rt.nginx('-t',config=paths[entry])
            rt.write(paths[owner],staged[owner],0o600)
        return {'validated':True}
    finally:
        rt.cmd(['rm','-rf','--',base])

def release_paths(rt,files,operation):
    config=rt.s['configPath']
    if not re.fullmatch(r'/etc/autoops-web/[A-Za-z0-9_-]+/current/nginx.conf',config):return None
    base=config[:-len('/current/nginx.conf')];current=base+'/current';old=rt.realpath(current)
    if not old.startswith(base+'/releases/'):raise ValueError('release pointer is outside instance directory')
    if any(not f['path'].startswith(current+'/') for f in files):raise ValueError('platform instances require all changed files under '+current)
    new=base+'/releases/op-'+str(operation)
    if rt.exists(new):raise ValueError('release already exists; reconcile the original operation')
    rt.cmd(['cp','-a','--',old,new])
    return current,old,new

def switch_release(rt,current,target):
    temporary=current+'.swap-'+str(os.getpid())
    rt.cmd(['ln','-s','--',target,temporary]);rt.cmd(['mv','-Tf','--',temporary,current])

def recover_journal(rt,journal,record,identity):
    if record.get('state') not in ('prepared','written'):raise ValueError('journal does not permit deterministic recovery')
    original=record.get('original',{})
    if not identity or identity!=original.get('identity'):raise ValueError('recovery identity mismatch')
    if rt.container and rt.identity!=identity:raise ValueError('container identity changed')
    hashes=record.get('candidate',{});backups={f['path']:f for f in record['backup']}
    for path,before in backups.items():
        rt.guard_path(path);actual=sha(rt.read(path)) if rt.exists(path) else None
        old=sha(before['content']) if before['exists'] else None
        if actual not in (old,hashes.get(path)):raise ValueError('external modification blocks recovery: '+path)
    for f in original['files']:
        if f['path'] not in backups and (not rt.exists(f['path']) or sha(rt.read(f['path']))!=f['hash']):raise ValueError('untouched configuration drift blocks recovery')
    changes=[dict(f,deleted=not f['exists']) for f in record['backup']]
    staging(rt,merge(original['files'],changes),changes)
    record['state']='recovering';save_journal(journal,record)
    if record.get('release'):
        current=rt.s['configPath'][:-len('/nginx.conf')];release=record['release']
        if rt.realpath(current) not in (release['previous'],release['candidate']):raise ValueError('release pointer changed outside operation')
        switch_release(rt,current,release['previous'])
    else:
        for f in changes:
            path=rt.guard_path(f['path'])
            if f.get('deleted'):rt.cmd(['rm','-f','--',path])
            else:rt.write(path,f['content'],f['mode'])
    rt.nginx('-t');rt.control_guard('reload');rt.nginx('-s','reload');time.sleep(.3);rt.health()
    record['state']='rolled-back';record['observed']=rt.discover();save_journal(journal,record)
    return {'state':'rolled-back','observed':record['observed']}


def discovery_options(command, version):
    """Resolve the master command against that exact binary's compile options."""
    tokens=shlex.split(command)
    compiled={}
    match=re.search(r'configure arguments:\s*(.*)',version)
    if match:
        for token in shlex.split(match.group(1)):
            if token.startswith('--') and '=' in token:
                key,value=token[2:].split('=',1);compiled[key]=value
    def option(key,default):
        if key in tokens:
            index=tokens.index(key)
            if index+1>=len(tokens):raise ValueError('incomplete master option '+key)
            return tokens[index+1]
        return default
    prefix=option('-p',compiled.get('prefix','/usr/local/nginx'))
    if not prefix.startswith('/'):raise ValueError('relative master prefix needs manual configuration')
    config=option('-c',compiled.get('conf-path','conf/nginx.conf'))
    if not config.startswith('/'):config=os.path.join(prefix,config)
    return os.path.normpath(prefix),os.path.normpath(config)

def source_discovery(spec):
    candidates=[];warnings=[]
    environments=[None]
    if spec['runtime']=='docker':
        if spec.get('container'):
            environments=[spec['container']]
        else:
            environments=run(['docker','ps','--no-trunc','--format','{{.ID}}'],timeout=15)[0].splitlines()
            if len(environments)>100:raise ValueError('more than 100 containers; choose a container explicitly')
    for container in environments:
        rt=Runtime(dict(spec,container=container or ''))
        if not rt.running:continue
        # Read only /proc and the selected binary; no service control or file changes.
        if container:
            processes=rt.cmd(['sh','-c','for p in /proc/[0-9]*/cmdline; do printf "%s\\t" "$p"; tr "\\000" " " < "$p" 2>/dev/null; printf "\\n"; done'])[0].splitlines()
        else:
            processes=[]
            for path in glob.glob('/proc/[0-9]*/cmdline'):
                try:processes.append(path+'\t'+pathlib.Path(path).read_bytes().replace(b'\x00',b' ').decode(errors='replace'))
                except (OSError,ProcessLookupError):continue
        for process in processes:
            if '\t' not in process:continue
            path,command=process.split('\t',1);command=command.strip()
            if not command.startswith('nginx: master process '):continue
            pid=path.split('/')[2]
            try:
                binary=rt.cmd(['readlink','-f','/proc/'+pid+'/exe'])[0].strip()
                if not binary.startswith('/') or binary.endswith(' (deleted)'):raise ValueError('running binary cannot be resolved')
                _,version,_=rt.cmd([binary,'-V'])
                prefix,config=discovery_options(command[len('nginx: master process '):],version)
                service=''
                if not container:
                    cgroup=pathlib.Path('/proc/'+pid+'/cgroup').read_text()
                    units=re.findall(r'/([^/\n]+\.service)(?:/|$)',cgroup,re.M)
                    if units:
                        unit=units[-1]
                        main=run(['systemctl','show','--property=MainPID','--value',unit],check=False)[0].strip()
                        if main==pid:service=unit.removesuffix('.service')
                candidate=dict(spec,binary=binary,configPath=config,prefix=prefix,service=service,container=rt.container or '',managedPaths=[os.path.dirname(config)],expectedStatus=200)
                probe=Runtime(candidate)
                if probe.process()['pid']!=pid:raise ValueError('master PID/configuration identity mismatch')
                observed=probe.discover()
                candidate_warnings=[]
                if not candidate.get('healthUrl'):candidate_warnings.append('接管前请填写业务健康检查地址')
                candidates.append({'id':sha(str(spec['hostId'])+':'+spec['runtime']+':'+observed['identity']), 'name':(container[:12]+' · ' if container else '')+config, 'spec':candidate, 'identity':observed['identity'], 'version':version.strip(), 'warnings':candidate_warnings})
            except Exception as ex:
                warnings.append(('container '+str(container)[:12]+' ' if container else '')+'nginx PID '+pid+': '+str(ex)[-500:])
    if not candidates:warnings.append('未发现可识别的运行中 Nginx；请检查权限或使用高级参数手动接入')
    return {'candidates':candidates,'warnings':warnings}


def vip_paths(node):
    cluster=int(node['clusterId']);instance=int(node['instanceId'])
    if cluster<1 or instance<1:raise ValueError('invalid VIP target')
    base='/etc/autoops-web-vip/'+str(cluster)
    if os.path.realpath('/etc/autoops-web-vip')!='/etc/autoops-web-vip':raise ValueError('VIP configuration parent is a symlink')
    return base,'autoops-web-vip-'+str(cluster),'/etc/systemd/system/autoops-web-vip-'+str(cluster)+'.service'

def vip_package_candidate():
    output=run(['env','LC_ALL=C','apt-cache','policy','keepalived'])[0]
    candidate=re.search(r'^\s*Candidate:\s*(\S+)',output,re.M)
    if not candidate or not re.fullmatch(r'[0-9][A-Za-z0-9.+:~_-]*',candidate[1]):raise ValueError('keepalived package candidate unavailable; refresh package metadata first')
    return candidate[1]

def vip_observe(node,check=False,allow_partial=False):
    import ipaddress,struct
    p=node['config'];target=ipaddress.ip_interface(p['cidr']);iface=p['interface'];base,unit,unitpath=vip_paths(node)
    if target.version!=4 or not 8<=target.network.prefixlen<=30 or target.ip in (target.network.network_address,target.network.broadcast_address):raise ValueError('invalid VIP address')
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.-]{0,14}',iface) or not 1<=int(p['routerId'])<=255:raise ValueError('invalid VIP interface or VRID')
    if os.path.islink(base) or os.path.islink(unitpath):raise ValueError('VIP path is a symlink')
    interfaces=json.loads(run(['ip','-j','-4','address','show','dev',iface])[0])
    if len(interfaces)!=1 or 'UP' not in interfaces[0]['flags']:raise ValueError('VIP interface unavailable')
    addresses=[a for a in interfaces[0]['addr_info'] if a['family']=='inet']
    primary=[a['local'] for a in addresses if ipaddress.ip_address(a['local']) in target.network and a['local']!=str(target.ip) and a['prefixlen']==target.network.prefixlen and not a.get('secondary',False)]
    if len(primary)!=1 and not node.get('address'):raise ValueError('VIP interface must have one primary address in the requested subnet')
    address=node.get('address') or primary[0]
    if address not in [a['local'] for a in addresses] or address==str(target.ip):raise ValueError('VIP host address missing from requested interface')
    if node.get('address') and node['address']!=address:raise ValueError('VIP node address changed since preflight')
    mac=json.loads(run(['ip','-j','link','show','dev',iface])[0])[0]['address']
    active=run(['systemctl','is-active','--quiet',unit],check=False)[2]==0
    marker='# AutoOps Web VIP '+str(node['clusterId'])+'\n'
    paths=[base+'/keepalived.conf',base+'/health.py',unitpath]
    existing=[os.path.isfile(path) for path in paths]
    owned=all(existing) and all(marker in pathlib.Path(path).read_text() for path in paths)
    if any(existing) and not owned and not allow_partial:raise ValueError('VIP files have incomplete or foreign ownership; reconcile before changes')
    digest=sha(''.join(pathlib.Path(path).read_text() for path in paths)) if owned else ''
    result={'bound':str(target.ip) in [a['local'] for a in addresses],'active':active,'owned':owned,'address':address,'mac':mac,'configDigest':digest,'arpOwners':[]}
    if not check:return result
    if os.geteuid()!=0:raise ValueError('VIP maintenance requires root or passwordless sudo')
    if not shutil.which('systemctl') or not os.path.isdir('/run/systemd/system'):raise ValueError('VIP requires systemd')
    if not shutil.which('keepalived'):
        info=dict(line.split('=',1) for line in pathlib.Path('/etc/os-release').read_text().splitlines() if '=' in line)
        if info.get('ID','').strip('"') not in ('ubuntu','debian'):raise ValueError('install keepalived before configuring VIP on this OS')
        result['installPackageVersion']=vip_package_candidate()
    # Inspect all platform-owned and system configurations, never change them.
    own=pathlib.Path(base+'/keepalived.conf')
    for root in ['/etc/keepalived','/etc/autoops-database-vip','/etc/autoops-web-vip']:
        for config in pathlib.Path(root).rglob('*.conf'):
            if config==own:continue
            if re.search(r'virtual_router_id\s+'+str(p['routerId'])+r'\b',config.read_text(errors='ignore')):raise ValueError('VRID already configured by another service')
    # Probe ARP and listen for conflicting VRRP advertisements in the same bounded window.
    hw=bytes.fromhex(mac.replace(':',''));packet=b'\xff'*6+hw+struct.pack('!H',0x0806)+struct.pack('!HHBBH',1,0x0800,6,4,1)+hw+socket.inet_aton(address)+b'\x00'*6+socket.inet_aton(str(target.ip))
    owners=set()
    with socket.socket(socket.AF_PACKET,socket.SOCK_RAW,socket.htons(0x0003)) as sock:
        sock.bind((iface,0));sock.settimeout(.2)
        for retry in range(3):
            sock.send(packet);end=time.monotonic()+.7
            while time.monotonic()<end:
                try:data=sock.recv(2048)
                except socket.timeout:continue
                if len(data)>=42 and data[12:14]==b'\x08\x06' and data[28:32]==socket.inet_aton(str(target.ip)):
                    owners.add(':'.join('%02x'%b for b in data[22:28]))
                if len(data)>=36 and data[12:14]==b'\x08\x00' and data[23]==112:
                    start=14+(data[14]&15)*4
                    if len(data)>start+1 and data[start+1]==int(p['routerId']) and not owned and data[26:30] not in [socket.inet_aton(peer) for peer in node.get('peers',[])]:raise ValueError('VRID is already advertised on the network')
    result['arpOwners']=sorted(owners)
    return result

def vip_render(node,spec,pidpath):
    base,unit,unitpath=vip_paths(node);p=node['config'];marker='# AutoOps Web VIP '+str(node['clusterId'])+'\n'
    import ipaddress
    ipaddress.ip_address(node['address'])
    if not 1<=int(node['priority'])<=254:raise ValueError('invalid VIP priority')
    peers=node.get('peers',[])
    if not peers or len(peers)>9:raise ValueError('VIP peers missing')
    for peer in peers:ipaddress.ip_address(peer)
    config=marker+'''global_defs {
    router_id AUTOOPS_WEB_%s_%s
    enable_script_security
    script_user root
}
vrrp_script nginx_health {
    script "%s/health.py"
    interval 2
    timeout 4
    fall 2
    rise 2
}
vrrp_instance AUTOOPS_WEB_%s {
    state BACKUP
    interface %s
    virtual_router_id %s
    priority %s
    advert_int 1
    nopreempt
    unicast_src_ip %s
    unicast_peer {
%s
    }
    virtual_ipaddress {
        %s dev %s
    }
    track_script {
        nginx_health
    }
    garp_master_delay 1
    garp_master_repeat 3
}
'''%(node['clusterId'],node['instanceId'],base,node['clusterId'],p['interface'],p['routerId'],node['priority'],node['address'],'\n'.join('        '+x for x in peers),p['cidr'],p['interface'])
    # PID path is obtained from the actual Nginx entry point, not the host ID.
    values={'pidPath':pidpath,'binary':os.path.realpath(spec['binary']),'prefix':spec['prefix'],'configPath':spec['configPath'],'healthUrl':spec['healthUrl'],'healthHost':spec.get('healthHost',''),'expectedStatus':spec.get('expectedStatus',200)}
    health='#!/usr/bin/python3\n'+marker+'P='+repr(values)+'\n'+'''import os,pathlib,shlex,subprocess,urllib.request
try:
 pid=pathlib.Path(P['pidPath']).read_text().strip()
 assert pid.isdigit() and int(pid)>1
 assert os.path.realpath('/proc/'+pid+'/exe')==P['binary']
 command=pathlib.Path('/proc/'+pid+'/cmdline').read_bytes().replace(b'\\x00',b' ').decode().strip()
 assert command.startswith('nginx: master process ')
 tokens=shlex.split(command[len('nginx: master process '):])
 if '-c' in tokens:
  path=tokens[tokens.index('-c')+1]
  if not path.startswith('/'):path=os.path.join(P['prefix'],path)
  assert os.path.realpath(path)==os.path.realpath(P['configPath'])
 subprocess.run([P['binary'],'-p',P['prefix'],'-c',P['configPath'],'-t'],check=True,timeout=2,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
 headers={'Host':P['healthHost']} if P['healthHost'] else {}
 request=urllib.request.Request(P['healthUrl'],headers=headers)
 with urllib.request.build_opener(urllib.request.ProxyHandler({})).open(request,timeout=2) as response:assert response.status==P['expectedStatus']
except Exception:
 raise SystemExit(1)
'''
    unittext=marker+'''[Unit]
Description=AutoOps Web cluster %s independent VIP
After=network-online.target
Wants=network-online.target
[Service]
Type=simple
ExecStart=/usr/sbin/keepalived --dont-fork --log-console --vrrp --use-file=%s/keepalived.conf --pid=/run/%s.pid --vrrp_pid=/run/%s-vrrp.pid
Restart=on-failure
RestartSec=3
[Install]
WantedBy=multi-user.target
'''%(node['clusterId'],base,unit,unit)
    return [(base+'/keepalived.conf',config,0o600),(base+'/health.py',health,0o700),(unitpath,unittext,0o644)]

def vip_install(version):
    if shutil.which('keepalived'):return
    if not re.fullmatch(r'[0-9][A-Za-z0-9.+:~_-]*',version or ''):raise ValueError('pinned keepalived package version required')
    if run(['systemctl','is-active','--quiet','keepalived.service'],check=False)[2]==0:raise ValueError('unmanaged keepalived service is active')
    masked=os.path.lexists('/run/systemd/system/keepalived.service')
    policy='/usr/sbin/policy-rc.d';created=False
    try:
        if not masked:run(['systemctl','mask','--runtime','keepalived.service'])
        if not os.path.exists(policy):
            with open(policy,'x') as f:f.write('#!/bin/sh\nexit 101\n')
            os.chmod(policy,0o755);created=True
        run(['env','DEBIAN_FRONTEND=noninteractive','apt-get','install','-y','--no-install-recommends','keepalived='+version],timeout=180)
        run(['systemctl','disable','keepalived.service'])
    finally:
        if created:os.unlink(policy)
        if not masked:run(['systemctl','unmask','--runtime','keepalived.service'])
    if not shutil.which('keepalived'):raise ValueError('keepalived installation did not provide the binary')

def vip_execute(req):
    node=req['vip'];action=req['action'];base,unit,unitpath=vip_paths(node)
    if req['spec']['runtime']!='host':raise ValueError('VIP requires host runtime')
    if action in ('vip-check','vip-inspect'):
        result=vip_observe(node,check=action=='vip-check');rt=Runtime(req['spec'])
        try:
            rt.health();result['healthy']=True
            if action=='vip-check':
                endpoint=urllib.parse.urlparse(req['spec']['healthUrl'])
                if endpoint.hostname not in ('localhost','127.0.0.1','::1',result['address']):raise ValueError('VIP health URL must address this Nginx host')
                port=endpoint.port or (443 if endpoint.scheme=='https' else 80);pid=rt.process()['pid']
                lines=run(['ss','-H','-ltnp','sport','=',':'+str(port)])[0].splitlines()
                target=node['config']['cidr'].split('/')[0]
                accepted=['0.0.0.0:'+str(port),'*:'+str(port),target+':'+str(port)]
                if not any(len(line.split())>3 and line.split()[3] in accepted and 'pid='+pid+',' in line for line in lines):raise ValueError('Nginx health port must listen on a wildcard or VIP address, owned by this exact master')
        except Exception as ex:
            result['healthy']=False;result['error']=str(ex)
            if action=='vip-check':raise
        return result
    operation=int(req.get('operationId',0))
    if operation<1:raise ValueError('VIP operation ID required')
    parent='/var/tmp/autoops-web-vip'
    os.makedirs(parent,mode=0o700,exist_ok=True)
    if os.path.islink(parent) or os.stat(parent).st_uid!=0 or os.stat(parent).st_mode&0o022:raise ValueError('unsafe VIP journal parent')
    root=parent+'/'+str(node['clusterId'])
    os.makedirs(root,mode=0o700,exist_ok=True)
    if os.path.islink(root) or os.stat(root).st_uid!=0 or os.stat(root).st_mode&0o022:raise ValueError('invalid VIP execution root')
    with open(root+'/lock','a') as lock:
        fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
        journal=root+'/'+str(operation)+'.json'
        observed=vip_observe(node,allow_partial=action in ('vip-reconcile','vip-rollback'))
        if action in ('vip-resume','vip-reconcile','vip-rollback'):
            rt=Runtime(req['spec']);current=rt.discover()
            if current['identity']!=req.get('identity') or current['digest']!=req.get('baseline'):raise ValueError('VIP recovery Nginx identity or baseline changed')
            if action in ('vip-reconcile','vip-rollback') and not os.path.exists(journal):
                if observed['owned'] or observed['active']:raise ValueError('VIP service exists without this operation journal')
                observed['state']='not-started';return observed
            record=json.loads(pathlib.Path(journal).read_text())
            if record.get('identity')!=req.get('identity') or record.get('baseline')!=req.get('baseline'):raise ValueError('VIP execution journal identity mismatch')
            if record.get('action')=='vip-configure':
                expected={base+'/keepalived.conf',base+'/health.py',unitpath}
                if not record.get('files'):
                    if record.get('state')=='prepared' and not any(os.path.exists(path) for path in expected):
                        observed['state']='not-started';return observed
                    raise ValueError('VIP deployment manifest missing')
                if set(record['files'])!=expected:raise ValueError('VIP manifest path mismatch')
                for path,digest in record['files'].items():
                    if not os.path.exists(path) and record.get('state') in ('prepared','rolled-back'):continue
                    if os.path.islink(path) or not os.path.isfile(path) or sha(pathlib.Path(path).read_text())!=digest:raise ValueError('VIP file drift blocks recovery')
            elif record.get('configDigest')!=observed['configDigest']:raise ValueError('VIP configuration changed after switch')
            if action=='vip-reconcile':
                observed['state']=record.get('state','unknown');return observed
        if action=='vip-rollback':
            if record.get('action')!='vip-configure':raise ValueError('only a new VIP deployment can be rolled back')
            if os.path.isdir(base) and set(os.listdir(base))-{'keepalived.conf','health.py'}:raise ValueError('unexpected VIP files block rollback')
            run(['systemctl','disable','--now',unit],check=False)
            if run(['systemctl','is-active','--quiet',unit],check=False)[2]==0:raise ValueError('VIP service did not stop')
            for path in record['files']:
                if os.path.exists(path):os.unlink(path)
            if os.path.isdir(base):os.rmdir(base)
            run(['systemctl','daemon-reload'])
            record['state']='rolled-back';save_journal(journal,record)
            observed=vip_observe(node);observed['state']='rolled-back';return observed
        if action=='vip-resume':
            record=json.loads(pathlib.Path(journal).read_text())
            if record.get('action')!='vip-switch' or record.get('configDigest')!=observed['configDigest']:raise ValueError('VIP rescue journal or configuration changed')
            run(['systemctl','start',unit])
            run(['systemctl','stop',unit+'-rescue-'+str(operation)+'.timer'],check=False)
            record['state']='resumed';save_journal(journal,record)
            return vip_observe(node)
        if os.path.exists(journal):raise ValueError('VIP operation already has a journal; reconcile before retrying')
        if observed['configDigest']!=node.get('configDigest',''):raise ValueError('VIP configuration changed since preflight')
        rt=Runtime(req['spec']);current=rt.discover()
        if current['identity']!=req['identity'] or current['digest']!=req['baseline']:raise ValueError('Nginx identity or baseline changed')
        rt.health()
        if action=='vip-configure':
            if observed['owned'] or observed['bound'] or observed['active']:raise ValueError('VIP service already exists')
            checks=vip_observe(node,check=True)
            if any(mac not in node.get('peerMACs',[]) for mac in checks['arpOwners']):raise ValueError('VIP became occupied before deployment')
            record={'action':action,'state':'prepared','identity':req['identity'],'baseline':req['baseline']};save_journal(journal,record)
            vip_install(node.get('installPackageVersion'))
            files=vip_render(node,req['spec'],rt.process()['pidPath'])
            if os.path.exists(base):raise ValueError('VIP directory exists; do not overwrite')
            os.makedirs(base,mode=0o700)
            record['files']={path:sha(content) for path,content,_ in files};save_journal(journal,record)
            for path,content,mode in files:
                with open(path,'x') as f:f.write(content);f.flush();os.fsync(f.fileno())
                os.chmod(path,mode)
            run(['/usr/sbin/keepalived','--config-test','--use-file='+base+'/keepalived.conf'])
            run(['/usr/bin/python3',base+'/health.py'])
            record['state']='written';save_journal(journal,record)
            run(['systemctl','daemon-reload']);run(['systemctl','enable','--now',unit])
            record['state']='succeeded';record['configDigest']=vip_observe(node)['configDigest'];save_journal(journal,record)
            return vip_observe(node)
        if action=='vip-switch':
            if not observed['owned'] or not observed['active'] or not observed['bound']:raise ValueError('VIP current owner changed')
            record={'action':action,'state':'prepared','configDigest':observed['configDigest'],'identity':req['identity'],'baseline':req['baseline']};save_journal(journal,record)
            run(['systemd-run','--unit='+unit+'-rescue-'+str(operation),'--on-active=120s','/usr/bin/systemctl','start',unit])
            run(['systemctl','stop',unit]);record['state']='stopped';save_journal(journal,record)
            return vip_observe(node)
        raise ValueError('unsupported VIP action')

# Incremental access-log aggregates. Only registered regular files are read and no
# raw line or query string leaves the host. Cursors live in the metadata DB.
def traffic_parse(line):
    import math, ipaddress
    try:
        if line.startswith('{'):
            d=json.loads(line)
            if not isinstance(d,dict):return None
            ip=str(d.get('remote_addr',''));uri=d.get('uri') or d.get('request_uri')
            if uri is None and isinstance(d.get('request'),str):
                fields=d['request'].split();uri=fields[1] if len(fields)>1 else ''
            status=int(d.get('status',0));size=d.get('body_bytes_sent',d.get('bytes_sent'))
            if not uri or size is None:return None
            result={'ip':ip,'uri':str(uri),'status':status,'bytes':int(size)}
            if d.get('request_length') not in (None,'','-'):result['requestBytes']=int(d['request_length'])
            if d.get('request_time') not in (None,'','-'):result['latencyMs']=float(d['request_time'])*1000
            upstream=d.get('upstream_addr')
            if upstream and upstream!='-':result['upstream']=str(upstream)[:256]
        else:
            m=re.fullmatch(r'(\S+) \S+ \S+ \[[^\]]+\] "[^" ]+ ([^" ]+) [^" ]+" ([1-5][0-9]{2}) ([0-9]+)(?: "(?:[^"\\]|\\.)*" "(?:[^"\\]|\\.)*")?(?:\s*)',line)
            if not m:return None
            result={'ip':m[1],'uri':m[2],'status':int(m[3]),'bytes':int(m[4])}
        ipaddress.ip_address(result['ip'])
        if not 100<=result['status']<=599 or result['bytes']<0:return None
        if 'requestBytes' in result and result['requestBytes']<0:return None
        if 'latencyMs' in result and (not math.isfinite(result['latencyMs']) or result['latencyMs']<0):return None
        # Queries may contain credentials and have unbounded cardinality.
        value=result['uri'].split('?',1)[0]
        if value.startswith(('http://','https://')):
            import urllib.parse
            value=urllib.parse.urlsplit(value).path or '/'
        result['uri']=value[:512]
        return result
    except (ValueError,TypeError,OverflowError):return None

def traffic_file(rt, entry, cursor, budget):
    name=entry['path'];approved=entry.get('realPath')
    if not approved or not name.startswith('/') or rt.realpath(name)!=approved:raise ValueError('log path left its registered scope')
    if rt.container:
        # Keep fd 3 open across stat and read so rename rotation cannot switch the
        # file midway. All untrusted values are positional shell parameters.
        def read(offset,count):
            command='exec 3< "$1" || exit; test -f /proc/self/fd/3 || exit 2; test "$(readlink -f /proc/self/fd/3)" = "$4" || exit 3; stat -Lc "%d:%i" /proc/self/fd/3; tail -c +"$2" /proc/self/fd/3 | head -c "$3" | base64'
            output=rt.cmd(['sh','-c',command,'traffic',name,str(offset+1),str(count),approved])[0]
            actual,body=output.split('\n',1)
            if actual.strip()!=identity:raise ValueError('log rotated during sampling; retry on next collection')
            return base64.b64decode(body)
        meta=rt.cmd(['stat','-Lc','%d:%i:%s','--',name])[0].strip().split(':')
        if rt.cmd(['test','-f',name],check=False)[2]!=0:raise ValueError('log is not a regular file')
        identity=':'.join(meta[:2]);size=int(meta[2]);fd=None
    else:
        import stat
        parent=os.open('/',os.O_RDONLY|os.O_DIRECTORY)
        try:
            components=pathlib.PurePosixPath(approved).parts[1:]
            for component in components[:-1]:
                child=os.open(component,os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW,dir_fd=parent);os.close(parent);parent=child
            fd=os.open(components[-1],os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK,dir_fd=parent)
        finally:os.close(parent)
        st=os.fstat(fd)
        if not stat.S_ISREG(st.st_mode):os.close(fd);raise ValueError('log is not a regular file')
        identity=str(st.st_dev)+':'+str(st.st_ino);size=st.st_size
        def read(offset,count):return os.pread(fd,count,offset)
    try:
        previous=cursor or {};offset=int(previous.get('offset',0));reset=False
        if offset<0:raise ValueError('invalid traffic cursor')
        if not previous:
            offset=size
            return b'',dict(identity=identity,offset=offset,anchor=sha(read(max(0,offset-64),min(64,offset))),capabilities={}),False,True
        if previous.get('identity')!=identity or size<offset or (offset and previous.get('anchor')!=sha(read(max(0,offset-64),min(64,offset)))):
            offset=0;reset=True
        limited=size-offset>budget
        data=read(offset,min(budget,max(0,size-offset)));read_bytes=len(data)
        # Commit only complete lines; incomplete writes are revisited next tick.
        last=data.rfind(b'\n')
        if last>=0:data=data[:last+1]
        elif len(data)==budget:
            # An overlong line cannot pin the cursor forever; disclose the gap.
            return b'',dict(identity=identity,offset=offset+len(data),anchor=sha(data[-64:]),capabilities=previous.get('capabilities',{}),gap=True,readBytes=read_bytes),True,False
        else:data=b''
        offset+=len(data)
        nextcursor=dict(identity=identity,offset=offset,anchor=sha(read(max(0,offset-64),min(64,offset))),capabilities=previous.get('capabilities',{}),gap=limited,readBytes=read_bytes)
        return data,nextcursor,reset or offset<size,False
    finally:
        if fd is not None:os.close(fd)

def traffic_collect(rt,req):
    cursors=req.get('trafficCursors') or {};result={'cursors':{},'requests':0,'responseBytes':0,'requestBytes':0,'requestByteCount':0,'latencyTotalMs':0,'latencyCount':0,'errors':0,'unparsed':0,'scannedBytes':0,'supportedLogs':0,'rateValid':True,'partial':False,'initialized':False,'statusCodes':{},'ips':{},'uris':{},'upstreams':{},'messages':[]}
    logs=[x for x in req.get('logs',[]) if x.get('kind')=='access'];budget=4*1024*1024
    dedicated=[x for x in logs if x.get('format','').startswith('autoops_traffic_')]
    if dedicated:
        logs=dedicated;result['partial']=True;result['messages'].append('Dedicated traffic logs selected; byte and route metrics cover only sites with traffic logging enabled, ordinary access logs are excluded to prevent duplicate counting')
    if len(logs)>8:result['partial']=True;result['messages'].append('Only the first 8 registered access logs were scanned')
    result['cursors']={x['id']:cursors[x['id']] for x in logs[:8] if x['id'] in cursors}
    def add(group,key,size):
        values=result[group]
        if key not in values and len(values)>=1000:result['partial']=True;return
        v=values.setdefault(key,{'requests':0,'responseBytes':0});v['requests']+=1;v['responseBytes']+=size
    for entry in logs[:8]:
        ident=entry['id']
        try:
            if budget<=0:
                result['partial']=True;result['rateValid']=False;result['cursors'].pop(ident,None);result['messages'].append('Access-log scan reached 4 MiB limit');continue
            data,cursor,partial,initial=traffic_file(rt,entry,cursors.get(ident),min(budget,2*1024*1024));read_bytes=cursor.pop('readBytes',len(data));budget-=read_bytes
            result['partial']|=partial;result['initialized']|=initial;result['scannedBytes']+=read_bytes
            parsed_before=result['requests'];unparsed_before=result['unparsed']
            for line in data.decode(errors='replace').splitlines():
                item=traffic_parse(line) if len(line)<=65536 else None
                if not item:result['unparsed']+=1;continue
                caps=cursor['capabilities'];caps['accessLog']=True
                result['requests']+=1;result['responseBytes']+=item['bytes'];result['errors']+=int(item['status']>=500)
                key=str(item['status']);result['statusCodes'][key]=result['statusCodes'].get(key,0)+1
                add('ips',item['ip'],item['bytes']);add('uris',item['uri'],item['bytes'])
                if 'requestBytes' in item:result['requestBytes']+=item['requestBytes'];result['requestByteCount']+=1;caps['requestBytes']=True
                if 'latencyMs' in item:result['latencyTotalMs']+=item['latencyMs'];result['latencyCount']+=1;caps['latency']=True
                if 'upstream' in item:add('upstreams',item['upstream'],item['bytes']);caps['upstream']=True
            if result['requests']==parsed_before and result['unparsed']>unparsed_before:
                cursor['capabilities']={};result['rateValid']=False
            if cursor.get('gap'):
                result['cursors'].pop(ident,None);result['rateValid']=False;result['messages'].append('Log scan was incomplete; its cursor will be initialized again to avoid reporting backlog as current traffic')
            else:result['cursors'][ident]=cursor
            if cursor['capabilities'].get('accessLog'):result['supportedLogs']+=1
        except (OSError,ValueError,TimeoutError,KeyError) as e:
            result['partial']=True;result['rateValid']=False;result['messages'].append(str(e)[:300]);result['cursors'].pop(ident,None)
            result['messages'].append('Failed log cursor cleared; recovery starts a new sampling baseline')
    if (result['requestByteCount'] and result['requestByteCount']<result['requests']) or (result['latencyCount'] and result['latencyCount']<result['requests']):result['partial']=True
    if result['unparsed']:result['partial']=True;result['messages'].append('Some access-log records use an unsupported format')
    if result['initialized']:result['messages'].append('Log cursor initialized; only new requests are counted')
    if not logs:result['messages'].append('No registered access-log file; container stdout and custom formats require a dedicated supported access log')
    return result

def execute(req):
    action=req['action']
    if action=='source-discovery':return source_discovery(req['spec'])
    if action.startswith('vip-'):return vip_execute(req)
    if action=='provision-check':return provision_preflight(req['spec'])
    if action=='provision':return provision(req)
    rt=Runtime(req['spec'])
    if action=='inspect':return {'identity':rt.identity,'state':'running' if rt.running else 'stopped'}
    if action=='discover':return rt.discover()
    if action=='health':return rt.health()
    if action=='tls':return {'certificates':verify_tls(req.get('tls',[]))}
    if action=='traffic':
        if rt.container and req.get('identity')!=rt.identity:raise ValueError('container identity changed during traffic collection')
        return traffic_collect(rt,req)
    if action=='metrics':
        url=rt.s.get('statusUrl')
        if not url:return {'supported':False}
        with urllib.request.build_opener(urllib.request.ProxyHandler({})).open(url,timeout=5) as response:text=response.read(16384).decode()
        active=re.search(r'Active connections:\s*(\d+)',text);counts=re.search(r'\n\s*(\d+)\s+(\d+)\s+(\d+)\s*\n',text);states=re.search(r'Reading:\s*(\d+)\s+Writing:\s*(\d+)\s+Waiting:\s*(\d+)',text)
        if not active or not counts or not states:raise ValueError('endpoint is not an Nginx stub_status response')
        return {'supported':True,'metrics':{'connections_active':int(active[1]),'connections_accepted':int(counts[1]),'connections_handled':int(counts[2]),'requests_total':int(counts[3]),'connections_reading':int(states[1]),'connections_writing':int(states[2]),'connections_waiting':int(states[3])}}
    if action in ('challenge-present','challenge-cleanup'):
        f=req['files'][0]; rt.guard_path(f['path'])
        if '/.well-known/acme-challenge/' not in f['path'] or not re.fullmatch(r'[A-Za-z0-9_-]+',f['path'].rsplit('/',1)[1]):raise ValueError('invalid challenge path')
        if action=='challenge-present':rt.write(f['path'],f['content'])
        elif rt.exists(f['path']) and rt.read(f['path'])==f['content']:rt.cmd(['rm','-f','--',f['path']])
        return {'state':'succeeded'}
    if action=='logs':
        observed={'logs':req['logs']} if req.get('logs') else rt.discover(); ident=req['logId'];registered={x['id']:x for x in observed['logs']};known={x['id']:x['path'] for x in observed['logs']}
        if ident not in known:raise ValueError('unknown log identifier')
        lines=max(1,min(int(req.get('lines',500)),10000));offset=max(0,int(req.get('offset',0)))
        if offset>=100000:raise ValueError('log scan limit reached')
        lines=min(lines,100000-offset);window=offset+lines;name=known[ident]
        if ident=='stdout':
            stdout,stderr,_=run(['docker','logs','--timestamps','--tail',str(window),rt.container]);text='\n'.join(sorted((stdout+'\n'+stderr).splitlines()))
        else:
            rotation=req.get('rotation','')
            if rotation and not re.fullmatch(r'\.[0-9]+(?:\.gz)?',rotation):raise ValueError('invalid rotation')
            name+=rotation
            if not rt.exists(name):raise ValueError('log file no longer exists')
            if rt.cmd(['test','-f',name],check=False)[2]!=0:raise ValueError('log is not a regular file')
            real=rt.realpath(name);approved=registered[ident].get('realPath',rt.realpath(known[ident]))
            if (not rotation and real!=approved) or (rotation and os.path.dirname(real)!=os.path.dirname(approved)):raise ValueError('log path left its registered scope')
            if name.endswith('.gz'):
                size=int(rt.cmd(['stat','-c','%s','--',name])[0]);
                if size>16*1024*1024:raise ValueError('compressed log scan exceeds 16 MiB')
                text=rt.cmd(['gzip','-cd','--',name])[0];text='\n'.join(text.splitlines()[-window:])
            else:text=rt.cmd(['tail','-n',str(window),'--',name])[0]
        keyword=req.get('keyword','');result=[];scanned=text.splitlines();more=len(scanned)>=window and window<100000
        sliced=scanned[max(0,len(scanned)-offset-lines):len(scanned)-offset if offset else None]
        result=[line for line in sliced if log_matches(line,req)]
        return {'text':'\n'.join(result),'path':name,'lines':len(result),'at':time.time(),'window':lines,'cursor':sha(text),'offset':offset,'nextOffset':offset+lines,'hasMore':more,'scanLimit':100000}
    key=sha(str(req['spec']['hostId'])+':'+req['spec']['runtime']+':'+req['spec'].get('container','')+':'+req['spec']['configPath'])
    root='/var/tmp/autoops-web/'+key
    os.makedirs(root,mode=0o700,exist_ok=True)
    if os.path.islink(root):raise ValueError('invalid execution root')
    with open(root+'/lock','a') as lock:
        fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
        journal=root+'/'+str(int(req.get('operationId',0)))+'.json'
        if action=='recover':
            return recover_journal(rt,journal,json.loads(pathlib.Path(journal).read_text()),req.get('identity'))
        if action=='reconcile':
            if not os.path.exists(journal):return {'state':'not-started','observed':rt.discover()}
            record=json.loads(pathlib.Path(journal).read_text());record.pop('backup',None);record.pop('original',None);record.pop('candidate',None)
            if record.get('state') in ('prepared','written'):return {'state':'recoverable'}
            if record.get('action')=='stop':
                stopped=not rt.running if rt.container else not rt.process(required=False)['running']
                return record if stopped else {'state':'conflict'}
            live=rt.discover()
            if req.get('identity') and req['identity']!=live['identity']:return {'state':'identity-conflict','observed':live}
            if record.get('observed',{}).get('digest')!=live['digest']:return {'state':'conflict','observed':live}
            if record.get('state') in ('succeeded','rolled-back'):
                rt.health()
                if record['state']=='succeeded':verify_tls(req.get('tls',[]))
            record['observed']=live;return record
        if action=='provision':
            if rt.s['runtime']=='docker':raise ValueError('provision Docker using host provision entry')
        if action=='start' and rt.container and not rt.running:
            if req.get('identity')!=rt.identity:raise ValueError('instance identity changed')
            save_journal(journal,{'state':'running','action':'start'})
            run(['docker','start',rt.container]);time.sleep(.3);rt.health()
            record={'state':'succeeded','observed':rt.discover()};save_journal(journal,record);return record
        observed=rt.discover()
        if req.get('identity') and req['identity']!=observed['identity']:raise ValueError('instance identity changed')
        if req.get('baseline') and req['baseline']!=observed['digest']:raise ValueError('configuration drift detected')
        restoring=action=='restore'
        if restoring:
            previous=json.loads(pathlib.Path(journal).read_text())
            if previous.get('state')!='succeeded' or previous.get('observed',{}).get('digest')!=observed['digest']:raise ValueError('restore evidence does not match current state')
            req['files']=[dict(f,deleted=not f['exists']) for f in previous['backup']];action='rollback'
        if action=='validate':
            files=merge(observed['files'],req['files']);
            for f in req['files']:rt.guard_path(f['path'])
            return staging(rt,files,req['files'])
        if action in ('apply','rollback'):
            rt.control_guard('reload')
            files=req['files'];before=[]
            for f in files:
                rt.guard_path(f['path'])
                exists=rt.exists(f['path']);before.append({'path':f['path'],'content':rt.read(f['path']) if exists else '', 'mode':rt.mode(f['path']), 'exists':exists})
            candidate=merge(observed['files'],files)
            staging(rt,candidate,files)
            if rt.discover()['digest']!=observed['digest']:raise ValueError('configuration changed during validation')
            record={'state':'prepared','backup':before,'digest':observed['digest'],'original':observed,'candidate':{f['path']:None if f.get('deleted') else sha(f['content']) for f in files}}
            save_journal(journal,record)
            changed=[];release=release_paths(rt,files,str(req.get('operationId',0))+('-restore' if restoring else ''))
            if release:record['release']={'previous':release[1],'candidate':release[2]};save_journal(journal,record)
            try:
                for f in files:
                    target=release[2]+f['path'][len(release[0]):] if release else rt.guard_path(f['path'])
                    if f.get('deleted'):rt.cmd(['rm','-f','--',target])
                    else:rt.write(target,f['content'],f.get('mode') or rt.mode(target))
                    changed.append(f['path'])
                if release:switch_release(rt,release[0],release[2])
                record['state']='written';save_journal(journal,record)
                rt.nginx('-t');rt.nginx('-s','reload');time.sleep(.3);rt.health();verify_tls(req.get('tls',[]))
                for probe in req.get('healthChecks',[]):
                    original=rt.s;rt.s=dict(original,healthUrl=probe['url'],healthHost=probe.get('host',''),expectedStatus=probe['expectedStatus'])
                    try:rt.health()
                    finally:rt.s=original
                record['state']='rolled-back' if restoring else 'succeeded';record['observed']=rt.discover();save_journal(journal,record)
                return {k:v for k,v in record.items() if k not in ('backup','original','candidate')}
            except Exception as ex:
                if release:switch_release(rt,release[0],release[1])
                for f in ([] if release else reversed(before)):
                    if f['path'] not in changed:continue
                    if f['exists']:rt.write(rt.guard_path(f['path']),f['content'],f['mode'])
                    else:rt.cmd(['rm','-f','--',f['path']])
                rt.nginx('-t');rt.nginx('-s','reload');time.sleep(.3);rt.health()
                record['state']='restore-failed' if restoring else 'rolled-back';record['observed']=rt.discover();save_journal(journal,record)
                return {'state':record['state'],'message':str(ex),'observed':rt.discover()}
        if action in ('reload','start','stop','restart'):
            rt.control_guard(action)
            save_journal(journal,{'state':'running'})
            if action=='reload':rt.nginx('-t');rt.nginx('-s','reload')
            elif action=='stop':
                process=rt.process();rt.nginx('-s','quit')
                for _ in range(100):
                    if rt.container:
                        if json.loads(run(['docker','inspect',rt.container])[0])[0]['State']['Running']==False:break
                    elif not rt.exists('/proc/'+process['pid']+'/cmdline') or not rt.read('/proc/'+process['pid']+'/cmdline'):break
                    time.sleep(.1)
                else:raise ValueError('graceful stop is still pending')
            elif rt.container:run(['docker',action,rt.container])
            elif rt.s.get('service'):run(['systemctl',action,rt.s['service']])
            elif action=='start':rt.cmd([rt.s['binary'],'-p',rt.s['prefix'],'-c',rt.s['configPath']])
            else:raise ValueError('restart requires systemd service or container')
            if action!='stop':rt.health()
            record={'state':'succeeded','observed':observed if action=='stop' else rt.discover(),'action':action};save_journal(journal,record);return record
        raise ValueError('unsupported action')

def provision_preflight(spec):
    config=spec['configPath'];port=int(spec.get('port',80))
    if not re.fullmatch(r'/etc/autoops-web/[A-Za-z0-9_-]+/current/nginx.conf',config):raise ValueError('new instances require /etc/autoops-web/<name>/current/nginx.conf')
    base=config[:-len('/current/nginx.conf')]
    if os.path.lexists(base):raise ValueError('instance directory exists; inspect before retrying')
    if spec.get('prefix')!=base+'/current':raise ValueError('new instances require the current directory as runtime prefix')
    if spec['binary']!='/usr/sbin/nginx':raise ValueError('new instances require /usr/sbin/nginx')
    if not 1<=port<=65535:raise ValueError('invalid port')
    with socket.socket() as sock:sock.bind(('0.0.0.0',port))
    if shutil.disk_usage('/etc').free<256*1024*1024:raise ValueError('insufficient free disk space')
    if spec['runtime']=='host':
        info=dict(line.split('=',1) for line in pathlib.Path('/etc/os-release').read_text().splitlines() if '=' in line)
        if (info.get('ID','').strip('"'),info.get('VERSION_ID','').strip('"')) not in [('ubuntu','22.04'),('ubuntu','24.04'),('debian','12')]:raise ValueError('unsupported deployment OS')
        package=spec.get('packageVersion','');available=[line.split('|')[1].strip() for line in run(['apt-cache','madison','nginx'])[0].splitlines() if '|' in line]
        if package not in available:raise ValueError('pinned package version unavailable')
        if not re.fullmatch(r'autoops-web-[A-Za-z0-9_-]+',spec.get('service','')):raise ValueError('new service name must start with autoops-web-')
        if os.path.exists('/etc/systemd/system/'+spec['service']+'.service'):raise ValueError('systemd service already exists')
    else:
        if not re.fullmatch(r'(?:[A-Za-z0-9][A-Za-z0-9._/:-]*@)?sha256:[a-f0-9]{64}',spec.get('image','')):raise ValueError('pin the Docker image digest')
        if run(['docker','inspect',spec['container']],check=False)[2]==0:raise ValueError('container already exists')
        run(['docker','version','--format','{{.Server.Version}}'])
    return {'checked':True,'port':port,'configPath':config}

def provision(req):
    spec=req['spec'];provision_preflight(spec);config=spec['configPath'];port=int(spec.get('port',80))
    if not config.startswith('/etc/autoops-web/') or not config.endswith('/current/nginx.conf') or os.path.normpath(config)!=config:
        raise ValueError('new instances require /etc/autoops-web/<name>/current/nginx.conf')
    base=config[:-len('/current/nginx.conf')]
    if not re.fullmatch(r'/etc/autoops-web/[A-Za-z0-9_-]+',base):raise ValueError('invalid instance directory')
    if os.path.lexists(base):raise ValueError('instance directory already exists; discover it before retrying')
    if not 1<=port<=65535:raise ValueError('invalid port')
    if spec['runtime']=='host':
        info=dict(line.split('=',1) for line in pathlib.Path('/etc/os-release').read_text().splitlines() if '=' in line)
        distro=info.get('ID','').strip('"'); version=info.get('VERSION_ID','').strip('"')
        if (distro,version) not in [('ubuntu','22.04'),('ubuntu','24.04'),('debian','12')]:raise ValueError('unsupported deployment OS')
        package=spec.get('packageVersion','')
        if not re.fullmatch(r'[0-9][A-Za-z0-9.+:~_-]+',package):raise ValueError('pin the nginx package version')
        if package not in run(['apt-cache','madison','nginx'])[0]:raise ValueError('pinned version unavailable in configured package repository')
        installed=run(['dpkg-query','-W','-f=${Version}','nginx'],check=False)
        if installed[2]==0 and installed[0]!=package:raise ValueError('existing nginx package differs; automatic upgrades are not permitted')
        service=spec.get('service','')
        if not re.fullmatch(r'autoops-web-[A-Za-z0-9_-]+',service):raise ValueError('new service name must start with autoops-web-')
        if installed[2]!=0:
            policy='/usr/sbin/policy-rc.d';created=False
            try:
                if not os.path.exists(policy):
                    with open(policy,'x') as f:f.write('#!/bin/sh\nexit 101\n')
                    os.chmod(policy,0o755);created=True
                run(['apt-get','install','-y','--no-install-recommends','nginx='+package],timeout=180)
                # The package enables its default unit even when policy-rc.d
                # prevented startup. Only the newly installed package's unused
                # default service is disabled; existing instances are untouched.
                run(['systemctl','disable','nginx.service'])
            finally:
                if created:os.unlink(policy)
    else:
        image=spec.get('image','')
        if not re.fullmatch(r'(?:[A-Za-z0-9][A-Za-z0-9._/:-]*@)?sha256:[a-f0-9]{64}',image):raise ValueError('pin the Docker image digest')
        if run(['docker','inspect',spec['container']],check=False)[2]==0:raise ValueError('container already exists')
        if run(['docker','image','inspect',image],check=False)[2]!=0:run(['docker','pull',image],timeout=180)
    release=base+'/releases/bootstrap';os.makedirs(release,mode=0o755)
    pid=base+'/nginx.pid' if spec['runtime']=='host' else '/tmp/autoops-nginx.pid'
    text='pid '+pid+';\nevents { worker_connections 1024; }\nhttp { access_log '+base+'/access.log; error_log '+base+'/error.log; include '+base+'/current/conf.d/*.conf; server { listen '+str(port)+'; location = /__autoops_health { return 200 "ok"; } } }\n'
    pathlib.Path(release+'/nginx.conf').write_text(text);os.makedirs(release+'/conf.d');pathlib.Path(release+'/conf.d/placeholder.conf').write_text('# AutoOps sites\n')
    os.symlink('releases/bootstrap',base+'/current')
    if spec['runtime']=='docker':
        run(['docker','run','-d','--name',spec['container'],'--restart','unless-stopped','-p',str(port)+':'+str(port),'-v',base+':'+base,spec['image'],spec['binary'],'-p',spec['prefix'],'-c',config,'-g','daemon off;'])
    else:
        command=spec['binary']+' -p '+spec['prefix']+' -c '+config
        unit='[Unit]\nDescription=AutoOps Web instance\nAfter=network.target\n[Service]\nType=forking\nPIDFile='+pid+'\nExecStartPre='+command+' -t\nExecStart='+command+'\nExecReload='+command+' -s reload\nExecStop='+command+' -s quit\n[Install]\nWantedBy=multi-user.target\n'
        unitpath='/etc/systemd/system/'+spec['service']+'.service'
        if os.path.exists(unitpath):raise ValueError('service unit already exists')
        pathlib.Path(unitpath).write_text(unit);run(['systemctl','daemon-reload']);run(['systemctl','enable','--now',spec['service']])
    rt=Runtime(spec);rt.health();return {'state':'succeeded','observed':rt.discover()}

def merge(existing,changes):
    files={f['path']:f.copy() for f in existing}
    for f in changes:
        if f.get('deleted'):files.pop(f['path'],None)
        else:files[f['path']]=f.copy()
    return list(files.values())

def log_matches(line, req):
    """Shared filtering for historical windows and live events (combined/JSON/raw)."""
    if req.get('keyword') and req['keyword'] not in line:return False
    parsed=traffic_parse(line)
    if req.get('status') and (not parsed or str(parsed['status'])!=str(req['status'])):return False
    if req.get('ip') and (not parsed or parsed['ip']!=req['ip']):return False
    if req.get('uri') and (not parsed or req['uri'] not in parsed['uri']):return False
    if req.get('from') or req.get('to'):
        def stamp(value):return datetime.datetime.fromisoformat(value.replace('Z','+00:00')).timestamp()
        try:
            if line.startswith('{'):
                obj=json.loads(line);when=stamp(obj.get('time_iso8601',obj.get('timestamp','')))
            else:
                match=re.search(r'\[([0-9]{2}/[A-Za-z]{3}/[0-9]{4}:[^]]+)\]',line)
                when=datetime.datetime.strptime(match[1],'%d/%b/%Y:%H:%M:%S %z').timestamp() if match else stamp(line.split(' ',1)[0])
            if req.get('from') and when<stamp(req['from']):return False
            if req.get('to') and when>stamp(req['to']):return False
        except (ValueError,TypeError,KeyError):return False
    return True

def log_open(root_fd, approved):
    import stat
    components=pathlib.PurePosixPath(approved).parts[1:]
    if not components or any(x in ('.','..') for x in components):raise ValueError('invalid registered log path')
    parent=os.dup(root_fd)
    try:
        for component in components[:-1]:
            child=os.open(component,os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW,dir_fd=parent);os.close(parent);parent=child
        fd=os.open(components[-1],os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK,dir_fd=parent)
    finally:os.close(parent)
    if not stat.S_ISREG(os.fstat(fd).st_mode):os.close(fd);raise ValueError('log is not a regular file')
    return fd

def log_follow(req, emit, stop=None):
    """One persistent SSH process; bounded incremental reads, no repeated log queries."""
    rt=Runtime(req['spec']);entry=next((x for x in req.get('logs',[]) if x['id']==req.get('logId')),None)
    if not entry:raise ValueError('unknown log identifier')
    if req.get('rotation') or req.get('from') or req.get('to'):raise ValueError('live logs do not accept historical file or time ranges')
    if rt.container and (not req.get('identity') or rt.identity!=req['identity']):raise ValueError('container identity changed')
    if not rt.container:
        entry_path=rt.s['configPath'] if re.fullmatch(r'/etc/autoops-web/[A-Za-z0-9_-]+/current/nginx.conf',rt.s['configPath']) else rt.realpath(rt.s['configPath'])
        identity=sha(rt.realpath(rt.s['binary'])+':'+entry_path)
        if identity!=req.get('identity'):raise ValueError('instance identity changed')
    lines=max(1,min(int(req.get('lines',500)),10000));deadline=time.monotonic()+20*60
    def active():return time.monotonic()<deadline and not (stop and stop())
    pending=b''
    def consume(data, final=False):
        nonlocal pending
        pending+=data
        if len(pending)>2*1024*1024:raise ValueError('log line exceeds live stream limit')
        chunks=pending.split(b'\n');pending=chunks.pop()
        if final and pending:chunks.append(pending);pending=b''
        batch=[];size=0
        for raw in chunks:
            if len(raw)>65536:raise ValueError('log line exceeds 64 KiB; use historical download')
            line=raw.decode(errors='replace').rstrip('\r')
            if not log_matches(line,req):continue
            if size+len(line)>32768 and batch:emit('log',{'text':'\n'.join(batch)+'\n','at':time.time()});batch=[];size=0
            batch.append(line);size+=len(line)
        if batch:emit('log',{'text':'\n'.join(batch)+'\n','at':time.time()})
    def container_current(expected_pid=None):
        current=json.loads(run(['docker','inspect',req['spec']['container']])[0])[0]
        if current['Id']!=rt.identity or not current['State']['Running'] or (expected_pid and current['State']['Pid']!=expected_pid):raise ValueError('container identity or process changed; discover and reconnect')
        return current['State']['Pid']
    if entry['id']=='stdout':
        if not rt.container:raise ValueError('container log requires Docker runtime')
        container_current()
        p=subprocess.Popen(['docker','logs','--follow','--timestamps','--tail',str(lines),rt.container],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,start_new_session=True)
        sel=selectors.DefaultSelector();sel.register(p.stdout,selectors.EVENT_READ);last_check=time.monotonic()
        emit('ready',{'path':'stdout/stderr','at':time.time()})
        try:
            while active():
                for key,_ in sel.select(.25):
                    data=os.read(key.fileobj.fileno(),32768)
                    if not data:
                        consume(b'',True);return
                    consume(data)
                if time.monotonic()-last_check>=5:container_current();emit('heartbeat',{'at':time.time()});last_check=time.monotonic()
        finally:
            import signal
            sel.close()
            with contextlib.suppress(ProcessLookupError):os.killpg(p.pid,signal.SIGTERM)
            try:p.wait(timeout=2)
            except subprocess.TimeoutExpired:
                with contextlib.suppress(ProcessLookupError):os.killpg(p.pid,signal.SIGKILL)
                p.wait()
            p.stdout.close()
        return
    approved=entry.get('realPath');name=entry['path']
    if not approved or not approved.startswith('/') or rt.realpath(name)!=approved:raise ValueError('log path left its registered scope')
    pid=container_current() if rt.container else None
    root=os.open('/proc/'+str(pid)+'/root' if pid else '/',os.O_RDONLY|os.O_DIRECTORY)
    fd=None
    try:
        fd=log_open(root,approved);st=os.fstat(fd);offset=st.st_size
        start=max(0,offset-1024*1024);data=os.pread(fd,offset-start,start)
        if start and b'\n' in data:data=data.split(b'\n',1)[1]
        # Tail complete lines, retaining partial writes until the next newline.
        tail=data.splitlines(keepends=True)[-lines:];data=b''.join(tail)
        emit('ready',{'path':name,'at':time.time(),'bounded':start>0})
        consume(data);anchor=os.pread(fd,min(offset,64),max(0,offset-64));last_check=time.monotonic()
        missing_since=None
        while active():
            time.sleep(.25)
            try:newfd=log_open(root,approved)
            except FileNotFoundError:
                if missing_since is None:missing_since=time.monotonic()
                if time.monotonic()-missing_since>30:raise ValueError('log file remains unavailable after rotation')
                continue
            missing_since=None;newst=os.fstat(newfd)
            rotated=(st.st_dev,st.st_ino)!=(newst.st_dev,newst.st_ino)
            if rotated:
                # Drain the renamed inode before attaching to the new registered path.
                remaining=max(0,os.fstat(fd).st_size-offset)
                if remaining>2*1024*1024:emit('notice',{'message':'轮转期间积压超过读取上限，部分日志未展示'})
                consume(os.pread(fd,min(remaining,2*1024*1024),offset),True)
                os.close(fd);fd=newfd;newfd=None;st=newst;offset=0;anchor=b'';pending=b''
                emit('reset',{'reason':'rotated','at':time.time()})
            elif newst.st_size<offset or os.pread(fd,min(offset,64),max(0,offset-64))!=anchor:
                offset=0;anchor=b'';pending=b'';emit('reset',{'reason':'truncated','at':time.time()})
            if newfd is not None:os.close(newfd)
            size=os.fstat(fd).st_size
            if size-offset>2*1024*1024:raise ValueError('live log backlog exceeds 2 MiB; narrow the query and reconnect')
            data=os.pread(fd,min(65536,max(0,size-offset)),offset);offset+=len(data)
            anchor=os.pread(fd,min(offset,64),max(0,offset-64));consume(data)
            if time.monotonic()-last_check>=5:
                if pid:container_current(pid)
                emit('heartbeat',{'at':time.time()})
                # Registered symlinks may survive rotation, but cannot move to a different scope.
                if rt.realpath(name)!=approved:raise ValueError('log path left its registered scope')
                last_check=time.monotonic()
    finally:
        if fd is not None:os.close(fd)
        os.close(root)

if __name__=='__main__':
    request={}
    def event(kind,data):
        print(json.dumps({'event':kind,'data':data}),flush=True)
    try:
        request=json.loads(sys.stdin.buffer.read(LIMIT+1))
        if request.get('action')=='log-follow':
            log_follow(request,event)
            event('end',{'message':'日志连接已结束，可重新连接继续追踪'})
        else:
            result=execute(request)
            print(json.dumps({'ok':True,'data':result}))
    except Exception as error:
        if request.get('action')=='log-follow':event('error',{'message':str(error)[-3000:]})
        else:print(json.dumps({'ok':False,'error':str(error)[-3000:]}))
        sys.exit(1)
