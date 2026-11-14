import importlib.util
import json
import os
from pathlib import Path
import selectors
import subprocess
import tempfile
import time
import unittest

READER = Path(__file__).parents[1] / 'log_reader.py'
spec = importlib.util.spec_from_file_location('reader', READER)
reader = importlib.util.module_from_spec(spec)
spec.loader.exec_module(reader)


class LogsTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.path = self.root / 'mysql.log'
        self.path.write_text('旧日志\nsecond\n', encoding='utf8')
        self.input = dict(dataDir=str(self.root), service='', lines=500)
        self.log_id = reader.sources(self.input)[0]['id']

    def run_reader(self, **changes):
        payload = dict(self.input, action='read', logId=self.log_id, follow=False)
        payload.update(changes)
        completed = subprocess.run(['python3', str(READER)], input=json.dumps(payload),
                                   capture_output=True, text=True, timeout=5)
        return completed, [json.loads(line) for line in completed.stdout.splitlines()]

    def test_sources_only_logs_and_no_links(self):
        (self.root / 'credentials.json').write_text('private')
        (self.root / 'mysql-bin.000001').write_text('binary')
        (self.root / 'escape.log').symlink_to('/etc/passwd')
        (self.root / 'logs').symlink_to('/var/log')
        rows = reader.sources(self.input)
        self.assertEqual([row['name'] for row in rows], ['mysql.log'])
        completed, events = self.run_reader(logId='/etc/passwd')
        self.assertNotEqual(completed.returncode, 0)
        self.assertEqual(events[0]['event'], 'error')

    def test_history_unicode_and_limits(self):
        completed, events = self.run_reader(lines=1)
        self.assertEqual(completed.returncode, 0)
        self.assertEqual(''.join(e['data'].get('text', '') for e in events), 'second\n')
        self.path.write_text('中文🙂' * 400000 + '\nTAIL\n')
        completed, events = self.run_reader()
        self.assertEqual(completed.returncode, 0)
        self.assertTrue(any(e['event'] == 'notice' for e in events))
        self.assertLess(max(len(json.dumps(e)) for e in events), 256000)
        self.assertTrue(''.join(e['data'].get('text', '') for e in events).endswith('TAIL\n'))

    def test_symlink_replacement_rejected(self):
        self.path.unlink()
        self.path.symlink_to('/etc/passwd')
        completed, events = self.run_reader()
        self.assertNotEqual(completed.returncode, 0)
        self.assertEqual(events[0]['event'], 'error')

    def test_follow_append_rotation_and_stop(self):
        payload = dict(self.input, action='read', logId=self.log_id, follow=True)
        proc = subprocess.Popen(['python3', '-u', str(READER)], stdin=subprocess.PIPE, stdout=subprocess.PIPE)
        self.addCleanup(lambda: proc.poll() is None and proc.kill())
        proc.stdin.write(json.dumps(payload).encode())
        proc.stdin.close()
        selector = selectors.DefaultSelector()
        self.addCleanup(selector.close)
        selector.register(proc.stdout, selectors.EVENT_READ)
        collected = bytearray()

        def until(expected):
            deadline = time.monotonic() + 5
            while expected not in collected:
                if time.monotonic() > deadline:
                    self.fail('missing live event: ' + repr(collected))
                if selector.select(.25):
                    chunk = os.read(proc.stdout.fileno(), 65536)
                    if not chunk:
                        self.fail('reader stopped unexpectedly')
                    collected.extend(chunk)

        until(b'second')
        with self.path.open('a') as stream:
            stream.write('APPENDED\n')
        until(b'APPENDED')
        self.path.rename(self.root / 'mysql.log.1')
        self.path.write_text('ROTATED\n')
        until(b'ROTATED')
        self.path.write_text('NEW\n')
        until(b'NEW')
        proc.terminate()
        proc.wait(timeout=3)
        proc.stdout.close()
        self.assertEqual(proc.returncode, 0)


if __name__ == '__main__':
    unittest.main()
