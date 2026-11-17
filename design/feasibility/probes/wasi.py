"""python3 wasi.py /path/to/python/dist — offline, disposable Wasmtime CLI probe."""
import base64
import json
from pathlib import Path
import shutil
import sqlite3
import subprocess
import sys
import tempfile

bridge = (Path(__file__).resolve().parents[3] / 'datasette.py').read_text()
base = '/party/data/t/'
with tempfile.TemporaryDirectory() as tmp:
    # The CLI preopens directories, not a file capability. Give it copies only.
    home = Path(tmp) / 'python'
    shutil.copytree(sys.argv[1], home)
    data = Path(tmp) / 'data'
    data.mkdir()
    db = sqlite3.connect(data / 'films.db')
    db.executescript("create table films(title); insert into films values ('Cléo from 5 to 7');")
    db.close()
    cases = [('films.json', 'sql=select+title+from+films&_shape=array'), ('films.db', '')]
    requests = [dict(method='GET', path=base + path, query=q, headers=[]) for path, q in cases]
    result = subprocess.run(['wasmtime', 'run', '--dir', f'{home}::/', '--dir', f'{data}::/data',
                             '--env', 'PYTHONHOME=/', '--env', 'PYTHONPATH=/site', '--env', 'HOME=/',
                             '--env', 'PYTHONDONTWRITEBYTECODE=1', home / 'python.wasm', '-P', '-c',
                             bridge, '/data/films.db', 'Films', base],
                            input=b''.join(json.dumps(req).encode() + b'\n' for req in requests),
                            stdout=subprocess.PIPE, check=True, timeout=120)
    ready, query, download = [json.loads(line[4:]) for line in result.stdout.splitlines() if line.startswith(b'\0MP ')]
    rows = json.loads(base64.b64decode(query['body']))
    if not (ready['ready'] and query['status'] == 200 and rows == [{'title': 'Cléo from 5 to 7'}]
            and download['status'] in (403, 404)):
        sys.exit(f"FAIL: {ready}, SQL {query['status']} {rows}, download {download['status']}")
    print(f"PASS: SQL {query['status']} {rows}, download {download['status']}")
