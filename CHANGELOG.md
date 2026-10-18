# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Content tree scanning: a `content.Source` interface with a confined
  filesystem implementation, slug-form names with derived titles and URLs,
  TOML `.configuration` files and `+++` frontmatter cascading root →
  directory → article, and a validated manifest with the navigation tree, an
  article index including drafts, a separate asset list, and per-item
  validation errors. The [authoring schema](docs/specs/content-authoring.md) documents the format.
- Versioned performance history machinery: a declarative workload registry
  (`perf/workloads.json`), a vegeta-based backend runner producing gitignored
  raw run artifacts, compression into committed per-merge records under
  `perf/history/`, and a comparison/report tool computing deltas with 95%
  confidence intervals from stored statistics. JSON schemas under
  `perf/schema/` define the formats for non-Go producers; `make check`
  parse-validates the registry and committed records. First canonical
  baseline recorded for `GET /healthcheck`.
- Frontend performance measurements in the shared history: a Playwright
  runner records route-level TTFB, FCP and LCP against production servers,
  and a build runner records client JavaScript and CSS bundle sizes. Both
  are registry workloads (`frontend-root`, `frontend-bundle`) run by
  `make perf-run`. Schema version 2 moves the toolchain to each workload,
  and comparisons flag toolchain changes per workload; version 1 records
  remain readable.
- Read-only Go, performance and security reviewer subagent profiles and a
  manual handoff procedure for planner, implementer and reviewer roles.
- Security review skill that reviews backend, frontend, dependency and CI changes
  against specific security practices in their exposure and execution context,
  using only CI/CD artifacts and proposing targeted tests only for specific suspicions.
- Performance review skill for backend and frontend changes that analyzes supplied
  performance artifacts against the changelog and diff, separates avoidable
  overhead from accepted cost and flags missing performance tests, without running anything.
- Go engineering review skill that applies the shared Go principles and separates
  defects, uncertainties, trade-offs and preferences, with synthetic review cases.
- Interactive implementation-planning skill with required discussion coverage,
  linked follow-up issue formats and scoped examples, without implementation authority.
- Portable Go engineering implementation skill with shared review principles,
  invariant-focused boundary/null-handling examples, fuzzing/benchmarks, and
  request-correlation checks, following the repository-wide JSON v2 preference.
- Agent-neutral development instructions, a verification-focused PR template, and
  a reviewable Git-backed development environment blueprint with manual activation.

### Changed

- Consolidated documentation around canonical sources and readable prose, moved
  existing ESLint rationale into an ADR, and replaced repeated skill/host policy
  with links while preserving the methods, safeguards and executable environment.
- Streamlined Go skill guidance with explicit on-demand examples and a colocated
  token-performance report convention, preserving shared principles and safeguards.

## [1.1.2] - 2024-09-27

### Added

### Changed

### Deprecated

### Removed

### Fixed
