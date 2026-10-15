# Instructions for all coding agents

## Authority and required context

- Planning approval, issue creation or status, and PR approval do not authorize
  execution or merge. Obtain Gabriel's explicit implementation authorization
  before creating a branch, changing files or settings, installing tools, running
  experiments, committing, or opening an implementation PR.
- Stay within the authorized scope. Escalate material requirements, architecture
  or dependency changes, security boundaries, infrastructure or cost, destructive
  operations, and production releases. Routine implementation choices may be autonomous.
- Gabriel reviews and decides whether to merge. Never merge or enable auto-merge.
- Skills grant no tools or permissions. They do not authorize provisioning,
  a new tracing backend, automatic fixes or review pipelines, global installation,
  agent or profile configuration, security-gate bypasses, or production load.
- Before implementation, read the current Linear issue and acceptance criteria,
  [README](README.md), [architecture](docs/specs/architecture.md), relevant
  [decisions and Engineering resources](docs/decisions/README.md), and
  [workflow guidance](docs/workflow.md). Report inaccessible or conflicting
  context instead of inventing it.
- For authorized Go implementation, follow the
  [Go skill](.agents/skills/go-engineering/SKILL.md), including its required principles.
- For an explicitly requested Go review, follow the
  [Go review skill](.agents/skills/go-engineering-review/SKILL.md).
- For an explicitly requested performance review, follow the
  [performance review skill](.agents/skills/performance-review/SKILL.md).
- For implementation planning, follow the
  [planning skill](.agents/skills/implementation-planning/SKILL.md) and complete
  its interactive discussion before settling the design.

## Delivery

- Start from current `main` on the issue's prescribed feature branch.
  Do not commit or push to `main`.
- Follow the linked [Engineering resources](docs/decisions/README.md#engineering-resources)
  for branch, PR and commit naming and closing versus non-closing issue linkage.
  Use the [PR template](.github/PULL_REQUEST_TEMPLATE.md) with actual issue links.
- Ready requires clear scope, verifiable acceptance criteria, understood
  dependencies, resolved consequential decisions, and explicit execution authorization.
- A verified PR awaiting human review is In Review, not Done. Done requires
  acceptance satisfied, relevant checks passing, documentation updated, human
  review complete, and human merge. Finish and record required operational
  activation or post-merge verification before closing the issue.
- Keep edits minimal and follow surrounding conventions. Reuse existing tooling;
  do not create frameworks or abstractions without a concrete need.
- Fix failed checks and actionable review feedback within scope. Reply to review
  threads individually and report deviations and anything not tested.

## Changelog and documentation

- Linear owns planned requirements, approval, acceptance criteria, and work status.
  GitHub owns source, current architecture, decisions, and verification history.
  Do not copy planning documents into Git or create issue-prefixed specifications.
- Update the [living architecture](docs/specs/architecture.md) when implemented
  facts change. Record enduring, consequential rationale in
  [ADRs](docs/decisions/README.md); routine edits do not need an ADR.
- Give each rule or fact one canonical home. Elsewhere, use a link or a short
  pointer rather than another policy copy. Link code-owned values such as version
  pins to their source artifacts instead of maintaining parallel tables. Public
  contract values, such as configuration defaults and timeouts, stay documented. Write plain sentences that
  humans can skim; token savings must not sacrifice clarity or safeguards.
- Every feature PR and notable behavior change needs a concise
  [changelog](CHANGELOG.md) entry under Unreleased in the appropriate category.
  Non-notable internal work may explain an exemption; feature work cannot.
  Maintain the same entry through review, not an entry per commit or a Git-log export.
- Release versions, dates, and release-history changes require separate authorization.

## Setup and verification

Use the pins in [backend/go.mod](backend/go.mod),
[frontend/.nvmrc](frontend/.nvmrc), and [frontend/package.json](frontend/package.json).
The [README](README.md) owns setup, native commands, browser installation and test ports.

- Run formatting and static checks and relevant unit, build and integration tests.
  Run `npm --prefix frontend audit` when dependencies change.
- Use the shared [verification method](.agents/skills/go-engineering/references/principles.md#flexible-invariant-focused-verification)
  for behavioral contracts, failure paths, and meaningful property or fuzz tests.
  Test async Server Components through integration tests, not unsupported unit mocks.
- Report actual commands, outcomes, evidence and gaps. Never invent coverage,
  present skipped checks as passing, or weaken verification or security gates.
  Record skill activation, exercise results, trade-offs and client limitations
  in PR verification rather than issue-specific repository reports.

## Repository preferences and boundaries

- For library selection, follow the shared
  [selection and measurement method](.agents/skills/go-engineering/references/principles.md#performance-conscious-design-and-early-measurements).
- Prefer mature, compatible dependency releases over newly published major or
  minor lines. Keep exact pins and install from the lockfile, not floating `latest`.
- Prefer `encoding/json/v2` for all repository Go code, tests and skill examples.
  An alternative needs an explained compatibility, contract or measured workload
  reason, not convenience. It is experimental and not yet covered by Go's
  compatibility guarantee. Native Go compilation, vet and tests require
  `GOEXPERIMENT=jsonv2`; Make and CI set this flag.
- Preserve the [architecture contracts](docs/specs/architecture.md#http-and-rendering-contract),
  including the Go/Next.js boundary, external-response validation and server-only configuration.
- Never commit or log credentials, local environment files, Terraform state or
  plans, or production secrets. AWS provisioning, deployment jobs and new
  credentials require explicit authorization.
- The Git-backed blueprint is environment configuration, not permission to change
  settings. Activation, builds and repository protection changes require approval.
  Do not add automatic sync/build pipelines or start services in snapshots.
