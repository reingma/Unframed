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
  configuration, timeouts, graceful shutdown and lifecycle, not route
  definitions. It runs the startup build before listening and then starts the
  builder and the content watcher.
- [backend/internal/router](../../backend/internal/router/router.go) registers
  explicit paths and methods through httprouter. Handlers use standard
  `http.HandlerFunc` signatures.
- [backend/internal/health](../../backend/internal/health/handler.go) implements
  the liveness endpoint.
- [backend/internal/content](../../backend/internal/content/manifest.go) owns the
  content domain types: the `Source` interface, names, configuration and the
  manifest. `FSSource` reads a local directory confined with `os.Root`.
  [content/scan](../../backend/internal/content/scan/scan.go) exports only `Scan`,
  which reads a `Source` into a validated manifest: the public navigation tree,
  an article index including drafts, each node's typed configuration and
  resolved theme, the separate list of assets and per-item validation errors. The
  [authoring schema](content-authoring.md) owns the format and
  [ADR 0004](../decisions/0004-content-source-and-authoring-format.md) its rationale.
  Article nodes keep their Markdown body from the scan's single read.
  [content/render](../../backend/internal/content/render/render.go) exports only
  `Render`, which converts Markdown to safe HTML with goldmark.
- [backend/internal/document](../../backend/internal/document/document.go) owns the
  versioned JSON documents, the article envelope and the public manifest, and
  their decoders; the [document contract](document-contract.md) owns the format.
- [backend/internal/build](../../backend/internal/build/build.go) exports `Build`,
  which scans a `Source`, renders every valid article, drafts included, into
  envelopes, and publishes the build's events. Each build has a random build
  ID on its log records and events.
- [backend/internal/event](../../backend/internal/event/event.go) owns the
  event contract: self-contained, versioned `article.published` and
  `validation.failed` events that consumers in other processes can decode.
- [backend/internal/publish](../../backend/internal/publish/publish.go) owns the
  build's single outbound boundary, `Publisher`, which puts a build's events on
  a bus. `Local` is the in-process bus; it delivers to every `Subscriber` and
  joins their errors. In production an SNS or EventBridge publisher is meant to
  replace it, and subscribers such as email Lambdas consume the bus outside
  this process; neither exists yet.
- [backend/internal/subscriber](../../backend/internal/subscriber) holds the
  local subscribers: structured logs, and JSON report files that refuse
  directories inside the content root and build IDs that are not safe file
  names.
  [ADR 0005](../decisions/0005-document-envelope-rendering-and-fanout.md)
  records the document, rendering and event rationale.
- [backend/internal/snapshot](../../backend/internal/snapshot/snapshot.go) turns
  one successful build into an immutable snapshot: the manifest and every
  envelope encoded once, each with a SHA-256 content hash, and the per-item
  diagnostics. Each encoded document is decoded back through the `document`
  decoders; one that fails, such as an unknown `schemaVersion`, is excluded and
  reported as a diagnostic. `Store` swaps snapshots atomically.
- [backend/internal/rebuild](../../backend/internal/rebuild/rebuild.go) owns the
  single builder. Startup, watch and reload triggers only signal it: at most
  one build runs and one is pending, and further signals join the pending
  build. The startup build must succeed; afterwards a failed build keeps the
  last good snapshot serving. Each build logs its trigger, coalesced signal
  count, duration and outcome.
- [backend/internal/watch](../../backend/internal/watch/watch.go) is the local
  change notifier, behind the builder's consumer-owned `Notifier` interface and
  outside the read-only `content.Source`. It watches every non-hidden directory
  with fsnotify, adds directories created later and reports a burst once after
  a 200 ms quiet window. Watcher errors and event overflow force a rebuild.
  Network filesystems may not deliver notifications; there is no polling.
- [backend/internal/api](../../backend/internal/api/api.go) serves the current
  snapshot through the server-owned `Snapshots` read interface, never the
  content source, and assigns request correlation IDs.
  [ADR 0006](../decisions/0006-snapshot-serving-and-rebuilds.md) records the
  serving and rebuild rationale.
- [backend/internal/perf](../../backend/internal/perf) and
  [backend/cmd/perf](../../backend/cmd/perf) own the performance measurement
  machinery: registry parsing, the vegeta-based HTTP runner, compression of
  raw artifacts into committed history records, and comparison/report analysis.
