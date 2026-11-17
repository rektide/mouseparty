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
    replacements = {str(root / 'probe_linux.go'): str(stub)}
    if '--selftest' in sys.argv:
        # The upstream check discards ffmpeg's log; keep it for diagnostics.
        verbose = Path(tmp) / 'selftest.go'
        verbose.write_text((root / 'selftest.go').read_text().replace('Log: func(string) {}', 'Log: func(s string) { fmt.Println(s) }'))
        replacements[str(root / 'selftest.go')] = str(verbose)
    overlay.write_text(json.dumps({'Replace': replacements}))
    if '--desktop' in sys.argv or '--selftest' in sys.argv:
        # Link the real Wails Linux frontend too. Do not launch/distribute it.
        subprocess.run(['go', 'build', '-tags', 'desktop,production,webkit2_41', '-overlay', overlay,
                        '-o', Path(tmp) / 'probe', '.'], check=True, cwd=root)
    if '--selftest' in sys.argv:
        # The upstream site binds 0.0.0.0. Give this test a loopback-only netns,
        # a private /tmp and no real home. Never fall back to the host namespace.
        subprocess.run(['bwrap', '--unshare-all', '--die-with-parent', '--clearenv',
                        '--ro-bind', '/usr', '/usr', '--symlink', 'usr/bin', '/bin',
                        '--symlink', 'usr/lib', '/lib', '--symlink', 'usr/lib64', '/lib64',
                        '--ro-bind', '/etc/ld.so.cache', '/etc/ld.so.cache',
                        '--ro-bind-try', '/etc/alternatives', '/etc/alternatives',
                        '--ro-bind', '/etc/fonts', '/etc/fonts', '--proc', '/proc', '--dev', '/dev',
                        '--tmpfs', '/tmp', '--dir', '/home/test', '--setenv', 'HOME', '/home/test',
                        '--setenv', 'PATH', '/tools:/usr/bin', '--dir', '/work', '--chdir', '/work',
                        '--ro-bind', (root / 'build/python/dist').resolve(), '/work/build/python/dist',
                        '--ro-bind', root / 'build/probes/linux-tools/mediamtx', '/tools/mediamtx',
                        '--ro-bind', Path(tmp) / 'probe', '/work/probe', '/work/probe', '--selftest'],
                       check=True, timeout=120)
        sys.exit(0)
    pattern = ('^(TestAppsOverChat|TestSocketGraphIsTypes)$' if '--social' in sys.argv else
               '^(TestOnlyTheKeyGetsIn|TestChatJoinSayLeave|TestOnlyFile|TestDatasette.*)$')
    subprocess.run(['go', 'test', '-count=1', '-v', '-overlay', overlay, '-run', pattern, '.'],
                   check=True, cwd=root)
