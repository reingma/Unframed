# Instructions for all coding agents

## Authority and required context

- Planning approval, issue creation/status, and PR approval are not execution or
  merge authorization. Obtain Gabriel's explicit implementation authorization
  before creating a branch, changing files/settings, committing, or opening a PR.
- Implement only the authorized scope. Escalate material requirements,
  architecture/dependency, security-boundary, infrastructure/cost, destructive,
  or production-release changes. Routine implementation choices may be autonomous.
- Gabriel reviews and decides whether to merge. Never merge or enable auto-merge.
- Before implementation, read the current Linear issue and acceptance criteria,
  [README](README.md), [living architecture](docs/specs/architecture.md), relevant
  [decisions and Engineering resources](docs/decisions/README.md), and
  [workflow/environment guidance](docs/workflow.md). If required context is
  inaccessible or conflicts with instructions, report the gap; do not invent it.
- Linear owns planned requirements, approval, acceptance criteria, and work status.
  GitHub owns source, current architecture, decisions, and verification history.
  Do not copy issue specifications into Git or create issue-prefixed spec files.

## Delivery

- Start from current `main` on the issue's prescribed feature branch, for example
  `eng-123-implement-markdown-renderer`. Do not commit or push to `main`.
- PR titles: `[ENG-123] Implement Markdown renderer`. Commit messages:
  `type(area): message`, for example `feature(workflow): add agent guidance`.
  The linked Engineering resources govern these conventions.
- Use the [repository PR template](.github/PULL_REQUEST_TEMPLATE.md). Replace its
  examples with actual issue linkage and verification results. Use `Fixes ENG-123`
  only when the PR completes the issue; use `Part of ENG-123` for partial work
  (`Refs` is a supported alias). Do not close issues with unverified post-merge work.
- Ready means clear scope, verifiable acceptance criteria, understood dependencies,
  consequential decisions resolved, and explicit execution authorization.
- A verified PR awaiting human review is In Review, not Done. Done requires
  acceptance satisfied, relevant checks passing, documentation updated, human
  review complete, and human merge. Record and finish any operational activation
  checks before declaring the overall issue complete.
- Keep edits minimal and follow surrounding conventions. Reuse existing tooling;
  do not build custom frameworks or introduce abstractions without a concrete need.
- Fix failing checks and address actionable review feedback within scope. Reply to
  review threads individually; surface deviations and anything not tested.

## Changelog and documentation

- Every feature PR must add or update a concise [changelog](CHANGELOG.md) entry
  under `Unreleased`, in the appropriate Keep a Changelog category. Include notable
  fixes and behavior changes too. Non-notable internal changes may explain why no
  entry is needed; feature work cannot claim this exemption.
- Maintain the same entry as the feature evolves during review. Do not add an
  entry per commit or generate the changelog from Git logs.
- Do not create release versions/dates or rewrite release history without release
  authorization.
- Update the living architecture when implemented facts change. Record enduring,
  consequential decision rationale in ADRs; routine edits do not need an ADR.

## Setup and verification

Use the pinned versions in `backend/go.mod`, `frontend/.nvmrc`, and
`frontend/package.json`; see README for native commands. Run from repository root:

| Command                                 | Purpose                                            |
| --------------------------------------- | -------------------------------------------------- |
| `make setup`                            | Fresh lockfile-based dependency installation       |
| `make dev-backend`, `make dev-frontend` | Run in separate terminals                          |
| `make fmt`                              | Format Go and frontend code                        |
| `make check`                            | Formatting, Go vet, ESLint, TypeScript             |
| `make test`                             | Go race tests and frontend unit tests              |
| `make build`                            | Independent production builds                      |
| `make test-e2e`                         | Builds and production Playwright integration tests |
| `npm --prefix frontend audit`           | Dependency audit when dependencies change          |

- Native Go compilation/vet/tests require `GOEXPERIMENT=jsonv2`; Make and CI set it.
- Before the first E2E run, install Chromium with `npx playwright install chromium`
  inside `frontend`; Linux may need `--with-deps`. See README for test ports.
- Run formatting/static checks and relevant unit, build, and integration checks.
  Test behavior and failure paths, not incidental implementation details. Use
  property/fuzz tests only for meaningful invariants. Test async Server Components
  through integration tests rather than unsupported unit-test mocks.
- Report actual commands, outcomes, evidence, and gaps. No invented coverage,
  skipped checks presented as passing, or weakening verification/security gates.

## Boundaries and security

- Go owns backend content/configuration processing and JSON-over-HTTP; Next.js owns
  complete reader-facing pages. Preserve external-response Zod validation.
- `BACKEND_URL` is server-only, never `NEXT_PUBLIC_`. The browser does not call Go
  directly in the current slice. Keep router-specific details at the HTTP boundary.
- Never commit or log credentials, local environment files, Terraform state/plans,
  or production secrets. No AWS provisioning, deployment jobs, or new credentials
  without explicit authorization.
- The Git-backed blueprint is environment configuration, not authority to change
  settings. Activation/builds and repository protection changes require approval;
  do not add automatic sync/build pipelines. Do not start services in snapshots.
