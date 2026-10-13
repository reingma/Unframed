# Repository workflow and environment

[AGENTS.md](../AGENTS.md) is the entry point for all coding agents. The
[Engineering resources](decisions/README.md) in Linear remain authoritative for
collaboration policy; this document describes repository entry points and
environment operations, not a second implementation specification.

## Entry points

- Linear holds planned requirements, approval, acceptance criteria, and status.
- [Current architecture](specs/architecture.md) describes implemented facts;
  [ADRs](decisions/README.md) record enduring decisions.
- [README](../README.md) and [Makefile](../Makefile) provide setup and verification.
- The [PR template](../.github/PULL_REQUEST_TEMPLATE.md) requests actual issue
  linkage, verification evidence, changelog accountability, and reviewer context.
- The [changelog](../CHANGELOG.md) records notable changes under `Unreleased`.
  Feature work needs an entry; keep it current through review rather than logging
  every commit. Release versions and dates require separate authorization.

## Environment definition

[.devin/blueprint.yaml](../.devin/blueprint.yaml) is the reviewed Git-backed
definition for Linux x86-64. Its installation reads Go/Node versions from the
repository metadata, verifies archive checksums, checks bundled npm against
`packageManager`, and persists the local toolchain PATH through Devin's `ENVRC`.
Make, curl, tar, xz, sha256sum, and awk are prerequisites of the base Linux image.

When upgrading Go or Node, update repository metadata and the reviewed checksum
together. An old checksum intentionally rejects a new archive. npm must match the
version bundled with the selected Node release or the install fails.

Initialization installs tools. Maintenance incrementally prepares Go/npm
dependencies and Playwright Chromium, including Linux system libraries. System
library installation needs root or passwordless elevation; a restricted image
must have those libraries prepared by its owner. It must not skip them silently.
Maintenance runs during snapshot builds; its commands are surfaced, not
automatically re-executed, at session start. No running servers survive a snapshot.

Fresh checkout and CI verification still use `make setup` (`npm ci`). Blueprint
maintenance uses incremental `npm install`; check for unexpected lockfile changes.
Environment knowledge points to the agent instructions instead of copying policy.
