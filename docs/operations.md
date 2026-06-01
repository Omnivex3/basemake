# Operations Infrastructure

This document covers the operational infrastructure for the basemake project — including CI/CD runners, build pipelines, cost optimization, license verification, and local development tooling.

## Self-Hosted GitHub Actions Runner

To reduce reliance on GitHub-hosted runners for heavy builds, a self-hosted runner is deployed on a VPS.

### Server Specifications

| Detail | Value |
|--------|-------|
| **Provider** | Hetzner |
| **Plan** | CX33 |
| **OS** | PopOS (Ubuntu-based) |
| **vCPUs** | 4 |
| **RAM** | 8 GB |
| **Disk** | 80 GB NVMe SSD |

### Runner Configuration

| Detail | Value |
|--------|-------|
| **Runner user** | `gh-runner` |
| **Installation path** | `/home/gh-runner/actions-runner` |
| **Service name** | `actions.runner.karabo-labs-basemake.basemake-runner.service` |
| **Labels** | `[self-hosted, linux, x64]` |
| **Docker access** | Via `docker` group membership |

### Checking Runner Status

Check if the service is running:

```bash
sudo systemctl status actions.runner.karabo-labs-basemake.basemake-runner.service
```

View recent logs:

```bash
journalctl -u actions.runner.karabo-labs-basemake.basemake-runner.service -n 50 -f
```

Check the runner's registration and status from GitHub CLI:

```bash
gh run list --repo karabo-labs/basemake --limit 10
```

### Restarting the Runner

```bash
cd /home/gh-runner/actions-runner
./svc.sh stop
./svc.sh start
```

### Runner Maintenance Notes

- The runner connects to GitHub via a long-lived registration token. If the token expires, re-register using the GitHub Actions runner setup script.
- Docker is available inside CI jobs because the `gh-runner` user is in the `docker` group. This is required for the Docker build job.
- The runner runs as a systemd service, so it auto-starts on boot.
- Disk cleanup: runner workspace directories accumulate over time. Run `docker system prune -f` periodically if disk usage climbs.

---

## CI/CD Pipeline

The GitHub Actions workflow (`.github/workflows/release.yml`) is structured to balance cost, speed, and capability.

### Pipeline Overview

```yaml
on:
  push:
    branches: [main]
    tags: ["v*"]
  pull_request:
    branches: [main]
```

Six jobs: **lint**, **test**, **build**, **build-all**, **docker**, **release**

### Job-to-Runner Mapping

| Job | Runner | When |
|-----|--------|------|
| **lint** | `ubuntu-latest` (GitHub-hosted) | All pushes + PRs |
| **test** | `ubuntu-latest` (GitHub-hosted) | All pushes + PRs |
| **build** (single linux/amd64) | `[self-hosted, linux, x64]` | PRs + pushes to `main` |
| **build-all** (full matrix) | `[self-hosted, linux, x64]` | Tags only (`v*`) |
| **docker** | `[self-hosted, linux, x64]` | `main` pushes + tags |
| **release** | `ubuntu-latest` (GitHub-hosted) | Tags only (`v*`) |

### build — Single Platform (Self-Hosted)

Runs on every PR and push to `main`. Produces a single `linux/amd64` binary for quick validation:

```yaml
build:
  if: github.event_name == 'pull_request' || (github.event_name == 'push' && github.ref == 'refs/heads/main')
  runs-on: [self-hosted, linux, x64]
  steps:
    - run: go build -ldflags="-s -w" -o basemake .
```

### build-all — Full Matrix (Tags Only)

Only triggered on `v*` tags. Cross-compiles for all 5 supported targets:

| Platform | Arch |
|----------|------|
| Linux | amd64, arm64 |
| macOS (Intel) | amd64 |
| macOS (Apple Silicon) | arm64 |
| Windows | amd64 |

```yaml
build-all:
  if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')
  runs-on: [self-hosted, linux, x64]
  strategy:
    matrix:
      goos: [linux, darwin, windows]
      goarch: [amd64, arm64]
      exclude:
        - goos: windows
          goarch: arm64
```

### docker — Container Image (Self-Hosted)

Builds multi-arch Docker images (`linux/amd64`, `linux/arm64`) and pushes to GHCR:

```yaml
docker:
  if: github.ref == 'refs/heads/main' || startsWith(github.ref, 'refs/tags/v')
  runs-on: [self-hosted, linux, x64]
```

Image location: `ghcr.io/dynamickarabo/basemake`

Tags:
- `latest` — on every `main` push
- `vX.Y.Z`, `vX.Y` — on semver tags

### release — GitHub Release (GitHub-Hosted)

Assembles tarballs and publishes a GitHub Release with auto-generated release notes. Runs on the GitHub-hosted runner since it's a lightweight orchestration step.

