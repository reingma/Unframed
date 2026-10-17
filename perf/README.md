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
  `NNNN-<sha8>.json` (zero-padded sequence plus short measured revision).
- `runs/` — raw run artifacts, ephemeral and gitignored.
- `reports/` — generated comparison reports, gitignored.

## Workloads

Registry entries are `{id, kind, target}` plus kind-specific fields:

- `http`: `rate` (requests/second) and `duration` (Go duration syntax). The
  target is a loopback GET URL. The Go runner measures it as `latency_ns`.
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

On the designated machine, start the backend that the `http` workloads
target, then run every workload into one run directory:

```sh
# terminal 1: serve the http workload targets
make build-backend
./backend/bin/server

# terminal 2: build both apps, then run every registry workload
PERF_MACHINE=<label> make perf-run
```

`make perf-run` builds the backend, the frontend and the perf tool. It runs
the `http` workloads with `backend/bin/perf run --run <dir>` and then the
`browser` and `build` workloads with `npm --prefix frontend run perf`. The
frontend runner starts its own production servers: the Go backend on
`127.0.0.1:18090` and `next start` on `127.0.0.1:13090`
([playwright.perf.config.ts](../frontend/playwright.perf.config.ts)).
`make perf-run-frontend` runs only the frontend workloads. `PERF_RUN=<dir>`
selects the run directory for either target.

Artifacts land under `perf/runs/<UTC timestamp>/<workload-id>.json`. The
machine label comes from `PERF_MACHINE` (or `--machine` for the Go runner)
and must be an opaque label, not a hostname. The canonical environment is
`workstation-1`.

Compress the latest run into a committed record for the merge it belongs to:

```sh
make perf-record TAG=ENG-<n>       # or backend/bin/perf compress --tag ENG-<n>
```

`compress` reads the newest run directory (`--run` selects another), records
`git rev-parse HEAD` as the measured revision (`--revision` overrides) and
writes the next sequenced file under `perf/history/`. It refuses a run whose
artifacts come from different hosts. Commit the record with the merge it
measures.

## Analysis

```sh
make perf-compare                  # latest two records, text table
make perf-report                   # HTML + SVG report under perf/reports/
```

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
- `make check` parse-validates the registry and every committed record
  (schema version, name/sequence agreement, statistics sanity). It is a
  parse gate, not a performance gate.
- When a delta resists attribution, capture a pprof profile during an
  ephemeral run rather than adding telemetry; see ADR 0003.
- Cross-machine runs are valid data but a different environment factor;
  canonical history is the designated machine only.
