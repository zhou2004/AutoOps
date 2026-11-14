"""Persistent log stream tests against isolated files (no service changes)."""
import importlib.util,json,pathlib,tempfile,unittest,threading,queue,time,os,sys
spec=importlib.util.spec_from_file_location('follow_runner',pathlib.Path(__file__).resolve().parents[1]/'runner.py')
runner=importlib.util.module_from_spec(spec);spec.loader.exec_module(runner)
class LiveLogs(unittest.TestCase):
 def setUp(self):
  self.temp=tempfile.TemporaryDirectory();self.root=pathlib.Path(self.temp.name);self.file=self.root/'access.log';self.file.write_text('initial\n');config=self.root/'nginx.conf';config.write_text('')
  self.req={'spec':{'runtime':'host','binary':sys.executable,'configPath':str(config)},'identity':runner.sha(os.path.realpath(sys.executable)+':'+str(config)),'logs':[{'id':'a','path':str(self.file),'realPath':str(self.file)}],'logId':'a','lines':100}
  self.stop=threading.Event();self.events=queue.Queue();self.errors=[];self.thread=None
 def tearDown(self):
  self.stop.set()
  if self.thread:self.thread.join(3);self.assertFalse(self.thread.is_alive())
  self.temp.cleanup()
 def start(self):
  def run():
   try:runner.log_follow(self.req,lambda event,data:self.events.put((event,data)),self.stop.is_set)
   except Exception as e:self.errors.append(e)
  self.thread=threading.Thread(target=run);self.thread.start();self.wait('ready')
 def wait(self,kind,contains=None):
  deadline=time.monotonic()+4
  while time.monotonic()<deadline:
   try:event,data=self.events.get(timeout=.1)
   except queue.Empty:continue
   if event==kind and (contains is None or contains in json.dumps(data)):return data
  self.fail('event missing '+kind+' '+str(self.errors))
 def test_incremental_partial_rotation_and_copytruncate(self):
  self.start();self.assertEqual(self.wait('log')['text'],'initial\n')
  with self.file.open('a') as f:f.write('part')
  time.sleep(.35);self.assertTrue(self.events.empty())
  with self.file.open('a') as f:f.write('ial\n')
  self.assertEqual(self.wait('log')['text'],'partial\n')
  self.file.rename(self.root/'access.log.1');self.file.write_text('rotated\n')
  self.wait('reset','rotated');self.assertEqual(self.wait('log')['text'],'rotated\n')
  self.file.write_text('short\n');self.wait('reset','truncated');self.assertEqual(self.wait('log')['text'],'short\n')
  time.sleep(.35);self.assertTrue(self.events.empty());self.assertFalse(self.errors)
 def test_symlink_escape_stops_without_returning_target(self):
  self.start();self.wait('log');secret=self.root/'other';secret.write_text('private-must-not-escape\n');self.file.unlink();self.file.symlink_to(secret)
  self.thread.join(3);self.assertFalse(self.thread.is_alive());self.assertTrue(self.errors)
  while not self.events.empty():self.assertNotIn('private-must-not-escape',str(self.events.get()))
 def test_json_status_ip_uri_and_time_filters(self):
  row=json.dumps({'time_iso8601':'2026-11-13T15:00:00+00:00','remote_addr':'192.0.2.4','uri':'/api','status':429,'body_bytes_sent':10})
  self.assertTrue(runner.log_matches(row,{'status':'429','ip':'192.0.2.4','uri':'/api','from':'2026-11-13T14:00:00Z','to':'2026-11-13T16:00:00Z'}))
  for query in [{'status':'200'},{'ip':'192.0.2.5'},{'uri':'/private'},{'from':'2026-11-14T00:00:00Z'}]:self.assertFalse(runner.log_matches(row,query))
  self.assertTrue(runner.log_matches('arbitrary error',{'keyword':'error'}));self.assertFalse(runner.log_matches('arbitrary error',{'status':'500'}))
 def test_identity_unknown_identifier_and_history_rejected(self):
  for changes in [{'identity':'wrong'},{'logId':'../../etc/passwd'},{'rotation':'.1'},{'from':'2026-11-01T00:00:00Z'}]:
   with self.subTest(changes=changes),self.assertRaises(ValueError):runner.log_follow(dict(self.req,**changes),lambda *_:None,self.stop.is_set)
 def test_stream_bounds_long_lines(self):
  self.file.write_text('x'*70000+'\n')
  self.start();self.thread.join(2);self.assertFalse(self.thread.is_alive());self.assertTrue(self.errors)
 def test_no_remote_file_is_created(self):
  before=set(self.root.iterdir());self.start();self.wait('log');self.stop.set();self.thread.join(3);self.assertEqual(before,set(self.root.iterdir()))
if __name__=='__main__':unittest.main()
