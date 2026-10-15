# Current architecture

This document describes Unframed as implemented. Its
[maintenance and ownership policy](../../AGENTS.md#changelog-and-documentation)
and [decision records](../decisions/README.md) are separate.

## Structure and responsibilities

The repository is a shallow monorepo with independently runnable and buildable
Go and Next.js applications. There is one Go module and one npm project.
[ADR 0001](../decisions/0001-monorepo-and-rendering-boundary.md) records the choices
and future rendering boundary.

- [backend/cmd/server](../../backend/cmd/server/main.go) owns server setup,
  timeouts, graceful shutdown and lifecycle, not route definitions.
- [backend/internal/router](../../backend/internal/router/router.go) registers
  explicit paths and methods through httprouter. Handlers use standard
  `http.HandlerFunc` signatures.
- [backend/internal/health](../../backend/internal/health/handler.go) implements
  the liveness endpoint.
- [frontend/src/app](../../frontend/src/app/page.tsx) owns Next.js App Router pages,
  layout and presentation, using React and strict TypeScript.
- [frontend/src/lib](../../frontend/src/lib/backend.ts) owns server-side
  communication and Zod validation of external responses. Internal components do
  not need Zod wrappers; Go retains backend validation responsibility.
- [frontend/e2e](../../frontend/e2e) contains production integration tests.
- [infra/terraform](../../infra/terraform/README.md) contains guidance only.
- [.agents/skills](../../.agents/skills) contains repository-local methods and
  isolated examples, not another application module.

There is no workspace manager, shared package, generic service/repository layer,
custom routing framework, Docker or task-orchestration framework.
[.gitignore](../../.gitignore) owns generated-output, environment and state exclusions.

## Tooling sources

Exact runtime and direct dependency pins live in
[backend/go.mod](../../backend/go.mod), [frontend/package.json](../../frontend/package.json)
and [frontend/.nvmrc](../../frontend/.nvmrc). Resolved dependencies are recorded in
[go.sum](../../backend/go.sum) and [package-lock.json](../../frontend/package-lock.json).
The repository's [JSON API preference](../../AGENTS.md#repository-preferences-and-boundaries)
also applies to isolated Go examples.

Styling uses minimal CSS without a component library. The
[ESLint configuration](../../frontend/eslint.config.mjs) uses the core,
TypeScript and React Hooks recommended rules. [ADR 0002](../decisions/0002-eslint-preset-selection.md)
records the preset decision and its revisit conditions.

## HTTP and rendering contract

Go exposes `GET /healthcheck` with HTTP 200, `Content-Type: application/json` and
`{"status":"ok"}`. It checks liveness, not readiness of future dependencies.

The request path is **browser → Next.js → Go**. The browser does not call Go
directly; this slice needs no browser CORS configuration. Next.js fetches on each
page request without caching. Reloading checks again; there is no polling or
refresh widget. Its production build succeeds without a running backend.

`BACKEND_URL` is a configurable, server-only absolute backend origin.
Never prefix it with `NEXT_PUBLIC_`. Go's `BACKEND_ADDR` controls its listen address;
changing either address requires no source edit. `BACKEND_URL` defaults to
`http://127.0.0.1:8080` and `BACKEND_ADDR` to `:8080`. Each health fetch has a
two-second timeout.

The helper resolves the health endpoint from the origin root and infers its
response type from a Zod schema requiring `status: "ok"`. Valid data produces the
available presentation and displays the response. Connection failures, timeouts,
non-success HTTP responses, malformed JSON and schema-invalid data produce the
unavailable presentation, not healthy data.

Content ingestion, article rendering, drafts and publishing, appearance
configuration, S3 or other content providers, math, authentication and database
storage are not implemented. The API has no generic response envelope,
GraphQL/gRPC or code generation. Future article and binary-response contracts
remain undecided; see ADR 0001 rather than treating them as implemented features.

## Verification and environment entry points

[README](../../README.md#commands-and-verification) owns runnable setup and check
instructions. [Makefile](../../Makefile) delegates to native commands, and
[CI](../../.github/workflows/ci.yml) verifies the applications and isolated Go
examples. There are no deployment jobs, cloud credentials or coverage-percentage gates.

Behavior is exercised by the [health](../../backend/internal/health/handler_test.go),
[router](../../backend/internal/router/router_test.go),
[communication](../../frontend/src/lib/backend.test.ts),
[presentation](../../frontend/src/app/health-status.test.tsx) and
[integration](../../frontend/e2e) tests. The [agent verification policy](../../AGENTS.md#setup-and-verification)
governs test selection.

The [Go skill](../../.agents/skills/go-engineering/SKILL.md) provides one shared
implementation/review method and standalone examples compiled by file list.
The [Go review skill](../../.agents/skills/go-engineering-review/SKILL.md) applies
that method to requested reviews, with optional synthetic review cases.
The [performance review skill](../../.agents/skills/performance-review/SKILL.md)
analyzes supplied performance artifacts against the diff for requested reviews
and runs nothing. There is no CI performance job, performance history, frontend
measurement or performance gate.
The [security review skill](../../.agents/skills/security-review/SKILL.md) reviews
changed code against specific security practices in its exposure and execution
context for requested reviews, using only CI/CD artifacts. CI produces no
dependency audit or secret-scan output, and there is no security gate.
The [planning skill](../../.agents/skills/implementation-planning/SKILL.md)
provides its interactive method and optional issue formats and examples.
[Workflow guidance](../workflow.md#repository-skills) owns discovery, loading and
report conventions. No reviewer profiles, automatic review wiring, global
installation or automatic issue execution are configured.

The [Linux development blueprint](../../.devin/blueprint.yaml) owns environment
initialization and maintenance. [Workflow guidance](../workflow.md#manual-activation-after-merge)
owns activation and fresh-session verification; this document does not assert
that a reviewed definition has been activated. No cloud resources, Terraform
providers or state configuration have been created.
