# Security

Please report a security problem privately, through GitHub's
[Report a vulnerability](https://github.com/ftrain/mouseparty/security/advisories/new)
form, not in a public issue.

Only the latest release is fixed. Mouse Party is one person's project, with no
promise of how soon a fix comes.

The macOS sandbox policies are in [`sandbox.go`](sandbox.go).

On experimental Linux builds, the **viewer site** runs under [bubblewrap](site_linux.go)
with private namespaces, embedded pages, read-only runtime libraries and WASM tools,
and disposable scratch space. It accepts on the inherited host listener. The parent
hands it connected sockets for just MediaMTX HLS and WebRTC; a fixed RPC reports
live status without exposing the control API. A [seccomp filter](seccomp_linux.go)
blocks opening or reconnecting sockets and executing programs. With sandboxing
enabled, failure to set up either layer stops hosting.

This requires amd64 or arm64, `bwrap` on PATH with `--disable-userns` support,
and working unprivileged user namespaces and seccomp. Linux MediaMTX, ffmpeg
and cloudflared do **not** yet have their own sandbox policies. To check the
viewer boundary, run `go test -run TestLinuxViewerIsolation -v .`; with the
Linux media tools on PATH, `go run -tags 'desktop production' . --selftest`
also exercises the viewer against real MediaMTX (requires Wails build dependencies).
