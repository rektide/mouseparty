---
type: Experiment
title: Run the two Datasette paths on Linux
description: Narrow reproduction recipes for the released WASI runtime, a Rust browser client of Datasette Lite, and selected upstream Go tests.
resource: /design/feasibility/probes/README.md
status: draft
generated: { by: 'model:openai/gpt-6.2-astra', at: 2026-11-17 }
sources:
  - { resource: 'opencode://ses_e1479787affey7NRxOch4DM678', title: Linux build and Datasette runtime probes }
  - { resource: 'https://github.com/ftrain/mouseparty/releases/tag/v3.5.1', title: Tested DMG and published digest }
  - { resource: 'https://github.com/simonw/datasette-lite/blob/f49662beda68d787ce9c8f24f6eb0b452d90f2c7/webworker.js', title: Tested worker protocol and Pyodide pin }
---

# Run the two Datasette paths on Linux

Run from the Mouseparty checkout. Both tests create a synthetic film database. They test opposite ends of the deployment decision: running Datasette **at the host** vs. giving a **browser** its own copy of the data.

## 1. Host: unchanged WASI payload, Rust runtime

If `build/python/dist` is populated, just run `python3 design/feasibility/probes/wasi.py build/python/dist`. Otherwise the tested shortcut needs `curl`, `sha256sum`, `7z` (with HFS support), Python 3 and Wasmtime (tested with 44.0.0). From a fresh checkout:

```sh
mkdir -p build/probes
curl -fLsS https://github.com/ftrain/mouseparty/releases/download/v3.5.1/MouseParty-macos-arm64.dmg -o build/probes/release.dmg
echo 'c7e6558255a0e19368c64d316a91b4283dc8a8e225ee8b3a3650bb2f0066bfbb  build/probes/release.dmg' | sha256sum --check && \
  7z x -obuild/probes/extracted build/probes/release.dmg 'Mouse Party/Mouse Party.app/Contents/Resources/python/*'
runtime="$PWD/build/probes/extracted/Mouse Party/Mouse Party.app/Contents/Resources/python"
python3 design/feasibility/probes/wasi.py "$runtime"
```

Observed: `PASS: SQL 200 [{'title': 'Cléo from 5 to 7'}], download 403`. The ~73 MiB unpacked Python resource includes a WASIp1 binary, standard library and packages; its `VERSION` matched `scripts/build_python.sh --stamp`. No Mac executable is launched. [The probe](wasi.py) passes real requests to the repository's [datasette.py](../../../datasette.py) over stdio using Wasmtime's CLI. This exercises the guest ABI, not a Rust crate embedded in Rune. CLI preopens point at **temporary copies**, not production read-only mounts. A Rust embedder needs explicit filesystem rights, bounded input/output and memory, cancellation, lifecycle management and tests for adjacent-file denial. The source recipe and licenses remain relevant for redistribution; using a DMG is a test shortcut, not a proposed dependency resolver.

## 2. Browser: Rust client → Lite worker → Datasette

Prerequisites: Rust with `wasm32-unknown-unknown` installed, `uv`, a system Chromium on PATH, network access to Pyodide's CDN and Python packages, and a checkout of Lite at `f49662beda68d787ce9c8f24f6eb0b452d90f2c7`.

```sh
uv run --with playwright design/feasibility/probes/lite/run.py ~/archive/simonw/datasette-lite
```

Observed: `PASS: HTTP 200 [{"title": "Cl\u00e9o from 5 to 7"}]`. [Rust](lite/client.rs) issues a GET and checks returned bytes; a [small JS adapter](lite/index.html) posts it to the **unmodified module worker**, copies the response back to Rust, and checks the parsed JSON. The worker uses Pyodide 314.0.8 and Datasette 0.65.5 (the Mouseparty pin). The [driver](lite/run.py) starts a loopback fixture server and terminates on a wrong response or timeout. This bypasses Lite's service worker and iframe deliberately: it tests a data API. It does not establish native Rust embedding, a fully offline bundle, compatibility with Rune's JS runtime, or two-party synchronized navigation. The worker downloads third-party dependencies; pin/vendor them before making offline claims.

