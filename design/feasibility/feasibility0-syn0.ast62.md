---
type: Assessment
title: A Linux Mouseparty host with a Rune window
description: Which parts already work on Linux, two working Datasette experiments, and a reversible first implementation choice.
resource: /design/feasibility/feasibility0-syn0.ast62.md
status: draft
generated: { by: 'model:openai/gpt-6.2-astra', at: 2026-11-17 }
sources:
  - { resource: 'opencode://ses_e1479787affey7NRxOch4DM678', title: User direction, runtime experiments and synthesis }
  - { resource: /design/feasibility/mouseparty-components0.sol62hi.md, title: Sol's component and compatibility audit }
  - { resource: /design/feasibility/rune-feasibility.glm53m.md, title: G's Rune host and web integration review }
  - { resource: /design/feasibility/pyodide0.glm53m.md, title: G's Pyodide and Datasette Lite assessment }
  - { resource: /design/feasibility/probes/README.md, title: Linux, Wasmtime and browser test recipes and outcomes }
---

# A Linux Mouseparty host with a Rune window

**Build a Rune control window if that's the interface you want; keep the Go host and stock browser guests for the first version.** A Linux port of the existing Wails app is the shortest route to the current product. A new UI does not oblige us to rewrite the server, media pipeline or database viewer. The experiments change the priority: Linux guest isolation and packaging are release work; getting real Datasette to execute is already demonstrated.

## What is an app instance here?

| Piece | What actually runs | What the Rune route changes |
|---|---|---|
| Controller | Web UI (`web/window` → `frontend`) inside a native Wails webview. It calls a privileged Go `App`, which owns queue, paths, settings and process lifecycle. | Redraw the host controls with Rune and replace the Wails bridge with a narrow private local control channel. The webview's HTML is not a native Rune view. |
| Party | Go serves keyed HTTP, a WebSocket, state, chat and activities. On sandboxed macOS a second invocation of the same executable (`--site`) serves guests and is handed document FDs. ffmpeg publishes to MediaMTX; cloudflared is optional. | Keep this machinery behind the controller. Extract a headless entry point: `--site` is a child of the privileged host and cannot be launched as a standalone server. Keep privileged commands off the guest routes. |
| Guests | Ordinary browsers load `web/viewer` → `host_web`, including the media player and lazy-loaded game/editor modules. No Rune or Mouseparty install. | Keep the keyed URL, routes and WebSocket behavior and existing guests continue to be the target. A native host preview is a separate video problem from streaming to guests. |
| Document tools | Go runs CPython+Datasette and Pandoc as local WASI guests in wazero. Settings and saved activities use a separate SQLite implementation. | Keep the tools with Go, or consume the same Python resource from a native Rust WASI host. Shared database navigation still belongs to the party session. |

## Which route buys what?

- **Linux Wails port — lowest implementation cost.** Seven temporary AppKit placeholders allowed a production Wails link. An isolated Linux host served the existing guest page, HLS playlist and Datasette HTML. The countdown publish check missed once and passed once; neither a full desktop session nor remote playback was tested. Port the shell behavior (drop/open/menu/relaunch/update), package the Linux helpers and fonts, and replace the per-process Seatbelt policies. Today Linux otherwise falls back to in-process guest handlers and unboxed helpers.
- **Rune controller + Go host — best fit for a new app that remains compatible.** Native Linux/Mac Rune views, file drops and headless UI testing are a good match. Rune supplies neither the media service nor the authority that keeps a session consistent. The real new work is command/event IPC, process teardown, and the controller UI; an app crate can consume a pinned Rune revision without changing the framework. Protocol compatibility with guests is much cheaper if the original host still emits the messages. Existing encrypted party files are a separate compatibility obligation if storage is replaced later.
- **Web-first — cheap as a client, expensive as the host.** A new guest can speak to an existing keyed session; a browser controller could operate a local helper that owns the files. A static page alone cannot listen for incoming HTTP/WebSocket connections or supervise native ffmpeg/MediaMTX. A browser-hosted remake would have to choose different discovery/signaling, media distribution and file-sharing semantics. Rune's canvas web target doesn't make the existing DOM apps or Datasette iframe native elements; browser APIs and DOM integration would remain app-owned.
- **All-Rust reimplementation — choose it for specific changes.** Reproducing per-user activity views, synchronized document history, persistence/encryption, media supervision and guest isolation is a substantially larger job than changing the window. Keep the original code/tests as a behavioral reference; choose which of wire, file-format and feature compatibility actually matter.

## Datasette: choose where the file lives

**Original semantics:** One database stays on the host, mounted as a single read-only file. Guests get Datasette pages and query results, not a raw database download. Python calls Datasette's ASGI app and speaks marked JSON lines on stdio; Go owns HTTP, the key/token, shared navigation and Back. Queries can disclose rows—refusing a file download is not row-level access control.

**Native result:** The Python payload extracted from v3.5.1's Mac DMG worked unchanged on Linux. Wasmtime's CLI answered real SQL with `200` and the `.db` route with `403`. The original Go/wazero implementation also passed its table, HTML, assets, odd-name, large-table and restriction tests. If a standalone Rust component is desirable, package the pinned WASI payload plus a Rust host for the existing request protocol. Start with `open(selected_file)`, `request(method,path,query,headers)` and `close`, then enforce the one-file mount, limits, shutdown and cancellation. A CLI smoke test is not a production Rust embedding.

**Browser result:** A Rust `wasm32` client sent a GET to an unmodified Datasette Lite worker via JS, using the same Datasette 0.65.5 pin, and got the expected SQLite row. The browser first fetched the **whole fixture DB**. Lite runs Pyodide/Emscripten with JS glue; it is not a WASIp1 module to drop into wazero/Wasmtime. Lite is promising for a personal or explicitly shareable database. Its normal per-tab download, mutable local copy, network-fetched runtime/dependencies and unshared navigation are different from the current party contract. For a static web client that must leave the DB on the host, call the host's existing HTTP API instead. The two probes validate different things, not interchangeable engines.

## First milestone

Make a small **Linux Rune party panel**: drop a fixture `.db`, start a loopback Go companion over private IPC, show its ready state and guest link, and stop it cleanly. On the guest side, open that link in **two stock browser tabs**. Navigate to a table in one; verify the other follows, Back works, `.db` is refused, and quitting removes the session. Use a document-only launch path (current `Host.Start` also starts MediaMTX). This proves the new window, compatibility and data boundary together, before the video decision. Keep the app in a development workspace, consuming `~/a/a/rune`; [Rune PR #50](https://github.com/anomalyco/rune/pull/50) and the user's `~/src/rune-video` can progress independently. If the web guest is the more interesting product, invert the milestone: make a new web guest do one activity such as Polls against the original host, including a late join.

**Start here:** [reproduction commands and measured results](probes/README.md). The fastest baseline is `python3 design/feasibility/probes/linux.py --social`. For implementation detail see [Sol's component map](mouseparty-components0.sol62hi.md), [G's Rune integration assessment](rune-feasibility.glm53m.md), and [G's Pyodide/Lite research](pyodide0.glm53m.md).
