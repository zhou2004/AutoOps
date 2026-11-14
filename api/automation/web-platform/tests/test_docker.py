"""Optional Docker integration. Uses an already present image; never pulls or touches existing containers.
AUTOOPS_WEB_DOCKER_TEST=1 python3 -m unittest discover -s api/automation/web-platform/tests -v
"""
import json,os,pathlib,shutil,socket,tempfile,time,unittest,urllib.request,uuid
from test_runner import runner

@unittest.skipUnless(os.getenv('AUTOOPS_WEB_DOCKER_TEST')=='1','set AUTOOPS_WEB_DOCKER_TEST=1 for Docker integration')
class DockerIntegration(unittest.TestCase):
 def setUp(self):
  self.name='autoops-web-test-'+uuid.uuid4().hex[:12];self.tmp=tempfile.TemporaryDirectory(prefix=self.name);self.root=pathlib.Path(self.tmp.name);self.container=None
  image=runner.run(['docker','image','inspect','nginx:stable-alpine','--format','{{.Id}}'])[0].strip()
  (self.root/'conf.d').mkdir();(self.root/'nginx.conf').write_text('pid /tmp/nginx.pid;\nevents {}\nhttp { access_log /dev/stdout; error_log /dev/stderr; include /etc/nginx/conf.d/*.conf; }')
  (self.root/'conf.d/site.conf').write_text('server {listen 8080; location / {return 200 "old";}}')
  with socket.socket() as s:s.bind(('127.0.0.1',0));hostport=s.getsockname()[1]
  self.container=runner.run(['docker','run','-d','--pull','never','--name',self.name,'-p','127.0.0.1:'+str(hostport)+':8080','-v',str(self.root)+':/etc/nginx',image,'nginx','-c','/etc/nginx/nginx.conf','-g','daemon off;'])[0].strip()
  try:
   raw=json.loads(runner.run(['docker','inspect',self.container])[0])[0];port=raw['NetworkSettings']['Ports']['8080/tcp'][0]['HostPort']
   self.source={'hostId':os.getpid(),'runtime':'docker','container':self.name,'binary':'/usr/sbin/nginx','configPath':'/etc/nginx/nginx.conf','prefix':'/etc/nginx','managedPaths':['/etc/nginx'],'healthUrl':'http://127.0.0.1:'+port,'expectedStatus':200}
   self.rt=runner.Runtime(self.source);self.wait('old')
  except BaseException:self.tearDown();raise
 def tearDown(self):
  if self.container:runner.run(['docker','rm','-f',self.container])
  key=runner.sha(str(os.getpid())+':docker:'+self.name+':/etc/nginx/nginx.conf');shutil.rmtree('/var/tmp/autoops-web/'+key,ignore_errors=True);self.tmp.cleanup()
 def wait(self,expected):
  for _ in range(100):
   try:
    with urllib.request.build_opener(urllib.request.ProxyHandler({})).open(self.source['healthUrl'],timeout=.5) as r:
     if r.read().decode()==expected:return
   except Exception:pass
   time.sleep(.03)
  self.fail('container did not serve '+expected)
 def test_publish_restore_and_logs(self):
  obs=self.rt.discover();request={'action':'apply','spec':self.source,'operationId':1,'baseline':obs['digest'],'identity':obs['identity'],'files':[{'path':'/etc/nginx/conf.d/site.conf','content':'server { listen 8080; location / {return 200 "new";}}','mode':0o644}]}
  result=runner.execute(request);self.assertEqual(result['state'],'succeeded');self.wait('new')
  logs=runner.execute({'action':'logs','spec':self.source,'logId':'stdout','lines':100});self.assertIn('GET',logs['text'])
  request.update(action='restore',baseline=result['observed']['digest']);result=runner.execute(request);self.assertEqual(result['state'],'rolled-back');self.wait('old')
 def test_incremental_traffic_file_and_container_identity(self):
  path='/etc/nginx/nginx.conf';self.rt.write(path,self.rt.read(path).replace('access_log /dev/stdout;','access_log /etc/nginx/traffic.log;'))
  self.rt.nginx('-t');self.rt.nginx('-s','reload');time.sleep(.2)
  obs=self.rt.discover();request={'action':'traffic','spec':self.source,'identity':obs['identity'],'logs':obs['logs'],'trafficCursors':{}}
  with self.assertRaisesRegex(ValueError,'identity'):runner.execute(dict(request,identity='recreated-container'))
  baseline=runner.execute(request);self.assertTrue(baseline['initialized'])
  for _ in range(3):self.wait('old')
  request['trafficCursors']=baseline['cursors'];sample=runner.execute(request);self.assertGreaterEqual(sample['requests'],3);self.assertGreater(sample['responseBytes'],0)
  request['trafficCursors']=sample['cursors'];self.assertEqual(runner.execute(request)['requests'],0)
 def test_live_stdout_and_file_streams(self):
  import threading,queue
  obs=self.rt.discover();events=queue.Queue();stop=threading.Event();errors=[]
  def start(request):
   def work():
    try:runner.log_follow(request,lambda kind,data:events.put((kind,data)),stop.is_set)
    except Exception as e:errors.append(e)
   thread=threading.Thread(target=work);thread.start();return thread
  def wait(kind,text=None):
   deadline=time.monotonic()+5
   while time.monotonic()<deadline:
    try:k,d=events.get(timeout=.1)
    except queue.Empty:continue
    if k==kind and (text is None or text in d.get('text','')):return d
   self.fail('missing '+kind+' '+str(errors))
  request={'spec':self.source,'identity':obs['identity'],'logs':obs['logs'],'logId':'stdout','lines':1}
  thread=start(request)
  try:wait('ready');self.wait('old');wait('log','GET')
  finally:stop.set();thread.join(3);self.assertFalse(thread.is_alive())
  self.assertFalse(errors)
  (self.root/'stream.log').write_text('first\n');events=queue.Queue();stop.clear()
  request.update(logId='file',logs=[{'id':'file','path':'/etc/nginx/stream.log','realPath':'/etc/nginx/stream.log'}])
  thread=start(request)
  try:
   wait('ready');wait('log','first')
   with (self.root/'stream.log').open('a') as f:f.write('next\n')
   wait('log','next')
   (self.root/'stream.log').rename(self.root/'stream.log.1');(self.root/'stream.log').write_text('rotated\n');wait('reset');wait('log','rotated')
  finally:stop.set();thread.join(3);self.assertFalse(thread.is_alive())
  self.assertFalse(errors)

 def test_stopped_container_start_checks_identity(self):
  obs=self.rt.discover();runner.run(['docker','stop','--time','3',self.container]);identity=runner.execute({'action':'inspect','spec':self.source});self.assertEqual(identity['state'],'stopped')
  result=runner.execute({'action':'start','spec':self.source,'identity':obs['identity'],'operationId':2});self.assertEqual(result['state'],'succeeded');self.wait('old')

 @unittest.skipUnless(os.geteuid()==0,'requires an isolated directory under /etc/autoops-web')
 def test_platform_provision_uses_immutable_release_pointer(self):
  name='test-'+uuid.uuid4().hex[:12];base=pathlib.Path('/etc/autoops-web')/name
  image=json.loads(runner.run(['docker','image','inspect','nginx:stable-alpine'])[0])[0]['Id']
  with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
  source={'hostId':os.getpid(),'runtime':'docker','container':name,'binary':'/usr/sbin/nginx','configPath':str(base/'current/nginx.conf'),'prefix':str(base/'current'),'managedPaths':[str(base/'current')],'healthUrl':'http://127.0.0.1:'+str(port)+'/__autoops_health','expectedStatus':200,'port':port,'image':image}
  try:
   runner.execute({'action':'provision-check','spec':source});result=runner.execute({'action':'provision','spec':source,'operationId':1});self.assertEqual(result['state'],'succeeded')
   rt=runner.Runtime(source);old=(base/'current').resolve();content=(old/'nginx.conf').read_text();obs=rt.discover()
   changed=content.replace('location = /__autoops_health','location /app { return 200 "app"; } location = /__autoops_health')
   result=runner.execute({'action':'apply','spec':source,'operationId':2,'identity':obs['identity'],'baseline':obs['digest'],'files':[{'path':source['configPath'],'content':changed,'mode':0o644}]})
   self.assertEqual(result['state'],'succeeded');self.assertNotEqual((base/'current').resolve(),old);self.assertEqual((old/'nginx.conf').read_text(),content)
   self.assertEqual(rt.discover()['identity'],obs['identity'])
   result=runner.execute({'action':'restore','spec':source,'operationId':2,'identity':obs['identity'],'baseline':result['observed']['digest']});self.assertEqual(result['state'],'rolled-back');self.assertEqual((base/'current/nginx.conf').read_text(),content)
  finally:
   runner.run(['docker','rm','-f',name],check=False);shutil.rmtree(base,ignore_errors=True)
   key=runner.sha(str(os.getpid())+':docker:'+name+':'+source['configPath']);shutil.rmtree('/var/tmp/autoops-web/'+key,ignore_errors=True)
