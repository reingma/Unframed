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

## Manual activation after merge

The existing Unframed installation was settings-managed when this definition was
introduced. Committing the YAML does not switch that installation or prove a
snapshot usable. Activation and fresh-session verification remain human-controlled.

1. Gabriel reviews and merges the repository PR. Use `Part of` for an issue whose
   acceptance includes these post-merge steps; do not close it on the file merge.
2. In Devin **Settings → Environment → Blueprints**, select Unframed and switch
   its source from database/settings to Git. Confirm the imported default-branch
   commit and blueprint contents.
3. Sync/build through the UI and wait for a successful build. The UI Sync action
   builds changed contents; explicitly build if necessary when unchanged. Confirm
   the intended Linux snapshot succeeded, not merely a partial overall result.
4. Leave automatic sync/build-on-default-branch-commit disabled. No API token or
   GitHub sync workflow is required.
5. Start a fresh session using that snapshot. Confirm the synced commit, tool
   versions, and discovery of the root agent instructions. Have the agent report
   the authorization/no-merge/changelog guardrails before authorizing test work.
6. Verify documented startup and `make check`, `make test`, `make build`, and
   `make test-e2e` in that fresh session. Record results and any activation gaps
   on the implementation issue before marking it Done.

Subsequent blueprint changes use a feature-branch PR, human review/merge, and
manual sync/build. Do not edit settings as a competing source or introduce
automatic activation without approval. See
[Devin's activation documentation](https://docs.devin.ai/onboard-devin/environment/git-backed-blueprints).

## Main-branch protection review

Inspection during introduction of this workflow found an active repository
[ruleset, “Protect mian”](https://github.com/reingma/Unframed/rules/24961945),
applying to `main`. GitHub's active branch-rules response included:

- PRs required, with resolved review threads.
- Zero required approving reviews.
- No required status-check rule.
- Linear history, no force pushes, and no deletion.

The classic branch-protection endpoint returned HTTP 401 through the available
API access. These findings describe the visible ruleset, **not proof that no
additional classic protection exists**. No security settings were changed.

Gabriel should inspect both classic protection and rulesets, including bypass
permissions, and explicitly approve any changes needed to enforce human review
and these existing CI jobs on `main`:

- `Backend checks and build`
- `Frontend checks and build`
- `Production integration tests`

Until confirmed, agents must not equate “CI runs” with “CI is required for merge.”
The no-merge instruction applies regardless of technical permissions.
