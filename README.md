# basemake — AI-powered database CLI

[![Release](https://img.shields.io/github/v/release/DynamicKarabo/basemake?style=flat&label=release)](https://github.com/DynamicKarabo/basemake/releases)
[![CI](https://github.com/DynamicKarabo/basemake/actions/workflows/release.yml/badge.svg)](https://github.com/DynamicKarabo/basemake/actions/workflows/release.yml)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://go.dev)
[![GitHub Downloads](https://img.shields.io/github/downloads/DynamicKarabo/basemake/total?style=flat&label=downloads)](https://github.com/DynamicKarabo/basemake/releases)

> **All local. All private. All yours.**  
> Talk to your database in plain English. Queries, performance analysis, and insights — no data leaves your machine.

![basemake demo](basemake-demo.gif)

## Features

- **Natural language queries** — `basemake "show me users who signed up last week"` → SQL → results
- **Zero data exfiltration** — works with [Ollama](https://ollama.ai) locally or your own API keys
- **Performance analysis** — `basemake analyze "SELECT * FROM orders"` surfaces slow scans, missing indexes
- **EXPLAIN mode** — `basemake "top products" --explain` shows execution plan alongside results
- **Multi-dialect** — PostgreSQL, MySQL, SQLite with automatic SQL generation for each
- **Output formats** — table (default), `--json`, `--csv`
- **Streaming AI** — watch SQL generate token by token, or use `--no-stream` for instant results
- **CI/CD gate** — `basemake check "query"` exits 0 if fast, 1 if slow, 2 if dangerous. Drop it in your pipeline.
- **History compounding** — past queries inform future AI responses for context-aware SQL generation
- **Config persistence** — set once with `basemake config set`, forget it
- **Interactive REPL** — `basemake repl` for an AI-assisted SQL shell

## Quick Start

```bash
# Connect to any database
basemake connect postgres://user:password@localhost:5432/mydb

# Ask questions in plain English
basemake "show me users who signed up last week"

# Analyze query performance
basemake analyze "SELECT * FROM orders WHERE status = 'pending'"

# See execution plans alongside results
basemake "top 5 products by revenue" --explain

# Output as JSON or CSV
basemake "recent orders" --json
basemake "slow queries from yesterday" --csv
```

That's it. Two commands to go from zero to querying with AI.

## Install

### Binary (Linux / macOS)

```bash
# Linux amd64
curl -sfL https://github.com/DynamicKarabo/basemake/releases/latest/download/basemake-linux-amd64.tar.gz | tar xz
sudo mv basemake /usr/local/bin/

# macOS (Apple Silicon)
curl -sfL https://github.com/DynamicKarabo/basemake/releases/latest/download/basemake-darwin-arm64.tar.gz | tar xz
sudo mv basemake /usr/local/bin/

# macOS (Intel)
curl -sfL https://github.com/DynamicKarabo/basemake/releases/latest/download/basemake-darwin-amd64.tar.gz | tar xz
sudo mv basemake /usr/local/bin/
```

### Via Go

```bash
go install github.com/DynamicKarabo/basemake@latest
```

### Docker

```bash
docker pull ghcr.io/dynamickarabo/basemake:latest
docker run --rm ghcr.io/dynamickarabo/basemake --help
```

## AI Providers

basemake works with three AI providers. Choose the one that fits your workflow.

### Ollama (Local — recommended)

Zero API costs, zero data leaves your machine. Requires [Ollama](https://ollama.ai) running locally.

```bash
basemake config set ai_provider ollama
basemake config set ollama_model llama3
basemake "show me users who signed up last week"
```

### OpenAI

```bash
export OPENAI_API_KEY="sk-..."
basemake config set ai_provider openai
basemake "show me users who signed up last week"
```

### Anthropic

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
basemake config set ai_provider anthropic
basemake "show me users who signed up last week"
```

## Examples

### Connect and introspect

```bash
$ basemake connect postgres://user:***@localhost:5433/demodb
✓ Connected to PostgreSQL (***@localhost:5433/demodb)
  Schema loaded: 3 tables, 19 columns, 10 indexes

orders (7 columns, 5 indexes)
  ├─ id integer [PK]
  ├─ user_id integer nullable
  ├─ product_id integer nullable
  ├─ quantity integer
  ├─ total numeric
  ├─ status character varying nullable
  ├─ ordered_at timestamp without time zone nullable
products (6 columns, 2 indexes)
  ├─ id integer [PK]
  ├─ name character varying
  ├─ price numeric
  ...
users (6 columns, 3 indexes)
  ├─ id integer [PK]
  ├─ name character varying
  ├─ email character varying
  ...
```

### Natural language → SQL → results

```bash
$ basemake "show me users who signed up last week"

SELECT *
FROM users
WHERE created_at >= NOW() - INTERVAL '1 week';

id | name          | email             | plan       | country
---+---------------+---------------+------------+--------
 1 | Alice Mokoena | alice@example.com | pro        | ZA
 2 | Bob Smith     | bob@example.com   | free       | US
 3 | Carol Ndlovu  | carol@example.com | pro        | ZA
 4 | Dave Patel    | dave@example.com  | enterprise | IN
(4 rows)
```

### Performance analysis

```bash
$ basemake analyze "SELECT * FROM orders WHERE status = 'delivered'"
Running EXPLAIN ANALYZE...
Analysis completed in 3ms

Execution Time: 0.13 ms
Planning Time: 0.73 ms

Scan Summary:
  Sequential Scans: 0
  Index Scans: 1
  Heaviest Node: Bitmap Heap Scan on orders (0.0ms)

Plan Tree:
Bitmap Heap Scan on orders (0.0ms, 8 rows)
  Bitmap Index Scan (0.0ms, 8 rows)
```

### Execution plans with results

```bash
$ basemake "top 5 products by revenue" --explain

SELECT
    p.id,
    p.name,
    SUM(o.quantity * o.total) AS total_revenue
FROM orders o
JOIN products p ON o.product_id = p.id
GROUP BY p.id, p.name
ORDER BY total_revenue DESC
LIMIT 5;

Execution Plan:
Limit  (cost=41.22..41.24 rows=5 width=454) (actual time=0.268..0.274 rows=5 loops=1)

id | name                       | total_revenue
---+----------------------------+--------------
 5 | Ergonomic Chair            |       2399.96
10 | Notebook Set               |       1873.75
 4 | Standing Desk              |        799.98
 8 | Noise Canceling Headphones |        499.98
 1 | Wireless Mouse             |        389.87
(5 rows)
```

## CI/CD Integration

Use `basemake check` as a merge gate in your pipeline. Reads SQL inline or from a file, runs EXPLAIN + execution timing, and exits with a predictable code.

### GitHub Actions

```yaml
name: Query Gate
on: [pull_request]
jobs:
  check-queries:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
    steps:
      - uses: actions/checkout@v4

      - name: Install basemake
        run: |
          curl -sfL https://github.com/DynamicKarabo/basemake/releases/latest/download/basemake-linux-amd64.tar.gz | tar xz
          sudo mv basemake /usr/local/bin/

      - name: Run migrations
        run: psql "$DATABASE_URL" -f migrations/001.sql

      - name: Check queries
        run: |
          basemake connect "$DATABASE_URL"
          basemake check queries/report.sql --threshold 500ms
          basemake check queries/update.sql --threshold 200ms --dry-run
```

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | ✅ Pass — query is fast and safe |
| `1` | ❌ Slow — execution exceeded time threshold |
| `2` | 🔴 Dangerous — structural issues (seq scan on large table, missing index) |
| `3` | ⚠ Error — connection failed or query invalid |

## Commands

| Command | Description |
|---------|-------------|
| `basemake connect <dsn>` | Connect to a database, introspect schema, cache locally |
| `basemake query <question\|sql>` | Ask a question or run raw SQL |
| `basemake <question>` | Shorthand — same as `query` |
| `basemake analyze <query>` | Run EXPLAIN ANALYZE, surface performance issues |
| `basemake analyze --all` | Analyze all cached tables for issues |
| `basemake check <sql\|file.sql>` | CI gate — exits 0 (pass), 1 (slow), 2 (dangerous), 3 (error) |
| `basemake repl` | Interactive SQL shell with AI assistance |
| `basemake config show` | View all configuration |
| `basemake config set <key> <value>` | Persist a config value |
| `basemake version` | Print version information |

### Query flags

| Flag | Description |
|------|-------------|
| `--json` | Output results as JSON |
| `--csv` | Output results as CSV |
| `--dry-run` | Generate SQL without executing |
| `--explain` | Show execution plan alongside results |
| `--no-stream` | Wait for full AI response (disable streaming) |

### Check flags

| Flag | Description |
|------|-------------|
| `--threshold <duration>` | Max query time (default: `1s`, e.g. `500ms`, `2s`) |
| `--dry-run` | Analyze only — don't execute the query |

## Configuration

Config is stored in `~/.basemake/config.json`. Manage it with `basemake config` commands:

```bash
basemake config set ai_provider ollama
basemake config set ollama_model llama3
basemake config set output_format json
basemake config show
```

Environment variables override config values:

| Variable | Purpose |
|----------|---------|
| `AI_PROVIDER` | Provider: `openai`, `anthropic`, `ollama` |
| `OPENAI_API_KEY` | API key for OpenAI |
| `ANTHROPIC_API_KEY` | API key for Anthropic |
| `OPENAI_MODEL` | Model override (default: `gpt-4`) |
| `ANTHROPIC_MODEL` | Model override (default: `claude-sonnet-4-20250514`) |
| `OLLAMA_MODEL` | Model override (default: `llama3`) |
| `OLLAMA_BASE_URL` | Ollama server URL (default: `http://localhost:11434/v1`) |

## Shell Completion

```bash
eval "$(basemake completion bash)"       # bash
eval "$(basemake completion zsh)"        # zsh
basemake completion fish | source        # fish
basemake completion powershell | Out-String | Invoke-Expression  # PowerShell
```

## Supported Databases

| Database | Driver | Connection String |
|----------|--------|-------------------|
| PostgreSQL | `lib/pq` | `postgres://user:pass@host:5432/dbname` |
| MySQL | `go-sql-driver/mysql` | `mysql://user:pass@host:3306/dbname` |
| SQLite | `modernc.org/sqlite` | `sqlite:/path/to/file.db` (coming soon) |

## Documentation

Comprehensive documentation for every feature, command, and configuration option lives in [`docs/`](docs/README.md).

| Document | Covers |
|----------|--------|
| [Overview](docs/overview.md) | Architecture, design decisions, data flow |
| [Commands](docs/commands.md) | Full reference for all commands |
| [Configuration](docs/configuration.md) | Config file, env vars, precedence |
| [Drivers](docs/database-drivers.md) | PostgreSQL, MySQL, SQLite internals |
| [Output Formats](docs/output-formats.md) | Table, JSON, CSV formatting rules |
| [AI Integration](docs/ai-integration.md) | NL→SQL, model selection, API details |
| [Development](docs/development.md) | Build, test, CI/CD, contributing |

## License

MIT
