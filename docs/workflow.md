# Repository workflow and environment

Follow [AGENTS.md](../AGENTS.md) for authority, delivery and documentation policy.
This guide owns skill discovery and environment operations.
[README](../README.md) owns local setup and runnable verification instructions.

## Repository skills

Repository-local skills use the open
[Agent Skills format](https://agentskills.io/specification) in `.agents/skills/`:

- [go-engineering](../.agents/skills/go-engineering/SKILL.md) owns the Go procedure
  and its required shared principles.
- [go-engineering-review](../.agents/skills/go-engineering-review/SKILL.md) owns
  the Go review procedure and finding format, using those same principles. Its
  worked review cases are optional calibration.
- [performance-review](../.agents/skills/performance-review/SKILL.md) owns the
  performance review procedure and its Go benchmark comparison tooling, using the
  shared measurement method. Raw results stay in the gitignored `.perf/` directory.
- [implementation-planning](../.agents/skills/implementation-planning/SKILL.md)
  owns interactive planning and its required method. Its formats and worked
  examples are optional illustrations, not product decisions or executed tests.

[Devin CLI documents](https://docs.devin.ai/cli/extensibility/skills#where-skills-live)
this location and slash-command activation. In a checkout containing the skill,
confirm its name and description are discovered, invoke `/go-engineering`,
`/go-engineering-review`, `/performance-review` or `/implementation-planning`, and confirm its required reference is read.

Other clients must verify their discovery mechanism. If search misses a skill,
use explicit directory activation where supported, or read its linked entry point
and required references and report that fallback. File presence alone does not
prove runtime discovery.

Metadata contains only a portable name and description. There are no tool
auto-approvals, platform permissions, model overrides, subagent profiles or event
triggers. [Host authority](../AGENTS.md#authority-and-required-context) still governs use.
Global installation and reviewer wiring are separately scoped work.

The Go implementation skill owns the shared method; the review skill references
it rather than maintaining a competing copy. Its worked examples introduce
no application feature, dependency or module. Native static and race checks run
through Make and CI. Exercise-specific fuzz and benchmark commands live in
[the worked exercise](../.agents/skills/go-engineering/references/examples.md).
They and the performance review tooling are manual evidence, not performance gates.

### Skill token-performance reports

For skill implementation or optimization issues requiring analysis, apply clear
wins, then run a fresh final analysis and maintain one report at
`.agents/skills/<skill-name>/token-performance/analysis.md`. Record comparable
baseline and final counts, the measured scope, tokenizer and version, reproduction
commands, loading assumptions and observations, savings, retained behavior,
verification and gaps. These are factual verification records, not specifications.
There is no universal token budget or new CI gate.

Measure discovery metadata, activated instructions, required references, optional
resources and host context separately. Inventory report size independently.
Keep reports and worked exercises off mandatory loading paths; read them only
for relevant work or review. Running native example checks does not require
reading their source into context.

## Environment operations

The [blueprint](../.devin/blueprint.yaml) owns executable installation,
maintenance, archive checksums and toolchain validation for Linux x86-64.
Make, curl, tar, xz, sha256sum and awk must be present in the base image.

For a Go or Node upgrade, update the source metadata and reviewed archive checksum
together. An old checksum deliberately rejects a new archive. The selected Node
release must bundle the npm version required by `packageManager`.

Maintenance incrementally prepares dependencies and Playwright. Installing Linux
system libraries requires root or passwordless elevation; a restricted image
must have its owner prepare them, not silently skip them. Maintenance runs during
snapshot builds and is surfaced, not automatically rerun, at session start.
Snapshots retain files, not running servers; startup happens in the session.

Fresh checkouts and CI use the [README setup procedure](../README.md#fresh-checkout).
Blueprint maintenance uses incremental `npm install`; check for unexpected
lockfile changes. Its knowledge section is a quick reference to common commands
and the canonical entry points.

## Manual activation after merge

The installation was settings-managed when the Git-backed definition was
introduced. Committing it does not switch that source or prove a snapshot usable.
Follow [host activation authority](../AGENTS.md#repository-preferences-and-boundaries)
and [issue completion policy](../AGENTS.md#delivery) throughout these steps.

1. Link the PR to the issue without closing it while these steps remain.
   After the PR is human-reviewed and merged, open Devin
   **Settings → Environment → Blueprints**, select Unframed, and switch its source
   from database/settings to Git. Confirm the imported default-branch commit and contents.
2. Sync/build through the UI and wait for success. Sync builds changed contents;
   explicitly build if necessary when unchanged. Confirm the intended Linux
   snapshot succeeded, not merely a partial overall result.
3. Leave automatic sync/build-on-default-branch-commit disabled. No API token or
   GitHub sync workflow is required.
4. Start a fresh session from that snapshot. Confirm the synced commit, tool
   versions and root agent-instruction discovery. Have the agent report the
   linked guardrails before authorizing test work.
5. Verify the README startup procedure and `make check`, `make test`, `make build`
   and `make test-e2e` in that session. Record results and activation gaps on the issue.

Subsequent definition changes follow the same reviewed, manual procedure, not a
competing settings source. See [Devin's activation documentation](https://docs.devin.ai/onboard-devin/environment/git-backed-blueprints).
