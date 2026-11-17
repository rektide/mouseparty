---
type: Assessment
title: Rune feasibility for a Mouseparty host app
description: Evidence-based assessment of anomalyco/rune (dev @ 0d00d3f2, 2026-11-17) as the UI/runtime base for a Rust Mouseparty host — targets, browser/HTML scope, native capabilities, process/network/storage feasibility, the rune-video/PR-50 boundary, and a first integration slice.
resource: /design/feasibility/rune-feasibility.glm53m.md
status: draft
tags: [rune, mouseparty, feasibility, host-app, integration]
generated:
  at: 2026-11-17
  by: model:zai-coding-plan/glm-5.3#max
sources:
  - resource: opencode://ses_e1478f9d0ffe9cDKSqsN22HfU
    title: Independent Rune read-through for Mouseparty feasibility (this assessment's session)
  - resource: file:///home/rektide/a/a/rune/README.md
    title: Rune overview, agent usage, pre-release pinning, Windows toolchain notes
  - resource: file:///home/rektide/a/a/rune/WEB.md
    title: "Rune web target: WebGPU canvas, mounting, fonts/input, scope limits"
  - resource: file:///home/rektide/a/a/rune/crates/rune/README.md
    title: Core crate docs — view API, app lifetime, dialogs, drop files, video/audio, overlays, headless tooling
  - resource: file:///home/rektide/a/a/rune/Cargo.toml
    title: Workspace members, vendored patch table, profiles, lints
  - resource: file:///home/rektide/a/a/rune/crates/rune/Cargo.toml
    title: Feature flags (webgl, native-file-dialog, ogg-audio, dbus, metal-backend), dependencies
  - resource: file:///home/rektide/a/a/rune/crates/rune-kit/README.md
    title: Themable component kit (@opencode/ui names)
  - resource: file:///home/rektide/a/a/rune/crates/procwatch/Cargo.toml
    title: procwatch — self-monitoring with loopback JSON endpoint (dev tool, not a process manager)
  - resource: file:///home/rektide/a/a/rune/examples/video-tiles/README.md
    title: NativeSurface producer/view contract, zero-copy IOSurface import, benchmarks
  - resource: file:///home/rektide/a/a/rune/crates/rune-notify/README.md
    title: System notifications (macOS UNUserNotificationCenter, Windows toasts)
  - resource: file:///home/rektide/src/mouseparty-rune/CLAUDE.md
    title: Current Go/Wails Mouseparty architecture — MediaMTX, cloudflared, ffmpeg, seatbelt, wazero, viewer pages
related:
  - /design/feasibility/probes/wasi.py
  - /design/feasibility/probes/lite/run.py
verification:
  claims cite files in the rune checkout at rev 0d00d3f2 (read-only); no rune code was run or modified
---

# Rune feasibility for a Mouseparty host app

Assessed `~/a/a/rune` (anomalyco/rune, `dev` @ `0d00d3f2`, 2026-11-17, stable Rust
1.99.1) read-only, for the Rust-Mouseparty effort this repo explores. Companion
strands already probed in this tree: [`probes/wasi.py`](/design/feasibility/probes/wasi.py)
(Datasette/CPython under Wasmtime) and [`probes/lite/`](/design/feasibility/probes/lite/run.py)
(datasette-lite in a browser worker) — this note is the Rune strand.

## Verdict in one paragraph

Rune is a credible **host-window framework** for a Rust Mouseparty: native
GPU-rendered UI on macOS (Metal backend, zero-copy video path, file drop, dialogs,
menus, notifications, accessibility, and unusually strong headless agent
verification). It deliberately ships **no host services** — no networking, no
process management, no storage — so everything Mouseparty's Go half does today
(HTTP site, WebRTC proxy, chat, child-process supervision, seatbelt sandboxing,
SQLite party file, wazero WASM) remains app-owned Rust that must be built or
ported beside the UI. Rune's browser target is **not** the viewer side for
Mouseparty: it requires WebGPU and has no native audio/video and no networking
layer, while Mouseparty's promise is "friends watch in any browser." The viewer
pages should stay plain HTML/TS (or datasette-lite-style WASM), with Rune owning
the Mac host window.

## What Rune is (and how you depend on it)

- Native GPU-rendered Rust UI for **macOS, Windows, Linux and the web**, with its
  own renderer (one instanced uber-quad pipeline), text (Parley + custom
  rasterizers), layout (Taffy 0.15 flexbox/grid), and springs
  ([README](file:///home/rektide/a/a/rune/README.md)).
- Pre-release; consumers pin `rune = { git = "https://github.com/anomalyco/rune", rev = "<dev rev>" }`
  and **must mirror Rune's `[patch.crates-io]` table** (vendored `wgpu-hal`,
  `winit`, `parley_engine`, `accesskit_windows`, `accesskit_macos`) in their own
  workspace root — a dependency's patches don't propagate
  ([crates/rune/README.md, "Depending on Rune"](file:///home/rektide/a/a/rune/crates/rune/README.md)).
  This is a standing integration cost: every Rune bump means re-checking its
  patch table.
- Upstream takes **no pull requests** ("rune is agent-driven… have your agent
  open an issue", [PR #50 close comment, 2026-11-06][pr50]); plan on consuming
  `dev` revisions, not landing changes.
- Stable toolchain (`rust-toolchain.toml` pins 1.99.1 + clippy/rustfmt +
  wasm32-unknown-unknown), edition 2024, workspace lints deny `unwrap`/`expect`.

[pr50]: https://github.com/anomalyco/rune/pull/50

## UI/runtime targets

| Target | State (evidence) | Mouseparty fit |
| --- | --- | --- |
| macOS native | First-class: own Metal backend (`metal-backend`, default), precompiled uber shader, ~300 ms cold launch to content, reopen/Dock/URL-scheme handling, `rune-mac` relaunch tooling | **Primary target** — Mouseparty is macOS-only today |
| Windows native | Builds (MSVC or GNU+binutils); Media Foundation video, WASAPI audio, toasts | Future option; nothing blocks |
| Linux native (X11/Wayland) | Builds (`libxkbcommon`/`libwayland-dev` needed); overlays via layer-shell; **video playback unsupported upstream** ("every clip reports `Failed("video playback isn't supported on this platform yet")`", [video section](file:///home/rektide/a/a/rune/crates/rune/README.md)) | Host window would work; video gap is exactly PR-#50 territory (below) |
| Web (wasm32) | Experimental: one WebGPU canvas, same views/layout/glyphs; `webgl` feature flag exists as WebGL2 fallback (`platform/web.rs` `try_backend`) though [WEB.md](file:///home/rektide/a/a/rune/WEB.md) still scopes the target as WebGPU-only | Not the viewer page; possible future extras surface |
| iOS | Mentioned but "not wired" (winit owns the scene delegate; URL links don't reach it) | Out of scope |
| Headless (all OSes) | Every app runs headless: PNG snaps, element-tree JSON, scripted input, fixed virtual clock, screen-recording-style videos | **Major asset** — see verification below |

## Browser/HTML target, precisely

From [WEB.md](file:///home/rektide/a/a/rune/WEB.md):

- Same Rust views render through WebGPU into one canvas; runtime stays on the
  browser main thread; no COOP/COEP/SharedArrayBuffer needed; served as static
  files (`scripts/web-build.sh`), `--dev`/`--watch` loops ~2 s rebuilds.
- Mounting is flexible: own canvas (`canvas_id`), full-viewport SPA route
  (`full_viewport`), inline article with `fit_content_height`, or the whole page
  (`web_document`) with a **DOM mirror** (landmarks, headings, links as real
  `<a>`) for screen readers, selection, crawlers; `--html` can emit a static
  first frame readable without WebGPU.
- Works: browser-fetched images (`ImageCache::url` — browser codecs, so
  WebP/AVIF), fonts, clipboard (user-activation rules apply), `open_url`,
  safe-area insets, `AppHandle::quit` unmounting, Luau scripting via a WASI-SDK
  build (precedent for nontrivial native code on the page).
- Explicitly absent on web ([WEB.md, "Scope and verification"]): **native
  audio/video, networking layer, overlays, extra windows, native menus,
  drag-out, hot patching**; desktop host paths for config types are unused.

Consequence for Mouseparty: the guest experience (HLS/WebRTC video, chat
WebSocket, games) cannot be a Rune web app. The existing `host_web/` plain-HTML
viewer — or a datasette-lite-style WASM component like the parent's
`probes/lite/` — remains the right guest side. Rune's web target is only worth
considering later for auxiliary surfaces (a WebGPU control panel embedded in
the viewer page, say), and even then WebGPU-only support cuts against
"any browser."

## Native host API and capabilities, mapped to Mouseparty needs

All from [crates/rune/README.md](file:///home/rektide/a/a/rune/crates/rune/README.md)
unless noted.

| Mouseparty need (today's Go/Wails) | Rune support | Notes / gaps |
| --- | --- | --- |
| Window + list UI (drag movies in, reorder, icons, states) | View API `el/row/col/text`, rune-kit components (Button, Switch, Select… themable), rich text, virtual rows, sticky bars | Kit follows `@opencode/ui` names; a small theme can be derived from three tokens |
| Drag files **into** the window | `.on_drop_files` + `.accepts_extensions`, drop-hover/rejected handlers, drop-point | macOS/Windows/X11 only; Wayland reports no file drags in winit 0.30; web has no paths. Fine for a macOS-first host |
| Drag files **out** (future niceties) | `.drag_out` | macOS only |
| Play/Stop, Copy-link buttons with pending states | Handlers + `Latest<T>` off-main-thread requests with debounce; `Tx::copy` to clipboard | Clipboard write is a shell request; fine |
| Menus (Party menu, Edit) | Native `tx.popup_menu` (NSMenu / TrackPopupMenuEx) + macOS `standard_menus()` | Linux popup unsupported → in-app fallback needed there; macOS host is fine |
| Settings window | `App::window` secondary windows with own state; `fit_content_height`; window resize/center on springs | Multi-window retained trees share one GPU device |
| Notifications while behind | [rune-notify](file:///home/rektide/a/a/rune/crates/rune-notify/README.md): macOS UNUserNotificationCenter, Windows toasts, `app_focused()` gating | Matches current "only while behind" behavior |
| Hosting status from background services | `AppSpec::setup` waker, `poll`/`wake_after`, `AppHandle::update` from any thread, `Live` values, `rune::Pixels` thread-safe image slot | The load-bearing bridge; see slice below |
| Host watching/previewing the movie | `Player::file` (AVFoundation; IOSurface → `NativeSurface`, no CPU copy), audio clips, seek/rate/loop; poster frames; ≤4 parked decoders | Windows via Media Foundation; **Linux unsupported upstream** |
| Custom decode pipeline (if ever needed) | `NativeSurface::push` from a decoder thread (BGRA/NV12, BT.601/709), `push_bytes` off-macOS | Producer/view contract documented in [video-tiles](file:///home/rektide/a/a/rune/examples/video-tiles/README.md) |
| Voice-call mic (host side) | `Recorder` (macOS only; levels as `Live`) | Guests' mics ride the browser page, unchanged |
| Accessible window | AccessKit on all natives; tree only built when a client asks; `--tree` exposes it to tests | Better than the current Wails window story |
| Agent/self verification | Headless `--snap/--tree/--script/--video/--strip`, scripted drops/keys/menus/dialogs, virtual clock, `--desktop` multi-window composites | This is Rune's headline feature for agent-built apps |

Not provided by Rune (by design): HTTP/WebSocket serving or clients (the crate
has **no HTTP client** — `ImageCache::url` on native takes only `file:` paths;
tokio appears only under the Linux `dbus` feature), child-process spawning or
supervision, webview embedding, persistent storage, crypto.

## Process, network, storage feasibility

These are app-owned in a Rust rewrite; nothing in Rune blocks them, and its
thread/waker model is built for exactly this shape of work:

- **Processes** (MediaMTX, cloudflared, custom static ffmpeg, `sandbox-exec`):
  plain `std::process::Command` on background threads. Rune imposes no event-loop
  constraints on them (its own `Latest` workers and `Recorder`/`Player` threads
  set the pattern: work off-thread, land answers in `poll`). The Go app's shell
  watchdog ("die with the app") must be re-implemented — e.g. thread-scoped
  child reaping in `App::on_close`/drop, or a supervisor task; `procwatch`
  (loopback JSON self-monitoring) is a dev tool here, not a manager.
  Seatbelt: the "second copy of the app, `--site`" pattern ports directly
  (`std::env::args` dispatch before any UI), since Rune only starts after the
  app's own `main` chooses a role.
- **Network** (viewer site + reverse proxies + chat WS + WHEP/WHIP + ICE):
  bring tokio/axum (or equivalent) on background threads; wake the UI through
  the setup waker / `AppHandle::update` / `Latest`. One wakeup per landed answer
  is the intended pattern (`Latest` "one wake per answer"; idle windows stay
  idle). NAT-PMP/UPnP and quick tunnels are crates/child processes as today.
  `Player::url` (+ headers) exists as a *consumer* for authenticated media on
  macOS — potentially useful for "add download link" previews — but is not a
  server.
- **Storage**: the one-file `party.sqlite` pattern (journal DELETE, temp in
  memory, optional encryption) ports to `rusqlite`. The Go version's adiantum
  VFS encryption (ncruces/go-sqlite3) has no off-the-shelf Rust equivalent —
  SQLCipher or a custom VFS is a real work item and should be sized before
  promising feature parity on "Use Encryption."
- **WASM tools** (pandoc, Datasette/CPython-WASI under wazero today): the Rust
  equivalent is Wasmtime; the parent's `probes/wasi.py` already probes exactly
  this offline. Rune is uninvolved — keep it off the UI thread.

## PR #50 and the rune-video boundary

- [PR #50 "Linux Player: GStreamer, libmpv, and libVLC backends"][pr50]
  (ggallovalle, base `dev`, head `linux-player`) adds opt-in Linux `Player`
  backends, volume/track-selection/`decode_frame_at` scrub previews, and
  `video-player` example changes. **Closed unmerged 2026-11-06** with the
  no-PRs policy comment. Issues, not PRs, are the upstream route.
- The user's `~/src/rune-video` (anomalyco/rune clone at the same `dev` tip)
  has uncommitted working-copy changes in `crates/rune/Cargo.toml`,
  `src/video/{mod,tests,unsupported}.rs`, `examples/video-player/Cargo.toml` —
  the same surface PR #50 touched. That is the user's active area; this
  assessment treats `Player`/`NativeSurface`/video internals as **consumed API
  only** and proposes nothing that modifies or depends on modifying Rune
  internals.
- Planning consequence: Linux video remains an upstream gap; a Mouseparty
  macOS host needs nothing from it, and any Linux-host ambition should wait on
  the user's rune-video direction rather than fork Rune's video stack.

## Proposed first integration slice: the party panel window

A small consuming crate in this repo (e.g. `crates/party-panel`) that proves the
four load-bearing bridges with **zero changes to Rune**:

1. **Window + drop-in list.** AppSpec window with rune-kit-styled rows; root
   `.on_drop_files` + `.accepts_extensions` for the media extensions Mouseparty
   treats as playable (from today's `media.go` list); rows keyed by path with
   icon/text; Backspace removes the selected row. Play/Stop toggle and
   Copy-link button whose label reflects a pending state (`Latest`-shaped).
2. **Background-service bridge.** In `AppSpec::setup`: start a loopback HTTP
   server (axum or `std::net` + thread) on a free port and spawn one child
   process (`ffmpeg -version` is enough — real MediaMTX/ffmpeg hosting is a
   later slice); surface port/pid/status into the UI via the setup waker +
   `poll`, and one `Live` value for a ticking status line. On `App::on_close` /
   `Tx::quit`: stop the server, reap the child; a test asserts no orphan.
3. **Headless verification, from day one.** A `--script` JSON that taps Play,
   drops two files (`drop_files` events), taps Stop; plus `--tree` assertions
   that the list state and button labels are as expected, and a `--snap`
   regression PNG. This exercises Rune's agent-verification story — the actual
   reason to choose Rune over Wails-equivalents — as a CI-able check
   (`cargo test` + the tooling flags; debug builds always have them).
4. **Explicit non-goals of the slice.** No video playback (rune-video
   boundary), no viewer-page changes, no sandbox, no SQLite, no tunnel — each
   is a later slice with its own proof.

Acceptance: `cargo run -p party-panel` opens the panel; dragging two files in
lists them; Play starts the child and flips the button; Stop reaps it; quitting
leaves no process and no bound port; the headless script reproduces all of it
without a display.

Why this slice first: it validates file drop (macOS), the thread→UI wake path,
child-process ownership, and scripted verification — the four things every
later Mouseparty feature sits on — while touching only Rune's public API.

## Open questions

- Do we want the host window to ever *play* the movie (Rune `Player`) rather
  than only host it? Today's host window doesn't; if it ever does, the
  rune-video boundary makes that a coordination point, not an independent move.
- Encryption parity (adiantum VFS → SQLCipher/custom VFS) needs sizing before
  any parity claim.
- Rune's `[patch.crates-io]` mirroring means we track `dev` closely; decide the
  update cadence and record each pinned rev here.
