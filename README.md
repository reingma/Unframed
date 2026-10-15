# Unframed

A personal engineering blog, built in the open as an experiment in
specification-driven, AI-assisted development with [Linear](https://linear.app)
and Devin.

The [current architecture](docs/specs/architecture.md) describes the skeleton,
its layout and contracts. [ADR 0001](docs/decisions/0001-monorepo-and-rendering-boundary.md)
explains the application boundary and self-hosting choices.

## Fresh checkout

Install Go, Node with its bundled npm, and Make. Use the reviewed versions in
[backend/go.mod](backend/go.mod), [frontend/.nvmrc](frontend/.nvmrc) and the
`packageManager` field in [frontend/package.json](frontend/package.json).
With [nvm](https://github.com/nvm-sh/nvm), run `nvm install` and `nvm use` inside
`frontend/` to select the pinned Node release.

From the repository root:

```sh
make setup
make dev-backend
```

In a second terminal:

```sh
make dev-frontend
```

Open **http://localhost:3000**. With Go running, the page shows “Backend available”
and its actual `{"status":"ok"}` response. Stop Go and reload to see “Backend
unavailable.” Reloading checks again; there is no polling.

The page fetches on each request with no caching and a two-second timeout.
The endpoint checks backend liveness only, not readiness of future dependencies.
See the [HTTP contract](docs/specs/architecture.md#http-and-rendering-contract)
for the request path and failure behavior.

## Configuration

| Variable       | Process        | Default                 | Purpose                                                            |
| -------------- | -------------- | ----------------------- | ------------------------------------------------------------------ |
| `BACKEND_ADDR` | Go             | `:8080`                 | Listen address, such as `127.0.0.1:9090`.                          |
| `BACKEND_URL`  | Next.js server | `http://127.0.0.1:8080` | Absolute backend origin; `/healthcheck` is resolved from its root. |

Follow the [server-only configuration contract](docs/specs/architecture.md#http-and-rendering-contract).

For example, in the two terminals:

```sh
BACKEND_ADDR=127.0.0.1:9090 make dev-backend
BACKEND_URL=http://127.0.0.1:9090 make dev-frontend
```

Alternatively, copy [frontend/.env.example](frontend/.env.example) to
`frontend/.env.local`. Restart Next.js after changing its configuration.

## Commands and verification

Run commands from the repository root. [Makefile](Makefile) owns the targets and
their underlying native commands; [frontend/package.json](frontend/package.json)
owns npm scripts.

```sh
make fmt
make check
make test
make build
make test-e2e
```

You can build either application alone with `make build-backend` or
`make build-frontend`. The skill example checks also have `make check-skills` and
`make test-skills` targets.

Before the first integration run, install Chromium:

```sh
cd frontend
npx playwright install chromium
# Linux may also need: npx playwright install --with-deps chromium
```

The integration suite manages its own compiled Go server and two production
Next.js servers. Keep the ports declared in
[frontend/playwright.config.ts](frontend/playwright.config.ts) unused; it does not
reuse development servers. Failure traces are retained and generated artifacts
are ignored by Git.

For native Go commands, first follow the
[JSON API and build requirements](AGENTS.md#repository-preferences-and-boundaries).
For example:

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
# After both production builds:
npm run test:e2e
```

The `typecheck` script generates route types before checking TypeScript, so a
prior development run is unnecessary. For production-style local verification,
run `backend/bin/server` and `npm run start` inside `frontend/` after `make build`.
This is not a production deployment configuration.

## Documentation and workflow

- [Instructions and policy for all coding agents](AGENTS.md)
- [Current architecture](docs/specs/architecture.md)
- [Specification index](docs/specs/README.md)
- [Decisions and Engineering resources](docs/decisions/README.md)
- [Skill discovery and manual environment activation](docs/workflow.md)
- [Go implementation skill](.agents/skills/go-engineering/SKILL.md)
- [Go review skill](.agents/skills/go-engineering-review/SKILL.md)
- [Performance review skill](.agents/skills/performance-review/SKILL.md)
- [Security review skill](.agents/skills/security-review/SKILL.md)
- [Interactive planning skill](.agents/skills/implementation-planning/SKILL.md)
- [Infrastructure guide](infra/terraform/README.md)

The [blueprint](.devin/blueprint.yaml) defines the development environment.
Its activation procedure is in the [workflow guide](docs/workflow.md#manual-activation-after-merge).
