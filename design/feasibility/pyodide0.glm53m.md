---
type: FeasibilityResearch
title: Pyodide and Datasette Lite against Mouse Party's Datasette requirements
description: Evidence from the pyodide and datasette-lite sources on versions, limitations and fit; where each could sit in Mouse Party, and the Rust/browser/native option landscape.
resource: /design/feasibility/pyodide0.glm53m.md
status: draft
generated: { by: 'model:zai-coding-plan/glm-5.3#max', at: 2026-11-17 }
sources:
  - { resource: 'opencode://ses_e1478f9ccffe8sihh7Ceyr02Xs', title: 'Source research pass over the pyodide and datasette-lite archives and version registries' }
  - { resource: /datasette.go, title: 'Host-side Datasette protocol, sandbox mounts and base URL' }
  - { resource: /datasette.py, title: 'Datasette settings, no-socket asyncio and threadless ASGI bridge' }
  - { resource: /scripts/build_python.sh, title: 'CPython 3.15.2 WASI build with SQLite 3.54.1 and zlib' }
  - { resource: /datasette_test.go, title: 'The page, SQL and read-only behaviors the tests require' }
  - { resource: /wasm.go, title: 'wazero runtime envelope: memory cap, mounts, compile cache' }
  - { resource: 'file:///home/rektide/archive/pyodide/pyodide', title: 'Pyodide source at 793b02b7 (314.1.0-dev): build flags, docs, changelog' }
  - { resource: 'file:///home/rektide/archive/simonw/datasette-lite', title: 'Datasette Lite at f49662be: webworker.js, README, service worker' }
  - { resource: 'https://datasette.github.io/pysqlite3-wasm-wheels/wheels.json', title: 'SQLite wheel matrix for Pyodide 314.0.8, 3.21.0-3.54.1' }
  - { resource: 'https://pypi.org/pypi/datasette/json', title: 'Datasette release history through 1.0a43 / 0.65.6' }
  - { resource: 'https://api.github.com/repos/pyodide/pyodide/releases/tags/314.0.8', title: 'Pyodide 314.0.8 asset sizes' }
---

# Pyodide and Datasette Lite against Mouse Party's Datasette requirements

**Assessment:** Pyodide cannot replace the in-app engine — its Python is welded to a
JavaScript host, and the app's runtime ([wasm.go](../../wasm.go)) is wazero, which runs
WASI modules and provides no JavaScript. Datasette Lite is real Datasette run *per
browser*, which for Mouse Party means shipping the whole database to every viewer —
exactly what the design forbids. The interesting placements are narrower: Lite as the
parent experiment's reference architecture (a Rust WASM client already speaks to its
JSON API in [probes/lite/](probes/lite/)), and Pyodide-in-the-host-webview as a
heavier substitute for the working CPython-WASI engine. Neither beats what
[build_python.sh](../../scripts/build_python.sh) already produces on the
requirements that matter most: read-only confinement, zero guest downloads, and a
pure-Go app.

## What Mouse Party actually requires of Datasette

From the code, not aspiration. [datasette.go](../../datasette.go) runs one CPython
per open database, inside the app under wazero; [datasette.py](../../datasette.py)
is the Python side.

- **Real Datasette pages**, not a lookalike: HTML table pages with foreign-key row
  links (`href="/k/data/t/films/directors/2"`) and the JSON API
  (`?sql=...&_shape=array`) are asserted by
  [datasette_test.go](../../datasette_test.go).
- **The file never leaves the Mac.** Pages render host-side and are proxied under
  `/<key>/data/<token>/`; `allow_download` is off; the database is not served to
  viewers.
- **Read-only and confined.** The database opens immutable
  (`immutables=[db]`, SQLite URI `?immutable=1`); Python's filesystem is its own
  files plus a synthetic one-file folder ([`onlyFile`](../../datasette.go)); SQLite
  temp goes to memory (`PRAGMA temp_store = memory`) because nothing is writable; a
  secret placed beside the database must be unreachable
  ([TestDatasetteOnlyReads](../../datasette_test.go)).
- **Threadless, socketless, serialized.** `num_sql_threads: 0`; a no-socket asyncio
  selector (`NoSockets`); one request at a time under a mutex; `sql_time_limit_ms`
  3000, `max_returned_rows` 1000; a stuck run is killed and restarted.
