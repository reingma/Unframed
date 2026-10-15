.PHONY: setup dev-backend dev-frontend fmt check check-backend check-frontend check-skills test test-backend test-frontend test-skills build build-backend build-frontend test-e2e

GO_SKILL_FILES := $(wildcard .agents/skills/go-engineering/assets/*.go)
PERF_SKILL_FILES := $(wildcard .agents/skills/performance-review/scripts/*.go)

setup:
	cd backend && go mod download
	cd frontend && npm ci

dev-backend:
	cd backend && GOEXPERIMENT=jsonv2 go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

fmt:
	cd backend && gofmt -w cmd internal
	cd frontend && npm run fmt
	gofmt -w $(GO_SKILL_FILES) $(PERF_SKILL_FILES)

check: check-backend check-frontend check-skills

check-backend:
	cd backend && test -z "$$(gofmt -l cmd internal)"
	cd backend && GOEXPERIMENT=jsonv2 go vet ./...

check-frontend:
	cd frontend && npm run fmt:check
	cd frontend && npm run lint
	cd frontend && npm run typecheck

check-skills:
	test -z "$$(gofmt -l $(GO_SKILL_FILES) $(PERF_SKILL_FILES))"
	GOEXPERIMENT=jsonv2 go vet $(GO_SKILL_FILES)
	GOEXPERIMENT=jsonv2 go vet $(PERF_SKILL_FILES)

test: test-backend test-frontend test-skills

test-backend:
	cd backend && GOEXPERIMENT=jsonv2 go test -race ./...

test-frontend:
	cd frontend && npm test

test-skills:
	GOEXPERIMENT=jsonv2 go test -race $(GO_SKILL_FILES)
	GOEXPERIMENT=jsonv2 go test -race $(PERF_SKILL_FILES)

build: build-backend build-frontend

build-backend:
	cd backend && GOEXPERIMENT=jsonv2 go build -o bin/server ./cmd/server

build-frontend:
	cd frontend && npm run build

test-e2e: build
	cd frontend && npm run test:e2e
