.PHONY: setup dev-backend dev-frontend fmt check check-backend check-frontend test test-backend test-frontend build build-backend build-frontend test-e2e

setup:
	cd backend && go mod download
	cd frontend && npm ci

dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

fmt:
	cd backend && gofmt -w cmd internal
	cd frontend && npm run fmt

check: check-backend check-frontend

check-backend:
	cd backend && test -z "$$(gofmt -l cmd internal)"
	cd backend && go vet ./...

check-frontend:
	cd frontend && npm run fmt:check
	cd frontend && npm run lint
	cd frontend && npm run typecheck

test: test-backend test-frontend

test-backend:
	cd backend && go test -race ./...

test-frontend:
	cd frontend && npm test

build: build-backend build-frontend

build-backend:
	cd backend && go build -o bin/server ./cmd/server

build-frontend:
	cd frontend && npm run build

test-e2e: build
	cd frontend && npm run test:e2e