- **Shared viewing.** Everyone sees the same page: the injected
  `datasetteFollow` script posts `{type:'went'}` and follows `{type:'go'}`, with a
  shared Back. Any replacement must keep serving Datasette's own HTML/CSS/JS from the
  host under a base URL.
- **Envelope.** 2 GB max WASM memory ([wasm.go](../../wasm.go)); compiled module
  cached to `~/Library/Caches/Mouse Party/wasm` so later sessions start in under a
  second; a 2-minute first-start budget and a 20-second per-page budget. Pins:
  Datasette **0.65.5** (all pure Python), CPython **3.15.2** built for
  wasm32-wasip1 with SQLite **3.54.1** and zlib, via WASI SDK 33
  ([build_python.sh](../../scripts/build_python.sh)).

## Pyodide, from the source

Archive at `~/archive/pyodide/pyodide`, commit `793b02b7` (2026-11-17),
version **314.1.0-dev** — past the current release **314.0.8** (2026-11-13). Since
314.0.0 (2026-06-09) the project versions by Python minor: Python 3.14, Emscripten
5.0.3, and `sqlite3`/`lzma` bundled into the core (changelog,
`docs/project/changelog.md`).

- **It is Emscripten, not WASI.** The main module links
  `-sENVIRONMENT=web,webview,worker,node,shell` (`Makefile.envs`), and
  `pyodide.asm.mjs` — "the JavaScript half of the main binary … the
  JavaScript/Wasm runtime interop APIs" — is mandatory
  (`docs/usage/downloading-and-deploying.md`). Nothing in `src/js` mentions WASI.
  A JS host is part of the ABI; **wazero cannot instantiate it**. This closes the
  door marked "drop Pyodide into wasm.go".
- **Same wasm constraints Mouse Party already lives with:** no threads, no
  multiprocessing (`docs/usage/wasm-constraints.md`); sockets only experimentally in
  Node (`docs/usage/socket.md`). Datasette Lite therefore also forces
  `num_sql_threads: 0`.
- **SQLite is pinned old: 3.39.0 (2022)**, matching the Emscripten ports tree
  (`cpython/Makefile` downloads `sqlite-autoconf-3390000`; ABI tables in
  `docs/development/abi/314.md` and `315.md` both say 3.39.0). Mouse Party ships
  3.54.1. The escape hatch is the
  [pysqlite3-wasm-wheels](https://datasette.github.io/pysqlite3-wasm-wheels/) set —
  verified today: wheels for SQLite 3.21.0–3.54.1, `cp314` /
  `pyemscripten_2026_0_wasm32`, built against Pyodide 314.0.8, ~422 KB each,
  swapped in by `sys.modules["sqlite3"] = pysqlite3`.
- **Size and memory.** `pyodide-core-314.0.8.tar.bz2` is **6.76 MB** compressed
  (GitHub release asset); the full distribution is 337 MB. Heap ceiling is
  `-sMAXIMUM_MEMORY=4GB` with growth (`Makefile.envs`); a `maxMemoryBytes` cap is
  being added (unreleased changelog). Everything — Python heap, stdlib zip, the
  database — lives in that one heap; the stdlib mounts from
  `python_stdlib.zip` via ZipImporter.
- **Startup.** Downloads plus instantiation plus package installs; Pyodide has a
  memory-snapshot facility (`makeSnapshot`/`loadSnapshot` in `src/js/snapshot.ts`),
  bound to a `BUILD_ID`, but Datasette Lite does not use it — its cost model is CDN +
  HTTP cache, with a loading screen of parallel steps. I found no sourced
  startup-time numbers in either repo; treat "seconds, not minutes" as unverified
  here.
- **Self-hosting is supported**: serve the core files yourself and point `indexURL`
  at them (`docs/usage/downloading-and-deploying.md`) — relevant if the app ever
  served Pyodide from the host instead of a CDN. Offline pinning means vendoring
  the core plus every wheel and the lockfile.

## Datasette Lite, from the source

Archive at `~/archive/simonw/datasette-lite`, commit `f49662be` (2026-11-16). Live
architecture (`README.md`, `webworker.js`, `sw.js`, `shell.js`):

- One module Worker per tab runs Pyodide **314.0.8 from jsDelivr** (a pinned
  constant), installs micropip and setuptools, then **Datasette from PyPI at
  runtime** — defaulting to the newest preview release (today **1.0a43**), with
  `?ref=` for stable (**0.65.6**) or any version, branch, or commit (GitHub zips
  via a CORS proxy). Databases are **fetched whole over CORS into the in-memory
  filesystem** (`fetch_bytes` → `open(name, "wb")`), and opened *not* immutable:
  `Datasette(names, ...)`. `?csv=`, `?json=`, `?parquet=`, `?sql=` build a
  `data.db`; `?sqlite=` swaps SQLite via the pysqlite3 wheel.
- Requests to `/app/<session>/…` are intercepted by a **service worker** and
  relayed through the shell to the worker, which calls Datasette's ASGI app
  directly — so Datasette's own JS/CSS, POST forms, CSRF, cookies (a Python-side
  cookie jar) and streaming responses all work. The iframe mirrors its location
  into the `#fragment`; failed tabs answer 503/504 (four minutes for slow SQL)/500,
  and stale tabs get a 409.
