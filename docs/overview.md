# basemake — Architecture Overview

**basemake** is a Go CLI tool that gives an AI awareness of your database's behavior over time. It translates natural language to SQL, profiles every query it runs, detects plan regressions, and checks its own work before executing.

## Architecture

Basemake follows a layered architecture. The core insight is that the AI has **four tools** — not one. Each tool knows something different about the database, and together they enable the AI to behave like a DBA who's been watching the database for months, not someone who just read the schema for the first time.

```
main.go                  Entry point — calls rootCmd.Execute()
└── cmd/                 CLI commands (Cobra)
    ├── root.go          Root command, persistent flags
    ├── connect.go       Database connection & schema introspection
    ├── query.go         Natural language or SQL query execution
    │   └── PlanCheck    — compares plan against profile history before executing
    ├── analyze.go       EXPLAIN ANALYZE with performance issue detection
    ├── check.go         CI/CD gate (exits 0/1/2/3)
    ├── diff.go          Schema diff between databases
    ├── watch.go         Scheduled query monitoring
    ├── index.go         Index recommendation management
    ├── config.go        Configuration management
    ├── repl.go          Interactive shell (REPL) — delegates to tui package
    ├── server.go        Team sync daemon
    └── pg_helper.go     Partial SQL parser for dialect detection
    │
    └── internal/        Private packages
        │
        ├── ai/          AI provider abstraction (OpenAI, Anthropic, Ollama, OpenCode)
        │   └── ai.go    — NL→SQL generation, streaming, provider selection
        │
        ├── db/          Database abstraction layer
        │   ├── db.go         — Database interface (Query, Explain, Introspect, etc.)
        │   ├── postgres.go   — PostgreSQL driver (lib/pq)
        │   ├── mysql.go      — MySQL driver (go-sql-driver/mysql)
        │   ├── sqlite.go     — SQLite driver (modernc.org/sqlite — pure Go, no CGo)
        │   ├── cache.go      — Schema caching to ~/.basemake/schema.json
        │   └── guardrail.go  — SELECT * detection & rewrite
        │
        ├── profile/     ★ Core memory layer — stores query behavior over time
        │   ├── normalize.go  — SQL normalisation (literals → ?, whitespace, casing)
        │   ├── planparse.go  — PostgreSQL EXPLAIN JSON parser + plan comparison + plain English explanation
        │   ├── profile.go    — Load/save query profiles from ~/.basemake/profiles/<hash>.json
        │   └── plancheck.go  — PlanCheck: compare current plan against history, return warnings
        │
        ├── observe/     ★ Proactive observation — scans profiles on REPL startup
        │   └── observe.go    — Brief(): returns the one thing worth knowing, or stays silent
        │
        ├── tui/         Bubbletea interactive terminal UI
        │   └── repl.go       — Full TUI: scrolling, tabs, ghost autocomplete, streaming, observe integration
        │
        ├── analyze/     Performance analysis & index recommendations
        │   ├── cardinality.go    — Table row count estimates
        │   └── recommendations.go — Index suggestion engine
        │
        ├── display/     Output formatting
        │   └── display.go       — Table, JSON, CSV formatters
        │
        ├── history/     Query history tracking
        │   └── history.go       — ~/.basemake/history.db (SQLite)
        │
        ├── config/      Persistent configuration
        │   └── config.go       — ~/.basemake/config.json
        │
        ├── budget/      Performance policy engine
        │
        ├── diff/        Schema diff engine
        │
        ├── server/      Team sync daemon
        │
        └── license/     License key verification
```

### The Four Tools

Basemake's AI has access to four tools, each backed by persistent data:

| Tool | Package | What it knows | When it runs |
|---|---|---|---|
| **Schema** | `internal/db/` | Tables, columns, types, foreign keys, indexes | At connect time, cached to disk |
| **Profile** | `internal/profile/` | Query execution plans, timing, row counts, plan changes over time | On `--explain`, stored per-query in `~/.basemake/profiles/` |
| **PlanCheck** | `internal/profile/` | Comparison of current plan vs profile history | Before every NL-generated query execution |
| **Observe** | `internal/observe/` | Recent plan changes, slow queries, schema drift | On REPL startup |

These aren't features bolted onto a chatbot. They're a data pipeline:

```
Schema → AI generates SQL → PlanCheck compares current plan vs profile history
                                                         ↓
                                          Warning? → Show before executing
                                                         ↓
                                          Execute → Save new profile → REPL startup → Observe
```

The tool gets smarter the longer you use it. Day one it's a translator. Day thirty it's noticed three plan regressions, warned you about two dropped indexes, and learned the normal timing baseline for every query you run.

## Design Decisions

### Why Go?

Go gives us a single static binary with zero runtime dependencies. No Python venv, no Node.js, no JVM. Download and run. Cross-compilation for the 8-build CI matrix (5 platforms × 2 archs where applicable) "just works."

### Why SQLite for Query History?

Basemake uses two SQLite databases:

1. **`~/.basemake/history.db`** — query history (question, SQL, timing, database, AI provider). Used for context compounding in NL→SQL prompts (last 5 queries inform the current one).

2. **`~/.basemake/profiles/`** — JSON files, one per normalized query fingerprint. Profile data is JSON (not SQLite) to allow per-query atomic reads/writes without database locking. A single profile file contains all runs for that query on a specific database, making it easy to inspect, delete, or share individual query histories.

### Why JSON Profiles Instead of a Database?

The profile directory (`profiles/`) uses one JSON file per normalized query fingerprint because:

