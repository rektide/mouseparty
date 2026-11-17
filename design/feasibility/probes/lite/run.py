"""uv run --with playwright run.py /path/to/datasette-lite (requires chromium on PATH)."""
import functools
import http.server
import pathlib
import shutil
import subprocess
import sys
import tempfile
import threading

from playwright.sync_api import sync_playwright

here = pathlib.Path(__file__).resolve().parent
with tempfile.TemporaryDirectory() as tmp:
    root = pathlib.Path(tmp)
    shutil.copy(pathlib.Path(sys.argv[1]) / 'webworker.js', root)
    shutil.copy(here / 'index.html', root)
    (root / 'films.sql').write_text("create table films(title); insert into films values ('Cléo from 5 to 7');", encoding='utf-8')
    subprocess.run(['rustc', '--edition=2024', '--target=wasm32-unknown-unknown', '--crate-type=cdylib',
                    '-O', '-C', 'strip=symbols', here / 'client.rs', '-o', root / 'client.wasm'], check=True)
    handler = functools.partial(http.server.SimpleHTTPRequestHandler, directory=tmp)
    with http.server.ThreadingHTTPServer(('127.0.0.1', 0), handler) as server, sync_playwright() as pw:
        threading.Thread(target=server.serve_forever, daemon=True).start()
        browser = pw.chromium.launch(executable_path=shutil.which('chromium'))
        page = browser.new_page()
        page.on('console', lambda msg: print(msg.text, flush=True) if msg.type == 'error' else None)
        page.on('pageerror', lambda exc: print(exc, flush=True))
        page.goto(f'http://127.0.0.1:{server.server_port}')
        page.wait_for_function("/^(PASS|FAIL):/.test(document.querySelector('#result').textContent)", timeout=180_000)
        result = page.locator('#result').inner_text()
        print(result)
        browser.close()
        server.shutdown()
        sys.exit(0 if result.startswith('PASS:') else 1)
