# Security

Please report a security problem privately, through GitHub's
[Report a vulnerability](https://github.com/ftrain/mouseparty/security/advisories/new)
form, not in a public issue.

Only the latest release is fixed. Mouse Party is one person's project, with no
promise of how soon a fix comes.

The macOS sandbox policies are in [`sandbox.go`](sandbox.go).

On experimental Linux builds, the [viewer site](site_linux.go) and
[offline ffmpeg jobs](ffmpeg_linux.go) share a [bubblewrap launch policy](bubblewrap_linux.go):
private namespaces (including the network), a minimal environment, read-only
system runtime libraries, and disposable scratch space.

- The **viewer site** has embedded pages and read-only WASM tools. It accepts on
  the inherited host listener. The parent hands it connected sockets for just
  MediaMTX HLS and WebRTC; a fixed RPC reports live status without exposing the
  control API. A [seccomp filter](seccomp_linux.go) blocks opening or reconnecting
  sockets and executing programs. Failure to set up either layer stops hosting.
- **Offline ffmpeg** covers frame grabs (`Host.Frame`), media inspection, and
  text subtitle conversion. Each invocation sees one host-selected regular
  file, mounted read-only by descriptor, not its parent directory. Replacing the
  selected path cannot widen the mount. Outputs go to stdout/stderr. There are
  no host sockets or writable host file grants. These jobs do not add a
  seccomp filter or resource limits.

Linux sandboxing needs an up-to-date `bwrap` on `PATH` with `--disable-userns`
and working unprivileged user namespaces. Offline ffmpeg also needs `--ro-bind-fd`;
the viewer needs seccomp and amd64 or arm64. Setup failures never trigger an unboxed
retry. The party file's explicit `no_sandbox` disables these protections.
Linux publishing ffmpeg (including its cover-art preparation), MediaMTX, and
cloudflared are **not** covered by these policies.

Run `go test -run 'TestLinux(Viewer|OfflineFFmpeg)' -v .` to check both boundaries
and exercise real offline media operations with ffmpeg on `PATH`. With the Linux
media tools on `PATH`, `go run -tags 'desktop production' . --selftest` also
exercises the viewer against real MediaMTX (requires Wails build dependencies).