---

## Cost Optimization

GitHub Actions usage is minimized by routing heavy jobs to the self-hosted runner.

| Job | Runner Type | Est. Duration | GitHub Minutes |
|-----|-------------|---------------|----------------|
| lint | GitHub-hosted | ~30s | **0.5 min** |
| test | GitHub-hosted | ~30s | **0.5 min** |
| build | Self-hosted | ~30s | **0 min** |
| build-all (5 targets) | Self-hosted | ~2 min | **0 min** |
| docker (multi-arch) | Self-hosted | ~3 min | **0 min** |
| release | GitHub-hosted | ~1 min | **1 min** |

On a typical PR, only the lint and test jobs consume GitHub Actions minutes (~2 minutes total). The heavy lifting (compilation, cross-compilation, container builds) all runs on the Hetzner VPS at a flat monthly cost.

---

## Pre-Commit Hooks

Local development quality gates are enforced via a Git pre-commit hook in `.githooks/pre-commit`.

### Installation

```bash
make hooks
```

This runs `git config core.hooksPath .githooks`, configuring Git to use the project's `.githooks/` directory.

### Hook Steps

The pre-commit hook runs the following checks in sequence:

1. **`go vet ./...`** — Detect suspicious constructs, unused code, and common errors.
2. **`gofmt -d .`** — Ensure all Go source files are properly formatted. Fails on any unformatted file.
3. **`go build -o /dev/null .`** — Verify the project compiles cleanly (syntax and type check).
4. **`go test -count=1` on changed packages** — Run tests only for packages with staged `.go` changes. Falls back to no test run if no Go files are staged.

### Skipping Hooks

```bash
git commit --no-verify
```

### Makefile Integration

The same checks are available via the Makefile for explicit invocation:

```bash
make lint    # go vet + staticcheck
make fmt     # gofmt -w .  (auto-format)
make test    # go test -count=1 -v ./...
make ci      # lint + test + build (matches CI pipeline)
```

---

## Docker Compose (Integration Tests)

`docker-compose.yml` at the project root provides ephemeral database instances for integration testing:

| Service | Port | Image | Purpose |
|---------|------|-------|---------|
| `postgres-test` | `:5433` | `postgres:16-alpine` | PG driver + EXPLAIN tests |
| `mysql-test` | `:3307` | `mysql:8` | MySQL driver + EXPLAIN tests |

Both use `tmpfs` for data, so they're purely in-memory. Start with `docker compose up -d`.

---

## Infrastructure Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    GitHub.com                            │
│  ┌─────────────┐  ┌─────────────┐  ┌────────────────┐   │
│  │  lint (GH)   │  │  test (GH)   │  │  release (GH)   │   │
│  │ ~0.5 min     │  │ ~0.5 min     │  │ ~1 min          │   │
│  └──────┬───────┘  └──────┬───────┘  └───────┬─────────┘   │
│         │                 │                   │            │
│         ▼                 ▼                   ▼            │
│  ┌──────────────────────────────────────────────────┐     │
│  │       Hetzner CX33 (PopOS, 4vCPU, 8GB RAM)       │     │
│  │                                                   │     │
│  │  ┌──────────────────────────────────────────┐    │     │
│  │  │   actions.runner.*.service (systemd)     │    │     │
│  │  │   User: gh-runner                        │    │     │
│  │  │   Path: /home/gh-runner/actions-runner   │    │     │
│  │  │   Labels: [self-hosted, linux, x64]      │    │     │
│  │  │   Docker: ✔ (docker group)              │    │     │
│  │  └──────────────────────────────────────────┘    │     │
│  │                                                   │     │
│  │  ┌──────────────────────────────────────────┐    │     │
│  │  │  build (single linux/amd64)              │    │     │
│  │  │  build-all (full matrix, tags only)      │    │     │
│  │  │  docker (multi-arch → ghcr.io)           │    │     │
│  │  └──────────────────────────────────────────┘    │     │
│  └──────────────────────────────────────────────────┘     │
│                                                           │
│  ┌──────────────────────────────────────────────┐        │
│  │           ghcr.io/dynamickarabo/basemake      │        │
│  │           Container Registry                  │        │
│  └──────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────┘
```

## Quick Reference

### Common Commands

```bash
# Runner status
sudo systemctl status actions.runner.karabo-labs-basemake.basemake-runner.service

# Runner logs
journalctl -u actions.runner.karabo-labs-basemake.basemake-runner.service -f

# Restart runner
cd /home/gh-runner/actions-runner && sudo ./svc.sh stop && sudo ./svc.sh start

# View recent workflow runs
gh run list --repo karabo-labs/basemake --limit 10

# Install pre-commit hooks (local dev)
make hooks

# Run CI locally
make ci
```
