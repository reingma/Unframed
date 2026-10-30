# 0003 — Performance history records and measurement

Status: accepted in ENG-29 planning (2026-10-15); backend machinery implemented
under ENG-30; frontend measurements implemented under ENG-31. Record binding
amended by [ADR 0008](0008-content-bound-history-records.md) under ENG-47.

## Context

Unframed wants objective, repeatable performance data per change, so each
change's blast radius — including effects on paths it did not touch — is
visible and architectural intuitions rest on measured evidence. The
[Performance tracking project](https://linear.app/unframed-blog/project/performance-tracking-39935c0137a4)
requires that raw metrics stay out of git, that history is kept in the
repository as a statistical summary, and that adding a measurable path is
cheap. ENG-29 planning settled the design below; this record captures the
enduring rationale.

## Decision

- Measure through a declarative workload registry: entries `{id, kind,
  target}` plus kind-specific fields — `rate`/`duration` for `http` and
  `repetitions` for `browser`. Adding a measurable path is one registry
  entry, not new plumbing.
- Use vegeta for HTTP load: an open model at constant rate avoids
  coordinated-omission bias. Use Playwright and web Performance APIs for
  browser metrics, reusing the existing browser dependency.
- Keep two artifact tiers. Raw run artifacts are ephemeral, gitignored and
  regenerable at a git state. Compressed history records are committed per
  merge to `main` under `perf/history/` and carry revision SHA, a
  changelog/issue tag, a host environment (opaque machine label plus
  OS/arch/CPU) and per-workload toolchains and sufficient statistics (n,
  mean, variance, quantiles, histogram).
- The versioned JSON schema is stack-agnostic and consumable by non-Go
  producers; Go emit/parse code uses `encoding/json/v2` per repository
  policy. The shared design is proven on the Go backend first; the frontend
  implementation stress-tests it.
- Analysis compares consecutive records or arbitrary A/B pairs and computes
  confidence intervals from stored statistics. Reports (HTML/SVG) are
  generated on demand and never committed.
- The canonical environment is a designated local machine; environment is a
  modelable factor so other machines or CI-produced runs can join later
  rather than being excluded by today's noise.

## Consequences

- Git gains small committed data files per merge; raw samples never enter
  history. The record mechanics settled at implementation: each feature PR
  produces and commits the record measuring its head, as the
  [run procedure](../../perf/README.md) describes.
- The canonical environment is the designated local machine recorded as the
  opaque label `workstation-1`; new machines join under their own labels.
- Cross-machine comparison must be flagged, not silently pooled.
- ENG-31 moved the toolchain from the record environment to each workload
  (schema version 2). Backend, browser and build workloads run on the same
  host with different toolchains (Go; Node and Chromium; Node), so one
  record per merge holds all of them. A toolchain change is flagged only on
  the workloads it affects. Version 1 records stay unchanged on disk and
  are read as version 2.
- Deltas with confidence intervals are the v1 attribution. Richer
  probabilistic modeling is deferred; stored sufficient statistics keep it
  possible without resampling.
- External raw-artifact storage is deferred; local-only plus gitignored
  suffices while there is one canonical machine.
- CI-produced runs need separate infrastructure authorization. In-app
  telemetry is deferred; pprof capture during an ephemeral run is the
  documented escalation when a delta resists attribution. Per-component
  tracking is deferred to ENG-32 planning.
- The performance-review skill and reviewer profile begin reading committed
  history once records exist.

## References

- [Performance tracking project](https://linear.app/unframed-blog/project/performance-tracking-39935c0137a4)
- [ENG-29](https://linear.app/unframed-blog/issue/ENG-29/plan-a-performance-baseline-and-tracking-schema)
  and its [planning record](https://linear.app/unframed-blog/issue/ENG-29/plan-a-performance-baseline-and-tracking-schema#comment-f10e0309)
- [ENG-30](https://linear.app/unframed-blog/issue/ENG-30/implement-performance-history-schema-backend-measurement-harness-and),
  [ENG-31](https://linear.app/unframed-blog/issue/ENG-31/implement-frontend-performance-measurements-using-the-shared-history),
  [ENG-32](https://linear.app/unframed-blog/issue/ENG-32/plan-per-component-frontend-performance-tracking)
- [ADR 0001](0001-monorepo-and-rendering-boundary.md) and
  [current architecture](../specs/architecture.md)