## 3. Upstream Go tests: distinguish compile failure from runtime failure

```sh
go test -run '^$' .  # unmodified checkout: undefined native* on Linux
go test ./apps
# With the extracted runtime above, if the standard path is unoccupied:
mkdir -p build/python
ln -s "$runtime" build/python/dist
python3 design/feasibility/probes/linux.py --desktop
```

[linux.py](linux.py) adds seven AppKit no-ops through a temporary `go -overlay` file. `--desktop` also attempts a Wails production link using WebKitGTK 4.1; omit that flag for tests alone. The binary is discarded. The selected `httptest`/WASI tests require no tunnel, media server or public listener. **Check the verbose test results:** database tests can skip without the runtime. These are compile and feature diagnostics, not a runnable Linux distribution or a substitute for a sandbox.

Observed on Linux x86-64, Go 1.27.4: the unmodified root package failed on the seven missing `native*` symbols. With the overlay, the production link succeeded. With the extracted runtime, all eight selected tests passed, none skipped: `TestDatasetteOnlyReads`, `TestOnlyFile`, `TestDatasettePages`, `TestDatasetteStatic`, `TestDatasetteOddNames`, `TestDatasetteBigTable`, `TestChatJoinSayLeave`, `TestOnlyTheKeyGetsIn`. These tests use the original Go/wazero implementation. The two Wasmtime CLI checks above are a smaller, separate result.

For a fast compatibility baseline **without the DMG**, run `python3 design/feasibility/probes/linux.py --social`. Both `TestAppsOverChat` (two real WebSocket clients, shared activities, late-join state) and `TestSocketGraphIsTypes` (Go command vocabulary vs. viewer TypeScript) passed here. This is a starting fixture for a future Rune panel or alternate guest; it does not render a guest page.

Linux helper availability is separate from the Wails link. Both [MediaMTX v1.21.3](https://github.com/bluenviron/mediamtx/releases/tag/v1.21.3) and [cloudflared 2026.10.3](https://github.com/cloudflare/cloudflared/releases/tag/2026.10.3) publish Linux amd64 and arm64 binaries at the versions in [`fetch_tools.sh`](../../../scripts/fetch_tools.sh); the script only selects Darwin arm64. This machine's distro ffmpeg 9.0.2 reports the libx264 and libopus encoders, RTSP output muxer, drawtext, subtitles and showcqt filters. Availability is encouraging for a dev build, not a playback or packaging test: e.g. [`stream.go`](../../../stream.go) still selects macOS's Menlo font for title cards.

### Host smoke test (Linux amd64)

With the `build/python/dist` link above, `bwrap` with private user/network namespace support, Debian-style library paths, distro ffmpeg and the GTK/WebKit build dependencies:

```sh
mkdir -p build/probes/linux-tools
curl -fsSL https://github.com/bluenviron/mediamtx/releases/download/v1.21.3/mediamtx_v1.21.3_linux_amd64.tar.gz -o build/probes/linux-tools/release.tar.gz
echo '0a2ea7eba96023631efa1bea7804823a9145b043b6646a8a05798d4ac24ed635  build/probes/linux-tools/release.tar.gz' | sha256sum --check && \
  tar xzf build/probes/linux-tools/release.tar.gz -C build/probes/linux-tools mediamtx
python3 design/feasibility/probes/linux.py --selftest
```

Runs upstream `--selftest` on a private loopback network with a temporary home, no tunnel or router access. The first integrated run served keyed guest pages, HLS playlist, PDF and Datasette HTML, and reached the WHEP handler; its countdown never went live before the check expired. A second run **passed**, including publishing the countdown with ffmpeg, after enabling the test's previously discarded player log. The cause of the first miss is undiagnosed. This verifies neither reliable playback in a remote browser nor Linux confinement of individual guest-facing processes: the entire test lives in *one* outer namespace.
