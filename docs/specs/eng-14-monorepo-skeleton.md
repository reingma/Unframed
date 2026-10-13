# ENG-14 — Initial Go and Next.js monorepo skeleton

Status: approved for implementation by Gabriel on 2026-10-13.
Tracking: [ENG-14](https://linear.app/unframed-blog/issue/ENG-14/implement-initial-go-and-nextjs-monorepo-skeleton).
Planning: [ENG-13](https://linear.app/unframed-blog/issue/ENG-13/plan-for-monorepo-implementation).

This is the versioned, authoritative specification transferred from the approved
ENG-14 handoff. Linear tracks status and links here; do not maintain a second
requirements copy. Patch pins and concrete commands below resolve the approved
proposal's delegated implementation details without expanding its scope.

## Scope and layout

One shallow monorepo, with independently runnable/buildable applications:

```text
backend/cmd/server/       executable entry point
backend/internal/health/  health handler and colocated tests
frontend/src/app/         Next.js routes and layouts
frontend/src/lib/         backend communication helper
frontend/e2e/             browser integration tests
infra/terraform/         scope and future setup guidance
docs/specs/              implementation specifications
docs/decisions/          architectural decision records
.github/workflows/       CI
```

Use one Go module in the backend and one npm project in the frontend. No JavaScript
workspace manager, shared package, generic service/repository layers, custom routing
framework, Docker, or task-orchestration framework. Include `.gitignore` for generated
application output, local environment configuration, and Terraform state/caches/plans.

## Toolchain and reviewed pins

Use Go 1.26.x, Next.js 16.x App Router, React, strict TypeScript, Node 24 LTS, npm with
a committed lockfile, Zod at external TypeScript boundaries, and minimal CSS without
a component library. Use Go formatting/vet and frontend ESLint/Prettier/type checks.

Reviewed on 2026-10-13 using official runtime releases, npm package release dates,
engine/peer constraints, dependency audits, and local compatibility checks:

| Runtime / dependency                   | Exact pin         |
| -------------------------------------- | ----------------- |
| Go                                     | 1.26.9            |
| Node LTS / bundled npm                 | 24.21.0 / 11.19.0 |
| httprouter                             | 1.3.0             |
| Next.js                                | 16.3.8            |
| React / React DOM                      | 19.3.0            |
| Zod / server-only                      | 4.6.5 / 0.0.1     |
| TypeScript                             | 5.9.3             |
| ESLint / @eslint/js                    | 10.11.0 / 10.0.1  |
| typescript-eslint / React Hooks plugin | 8.71.0 / 7.1.1    |
| Prettier                               | 3.9.9             |
| Vitest / jsdom                         | 4.1.11 / 27.4.0   |
| React Testing Library / jest-dom       | 16.3.3 / 7.0.1    |
| Playwright                             | 1.63.0            |
| @types/node                            | 24.19.0           |
| @types/react / @types/react-dom        | 19.3.0 / 19.3.0   |

Exact direct pins live in `go.mod` and `package.json`; `go.sum` and `package-lock.json`
record dependency resolution. Installation must use `npm ci`, not floating `latest`.
Mature compatible releases are preferred over newly published major/minor lines.

The initial ESLint configuration uses core recommended, TypeScript recommended,
and React Hooks recommended rules. The optional Next.js preset is not selected:
its React/import/accessibility plugins declare peers excluding ESLint 10, while
ESLint 9 is EOL. Its fast-glob dependency also pulls unpatched braces
([GHSA-vfj7-8cjw-p6xm](https://github.com/advisories/GHSA-vfj7-8cjw-p6xm)).
Use compatible supported plugins, not peer-dependency bypasses, an EOL linter, or
audit suppression. This is an initial tooling choice, not a change to established
repository merge gates. Re-evaluate the optional preset when its ecosystem supports
the chosen linter and has patched dependencies.

## HTTP and rendering contract

- Standard `net/http` serves connections, timeouts, and graceful shutdown;
  httprouter provides explicit method/path registration. Use standard handlers
  where practical and keep router-specific parameters at the HTTP boundary.
- Go exposes `GET /healthz` returning HTTP 200, `Content-Type: application/json`,
  and `{"status":"ok"}`. This is liveness, not readiness of future dependencies.
- The request flow is **browser → Next.js → Go**. The browser does not call Go
  directly, and the initial slice needs no browser CORS setup.
- Next.js fetches the endpoint server-side on each page request, without caching
  and with a bounded two-second timeout. Reloading checks again; no polling or
  refresh widget. The production frontend build must succeed with Go stopped.
- `BACKEND_URL` is a configurable server-only absolute backend origin, defaulting
  locally to `http://127.0.0.1:8080`. Go's `BACKEND_ADDR` defaults to `:8080`.
  Changing either address must not require source edits.
- The helper uses a small Zod schema requiring `status: "ok"` and infers its
  TypeScript response type. Valid data produces “Backend available” and displays
  the response; unreachable/timeout/non-success responses, malformed JSON, and
  schema-invalid data produce a clear unavailable state rather than healthy data.
- Do not wrap every internal function or UI component in Zod. Go remains
  responsible for backend validation.

Go owns future content/configuration loading and validation, Markdown processing,
and preparation of article data. Next.js owns complete reader-facing HTML pages,
layout, navigation, and metadata. The future article representation remains
undecided. Future images/downloads/streams need not use JSON.

## Commands, tests, and CI

Small root Make targets delegate to native Go/npm commands:

- `setup`: Go dependencies and `npm ci`.
- `dev-backend`, `dev-frontend`: independent processes in two terminals.
- `fmt`, `check`, `test`, `build`: project-wide formatting, static checks, unit
  tests, and independently buildable applications.
- `test-e2e`: production builds and Playwright-managed startup/cleanup.

The README documents fresh installation, configuration, native commands, browser
installation, and production-style local execution.

- Go: standard `testing` and `httptest` verify status/content-type/JSON; formatting,
  `go vet`, and race-enabled tests.
- Frontend: Vitest verifies the communication boundary's valid and invalid data,
  connection/HTTP failures, configuration, and timeout. React Testing Library
  verifies synchronous success/unavailable presentation.
- Integration: Playwright against running applications verifies the actual Go
  health response appears on the page and backend unavailability produces the
  fallback. Test async Server Components here, not through Vitest mocks.
- GitHub Actions: formatting, lint/static checks, unit tests, independent production
  builds, and integration tests on PRs. Local and CI targets verify the same
  behavior. No AWS credentials, deployment jobs, or coverage-percentage gates.
- Property testing/fuzzing waits for meaningful content/configuration invariants;
  do not invent properties for a constant health response.

## Acceptance criteria

1. A fresh checkout can be installed and run using documented commands.
2. Backend and frontend build independently.
3. The page displays a real Go health response fetched server-side and handles
   backend failure and malformed/schema-invalid responses gracefully.
4. The backend address is configurable without source edits.
5. Local checks and CI verify the same behavior.
6. Documentation/infrastructure areas exist, with no cloud provisioning, secrets,
   or blog features introduced.
7. Go uses `net/http` + httprouter; the endpoint satisfies its JSON contract, and
   the frontend validates it with Zod using an inferred TypeScript type.

## Boundaries and handoff

No article endpoints, content ingestion, Markdown renderer, publishing/drafts
implementation, appearance configuration schema, generic API envelopes,
GraphQL/gRPC, or API code generation. No AWS resources/services/providers or
Terraform state configuration. Infrastructure guidance only; full agent
instructions and the environment blueprint remain separate follow-up work.

Linear tracks planning, status, and implementation; GitHub holds source,
specifications, decisions, and verification history. Material scope changes
require discussion and explicit approval. Use the Engineering resource naming
conventions, a feature branch, the available PR template, and `Fixes ENG-14` when
fully delivered. Gabriel reviews and decides whether to merge; agents never merge.
