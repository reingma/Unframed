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

## Repository skills

The [Go engineering skill](../.agents/skills/go-engineering/SKILL.md) uses the open
[Agent Skills format](https://agentskills.io/specification) under `.agents/skills/`.
Apply it to explicitly authorized Go implementation issues. Root agent
instructions link to it so clients can load the instructions even without skill
auto-discovery. Reusable methods live with the skill; Unframed-specific authority,
architecture and native commands remain in the existing repository entry points.

[Devin CLI documents](https://docs.devin.ai/cli/extensibility/skills#where-skills-live)
this project discovery location and `/go-engineering` invocation. In a session
using a checkout containing the skill, confirm the name and description are
discovered, invoke it, and confirm the shared principles are loaded before work.
Other clients must confirm their own discovery mechanism; do not assume support.
When repository search does not discover it, use explicit skill-directory
activation if supported, or read the linked instructions and references and
report that fallback. File presence alone is not proof of runtime discovery.

The skill has only portable name/description metadata: no tool auto-approvals,
platform permissions, model override, subagent profile, or event trigger. Loading
it cannot replace authorization or enforce a permission boundary. Global/account
installation and reviewer wiring require their separately authorized work.

The implementation skill owns
[shared Go principles](../.agents/skills/go-engineering/references/principles.md).
The future review skill should reference this source, not copy it. Approved
durable feedback refinements go through reviewed changes to that source.

[Worked examples](../.agents/skills/go-engineering/references/examples.md) include
an isolated parser/lookup/context exercise, contract tests, fuzzing and operation
benchmarks. They introduce no blog feature or application dependency/module.
`make check-skills` and `make test-skills` verify the example Go files, also through
root `check`/`test` and the backend CI job. Fuzz campaigns and measurements are
explicit native commands, not performance gates. Record actual activation,
exercise commands/results, trade-offs and client limitations in PR verification,
not as issue-specific repository reports or fabricated production evidence.

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
