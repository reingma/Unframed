# Unframed

A personal engineering blog, built in the open as an experiment in specification-driven,
AI-assisted development with [Linear](https://linear.app) and Devin. AWS is an intended
future hosting option; running the project does not require AWS or Vercel.

This first skeleton contains a Go health API and a Next.js page that fetches and
validates its response server-side. It does not implement blog content or deployment.

## Repository

```text
backend/cmd/server/       Go executable
backend/internal/router/  route registration and routing tests
backend/internal/health/  health handler and contract test
frontend/src/app/         Next.js App Router page, layout, and presentation
frontend/src/lib/         server-only API communication and Zod validation
frontend/e2e/             Playwright integration tests
infra/terraform/         future infrastructure scope and guardrails
docs/specs/              living architecture specification
docs/decisions/          architectural decisions and rationale
.github/workflows/       verification, not deployment
```

## Fresh checkout

Install **Go 1.26.9**, **Node 24.21.0 LTS**, **npm 11.19.0**, and **Make**.
Go's version is pinned in `backend/go.mod`; Node's version is in `frontend/.nvmrc`.
With [nvm](https://github.com/nvm-sh/nvm), run `nvm install` and `nvm use` inside
`frontend/` to select Node. Use the bundled npm 11.19.0.

The backend uses standard-library `encoding/json/v2`, which is experimental in
Go 1.26 and requires `GOEXPERIMENT=jsonv2` at compile time. Make and CI set this
flag; set it explicitly when using native Go commands. The API is not yet covered
by Go's compatibility guarantee.

From the repository root:

```sh
make setup
make dev-backend
```

In a second terminal:

```sh
make dev-frontend
```

Visit **http://localhost:3000**. With Go running, the page shows “Backend available”
and its actual `{"status":"ok"}` response. Stop Go and reload: the page shows
“Backend unavailable.” Reloading checks again; there is no polling.

The page fetches on each request with no caching and a two-second timeout.
The endpoint checks backend liveness only, not readiness of future dependencies.

## Configuration

| Variable       | Process        | Default                 | Purpose                                                        |
| -------------- | -------------- | ----------------------- | -------------------------------------------------------------- |
| `BACKEND_ADDR` | Go             | `:8080`                 | Listen address, such as `127.0.0.1:9090`.                      |
| `BACKEND_URL`  | Next.js server | `http://127.0.0.1:8080` | Absolute backend origin; `/healthcheck` is resolved from its root. |

For example, in the two terminals:

```sh
BACKEND_ADDR=127.0.0.1:9090 make dev-backend
BACKEND_URL=http://127.0.0.1:9090 make dev-frontend
```

Alternatively, copy `frontend/.env.example` to `frontend/.env.local` for local
Next.js configuration. Restart Next.js after changing configuration. `BACKEND_URL`
is server-only: never prefix it with `NEXT_PUBLIC_`. The browser does not fetch Go
directly, so this slice needs no CORS configuration.

## Commands

Run from the repository root:

| Command         | Behavior                                                                     |
| --------------- | ---------------------------------------------------------------------------- |
| `make setup`    | Download Go dependencies and install the npm lockfile with `npm ci`.         |
| `make fmt`      | Format Go and frontend source.                                               |
| `make check`    | Check Go formatting/vet and frontend Prettier/ESLint/TypeScript.             |
| `make test`     | Run Go race-enabled tests and frontend Vitest tests.                         |
| `make build`    | Build both applications independently; no live backend needed.               |
| `make test-e2e` | Build production applications and run managed Playwright integration checks. |

Before the first integration run, install the Chromium browser:

```sh
cd frontend
npx playwright install chromium
# Linux systems may also need system libraries: npx playwright install --with-deps chromium
```

The integration suite starts and stops the compiled Go server and two production
Next.js servers, for healthy and unreachable-backend scenarios. Ports `18080`,
`18081`, `13000`, and `13001` must be unused. It does not reuse development servers.
Traces are retained on failure; generated artifacts are ignored by Git.

Native commands are also supported:

```sh
# In backend/
export GOEXPERIMENT=jsonv2
go mod download
go run ./cmd/server
gofmt -w cmd internal
go vet ./...
go test -race ./...
go build -o bin/server ./cmd/server

# In frontend/
npm ci
npm run dev
npm run fmt:check
npm run lint
npm run typecheck
npm test
npm run build
npm run start
# After both production builds above:
npm run test:e2e
```

`npm run typecheck` generates Next.js route types before checking TypeScript, so
it works without a prior development run. `make build-backend` and
`make build-frontend` build either application alone. For production-style local
verification, run `backend/bin/server` and `npm run start` inside `frontend/` after
`make build`; this is not a production deployment configuration.

## Specifications and workflow

- [Current architecture — living specification](docs/specs/architecture.md)
- [Specification and handoff conventions](docs/specs/README.md)
- [Decisions and engineering-resource links](docs/decisions/README.md)
- [Terraform scope](infra/terraform/README.md)

Linear tracks work; repository specifications and decisions are durable technical
records. Implementation requires explicit human approval. Changes are delivered
on a feature branch as a linked PR; agents never merge. Full agent instructions
and a reproducible environment blueprint are follow-up work.
