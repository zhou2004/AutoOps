"""Read-only database logs. Input is JSON from the trusted service, never shell text."""
import codecs
import hashlib
import json
import os
import selectors
import signal
import stat
import subprocess
import sys
import time

MAX_BYTES = 2 * 1024 * 1024


def sources(spec):
    result = []
    unit = spec.get('service', '')
    if unit:
        result.append(dict(id='journal', kind='journal', path=unit, name='服务日志'))
    directory = spec.get('dataDir', '')
    if not directory or not os.path.isabs(directory) or directory == '/':
        return result
    root = os.path.realpath(directory)
    # Never search other instances or traverse arbitrary directories/symlinks.
    for folder in (root, os.path.join(root, 'log'), os.path.join(root, 'logs'),
                   os.path.join(root, 'data', 'log'), os.path.join(root, 'data', 'pg_log')):
        if os.path.islink(folder) or not os.path.isdir(folder):
            continue
        if os.path.commonpath([root, os.path.realpath(folder)]) != root:
            continue
        with os.scandir(folder) as entries:
            for index, item in enumerate(entries):
                if index >= 1000:
                    break
                if not item.is_file(follow_symlinks=False):
                    continue
                if not item.name.endswith(('.log', '.err', '.csv', '.json')):
                    continue
                # JSON config files are not logs; only PostgreSQL structured logs.
                if item.name.endswith(('.csv', '.json')) and not item.name.startswith('postgresql-'):
                    continue
                path = item.path
                result.append(dict(id=hashlib.sha256(path.encode()).hexdigest(), kind='file',
                                   path=path, name=item.name, size=item.stat().st_size))
    return sorted(result, key=lambda item: (item['kind'] != 'journal', item['path']))[:100]


def emit(event, **data):
    print(json.dumps(dict(event=event, data=data), ensure_ascii=True), flush=True)


def open_log(path):
    # Recheck parent links each time a rotated file is opened.
    if os.path.realpath(path) != path:
        raise ValueError('日志路径已变化，请重新加载')
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    stream = os.fdopen(fd, 'rb')
    if not stat.S_ISREG(os.fstat(fd).st_mode):
        stream.close()
        raise ValueError('日志不是普通文件')
    return stream


def follow_file(target, lines, follow):
    stream = open_log(target['path'])
    decoder = codecs.getincrementaldecoder('utf-8')('replace')
    try:
        size = os.fstat(stream.fileno()).st_size
        start = max(0, size - MAX_BYTES)
        stream.seek(start)
        data = stream.read(MAX_BYTES)
        if start:
            data = data.partition(b'\n')[2]
            emit('notice', message='历史日志已限制为最近 2 MiB')
        history = b''.join(data.splitlines(keepends=True)[-lines:]).decode('utf-8', 'replace')
        for offset in range(0, len(history), 8192):
            emit('log', text=history[offset:offset + 8192])
        last_heartbeat = time.monotonic()
        while follow:
            data = stream.read(32768)
            if data:
                emit('log', text=decoder.decode(data))
                continue
            try:
                current = os.stat(target['path'], follow_symlinks=False)
            except FileNotFoundError:
                current = None
            if current and (current.st_ino != os.fstat(stream.fileno()).st_ino or current.st_size < stream.tell()):
                stream.close()
                stream = open_log(target['path'])
                decoder.reset()
                emit('notice', message='日志已轮转，继续读取新文件')
            if time.monotonic() - last_heartbeat >= 10:
                emit('heartbeat', at=time.time())
                last_heartbeat = time.monotonic()
            time.sleep(.25)
    finally:
        stream.close()


def follow_journal(target, lines, follow):
    command = ['journalctl', '--no-pager', '--quiet', '--output=short-iso', '--unit', target['path'], '--lines', str(lines)]
    if follow:
        command.append('--follow')
    process = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    decoder = codecs.getincrementaldecoder('utf-8')('replace')
    selector = selectors.DefaultSelector()
    selector.register(process.stdout, selectors.EVENT_READ)
    try:
        while True:
            if not selector.select(10):
                emit('heartbeat', at=time.time())
                continue
            data = os.read(process.stdout.fileno(), 32768)
            if not data:
                break
            emit('log', text=decoder.decode(data))
        if process.wait() != 0:
            raise ValueError('服务日志读取失败，请检查 journal 权限')
    finally:
        selector.close()
        process.stdout.close()
        if process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()


def main():
    spec = json.load(sys.stdin)
    available = sources(spec)
    if spec['action'] == 'list':
        print(json.dumps(available))
        return
    target = next((item for item in available if item['id'] == spec['logId']), None)
    if target is None:
        raise ValueError('日志源不存在，请重新加载')
    emit('ready', path=target['path'])
    lines = max(1, min(int(spec.get('lines', 500)), 2000))
    if target['kind'] == 'file':
        follow_file(target, lines, spec.get('follow', True))
    else:
        follow_journal(target, lines, spec.get('follow', True))
    emit('end', message='日志读取完成')


if __name__ == '__main__':
    def stop(signum, frame):
        raise SystemExit(0)
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGHUP, stop)
    signal.signal(signal.SIGALRM, stop)
    signal.alarm(1200)
    try:
        main()
    except (BrokenPipeError, KeyboardInterrupt):
        pass
    except Exception as exc:
        emit('error', message=str(exc))
        sys.exit(1)
