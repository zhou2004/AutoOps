#!/usr/bin/env python3
"""Install checksum-pinned pure Python artifacts without changing system Python."""
import hashlib
import os
from pathlib import Path
import shutil
import tarfile
import zipfile

root = Path('/opt/autoops/patroni-4.1.5')
artifacts = {
    'patroni-4.1.5-py3-none-any.whl': '7aad8c1539e6c51c269a8005169dd54ddc7668437b7900fa6099d7894e224c84',
    'ydiff-1.3.tar.gz': '8a2e84588ef31d3e525700fa964b1f9aab61f0c9c46281e05341a56e01dcd7b8',
    'py-consul-1.1.1.tar.gz': '4debbbd4f5cab02385e8b1e7e66d998fbd0f48682839921c5ab25ff23ddb5fce',
}
for name, digest in artifacts.items():
    assert hashlib.sha256((root / name).read_bytes()).hexdigest() == digest
lib = root / 'lib'
lib.mkdir(mode=0o755, exist_ok=True)
with zipfile.ZipFile(root / 'patroni-4.1.5-py3-none-any.whl') as wheel:
    for item in wheel.infolist():
        target = lib / item.filename
        assert target.resolve().is_relative_to(lib.resolve())
    wheel.extractall(lib)
for name in ('ydiff-1.3.tar.gz', 'py-consul-1.1.1.tar.gz'):
    with tarfile.open(root / name) as archive:
        for item in archive:
            relative = Path(*Path(item.name).parts[1:])
            if not item.isfile() or not relative.parts:
                continue
            if relative != Path('ydiff.py') and relative.parts[0] != 'consul':
                continue
            target = lib / relative
            assert target.resolve().is_relative_to(lib.resolve())
            target.parent.mkdir(mode=0o755, parents=True, exist_ok=True)
            with archive.extractfile(item) as source, target.open('wb') as dest:
                shutil.copyfileobj(source, dest)
            target.chmod(0o644)
(root / 'bin').mkdir(mode=0o755, exist_ok=True)
script = root / 'bin' / 'patroni'
script.write_text('#!/usr/bin/python3\nimport sys,os\nos.environ["PYTHONPATH"]="/opt/autoops/patroni-4.1.5/lib"\nos.environ["PATH"]="/opt/autoops/patroni-4.1.5/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"\nsys.path.insert(0, "/opt/autoops/patroni-4.1.5/lib")\nfrom patroni.__main__ import main\nif __name__ == "__main__":\n    main()\n')
script.chmod(0o755)
print('Pinned Patroni 4.1.5 installed in private runtime')