- [frontend/perf](../../frontend/perf/frontend.perf.ts) owns the browser and
  build runners. They write raw artifacts in the shared schema, which the Go
  tool compresses: Playwright measures route TTFB, FCP and LCP against
  production servers, and the build runner measures client bundle bytes.
- [perf](../../perf/README.md) owns the workload registry, the versioned JSON
  schemas, committed history records under `perf/history/`, and gitignored raw
  runs and reports. [ADR 0003](../decisions/0003-performance-history-records-and-measurement.md)
  records the design.
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

Go serves the build output as JSON. `Content-Type` is `application/json`, and
the bodies are the snapshot's pre-encoded [documents](document-contract.md):

| Route                                | Response                                                                                                                                         |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| `GET /manifest`                      | The public manifest. Each nav node's `urlPath` addresses its article.                                                                            |
| `GET /articles/*path`                | The published envelope at the slug path; `/articles/` is the root landing page.                                                                  |
| `GET /preview/:token/articles/*path` | The draft at the path, otherwise the published envelope. A miss is a 404 listing the validation errors of the item and its ancestor directories. |
| `GET /preview/:token/diagnostics`    | The serving build's ID, time and counts, every validation error and the latest build outcome, which may be a failure.                            |
| `POST /admin/reload`                 | Runs a build that starts after the request and returns its `buildId`, `envelopes` and `validationErrors`.                                        |

- Article paths are lowercase slug segments. Drafts, invalid items, unknown
  paths, dot or empty segments, trailing slashes, encoded `/` or `\` and other
  non-slug segments all return the same `404 {"error":"not found"}`. Paths are
  never cleaned or redirected.
- `PREVIEW_TOKEN` gates both preview routes; a wrong token or an unset one
  returns the same public 404. Preview links are shareable, so the token is a
  shareable credential.
- `ADMIN_TOKEN` gates reload only, and only as `Authorization: Bearer <token>`.
  Reload returns 404 when the token is unset, 401 for a missing or wrong token,
  including the preview token, 500 when its build fails and 504 when it waits
  longer than 8 seconds. Tokens are compared in constant time and never logged.
- Next.js sends `X-Request-ID`. Go keeps an ID of at most 128 characters from
  `[A-Za-z0-9._-]` and otherwise generates one, and echoes it in the response.
  Each request log carries it, the URL path with any preview token redacted,
  the status, the duration and the serving build ID.
- Every response comes from one snapshot. Requests never read the content
  source; the watcher keeps snapshots fresh. Content hashes are computed but no
  caching headers are served.

Remote event delivery,
appearance configuration beyond the theme, S3 or other content providers,
math, authentication, rate limiting and database storage are not implemented. The API has
no generic response envelope, GraphQL/gRPC or code generation. Binary-response
contracts remain undecided; see ADR 0001 rather than treating them as implemented features.

## Verification and environment entry points

[README](../../README.md#commands-and-verification) owns runnable setup and check
instructions. [Makefile](../../Makefile) delegates to native commands, and
[CI](../../.github/workflows/ci.yml) verifies the applications and isolated Go
examples. There are no deployment jobs, cloud credentials or coverage-percentage gates.

Behavior is exercised by the [health](../../backend/internal/health/handler_test.go),
[content](../../backend/internal/content),
[document](../../backend/internal/document),
[build](../../backend/internal/build),
[event](../../backend/internal/event),
[publish](../../backend/internal/publish),
[subscriber](../../backend/internal/subscriber),
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
and runs nothing. Committed performance history lives under
[perf/history/](../../perf/README.md): one compressed record per merge to
`main`, produced by the local backend and frontend runners on the designated
machine and validated for shape by `make check`. A `pull_request` CI job runs
the `perf gate` presence-and-binding check: the diff must append a record,
committed records are immutable, and the newest record's measured revision
must match the merged content outside `perf/history/`. The gate measures
nothing — CI never produces records; there is no performance-threshold gate
and no per-component frontend measurement; raw run artifacts are local-only.
The [security review skill](../../.agents/skills/security-review/SKILL.md) reviews
changed code against specific security practices in its exposure and execution
context for requested reviews, using only CI/CD artifacts. CI produces no
dependency audit or secret-scan output, and there is no security gate.
The three reviews also run as read-only Devin CLI subagent profiles in
[.agents/agents/](../../.agents/agents/), started manually as
[review handoffs](../review-handoffs.md) describes; nothing triggers them.
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
