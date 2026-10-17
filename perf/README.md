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
`rate` (requests/second) and `duration` (Go duration syntax) for `http`,
`repetitions` for `browser`. `http` targets are GET endpoints and must be
loopback URLs; `browser` entries are run by the frontend tooling, not this
runner. `make check` validates the registry: unique ids, kind fields and
local-only targets.

## Running a measurement

The runner measures a live target; it does not start servers. On the
designated machine:

```sh
# terminal 1: serve the workload targets
make build-backend
./backend/bin/server

# terminal 2: run every http registry workload
make perf-run                      # or PERF_MACHINE=<label> backend/bin/perf run
```

Artifacts land under `perf/runs/<UTC timestamp>/<workload-id>.json`. The
machine label comes from `PERF_MACHINE` or `--machine` and must be an opaque
label, not a hostname; the canonical environment is `workstation-1`.

Compress the latest run into a committed record for the merge it belongs to:

```sh
make perf-record TAG=ENG-<n>       # or backend/bin/perf compress --tag ENG-<n>
```

`compress` reads the newest run directory (`--run` selects another), records
`git rev-parse HEAD` as the measured revision (`--revision` overrides) and
writes the next sequenced file under `perf/history/`. Commit the record with
the merge it measures.

## Analysis

```sh
make perf-compare                  # latest two records, text table
make perf-report                   # HTML + SVG report under perf/reports/
```

`compare --a <file> --b <file>` compares an arbitrary pair and `--json` emits
the same analysis for tooling. Deltas are `B - A` with a 95% confidence
interval on each mean, computed from stored n/mean/variance (Welch form,
normal approximation); stored quantiles are reported as point deltas.
Records from different environments are flagged, never pooled.

## Conventions and failure behavior

- Records are committed data; raw artifacts never enter git. A record's
  environment factor is the machine label plus OS, architecture, CPU and
  toolchain captured at run time. `toolchain` reports the Go toolchain that
  built the measuring binary (and, under this procedure, the measured
  server, since both build from the same go.mod pin).
- `make check` parse-validates the registry and every committed record
  (schema version, name/sequence agreement, statistics sanity). It is a
  parse gate, not a performance gate.
- When a delta resists attribution, capture a pprof profile during an
  ephemeral run rather than adding telemetry; see ADR 0003.
- Cross-machine runs are valid data but a different environment factor;
  canonical history is the designated machine only.
