.PHONY: build test test-race test-all lint fmt run ci \
        dev watch demo demo-debug release clean coverage \
        hooks integration-pg integration-mysql \
        changelog

BINARY   = basemake
APP_NAME = basemake

# ── Build ──────────────────────────────────────────────────

build:           ## Build binary (default)
	go build -ldflags="-s -w" -o $(BINARY) .

install: build   ## Build and install to /usr/local/bin
	sudo cp $(BINARY) /usr/local/bin/$(BINARY)
	@echo "  ✅ Installed $(BINARY) to /usr/local/bin/"

build-race:      ## Build with race detector
	go build -race -ldflags="-s -w" -o $(BINARY) .

# ── Test ───────────────────────────────────────────────────

test:            ## Run all unit tests
	go test -count=1 -v ./...

test-short:      ## Quick test (no verbose)
	go test -count=1 ./...

test-race:       ## Test with race detector
	go test -race -count=1 ./...

test-all: test integration-pg integration-mysql  ## Full suite

# ── Lint & Format ──────────────────────────────────────────

lint:            ## Run linters (vet + staticcheck)
	go vet ./...
	staticcheck ./...

fmt-check:       ## Check formatting (CI-safe)
	test -z "$$(gofmt -l .)"

fmt:             ## Auto-format code
	gofmt -w .

# ── CI (matches GitHub Actions) ────────────────────────────

ci: lint test build  ## Same pipeline as CI

# ── Dev Loop ───────────────────────────────────────────────

run: build       ## Build and launch REPL
	@echo "  Starting $(BINARY)..."
	@./$(BINARY)

dev:             ## Build + launch with demo DB
	@echo "  Starting $(BINARY) with demo DB..."
	@./$(BINARY) init --demo 2>/dev/null || true
	@./$(BINARY)

# ── File Watcher ───────────────────────────────────────────

watch:           ## Auto-rebuild on .go changes
	@if ! command -v entr >/dev/null 2>&1; then \
		echo "  Installing entr (file watcher)..."; \
		sudo apt-get install -y -qq entr >/dev/null 2>&1; \
	fi
	@echo "  Watching .go files — save to rebuild..."
	@find . -name '*.go' -not -path './vendor/*' | entr -c sh -c 'make build && echo "  ✅ Rebuilt"'

# ── Demo ───────────────────────────────────────────────────

demo: build       ## Build + REPL on demo SQLite DB
	@echo "  🚀 Launching $(BINARY) with demo data..."
	@./$(BINARY) init --demo --provider ollama

demo-debug: build-race  ## Build with race detector + launch
	@echo "  🚀 Launching $(BINARY) (race-detector build)..."
	@./$(BINARY)

# ── Release ────────────────────────────────────────────────

release:         ## Tag and push a new release
	@read -p "  Version (e.g. v0.3.0): " TAG; \
	echo ""; \
	sed -i "1s/.*/# $$TAG/" CHANGELOG.md; \
	git add -A; \
	git commit -m "chore: release $$TAG"; \
	git tag $$TAG; \
	echo "  ✅ Tagged $$TAG. Pushing..."; \
	git push origin main --tags; \
	echo "  🚀 CI will build and publish."

# ── Coverage ───────────────────────────────────────────────

coverage:        ## Generate coverage report
	go test -coverprofile=coverage.out -count=1 ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "  📊 Open coverage.html in your browser"

# ── Pre-commit Hooks ────────────────────────────────────────

hooks:           ## Install git hooks
	git config core.hooksPath .githooks
	@echo "  ✅ Hooks installed from .githooks/"

# ── Integration Tests ─────────────────────────────────────

integration-pg:  ## Run PostgreSQL integration tests (start postgres-test container first)
	@echo "  Starting PostgreSQL test container..."
	@docker compose up -d postgres-test 2>/dev/null; \
	echo "  Waiting for PG..."; \
	sleep 2; \
	BASEMAKE_TEST_PG="postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable" \
		go test -count=1 -v -run TestPostgres ./internal/db/

integration-mysql: ## Run MySQL integration tests (start mysql-test container first)
	@echo "  Starting MySQL test container..."
	@docker compose up -d mysql-test 2>/dev/null; \
	echo "  Waiting for MySQL..."; \
	sleep 3; \
	BASEMAKE_TEST_MYSQL="mysql://root:root@tcp(localhost:3307)/mysql" \
		go test -count=1 -v -run TestMySQL ./internal/db/

# ── Changelog ───────────────────────────────────────────────

changelog:       ## Insert new changelog entry scaffold
	@read -p "  Version (e.g. v0.3.0): " VER; \
	read -p "  Title: " TITLE; \
	DATE=$$(date +%Y-%m-%d); \
	ENTRY="# $$VER — $$TITLE ($$DATE)\n\n## Added\n\n-\n\n## Fixed\n\n-\n\n## Internal\n\n-\n\n---\n\n"; \
	printf "$$ENTRY" | cat - CHANGELOG.md > /tmp/changelog.tmp && mv /tmp/changelog.tmp CHANGELOG.md; \
	echo "  ✅ Changelog entry scaffolded at top of CHANGELOG.md"

# ── Clean ──────────────────────────────────────────────────

clean:           ## Remove build artifacts
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf release/
	@echo "  ✅ Clean"

# ── Help ───────────────────────────────────────────────────

help:            ## Show available commands
	@echo "  basemake — dev commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