- **`base_url` is forced and then patched around**: `webworker.js` carries
  `datasette_base_url_fixes()` for three documented holes (a hardcoded `/-/jump`
  fetch in the 1.0 alphas; doubled prefixes on export links and the 0.x filter
  form; `Location` headers missing the prefix). Mouse Party depends on `base_url`
  too (`/<key>/data/<token>/`); 0.65.5 works there today, but this is standing
  evidence that the prefix surface stays frangible across Datasette versions.
- **Pyodide portability tax, paid up front** (`STARTUP_PY`): `os.geteuid` shimmed
  (WASI needed the same shim in `datasette.py`), `os.link = os.symlink`, and a
  pure-Python `pbkdf2_hmac` because Pyodide ships without OpenSSL. Plugins must be
  pure Python without threads or subprocesses.
- **Browser floor:** service worker plus module worker — Chrome 80+, Safari 15+,
  Firefox 114+; some private-browsing modes (Firefox) break the service worker.
  That floor is per *viewer*, which matters if guests were ever the ones running
  it.

## Fit against the requirements

| Requirement | Today (CPython-WASI under wazero) | Pyodide in each guest browser (Lite as-is) | Pyodide in the host webview, host still proxying |
| --- | --- | --- | --- |
| File never sent | Holds — host renders pages | **Breaks** — every viewer downloads the whole DB plus ~7 MB core plus wheels; anyone with the token can keep the file | Holds |
| Read-only confinement | Immutable open; one-file FS mount; temp in memory; secret unreachable | MEMFS copy, non-immutable open; no equivalent of the one-file folder | Same as Lite unless the host feeds bytes through a narrow channel; still a writable heap copy |
| Real Datasette | 0.65.5 pinned, hash-checked, offline | Newest preview by default (1.0a43); `?ref=` pins; installs from PyPI at runtime — a network dependency per start | Whatever is vendored into the app |
| Threads/sockets | Engineered out in `datasette.py` | Engineered out in `webworker.js` (same tricks) | Same |
| Guest cost | Zero download, any browser the site supports | ~10+ MB and a modern browser before the first page; some private modes fail | Zero (host pays) |
| Host cost | Custom build pipeline (`build_python.sh`), pinned sources; runs in-process, pure Go | None (it's not host-side) | JS engine in the loop; WKWebView service-worker limits (app-bound domains — unverified here, docs URLs 404); wheel/lockfile logistics per Pyodide release |
| Startup | ~<1 s warm (compiled-module cache), minutes cold; 2-minute budget | Loading-screen minutes on first visit, HTTP-cache warm after | Two cold starts stacked (webview + Pyodide) unless snapshotted |
| Shared viewing | Host-side single instance; follow-script | Per-tab instances; the URL `#fragment` is the only shared thing — Mouse Party would still need its own follow protocol | Same as today (host owns the pages) |

Two further version couplings deserve naming. The wheel platform tag
(`pyemscripten_2026_0_wasm32`) ties every compiled package to a Pyodide/Emscripten
ABI generation — the pysqlite3 matrix is already per-Pyodide-release — so a
Mouse Party embedding would vendor and re-verify wheels on every Pyodide bump, a
maintenance profile similar to the pinned-hash discipline
[python-requirements.txt](../../scripts/python-requirements.txt) already has, but
with someone else's release calendar setting the pace. And Datasette's stable line
(0.65.x) is what the tests target; Lite's default track is the 1.0 alphas, whose
base_url and write-API behavior differ — pinning, not tracking, is the only sane
posture for an app.

## Rust, browser, native — the option landscape

- **Rust.** There is no Rust Datasette: a GitHub repository search for `datasette rust`
  finds only `simonw/datasette-rure`, a plugin using a Rust regex crate. A Rust
  route therefore means writing a Datasette-shaped surface. Two sub-routes: (a)
  browser-side Rust→WASM as a client of Datasette's JSON API — exactly what the
  parent's probe in [probes/lite/](probes/lite/) exercises (a `wasm32` Rust module
  issuing `GET /films.json?sql=...&_shape=array` through Lite's worker, `ref`
  pinned to 0.65.5, the DB built from `films.sql`); (b) server-side Rust→WASM
  under wazero, where `rusqlite` with bundled SQLite plausibly builds for
  wasm32-wasip1 with the same flags `build_python.sh` uses for SQLite
  (`SQLITE_THREADSAFE=0`, `OMIT_LOAD_EXTENSION`, `OMIT_WAL`) — plausible but
  untested here; it would reuse [wasm.go](../../wasm.go) unchanged and keep the
  confinement story, at the price of reimplementing the pages.
