.PHONY: setup dev-backend dev-frontend fmt check check-backend check-frontend check-skills check-perf test test-backend test-frontend test-skills build build-backend build-frontend test-e2e perf-build perf-run perf-run-frontend perf-record perf-compare perf-report perf-gate perf-content

GO_SKILL_FILES := $(wildcard .agents/skills/go-engineering/assets/*.go)

setup:
	cd backend && go mod download
	cd frontend && npm ci

dev-backend:
	cd backend && CONTENT_ROOT=$${CONTENT_ROOT:-../examples/site} GOEXPERIMENT=jsonv2 go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

fmt:
	cd backend && gofmt -w cmd internal
	cd frontend && npm run fmt
	gofmt -w $(GO_SKILL_FILES)

check: check-backend check-frontend check-skills check-perf

check-backend:
	cd backend && test -z "$$(gofmt -l cmd internal)"
	cd backend && GOEXPERIMENT=jsonv2 go vet ./...

check-frontend:
	cd frontend && npm run fmt:check
	cd frontend && npm run lint
	cd frontend && npm run typecheck

check-skills:
	test -z "$$(gofmt -l $(GO_SKILL_FILES))"
	GOEXPERIMENT=jsonv2 go vet $(GO_SKILL_FILES)

check-perf: perf-build
	backend/bin/perf validate

test: test-backend test-frontend test-skills

test-backend:
	cd backend && GOEXPERIMENT=jsonv2 go test -race ./...

test-frontend:
	cd frontend && npm test

test-skills:
	GOEXPERIMENT=jsonv2 go test -race $(GO_SKILL_FILES)

build: build-backend build-frontend

build-backend:
	cd backend && GOEXPERIMENT=jsonv2 go build -o bin/server ./cmd/server

build-frontend:
	cd frontend && npm run build

test-e2e: build
	cd frontend && npm run test:e2e

perf-build:
	cd backend && GOEXPERIMENT=jsonv2 go build -o bin/perf ./cmd/perf

ifndef PERF_RUN
PERF_RUN := perf/runs/$(shell date -u +%Y%m%dT%H%M%S)
endif

perf-run: perf-build build
	backend/bin/perf run --run $(PERF_RUN)
	cd frontend && PERF_RUN_DIR=$(PERF_RUN) npm run perf

perf-run-frontend: perf-build build
	backend/bin/perf validate
	cd frontend && PERF_RUN_DIR=$(PERF_RUN) npm run perf

perf-content: perf-build
	rm -rf perf/content
	backend/bin/perf content --out perf/content

perf-record: perf-build
	backend/bin/perf compress --tag $(TAG)

perf-compare: perf-build
	backend/bin/perf compare

perf-report: perf-build
	backend/bin/perf report --out perf/reports/comparison.html

ifndef PERF_BASE
PERF_BASE := origin/main
endif
ifndef PERF_HEAD
PERF_HEAD := HEAD
endif

perf-gate: perf-build
	backend/bin/perf gate --base $(PERF_BASE) --head $(PERF_HEAD)
