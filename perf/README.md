# Performance history and measurement

This directory owns the repository's performance data. The measurement
machinery lives in [backend/internal/perf](../backend/internal/perf) and
[backend/cmd/perf](../backend/cmd/perf); [ADR 0003](../docs/decisions/0003-performance-history-records-and-measurement.md)
records the design. Run all commands from the repository root.

## Layout

- [workloads.json](workloads.json) — the declarative workload registry.
  Adding a measurable path is one entry.
- [schema/](schema) — the versioned JSON schemas for the registry, raw run
  artifacts and history records. They are the contract for non-Go producers.
- `history/` — committed compressed records, one per merge to `main`, named
  `NNNN-<sha8>.json` (zero-padded sequence plus short measured revision),
  bound to the measured content by `content_hash`.
- `runs/` — raw run artifacts, ephemeral and gitignored.
- `reports/` — generated comparison reports, gitignored.

## Workloads

Registry entries are `{id, kind, target}` plus kind-specific fields:

- `http`: `rate` (requests/second), `duration` (Go duration syntax) and an
  optional expected `status`, 200 by default; other response codes count as
  errors. The target is a loopback GET URL. The Go runner records
  `latency_ns` and `response_bytes` (the response body size).
- `browser`: `repetitions`. The target is a route path, such as `/`, or a
  loopback URL. The frontend runner loads it in Chromium against a
  production `next start` server. Each repetition is one fresh browser
  context (no cache, cookies or storage), after one discarded warm-up load.
  It records `ttfb_ms`, `fcp_ms` and `lcp_ms` from the standard Navigation
  Timing, Paint Timing and Largest Contentful Paint APIs, plus the page's
  decoded `html_bytes`, `js_bytes` and `css_bytes` from Resource Timing.
  A non-2xx document or a missing paint entry is a sample error; the
  values that were measured are kept.
- `build`: no extra fields. The target is a repository-relative Next.js
  build directory. The frontend runner sums the raw and gzip bytes of all
  JavaScript and CSS under `<target>/static` into `js_bytes`,
  `js_gzip_bytes`, `css_bytes` and `css_gzip_bytes`. These values are
  deterministic for a build, so the artifact has exactly one sample.

Browser timings are the noisiest metrics. The seeded `frontend-root` entry
uses 50 repetitions. In a development run, that gave a 95% interval of
about ±1.2 ms on mean TTFB (about 10 ms) and about ±2.6 ms on mean FCP and
LCP (about 43 ms). Raise `repetitions` when a route's intervals are too
wide to separate the deltas you care about. To measure another route, add a `browser` entry with its path. `make check`
validates the registry: unique ids, kind fields and local-only targets.

## Running a measurement

On the designated machine, start the backends that the `http` workloads
target, then run every workload into one run directory. The `*-large`
workloads hit a second server on `127.0.0.1:8081` that serves a generated
1,000-article tree (20 sections of 50 articles, the shape `BenchmarkBuild`
measures). `make perf-content` regenerates it under the gitignored
`perf/content/`:

```sh
# terminal 1: serve the example site for the small http workloads
make build-backend
CONTENT_ROOT=examples/site ./backend/bin/server

# terminal 2: serve the generated large tree for the *-large workloads
make perf-content
BACKEND_ADDR=127.0.0.1:8081 CONTENT_ROOT=perf/content ./backend/bin/server

# terminal 3: build both apps, then run every registry workload
PERF_MACHINE=<label> make perf-run
```

`make perf-run` builds the backend, the frontend and the perf tool. It first
runs `backend/bin/perf content-hash`, which refuses a worktree with
uncommitted or untracked changes outside `perf/history/` (gitignored files
are fine) and prints the content hash of `HEAD`: the git tree ID of
everything outside `perf/history/`. Every artifact of the run stores that
hash as `content_hash`. It runs the `http` workloads with `backend/bin/perf run --run <dir>` and then the
`browser` and `build` workloads with `npm --prefix frontend run perf`. The
frontend runner starts its own production servers: the Go backend on
`127.0.0.1:18090` and `next start` on `127.0.0.1:13090`
([playwright.perf.config.ts](../frontend/playwright.perf.config.ts)).
`make perf-run-frontend` runs only the frontend workloads. `PERF_RUN=<dir>`
selects the run directory for either target. The frontend runner takes the
hash from `PERF_CONTENT_HASH`, which both targets set.

Artifacts land under `perf/runs/<UTC timestamp>/<workload-id>.json`. The
machine label comes from `PERF_MACHINE` (or `--machine` for the Go runner)
and must be an opaque label, not a hostname. The canonical environment is
`workstation-1`.

Compress the latest run into a committed record for the merge it belongs to:

```sh
make perf-record TAG=ENG-<n>       # or backend/bin/perf compress --tag ENG-<n>
```