- **Browser-native SQLite.** The official SQLite WASM (with OPFS) and
  `sql.js-httpvfs` (HTTP range serving) both still place the database bytes in the
  viewer's hands; httpvfs merely fetches them lazily. They fit "public data, thin
  server", not Mouse Party's file-privacy model.
- **Native macOS Python.** Bundling a real CPython and running Datasette as a child
  process buys fidelity and native speed at the cost of ~100 MB+ of signed,
  notarized runtime, a process-per-database (weaker than the WASI mount — a real
  process on the Mac, not a confined module), and the same package pinning.
  Nothing in the current tests needs the speed.
- **Go-native.** [store.go](../../store.go) already uses ncruces/go-sqlite3 (the
  real SQLite, translated to Go). A Go-side renderer could serve
  Datasette-*like* pages inside the existing sandbox with zero new runtimes — the
  cost is UI fidelity, which is the one thing "the real one" was chosen for.

## Version caveats, collected

- Pyodide: release **314.0.8** (2026-11-13); the archive is **314.1.0-dev**;
  versioning is now `314.x` (Python 3.14, Emscripten 5.0.3). Python in Pyodide
  (3.14) trails the app's own WASI build (3.15.2).
- SQLite: Pyodide bundles **3.39.0**; current is 3.54.x; the wheel swap reaches
  **3.54.1** for cp314 wheels built against Pyodide 314.0.8 only.
- Datasette: stable **0.65.6** and preview **1.0a43** (both 2026-11-15); the app
  pins **0.65.5**; Lite defaults to previews and patches around 1.0a-specific
  `base_url` holes.
- Emscripten wheel ABI tag `pyemscripten_2026_0_wasm32` gates every compiled wheel
  a Lite-style deployment loads; pure-Python wheels (all of Mouse Party's current
  pins) are unaffected.

## Not verified here

WKWebView's service-worker/app-bound-domain behavior against a Wails window (the
Apple and WebKit doc pages I tried were unreachable; a probe should settle it);
measured Pyodide 314 startup/memory numbers on target machines (none sourced in
either repo — Lite's own loading screens parallelize but don't promise seconds);
the rusqlite-wasip1 build claim (flagged plausible, untested); and Datasette
1.0a-vs-0.65 page-shape differences beyond the `base_url` fixes Lite documents.
The parent's independent Rust/Lite experiment owns the runtime measurements.

## Relation to sibling work

This note supplies the Pyodide/Lite research half;
[mouseparty-components0.sol62hi.md](mouseparty-components0.sol62hi.md) maps the
wider reuse boundaries (Wails window, guest page, WASI tools, Linux blockers), and
the [probes/lite/](probes/lite/) harness is the parent's live experiment this
assessment is meant to inform.