- **Atomicity** — reading/writing one query's history doesn't require a database connection or lock
- **Transparency** — `~/.basemake/profiles/*.json` is human-readable and trivially inspectable
- **Portability** — you can delete, archive, or share individual profile files with `cp` or `scp`
- **Simplicity** — no schema migrations needed for a data format that's append-only

### PostgreSQL JSON Plan Parsing

Plan change detection uses PostgreSQL's `EXPLAIN (FORMAT JSON)` output. The plan tree is recursively walked to find **table-access nodes** (nodes with a `Relation Name` field). Intermediate nodes like Sort, Hash, Bitmap Index Scan, and Aggregate are skipped — they don't represent table access decisions.

Node types compared:
- **Seq Scan** — full table scan (slow on large tables)
- **Index Scan** — index-based access (fast for selective queries)
- **Index Only Scan** — index-only access (fastest, no table heap access)
- **Bitmap Heap Scan** — bitmap-based access (medium, often with Bitmap Index Scan child)

A plan change between runs is detected when any table-access node's type changes, or when an Index Scan switches to a different index name.

### Pure Go SQLite (no CGo)

SQLite is powered by `modernc.org/sqlite` — a transpilation of the SQLite C library to Go. This means:
- No C compiler required on build
- No SQLite shared library on the target machine
- Cross-compilation "just works" (important for our build matrix)
- WAL mode enabled by default for concurrent read performance

### Interface-based Drivers

The `Database` interface in `internal/db/db.go` defines the contract:

```go
type Database interface {
    Name() string
    Dialect() string
    Close() error
    Introspect(ctx context.Context) (*Schema, error)
    Query(ctx context.Context, sql string) (*Rows, error)
    Explain(ctx context.Context, sql string) (string, error)
    ExplainJSON(ctx context.Context, sql string) (string, error)
    ExplainNoAnalyze(ctx context.Context, sql string) (string, error)
}
```

Each driver (PostgreSQL, MySQL, SQLite) implements this interface. The driver package uses a registry pattern — drivers self-register in `init()` and are selected by DSN prefix detection at connect time.

The `ExplainNoAnalyze` method is critical for PlanCheck — it returns a query plan without executing the query. This lets PlanCheck compare plans with zero side effects.

### Silent Observations

The observe module (`Brief()`) is designed to be silent by default. It only produces output when it detects a signal worth reporting. The constraints that make this work:

1. **Priority-ordered signals** — plan changes > slow queries > schema drift. Only the highest-priority signal is reported.
2. **Dedup by timestamp** — once a signal is reported, it won't repeat unless new profile data appears after the last report.
3. **First-run silence** — the first time observe runs, it stores state silently. No "welcome to observing!" output.
4. **Zero live DB calls** — reads only local cached state (profiles directory + schema.json). Works offline.

### PlanCheck Design

PlanCheck runs before every NL-generated query (not raw SQL, not `--explain` mode). It does NOT save to the profile — that's the caller's responsibility. This keeps PlanCheck fast and non-side-effecty:

1. Normalize the SQL and load the profile for this query
2. Run `ExplainNoAnalyze` to get the current plan (no execution)
3. Compare plan nodes against the previous run's plan
4. Return warnings in priority order: index dropped → Seq Scan → historical regression

## Data Flow

### NL Query Flow (with PlanCheck)

```
User: "show me orders with status 0"
    │
    ▼
Load cached schema
    │
    ▼
AI generates SQL: SELECT * FROM orders WHERE status = 0
    │
    ├──► PlanCheck: ExplainNoAnalyze → load profile → compare plans
    │       │
    │       ├── Plan unchanged → silent, proceed
    │       │
    │       └── Plan changed → ⚠ "idx_status_id was dropped"
    │                          Run anyway? [Y/n]
    │                               │
    │                          User confirms → proceed
    │
    ▼
Validate SQL via EXPLAIN
    │
    ▼
Execute query → measure timing → format results
    │
    ▼
Record in history + if --explain, save profile
```

### REPL Startup Flow (with Observe)

```
basemake
    │
    ▼
Load saved DSN → auto-connect → introspect schema
    │
    ▼
Observe: load state → scan profiles → check signals
    │       │
    │       ├── Plan change detected → show brief in viewport
    │       │
    │       └── Nothing → silent, show prompt
    │
    ▼
Display prompt: "Type .help for commands"
```

### Profile Flow

```
basemake "top products" --explain
    │
    ▼
Generate SQL (or use raw SQL)
    │
    ▼
Run EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) → show plan
    │
    ▼
Execute query → measure timing
    │
    ▼
Run ExplainNoAnalyze → get JSON plan
    │
    ▼
Normalize SQL → hash → load existing profile
    │
    ▼
Append new run (timing + plan + hash + db fingerprint)
    │
    ├── First run → "⚡ Profiled 1 time."
    │
    └── Repeated → "⚡ Profiled 3 times. Avg: 124ms."
                    "   ⚠ Plan changed: ..."
```

## Build & CI

### Development

```bash
go run .                        # Run the CLI
go run . connect <dsn>          # Connect and query
go test ./...                   # Run all tests
go vet ./...                    # Static analysis
```

### CI/CD Pipeline

The release pipeline (`.github/workflows/release.yml`) has 4 jobs:

1. **lint** — `go vet` + formatting check
2. **test** — `go test -v -count=1 ./...` (no cached results)
3. **build** — Matrix: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64. Stripped binaries with `-ldflags="-s -w"`
4. **release** — On tag push (`v*`): creates archives, SHA256 checksums, publishes GitHub Release

Total CI time: ~2 minutes.

### Pre-Commit Hook

The repository has a pre-commit hook that runs `go vet`, `gofmt`, `go build`, and test suites on changed packages. All checks must pass before commit.
