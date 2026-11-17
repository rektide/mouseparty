"""Linux compile diagnostic, not a port: simulate only the missing AppKit hooks in go test."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile

root = Path(__file__).resolve().parents[3]
with tempfile.TemporaryDirectory() as tmp:
    stub = Path(tmp) / 'hooks.go'
    stub.write_text('''package main
func nativeActive() bool { return true }
func nativeAbout(string) {}
func nativeSend(string) {}
func nativeBadge(string) {}
func nativeKeepFrame() {}
func nativeHideOthers() {}
func nativeShowAll() {}
''')
    overlay = Path(tmp) / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': {str(root / 'probe_linux.go'): str(stub)}}))
    if '--desktop' in sys.argv:
        # Link the real Wails Linux frontend too. Do not launch/distribute it.
        subprocess.run(['go', 'build', '-tags', 'desktop,production,webkit2_41', '-overlay', overlay,
                        '-o', Path(tmp) / 'probe', '.'], check=True, cwd=root)
    pattern = ('^(TestAppsOverChat|TestSocketGraphIsTypes)$' if '--social' in sys.argv else
               '^(TestOnlyTheKeyGetsIn|TestChatJoinSayLeave|TestOnlyFile|TestDatasette.*)$')
    subprocess.run(['go', 'test', '-count=1', '-v', '-overlay', overlay, '-run', pattern, '.'],
                   check=True, cwd=root)
