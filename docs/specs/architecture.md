# Current architecture

This living specification describes Unframed as implemented. Update this single
document as approved changes land, rather than adding issue-specific copies.
Linear remains the authoritative source for planned requirements and acceptance
criteria; decision records explain why architectural choices were made.

## Repository layout

One shallow monorepo, with independently runnable/buildable applications:

```text
backend/cmd/server/       executable entry point
backend/internal/router/  route registration and routing tests
backend/internal/health/  health handler and colocated tests
frontend/src/app/         Next.js routes and layouts
frontend/src/lib/         backend communication helper
frontend/e2e/             browser integration tests
infra/terraform/         scope and future setup guidance
docs/specs/              living architecture specification
docs/decisions/          architectural decision records
.github/workflows/       CI
```

There is one Go module in the backend and one npm project in the frontend, without
a workspace manager, shared package, generic service/repository layers, custom
routing framework, Docker, or task-orchestration framework. `.gitignore` excludes
generated application output, local environment configuration, and Terraform
state/caches/plans.

## Toolchain and reviewed pins

The stack is Go 1.26.x, Next.js 16.x App Router, React, strict TypeScript, Node 24
LTS, and npm with a committed lockfile. Zod validates external TypeScript data;
styling uses minimal CSS without a component library. Static checks use Go
formatting/vet and frontend ESLint/Prettier/type checking.

The backend serializes JSON with standard-library `encoding/json/v2` using
`json.MarshalWrite`. Go 1.26 requires the experimental `GOEXPERIMENT=jsonv2`
build flag; backend Make targets and CI set it, and native Go commands must do
the same. This API is not yet covered by Go's compatibility guarantee.

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
  httprouter provides explicit method/path registration in `backend/internal/router`.
  The executable handles server setup and lifecycle, not route definitions.
  Handlers use standard `http.HandlerFunc` signatures.
- Go exposes `GET /healthcheck` returning HTTP 200, `Content-Type: application/json`,
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
- Zod is used at the external response boundary, not around internal functions
  or UI components. Go retains responsibility for backend validation.

The agreed future responsibility boundary assigns content/configuration loading,
validation, Markdown processing, and article-data preparation to Go. Next.js owns
complete reader-facing HTML pages, layout, navigation, and metadata. Those content
features are not implemented. The article representation remains undecided;
future images/downloads/streams need not use JSON.

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
  `go vet`, and race-enabled tests. Router tests cover `/healthcheck`, rejected
  methods, unknown routes, and removal of `/healthz`. A failing response writer
  verifies that JSON write failures are logged with a descriptive message and
  the underlying error details.
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

## Infrastructure and unimplemented capabilities

`infra/terraform` contains guidance only; no AWS account is required. No cloud
resources, Terraform providers/state configuration, or deployment jobs exist.

Content ingestion, article rendering, drafts/publishing, appearance configuration,
S3/content providers, math support, authentication, and database storage are not
implemented. The API has no generic response envelope, GraphQL/gRPC, or code
generation. Go and Next.js responsibilities for future capabilities are outlined
above; their detailed contracts still require planning and approval.
