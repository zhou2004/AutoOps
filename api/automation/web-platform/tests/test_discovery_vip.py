"""Bounded, isolated discovery/VIP validation; never starts a system VIP."""
import importlib.util,pathlib,types,unittest,json
from unittest import mock
spec=importlib.util.spec_from_file_location('vip_runner',pathlib.Path(__file__).resolve().parents[1]/'runner.py')
runner=importlib.util.module_from_spec(spec);spec.loader.exec_module(runner)

class DiscoveryAndVIP(unittest.TestCase):
    def test_master_options_preserve_distinct_native_layouts(self):
        version='nginx version: nginx/1.22.1\nconfigure arguments: --prefix=/usr/share/nginx --conf-path=/etc/nginx/nginx.conf'
        self.assertEqual(runner.discovery_options('/usr/sbin/nginx -p /srv/blue -c conf/nginx.conf',version),('/srv/blue','/srv/blue/conf/nginx.conf'))
        self.assertEqual(runner.discovery_options('/usr/sbin/nginx',version),('/usr/share/nginx','/etc/nginx/nginx.conf'))
        with self.assertRaises(ValueError):runner.discovery_options('nginx -p relative -c nginx.conf',version)
    def test_vip_render_has_distinct_service_pidfiles_and_health_identity(self):
        node={'clusterId':42,'instanceId':2,'address':'192.0.2.52','peers':['192.0.2.51','192.0.2.53'],'priority':190,'config':{'cidr':'192.0.2.110/24','interface':'ens33','routerId':151}}
        spec={'binary':'/usr/sbin/nginx','prefix':'/srv/blue','configPath':'/srv/blue/nginx.conf','healthUrl':'http://127.0.0.1:18080/health','expectedStatus':200}
        files=runner.vip_render(node,spec,'/srv/blue/nginx.pid');text={p:c for p,c,_ in files}
        unit=text['/etc/systemd/system/autoops-web-vip-42.service']
        self.assertIn('--pid=/run/autoops-web-vip-42.pid',unit)
        self.assertIn('--vrrp_pid=/run/autoops-web-vip-42-vrrp.pid',unit)
        self.assertNotIn('autoops-db-',unit)
        config=text['/etc/autoops-web-vip/42/keepalived.conf'];self.assertIn('nopreempt',config);self.assertIn('192.0.2.53',config)
        health=text['/etc/autoops-web-vip/42/health.py'];compile(health,'<health>','exec')
        self.assertIn('/srv/blue/nginx.pid',health);self.assertIn("'/proc/'+pid+'/exe'",health)
    def test_high_log_offsets_progress_and_stop_at_scan_limit(self):
        logs=[f"{i:06d}" for i in range(1,100001)]
        def command(args,**kwargs):return '\n'.join(logs[-int(args[-2]):]),'',0
        req={'action':'logs','spec':{'runtime':'docker'},'logs':[{'id':'stdout','path':'stdout/stderr','kind':'container'}],'logId':'stdout','lines':100,'offset':95000}
        with mock.patch.object(runner,'Runtime',return_value=types.SimpleNamespace(container='isolated')),mock.patch.object(runner,'run',side_effect=command):
            first=runner.execute(req);second=runner.execute(dict(req,offset=first['nextOffset']))
            self.assertEqual(first['offset'],95000);self.assertEqual(second['offset'],95100)
            self.assertNotEqual(first['text'],second['text'])
            last=runner.execute(dict(req,offset=99950))
            self.assertFalse(last['hasMore']);self.assertEqual(last['nextOffset'],100000)
            with self.assertRaises(ValueError):runner.execute(dict(req,offset=100000))

    def test_existing_database_vips_do_not_become_primary_addresses(self):
        node={'clusterId':777,'instanceId':1,'config':{'cidr':'192.168.80.110/24','interface':'ens33','routerId':151}}
        addresses=[{'family':'inet','local':'192.168.80.51','prefixlen':24},{'family':'inet','local':'192.168.80.60','prefixlen':24,'secondary':True},{'family':'inet','local':'192.168.80.61','prefixlen':24,'secondary':True}]
        def command(args,**kwargs):
            if 'address' in args:return json.dumps([{'flags':['UP'],'addr_info':addresses}]),'',0
            if 'link' in args:return json.dumps([{'address':'00:00:00:00:00:51'}]),'',0
            return '', '', 3
        with mock.patch.object(runner,'run',side_effect=command),mock.patch.object(runner.os.path,'isfile',return_value=False),mock.patch.object(runner.os.path,'islink',return_value=False):
            observed=runner.vip_observe(node)
        self.assertEqual(observed['address'],'192.168.80.51');self.assertFalse(observed['bound'])

    def test_package_candidate_uses_stable_locale(self):
        def command(args,**kwargs):
            if 'LC_ALL=C' in args:return 'keepalived:\n  Installed: (none)\n  Candidate: 1:2.2.7-1+b2\n','',0
            return 'keepalived:\n  候选: 1:2.2.7-1+b2\n','',0
        with mock.patch.object(runner,'run',side_effect=command):
            self.assertEqual(runner.vip_package_candidate(),'1:2.2.7-1+b2')
