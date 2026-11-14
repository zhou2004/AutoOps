"""Isolated native-Nginx integration checks; never use system nginx.conf or services.
Run: python3 -m unittest discover -s api/automation/web-platform/tests -v
"""
import ssl, importlib.util, json, os, pathlib, shutil, socket, subprocess, tempfile, time, unittest, urllib.request

path=pathlib.Path(__file__).resolve().parents[1]/'runner.py'
spec=importlib.util.spec_from_file_location('runner',path);runner=importlib.util.module_from_spec(spec);spec.loader.exec_module(runner)

@unittest.skipUnless(shutil.which('nginx'), 'native nginx is required')
class NginxIntegration(unittest.TestCase):
    def setUp(self):
        self.temporary=tempfile.TemporaryDirectory(prefix='autoops-web-test-');self.root=pathlib.Path(self.temporary.name)
        with socket.socket() as s:s.bind(('127.0.0.1',0));self.port=s.getsockname()[1]
        self.conf=self.root/'nginx.conf';self.site=self.root/'sites/site.conf';self.site.parent.mkdir()
        (self.root/'nested').mkdir();(self.root/'nested/headers.conf').write_text('add_header X-AutoOps test; # keep this comment\n')
        self.site.write_text(self.site_text('old'))
        self.conf.write_text('pid '+str(self.root/'nginx.pid')+';\nerror_log '+str(self.root/'error.log')+';\nevents {}\nhttp { access_log '+str(self.root/'access.log')+'; include '+str(self.root/'sites/*.conf')+'; }\n')
        self.source={'hostId':os.getpid(),'runtime':'host','binary':shutil.which('nginx'),'configPath':str(self.conf),'prefix':str(self.root),'managedPaths':[str(self.root)],'healthUrl':'http://127.0.0.1:'+str(self.port)+'/','expectedStatus':200}
        self.rt=runner.Runtime(self.source);self.rt.cmd([self.source['binary'],'-p',str(self.root),'-c',str(self.conf)])
        try:self.wait_body('old')
        except BaseException:self.tearDown();raise
    def tearDown(self):
        self.rt.nginx('-s','quit')
        for _ in range(100):
            if not (self.root/'nginx.pid').exists():break
            time.sleep(.02)
        key=runner.sha(str(os.getpid())+':host::'+str(self.conf));shutil.rmtree('/var/tmp/autoops-web/'+key,ignore_errors=True)
        self.temporary.cleanup()
    def site_text(self,body,code=200):
        return 'server { listen 127.0.0.1:'+str(self.port)+'; include '+str(self.root/'nested/headers.conf')+'; location / { return '+str(code)+' "'+body+'"; } }\n'
    def wait_body(self,expected):
        for _ in range(100):
            try:
                with urllib.request.build_opener(urllib.request.ProxyHandler({})).open(self.source['healthUrl'],timeout=1) as res:
                    if res.read().decode()==expected:return
            except Exception:pass
            time.sleep(.03)
        self.fail('server did not serve '+expected)
    def call(self,action,files=None,**kw):
        ob=self.rt.discover();req={'action':action,'spec':self.source,'identity':ob['identity'],'baseline':ob['digest'],'operationId':42,'files':files or []};req.update(kw);return runner.execute(req)
    def file(self,content):return {'path':str(self.site),'content':content,'mode':0o644}
    def test_discovery_and_nested_candidate_publish(self):
        before=self.rt.discover();self.assertEqual(len(before['files']),3)
        original=self.site.read_text();self.call('validate',[self.file(self.site_text('new'))]);self.assertEqual(self.site.read_text(),original)
        result=self.call('apply',[self.file(self.site_text('new'))]);self.assertEqual(result['state'],'succeeded');self.wait_body('new')
        self.assertNotEqual(before['digest'],self.rt.discover()['digest'])
        result=self.call('restore');self.assertEqual(result['state'],'rolled-back');self.wait_body('old')
        self.assertEqual(self.call('reconcile')['state'],'rolled-back')
    def test_bad_syntax_preserves_live_files(self):
        original=self.site.read_text()
        with self.assertRaises(ValueError):self.call('apply',[self.file('this_directive_does_not_exist;')])
        self.assertEqual(self.site.read_text(),original);self.wait_body('old')
    def test_failed_health_restores_existing_and_removes_created_files(self):
        files=[self.file(self.site_text('broken',503)),{'path':str(self.root/'sites/extra.conf'),'content':'# new file\n','mode':0o644}]
        result=self.call('apply',files);self.assertEqual(result['state'],'rolled-back');self.wait_body('old')
        self.assertFalse((self.root/'sites/extra.conf').exists())
    def test_external_change_blocks_rollback(self):
        result=self.call('apply',[self.file(self.site_text('new'))]);self.wait_body('new')
        baseline=result['observed']['digest'];self.site.write_text(self.site_text('human'))
        with self.assertRaisesRegex(ValueError,'drift'):self.call('restore',baseline=baseline)
        self.assertIn('human',self.site.read_text())
    def test_out_of_scope_and_symlink_escape_are_rejected(self):
        with tempfile.TemporaryDirectory() as outside:
            link=self.root/'escape';link.symlink_to(outside,target_is_directory=True)
            with self.assertRaisesRegex(ValueError,'scope'):self.call('validate',[{'path':str(link/'secret'),'content':'x'}])
    def test_existing_symlink_is_preserved(self):
        actual=self.root/'nested/real-site';self.site.rename(actual);self.site.symlink_to(actual)
        self.call('apply',[self.file(self.site_text('new'))]);self.wait_body('new');self.assertTrue(self.site.is_symlink())
    def test_log_query_and_rotation(self):
        self.wait_body('old');res=self.call('logs',logId=runner.sha(str(self.root/'access.log')),lines=20,status='200');self.assertIn('200',res['text'])
        with self.assertRaisesRegex(ValueError,'rotation'):self.call('logs',logId=runner.sha(str(self.root/'access.log')),rotation='/../../passwd')
    def test_empty_glob_and_candidate_only_include(self):
        config=self.conf.read_text().replace('http {','http { include '+str(self.root/'missing/*.conf')+';')
        self.call('validate',[{'path':str(self.conf),'content':config,'mode':0o644}])
        with self.assertRaisesRegex(ValueError,'unresolved include'):self.call('validate',[{'path':str(self.conf),'content':config.replace('*.conf','must-exist.conf'),'mode':0o644}])

    def test_explicit_include_deletion_never_reads_live_dependency(self):
        dependency=self.root/'nested/headers.conf';original=dependency.read_text()
        deletion={'path':str(dependency),'deleted':True}
        for action in ('validate','apply'):
            with self.assertRaisesRegex(ValueError,'unresolved include'):self.call(action,[deletion])
            self.assertEqual(dependency.read_text(),original);self.wait_body('old')
        replacement=self.site_text('new').replace(' include '+str(dependency)+';','')
        result=self.call('apply',[deletion,self.file(replacement)])
        self.assertEqual(result['state'],'succeeded');self.assertFalse(dependency.exists());self.wait_body('new')
        self.call('restore');self.assertEqual(dependency.read_text(),original);self.wait_body('old')

    def test_inactive_confd_files_are_discovered_and_checked_without_activation(self):
        directory=self.root/'conf.d';directory.mkdir();inactive=directory/'inactive.conf'
        inactive.write_text('map $http_x_test $autoops_native_test { default old; }\n')
        observation=self.rt.discover();found={f['path']:f for f in observation['files']}
        self.assertIn(str(inactive),found);self.assertFalse(found[str(inactive)]['included']);self.assertTrue(found[str(inactive)]['editable'])
        for path in (inactive,directory/'new.conf',directory/'new.inc'):
            with self.assertRaises(ValueError):self.call('validate',[{'path':str(path),'content':'unknown_nginx_directive on;'}])
        self.assertEqual(inactive.read_text(),'map $http_x_test $autoops_native_test { default old; }\n')
        valid='map $http_x_test $autoops_native_test { default changed; }\n'
        result=self.call('apply',[{'path':str(inactive),'content':valid}]);self.assertEqual(result['state'],'succeeded')
        self.assertNotEqual(observation['digest'],result['observed']['digest'])
        self.assertNotIn(str(inactive),self.rt.nginx('-T')[0]);self.wait_body('old')
        self.call('restore');self.assertEqual(self.rt.discover()['digest'],observation['digest'])

    def test_new_confd_file_lifecycle_and_nested_http_context(self):
        directory=self.root/'conf.d';directory.mkdir();added=directory/'new.conf';http=self.root/'http.conf'
        original=self.conf.read_text();prefix,body=original.split('http {',1)
        http.write_text('http {'+body);self.conf.write_text(prefix+'include '+str(http)+';\n')
        before=self.rt.discover();new={'path':str(added),'content':'map $http_x_test $autoops_native_test { default one; }\n'}
        self.call('validate',[new]);self.assertFalse(added.exists())
        result=self.call('apply',[new]);self.assertEqual(result['state'],'succeeded');self.wait_body('old')
        changed=dict(new,content=new['content'].replace('one','two'))
        result=self.call('apply',[changed],operationId=43);self.assertEqual(result['state'],'succeeded');self.assertEqual(added.read_text(),changed['content'])
        result=self.call('apply',[{'path':str(added),'deleted':True}],operationId=44)
        self.assertEqual(result['state'],'succeeded');self.assertFalse(added.exists());self.assertEqual(before['digest'],self.rt.discover()['digest'])

    def test_omitted_file_mode_preserves_existing_permissions(self):
        os.chmod(self.site,0o640)
        change={'path':str(self.site),'content':self.site_text('new'),'mode':0}
        self.call('apply',[change]);self.assertEqual(self.site.stat().st_mode & 0o777,0o640);self.wait_body('new')

    def test_private_key_named_conf_is_not_returned_by_discovery(self):
        private=self.root/'misplaced.conf';private.write_text('-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----\n')
        with self.assertRaisesRegex(ValueError,'private key material'):self.rt.discover()
        private.unlink()

    @unittest.skipUnless(shutil.which('openssl'),'openssl required')
    def test_certificate_deployment_tls_pin_and_private_key_restore(self):
        def certificate(name):
            cert=self.root/(name+'.pem');key=self.root/(name+'.key')
            runner.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-subj','/CN=localhost','-addext','subjectAltName=DNS:localhost','-days','1','-keyout',str(key),'-out',str(cert)])
            return cert.read_text(),key.read_text()
        old,oldkey=certificate('old');new,newkey=certificate('new')
        certpath=self.root/'live.pem';keypath=self.root/'live.key';certpath.write_text(old);keypath.write_text(oldkey)
        with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
        tlsconfig='server {listen 127.0.0.1:'+str(port)+' ssl;ssl_certificate '+str(certpath)+';ssl_certificate_key '+str(keypath)+';location / {return 200 "tls";}}'
        tlsfile=self.root/'sites/tls.conf';tlsfile.write_text(tlsconfig);self.rt.nginx('-s','reload');time.sleep(.3)
        before=self.rt.discover()['digest']
        relative=tlsconfig.replace(str(certpath),'live.pem').replace(str(keypath),'live.key')
        mismatch=[{'path':str(tlsfile),'content':relative},{'path':str(certpath),'content':new},{'path':str(keypath),'content':oldkey,'mode':0o600}]
        with self.assertRaises(ValueError):self.call('validate',mismatch)
        self.assertEqual(certpath.read_text(),old);self.assertEqual(keypath.read_text(),oldkey)
        probes=[{'bindingId':1,'versionId':2,'host':'127.0.0.1','port':port,'serverName':'localhost','fingerprint':runner.sha(ssl.PEM_cert_to_DER_cert(new))}]
        files=[{'path':str(certpath),'content':new,'mode':0o644},{'path':str(keypath),'content':newkey,'mode':0o600}]
        result=self.call('apply',files,tls=probes);self.assertEqual(result['state'],'succeeded');self.assertNotEqual(before,self.rt.discover()['digest']);self.assertEqual(keypath.stat().st_mode&0o777,0o600)
        self.call('restore');self.assertEqual(keypath.read_text(),oldkey);self.assertEqual(certpath.read_text(),old)
        badprobe=[dict(probes[0],fingerprint='0'*64)]
        result=self.call('apply',files,tls=badprobe,operationId=43);self.assertEqual(result['state'],'rolled-back');self.assertEqual(keypath.read_text(),oldkey)

    def test_interrupted_partial_files_recover_without_replaying_publication(self):
        observation=self.rt.discover();original=self.site.read_text();newfile=self.root/'sites/new.conf'
        candidate='server {listen 127.0.0.1:'+str(self.port)+'; include '+str(newfile)+';}'
        # Simulate a worker loss after replacing the referencing file, before its include.
        key=runner.sha(str(os.getpid())+':host::'+str(self.conf));root=pathlib.Path('/var/tmp/autoops-web')/key;root.mkdir(parents=True,exist_ok=True)
        record={'state':'written','original':observation,'candidate':{str(self.site):runner.sha(candidate),str(newfile):runner.sha('location / {return 200 "next";}')},'backup':[{'path':str(self.site),'content':original,'mode':0o644,'exists':True},{'path':str(newfile),'content':'','mode':0o644,'exists':False}]}
        runner.save_journal(str(root/'42.json'),record);self.site.write_text(candidate)
        self.assertEqual(runner.execute({'action':'reconcile','spec':self.source,'operationId':42})['state'],'recoverable')
        result=runner.execute({'action':'recover','spec':self.source,'operationId':42,'identity':observation['identity']});self.assertEqual(result['state'],'rolled-back');self.wait_body('old')
        self.assertEqual(self.site.read_text(),original);self.assertFalse(newfile.exists())
    def test_interrupted_recovery_does_not_overwrite_human_changes(self):
        observation=self.rt.discover();original=self.site.read_text();candidate=self.site_text('next')
        key=runner.sha(str(os.getpid())+':host::'+str(self.conf));root=pathlib.Path('/var/tmp/autoops-web')/key;root.mkdir(parents=True,exist_ok=True)
        runner.save_journal(str(root/'42.json'),{'state':'written','original':observation,'candidate':{str(self.site):runner.sha(candidate)},'backup':[{'path':str(self.site),'content':original,'mode':0o644,'exists':True}]})
        self.site.write_text(self.site_text('human'))
        with self.assertRaisesRegex(ValueError,'external modification'):runner.execute({'action':'recover','spec':self.source,'operationId':42,'identity':observation['identity']})
        self.assertIn('human',self.site.read_text())

class BoundedExecution(unittest.TestCase):
    def test_output_limit(self):
        with self.assertRaisesRegex(ValueError,'output exceeded'):runner.run(['python3','-c','print("x"*(9*1024*1024))'])
    def test_timeout(self):
        with self.assertRaises(TimeoutError):runner.run(['python3','-c','import time;time.sleep(3)'],timeout=.1)

if __name__=='__main__':unittest.main()