`compress` reads the newest run directory (`--run` selects another) and
writes the next sequenced file under `perf/history/`. The record takes the
run's `content_hash` and records `HEAD` (or `--revision <sha>`) as the
measured `revision`, refusing a commit whose content differs from what the
run measured. It refuses a run whose artifacts come from different hosts or
carry different content hashes. Commit the record with the merge it measures.

The `revision` is informational: a squash or rebase merge drops it from
`main`. To find the code a record measured, run
`make perf-locate RECORD=perf/history/<file>` (or
`backend/bin/perf locate [--ref main] <file>`). It prints the oldest commit
on `main`'s first-parent history whose content outside `perf/history/`
matches the record's `content_hash`. Version 1 and 2 records predate the
hash and only have their revision.

## The merge gate

Every pull request must append at least one record under `perf/history/`, and
the newest record must be bound to what merges. The CI `perf-history` job runs
`perf gate --base <sha> --head <sha>` on `pull_request`; `make perf-gate`
(`PERF_BASE`, `PERF_HEAD`) runs it locally. The gate enforces:

- Committed records are immutable: the diff may only add files under
  `perf/history/`, never modify or delete existing ones.
- At least one `NNNN-<sha8>.json` record is added, and the head history
  satisfies the same name/sequence/parse rules as `perf validate`. Every
  added record is version 3, with a `content_hash`.
- The newest record's `content_hash` equals head's: everything outside
  `perf/history/` is identical to what was measured, so only record appends
  came after the measured commit. Its `revision` need not resolve, so squash,
  rebase and merge-commit merges all keep history verifiable
  ([ADR 0008](../docs/decisions/0008-content-bound-history-records.md)).
  A record therefore lands in its own commit (or commits) on top of the
  measured head, and **any push after measuring** — review feedback, rebase,
  typo fix — requires re-running `make perf-run` and `make perf-record` and
  appending a fresh record. A failure prints both hashes and, when the
  revision still resolves, the changed paths.
- `environment.machine != "workstation-1"` passes with a visible
  non-canonical warning.

The gate checks presence and binding only; it cannot produce measurements and
sets no performance thresholds. Binding proves the record names the merged
content, not that the measurement ran on `workstation-1`. A PR squashed while
`main` has moved on lands content nobody measured; the next PR's record
measures it again.

To satisfy it: finish the change, then on the designated machine run
`make perf-run` and `make perf-record TAG=ENG-<n>` at the final head, commit
the record file and push. A backfill of an old merge is measured at a checkout of that
commit and compressed with `compress --revision <sha>`; the newest record still must bind to this
merge's head, so a PR may not carry only a backfill.

Work authored somewhere other than the designated machine — a cloud session,
another host — cannot produce a canonical record. The handoff is: push the
branch, pull it on `workstation-1`, run `make perf-run` and
`make perf-record TAG=ENG-<n>` there, commit and push. A non-canonical record
is accepted but warned, so the drift stays visible in history and `compare`
flags it.

## Analysis

```sh
make perf-compare                  # latest two records, text table
make perf-report                   # HTML + SVG report under perf/reports/
```

Both name each record's located commit on `main` (`--ref` selects another
branch) next to its measured revision.
`compare --a <file> --b <file>` compares an arbitrary pair and `--json` emits
the same analysis for tooling. Deltas are `B - A` with a 95% confidence
interval on each mean, computed from stored n/mean/variance (Welch form,
normal approximation); stored quantiles are reported as point deltas.
Records from different hosts are flagged, never pooled. A workload whose
toolchain changed between the records is flagged on its own, because its
deltas include the toolchain effect.

## Conventions and failure behavior

- Records are committed data; raw artifacts never enter git. A record's
  host environment is the machine label plus OS, architecture and CPU,
  shared by every workload. OS and architecture use Go's `GOOS` and
  `GOARCH` names for every producer, so Node's `x64` is recorded as `amd64`.
- Each workload carries its own toolchain map. `http` workloads record
  `{"go": ...}`, the Go toolchain that built the measuring binary and, under
  this procedure, the measured server. `browser` workloads record
  `{"node": ..., "chromium": ...}`, and `build` workloads record
  `{"node": ...}`.
- Schema version 2 introduced per-workload toolchains. Version 1 records
  keep their file contents. They are read as version 2, with their
  record-level Go toolchain moved to each workload.
- Schema version 3 added `content_hash` to run artifacts and records.
  Version 2 records keep their file contents and have no hash; the workload
  registry stays at version 2.
- `make check` parse-validates the registry and every committed record
  (schema version, name/sequence agreement, statistics sanity). It is a
  parse gate, not a performance gate.
- When a delta resists attribution, capture a pprof profile during an
  ephemeral run rather than adding telemetry; see ADR 0003.
- Cross-machine runs are valid data but a different environment factor;
  canonical history is the designated machine only.
