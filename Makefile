.PHONY: help
help:
	@echo "Targets:"
	@echo "  frontend-install   Install pnpm deps under frontend/"
	@echo "  frontend-dev       Start the Vite+ dev server (verifier only)"
	@echo "  frontend-build     Build the SPA + PWA assets to frontend/dist/"
	@echo "  frontend-lint      Run vp lint (non-fatal — informational)"
	@echo "  frontend-typecheck Run tsc --noEmit"

# ---------- Frontend ----------

.PHONY: frontend-install
frontend-install:
	cd frontend && pnpm install

.PHONY: frontend-dev
frontend-dev:
	cd frontend && pnpm dev

.PHONY: frontend-build
frontend-build:
	cd frontend && pnpm build

.PHONY: frontend-lint
frontend-lint:
	cd frontend && pnpm lint || true

.PHONY: frontend-typecheck
frontend-typecheck:
	cd frontend && pnpm exec tsc --noEmit

# ---------- Backend ----------

.PHONY: backend-build
backend-build:
	cd backend && go build ./...

.PHONY: backend-test
backend-test:
	cd backend && go test ./... -count=1

.PHONY: backend-test-cover-domain
backend-test-cover-domain:
	cd backend && go test -coverprofile=cov.out ./internal/domain/... && go tool cover -func=cov.out | tail -1

.PHONY: backend-test-cover-app
backend-test-cover-app:
	cd backend && go test -coverprofile=cov.out ./internal/application/... && go tool cover -func=cov.out | tail -1

.PHONY: backend-vet
backend-vet:
	cd backend && go vet ./...

.PHONY: backend-tidy
backend-tidy:
	cd backend && go mod tidy

.PHONY: backend-lint
backend-lint:
	cd backend && golangci-lint run ./...

.PHONY: backend-run-api backend-run
backend-run-api:
	cd backend && go run ./cmd/api

# Friendly alias kept for parity with the task spec (`make backend-run`).
backend-run: backend-run-api

.PHONY: backend-cover-http
backend-cover-http:
	cd backend && go test -coverprofile=cov.http.out ./internal/interfaces/http/... && go tool cover -func=cov.http.out | tail -1

# ---------- Backend layer 2 (infra) ----------

.PHONY: backend-sqlc
backend-sqlc:
	@if command -v sqlc >/dev/null 2>&1; then \
	  cd backend && sqlc generate; \
	else \
	  echo "[backend-sqlc] sqlc binary not on PATH — skipping. Install via 'make backend-tools'."; \
	fi

.PHONY: backend-migrate-up
backend-migrate-up:
	@if command -v goose >/dev/null 2>&1; then \
	  for ctx in shared iam decks practice; do \
	    echo "[goose] $$ctx"; \
	    goose -dir backend/migrations/$$ctx postgres "$$DATABASE_URL" up || exit 1; \
	  done; \
	else \
	  echo "[backend-migrate-up] goose binary not on PATH — install via 'make backend-tools'. Tests use the in-process migrate package automatically."; \
	  exit 1; \
	fi

.PHONY: backend-migrate-down
backend-migrate-down:
	@if command -v goose >/dev/null 2>&1; then \
	  for ctx in practice decks iam shared; do \
	    echo "[goose] $$ctx"; \
	    goose -dir backend/migrations/$$ctx postgres "$$DATABASE_URL" down || exit 1; \
	  done; \
	else \
	  echo "[backend-migrate-down] goose binary not on PATH — install via 'make backend-tools'."; \
	  exit 1; \
	fi

.PHONY: backend-test-integration
backend-test-integration:
	cd backend && INTEGRATION=1 go test -tags=integration ./... -count=1 -timeout=300s

.PHONY: backend-tools
backend-tools:
	@echo "Installing sqlc + goose..."
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest || echo "  (sqlc install failed — may be offline)"
	go install github.com/pressly/goose/v3/cmd/goose@latest || echo "  (goose install failed — may be offline)"

.PHONY: postgres-up
postgres-up:
	docker compose -f infra/docker-compose.yaml up -d

.PHONY: postgres-down
postgres-down:
	docker compose -f infra/docker-compose.yaml down
