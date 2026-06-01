# basemake — DBA that checks its own work

[![Release](https://img.shields.io/github/v/release/karabo-labs/basemake?style=flat&label=release)](https://github.com/karabo-labs/basemake/releases)
[![CI](https://github.com/karabo-labs/basemake/actions/workflows/release.yml/badge.svg)](https://github.com/karabo-labs/basemake/actions/workflows/release.yml)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat&logo=go)](https://go.dev)

A CLI tool that wraps an AI around your database with memory. It translates natural language to SQL, profiles every query it runs, detects plan regressions over time, and checks its own work before executing.

![basemake demo](assets/basemake-demo.gif)

Basemake maintains a **profile history** for every query — storing execution plans, timings, row counts, and plan changes. That history powers four capabilities that get better the longer you use the tool:

| Capability | What it does |
|------------|-------------|
| **Schema** | Knows your tables, columns, types, foreign keys, indexes |
| **Profile** | Remembers every query's plan and timing across runs |
| **PlanCheck** | Compares current plan against history before executing |
| **Observe** | Scans all profiles on REPL startup, surfaces what changed |

The AI uses all four. It checks its work, notices regressions, and flags changes. When nothing's wrong, it stays quiet.

### What this looks like in practice

**A question goes through AI, then PlanCheck catches a regression before execution:**

```
You > show me orders with status 0

⚠ idx_status_id was dropped since the last profile.
  This query may be slower. Run ANALYZE or recreate the index.

Run anyway? [Y/n]:
```

The AI caught the plan change before executing — because it compared against the profile history.

**REPL startup surfaces observations:**

```
📦 1 tables — schema cached ✅

══════════════════════════════════════
  ⚠ Plan changed: The planner stopped
  using idx_status_id on orders.
  Run ANALYZE.
══════════════════════════════════════

Type .help for commands  ·  ask your question or enter SQL
>
```

One line. Then it gets out of the way.

**Nothing changed — says nothing:**

No confirmation. No dashboard. Silence is the signal that the database is healthy.

## Quick Start

```bash
# Connect to any database
basemake connect postgres://user:***@localhost:5432/mydb

# Ask questions in plain English — basemake checks before executing
basemake "show me users who signed up last week"

# Profile a query (saves plan + timing for future comparisons)
basemake "top 5 products by revenue" --explain

# Start the REPL — see observations on startup, ask questions interactively
basemake
```

### Interactive REPL

```bash
# Start the AI-assisted SQL shell
basemake

# Tab completion — press Tab to complete table/column names
You > SELECT * FROM or[Tab] → SELECT * FROM orders

# Cancel a running query with Ctrl+C or Escape
You > SELECT * FROM really_big_table
⏹  Query cancelled

# Save queries as named bookmarks
You > .save weekly-report

# Replay them later
You > .run weekly-report

# Export results to a file
You > .export results.csv
💾 Exported 42 rows to results.csv

# Toggle read-only mode for production safety
You > .readonly
✅ Read-only mode: ON
You > DELETE FROM users
⚠ Write queries are blocked in read-only mode.

# Check your setup
basemake doctor
basemake init   # one-command setup wizard
```

## Install

### Binary (Linux / macOS)

```bash
# Linux amd64
curl -sfL https://github.com/karabo-labs/basemake/releases/latest/download/basemake-linux-amd64.tar.gz | tar xz
sudo mv basemake /usr/local/bin/

# macOS (Apple Silicon)
curl -sfL https://github.com/karabo-labs/basemake/releases/latest/download/basemake-darwin-arm64.tar.gz | tar xz
sudo mv basemake /usr/local/bin/

# macOS (Intel)
curl -sfL https://github.com/karabo-labs/basemake/releases/latest/download/basemake-darwin-amd64.tar.gz | tar xz
sudo mv basemake /usr/local/bin/
```

### Via Go

```bash
go install github.com/karabo-labs/basemake@latest
```

### Docker

```bash
docker pull ghcr.io/karabo-labs/basemake:latest
docker run --rm ghcr.io/karabo-labs/basemake --help
```

## AI Providers

Works with four AI providers:

### Ollama (Local)

Zero API costs, no data leaves your machine. Requires [Ollama](https://ollama.ai) running locally.

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

### OpenCode

```bash
export OPENCODE_API_KEY="sk-..."
basemake config set ai_provider opencode
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
          curl -sfL https://github.com/karabo-labs/basemake/releases/latest/download/basemake-linux-amd64.tar.gz | tar xz
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
| `basemake diff` | Schema diff between two databases (live, cached, or file) |
| `basemake budget` | Database performance policy as code |
| `basemake watch` | Monitor a query on a schedule, alert on regression |
| `basemake server` | Start the team sync daemon |
| `basemake sync push` | Push a query event to the team server |
| `basemake sync history` | Show team query log |
| `basemake config show` | View all configuration |
| `basemake config set <key> <value>` | Persist a config value |
| `basemake repl` | Interactive SQL shell with AI assistance |
| `basemake init` | One-command setup: detect DB, configure AI, test query |
| `basemake doctor` | Diagnose connections, schema, AI config in one shot |
| `basemake use <name>` | Switch to a named connection |
| `basemake version` | Print version information |

### Query flags

| Flag | Description |
|------|-------------|
| `--json` | Output results as JSON |
| `--csv` | Output results as CSV |
| `--dry-run` | Generate SQL without executing |
| `--explain` | Show execution plan alongside results |
| `--no-stream` | Wait for full AI response (disable streaming) |
| `--readonly` | Block write queries (INSERT/UPDATE/DELETE/DROP/ALTER/CREATE/TRUNCATE) |

### REPL commands (interactive shell)

Enter `basemake repl` (or just `basemake`) for the AI-assisted SQL shell.

| Command | Description |
|---------|-------------|
| `.help` | Show help with all commands and keyboard shortcuts |
| `.quit` | Exit the REPL |
| `.tables` | List tables in the current database |
| `.schema` | Show full database schema |
| `.connect <dsn>` | Connect to a database |
| `.refresh` | Re-introspect and cache schema |
| `.history` | Show past queries (most recent first) |
| `.replay <N>` | Re-run query N from history (1 = most recent) |
| `.export <file>` | Save last result as CSV/JSON/MD |
| `.info` | Show connection, AI provider, version, read-only status |
| `.readonly` | Toggle write protection on/off |
| `.save <name>` | Bookmark the last query |
| `.run <name>` | Run a saved query |
| `.saved` | List all saved queries |

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Run query or send message |
| `Tab` | Complete table / column names |
| `Esc` / `Ctrl+C` | Cancel running query |
| `Ctrl+C` (idle) | Exit the REPL |

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
| `AI_PROVIDER` | Provider: `openai`, `anthropic`, `ollama`, `opencode` |
| `OPENAI_API_KEY` | API key for OpenAI |
| `ANTHROPIC_API_KEY` | API key for Anthropic |
| `OPENCODE_API_KEY` | API key for OpenCode |
| `OPENAI_MODEL` | Model override (default: `gpt-4`) |
| `ANTHROPIC_MODEL` | Model override (default: `claude-sonnet-4-20250514`) |
| `OLLAMA_MODEL` | Model override (default: `llama3`) |
| `OPENCODE_MODEL` | Model override (default: `deepseek-chat`) |
| `OLLAMA_BASE_URL` | Ollama server URL (default: `http://localhost:11434/v1`) |
| `OPENCODE_BASE_URL` | OpenCode server URL (default: `https://api.opencode.ai/v1`) |
| `OPENAI_BASE_URL` | OpenAI API base URL |

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
| PostgreSQL | `lib/pq` | `postgres://user:***@host:5432/dbname` |
| MySQL | `go-sql-driver/mysql` | `mysql://user:***@host:3306/dbname` |
| SQLite | `modernc.org/sqlite` | `sqlite:/path/to/file.db` |

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
