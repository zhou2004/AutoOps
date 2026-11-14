"""Incremental traffic sampling uses isolated files, never system logs."""
import importlib.util,json,pathlib,tempfile,unittest
spec=importlib.util.spec_from_file_location('traffic_runner',pathlib.Path(__file__).resolve().parents[1]/'runner.py')
runner=importlib.util.module_from_spec(spec);spec.loader.exec_module(runner)
class TrafficSampling(unittest.TestCase):
 def setUp(self):
  self.temp=tempfile.TemporaryDirectory();self.root=pathlib.Path(self.temp.name);self.log=self.root/'access.log';self.log.write_text('')
  self.rt=runner.Runtime({'runtime':'host'});self.entry={'id':'a','path':str(self.log),'realPath':str(self.log),'kind':'access'}
 def tearDown(self):self.temp.cleanup()
 def row(self,path='/hello?token=private',status=200):return f'192.0.2.1 - - [13/Nov/2026:01:02:03 +0800] "GET {path} HTTP/1.1" {status} 5 "-" "test"\n'
 def sample(self,cursors=None,logs=None):return runner.traffic_collect(self.rt,{'logs':logs or [self.entry],'trafficCursors':cursors or {}})
 def test_incremental_cursor_does_not_recount_and_queries_are_removed(self):
  self.log.write_text(self.row());first=self.sample();self.assertTrue(first['initialized']);self.assertEqual(first['requests'],0)
  with self.log.open('a') as f:f.write(self.row())
  second=self.sample(first['cursors']);self.assertEqual(second['requests'],1);self.assertEqual(second['responseBytes'],5);self.assertIn('/hello',second['uris']);self.assertNotIn('private',json.dumps(second))
  third=self.sample(second['cursors']);self.assertEqual(third['requests'],0);self.assertEqual(third['supportedLogs'],1)
 def test_rotation_and_copytruncate_reset_without_stale_offset(self):
  first=self.sample();self.log.write_text(self.row());second=self.sample(first['cursors'])
  self.log.rename(self.root/'access.log.1');self.log.write_text(self.row('/new'))
  third=self.sample(second['cursors']);self.assertEqual(third['requests'],1);self.assertTrue(third['partial']);self.assertIn('/new',third['uris'])
  self.log.write_text(self.row('/trunc'));fourth=self.sample(third['cursors']);self.assertEqual(fourth['requests'],1);self.assertTrue(fourth['partial'])
 def test_partial_lines_are_retained_until_newline(self):
  first=self.sample();self.log.write_text(self.row().rstrip('\n'));second=self.sample(first['cursors']);self.assertEqual(second['requests'],0)
  with self.log.open('a') as f:f.write('\n')
  third=self.sample(second['cursors']);self.assertEqual(third['requests'],1)
 def test_json_provides_actual_bytes_latency_upstream(self):
  first=self.sample();row={'remote_addr':'2001:db8::1','uri':'/api?secret=yes','status':'503','body_bytes_sent':'16','request_length':'80','request_time':'0.025','upstream_addr':'192.0.2.8:8080'};self.log.write_text(json.dumps(row)+'\n')
  result=self.sample(first['cursors']);self.assertEqual(result['errors'],1);self.assertEqual(result['requestBytes'],80);self.assertEqual(result['latencyTotalMs'],25);self.assertIn('192.0.2.8:8080',result['upstreams']);self.assertNotIn('secret',json.dumps(result))
 def test_unsupported_formats_and_nonfinite_latency_not_counted(self):
  first=self.sample();self.log.write_text('custom unknown\n'+json.dumps({'remote_addr':'192.0.2.1','uri':'/','status':200,'body_bytes_sent':3,'request_time':'NaN'})+'\n')
  result=self.sample(first['cursors']);self.assertEqual(result['requests'],0);self.assertTrue(result['partial']);self.assertEqual(result['unparsed'],2)
 def test_path_escape_is_rejected_and_failed_cursor_cleared(self):
  first=self.sample();outside=self.root/'secret';outside.write_text(self.row());self.log.unlink();self.log.symlink_to(outside)
  result=self.sample(first['cursors']);self.assertEqual(result['requests'],0);self.assertTrue(result['partial']);self.assertEqual(result['cursors'],{});self.assertFalse(result['rateValid'])
 def test_byte_limit_and_high_cardinality_are_disclosed(self):
  first=self.sample();self.log.write_text(''.join(self.row('/'+str(i)) for i in range(30000)))
  result=self.sample(first['cursors']);self.assertLessEqual(result['scannedBytes'],2*1024*1024);self.assertLessEqual(len(result['uris']),1000);self.assertTrue(result['partial'])
 def test_dedicated_logging_never_double_counts_ordinary_access(self):
  dedicated=self.root/'traffic.jsonl';dedicated.write_text('');special={'id':'b','path':str(dedicated),'realPath':str(dedicated),'kind':'access','format':'autoops_traffic_abc'}
  first=self.sample(logs=[self.entry,special]);self.log.write_text(self.row());dedicated.write_text(self.row())
  result=self.sample(first['cursors'],[self.entry,special]);self.assertEqual(result['requests'],1);self.assertNotIn('a',result['cursors']);self.assertTrue(result['partial'])

 def test_failed_file_recovery_initializes_instead_of_accumulating_two_intervals(self):
  first=self.sample();self.log.write_text(self.row());second=self.sample(first['cursors'])
  hidden=self.root/'hidden';self.log.rename(hidden);failed=self.sample(second['cursors']);self.assertEqual(failed['cursors'],{});self.assertFalse(failed['rateValid'])
  hidden.rename(self.log)
  with self.log.open('a') as f:f.write(self.row())
  recovered=self.sample(failed['cursors']);self.assertTrue(recovered['initialized']);self.assertEqual(recovered['requests'],0)
  with self.log.open('a') as f:f.write(self.row())
  current=self.sample(recovered['cursors']);self.assertEqual(current['requests'],1);self.assertTrue(current['rateValid'])
 def test_budget_gap_resets_only_incomplete_file_baseline(self):
  first=self.sample();self.log.write_text(self.row()*40000);bounded=self.sample(first['cursors']);self.assertFalse(bounded['rateValid']);self.assertNotIn('a',bounded['cursors'])
  recovered=self.sample(bounded['cursors']);self.assertTrue(recovered['initialized']);self.assertEqual(recovered['requests'],0)
  with self.log.open('a') as f:f.write(self.row())
  self.assertEqual(self.sample(recovered['cursors'])['requests'],1)
 def test_previously_supported_log_format_becoming_unknown_is_not_zero(self):
  first=self.sample();self.log.write_text(self.row());second=self.sample(first['cursors']);self.assertEqual(second['supportedLogs'],1)
  with self.log.open('a') as f:f.write('now this is an unknown format\n')
  unknown=self.sample(second['cursors']);self.assertEqual(unknown['requests'],0);self.assertEqual(unknown['supportedLogs'],0);self.assertFalse(unknown['rateValid'])
 def test_oversized_lines_cannot_bypass_total_scan_budget(self):
  entries=[]
  for index in range(3):
   file=self.root/('large-'+str(index)+'.log');file.write_text('');entries.append({'id':str(index),'path':str(file),'realPath':str(file),'kind':'access'})
  first=self.sample(logs=entries)
  for entry in entries:pathlib.Path(entry['path']).write_bytes(b'x'*(3*1024*1024))
  result=self.sample(first['cursors'],entries);self.assertEqual(result['scannedBytes'],4*1024*1024);self.assertFalse(result['rateValid']);self.assertEqual(result['cursors'],{})
  recovered=self.sample(result['cursors'],entries);self.assertTrue(recovered['initialized']);self.assertEqual(recovered['requests'],0)
