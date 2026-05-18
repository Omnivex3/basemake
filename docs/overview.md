# basemake — AI-Powered Database CLI

**basemake** is a Go CLI tool that lets you query, analyze, and optimize databases using natural language. It bridges the gap between plain English questions and SQL execution — connecting to PostgreSQL, MySQL, and SQLite databases, introspecting their schemas, and translating your questions into optimized queries.

## Architecture

`basemake follows a standard Go CLI architecture using the [Cobra](https://github.com/spf13/cobra) framework.

```
main.go                  Entry point — calls rootCmd.Execute()
└── cmd/                 CLI commands (Cobra commands)
    ├── root.go          Root command, persistent flags, subcommand registration
    ├── connect.go       Database connection & schema introspection
    ├── query.go         Natural language or SQL query execution
    ├── analyze.go       EXPLAIN ANALYZE with performance issue detection
    ├── repl.go          Interactive shell with dot-commands
    ├── completion.go    Shell completion script generation (bash/zsh/fish/powershell)
    ├── version.go       Build info (version, Go runtime, platform, commit)
    └── cmd_test.go      Command-level tests
└── internal/            Private packages (not importable outside the module)
    ├── ai/              OpenAI API integration for NL→SQL generation
    │   └── ai.go
    ├── analyze/         PostgreSQL JSON EXPLAIN plan parser & issue detector
    │   ├── plan.go
    │   └── plan_test.go
    ├── config/          Persistent JSON config (~/.basemake/config.json)
    │   ├── config.go
    │   └── config_test.go
    ├── db/              Database abstraction layer with 3 driver implementations
    │   ├── db.go         Database interface, Schema types, connection management
    │   ├── driver.go     Driver registration & DSN-based detection
    │   ├── postgres.go   PostgreSQL driver (lib/pq)
    │   ├── mysql.go      MySQL driver (go-sql-driver/mysql)
    │   ├── sqlite.go     SQLite driver (modernc.org/sqlite — pure Go, no CGo)
    │   ├── cache.go      Schema caching to JSON on disk (~/.basemake/schema.json)
    │   ├── config.go     DSN persistence (old format, superseded by config)
    │   ├── errors.go     Sentinel errors (ErrNoConnection, ErrUnsupported)
    │   └── *_test.go     Tests for each driver & schema
    └── display/         Output formatting engine (table, JSON, CSV, TSV)
        ├── display.go
        └── display_test.go
```

## Design Decisions

### Why Go 1.25?

Go gives us a single static binary with zero runtime dependencies. No Python venv, no Node.js, no JVM. Download and run. The 1.25 toolchain provides `debug.ReadBuildInfo()` for VCS commit embedding and modern generics (though basemake doesn't use generics — the type complexity didn't justify it).

### Why Cobra?

Cobra is the de-facto standard for Go CLIs. It provides:
- POSIX-compliant flag parsing via pflags
- Automatic help generation with `--help` / `-h`
- Built-in shell completion generators (bash, zsh, fish, powershell)
- Nested subcommand support
- Standard `Run` / `RunE` error handling patterns

### Pure Go SQLite (no CGo)

SQLite is powered by `modernc.org/sqlite` — a transpilation of the SQLite C library to Go. This means:
- No C compiler required on build
- No SQLite shared library on the target machine
- Cross-compilation "just works" (important for our 8-build CI matrix)
- WAL mode enabled by default for concurrent read performance

### Interface-based Drivers

The `Database` interface in `internal/db/db.go` defines the contract:

```go
type Database interface {
    Name() string
    Close() error
    Introspect(ctx context.Context) (*Schema, error)
    Query(ctx context.Context, sql string) (*Rows, error)
    Explain(ctx context.Context, sql string) (string, error)
    ExplainJSON(ctx context.Context, sql string) (string, error)
}
```

Each driver (PostgreSQL, MySQL, SQLite) implements this interface. The driver package uses a registry pattern — drivers self-register in `init()` and are selected by DSN prefix detection at connect time.

### Secrets Handling

API keys and database credentials are **never** stored in config files:
- `OPENAI_API_KEY` — environment variable only
- Database passwords — embedded in the DSN string (standard practice)
- Config stores only: default DSN (can include credentials), output format, AI model name
- Connection strings are masked in logs: `***@host:port/dbname`

### Auto-Reconnect

Commands that need a database connection don't require the user to explicitly reconnect. If no active connection exists, basemake attempts to reconnect using either:
1. The `BASEMAKE_DSN` environment variable
2. The `default_dsn` from `~/.basemake/config.json`
3. The legacy DSN cache (`~/.basemake/dsn.txt` — deprecated)

## Data Flow

```
User Input
    │
    ├── SQL? ──────────────────► Execute directly
    │
    └── Natural Language? ──► Load cached schema
                                │
                                ▼
                            OpenAI API (gpt-4 / configured model)
                                │
                                ▼
                            Generated SQL
                                │
                                ▼
                            Validate via EXPLAIN
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
              --dry-run?              Execute SQL
              (print & exit)              │
                                          ▼
                                    Format output
                                    (table/JSON/CSV/TSV)
                                          │
                                          ▼
                                    stdout + row count on stderr
```

## Build & Release Pipeline

The CI/CD pipeline (`.github/workflows/release.yml`) has 4 jobs:

1. **lint** — `go vet` + staticcheck
2. **test** — `go test -v -count=1 ./...` (all 44 tests, no flaky caching)
3. **build** — Matrix: 5 platform/arch combos (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64). Strip debug symbols with `-ldflags="-s -w"`
4. **release** — On tag push (`v*`): creates archives, SHA256 checksums, publishes GitHub Release via `softprops/action-gh-release`

Total CI time: ~2 minutes (lint + test run in parallel, build runs after both pass).
