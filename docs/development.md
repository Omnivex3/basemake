# Development Guide

## Prerequisites

- Go 1.25+ (matching `go.mod` / CI)
- Make (optional, for convenience)
- Access to a PostgreSQL, MySQL, or SQLite database for testing

## Quick Start

```bash
# Clone
git clone https://github.com/DynamicKarabo/dbai.git
cd dbai

# Build
go build -o dbai .

# Test
go test -v -count=1 ./...

# Lint
go vet ./...
```

## Project Structure

```
.
├── main.go                          # Entry point
├── cmd/                             # CLI commands (Cobra)
│   ├── root.go                      # Root command setup
│   ├── connect.go                   # Database connection
│   ├── query.go                     # Query execution
│   ├── analyze.go                   # EXPLAIN ANALYZE
│   ├── repl.go                      # Interactive shell
│   ├── completion.go                # Shell completion
│   ├── version.go                   # Build info
│   └── cmd_test.go                  # Command tests
├── internal/                        # Private packages
│   ├── ai/ai.go                     # OpenAI integration
│   ├── analyze/plan.go              # Plan parsing & issue detection
│   ├── config/config.go             # Persistent config
│   ├── db/                          # Database drivers
│   │   ├── db.go                    # Interface, types, connection management
│   │   ├── driver.go                # Driver registry
│   │   ├── cache.go                 # Schema caching
│   │   ├── config.go               # DSN persistence
│   │   ├── errors.go               # Sentinel errors
│   │   ├── postgres.go             # PostgreSQL driver
│   │   ├── mysql.go                 # MySQL driver
│   │   └── sqlite.go               # SQLite driver
│   └── display/display.go          # Output formatting
├── .github/workflows/release.yml   # CI/CD (build, test, release)
├── .golangci.yml                    # Linter configuration
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## Testing

### Running Tests

```bash
# All tests
go test -v -count=1 ./...

# Specific package
go test -v -count=1 ./internal/display/...
go test -v -count=1 ./internal/db/...

# With race detection
go test -race -count=1 ./...
```

### Test Counts

| Package | Tests | Description |
|---------|-------|-------------|
| `cmd` | 3 | Version, completion (4 shells + invalid) |
| `internal/analyze` | 7 | Plan parsing, issue detection, formatting |
| `internal/config` | 4 | Defaults, save/load, missing file, overwrite |
| `internal/db` | 8 | Schema round-trip, helpers, driver detection (6), SQLite (5) |
| `internal/display` | 7 | Table, JSON, CSV, TSV, empty, isNumeric, alignment |
| **Total** | **29 test functions** | |

Note: `go test` runs all test functions across all packages. The CI pipeline reports "44 tests" because individual test functions within table-driven tests and sub-tests are counted separately by `go test -v`.

### Test Patterns

- **No database dependency**: All tests use in-memory SQLite (`sqlite:file::memory:?mode=memory&cache=shared`) or mock data structures
- **Deterministic**: `-count=1` prevents test caching interference
- **Table-driven tests**: Used in `display_test.go` (isNumeric) and `driver_test.go` (DSN detection)
- **Golden file tests**: Not used — all assertions are inline

### Test Environment

CI runs on `ubuntu-latest` with Go 1.25. Tests don't require:
- PostgreSQL or MySQL servers (SQLite in-memory is sufficient)
- OpenAI API key
- Network access
- Docker

## Linting

### Local

```bash
# Go vet (built-in)
go vet ./...

# Staticcheck (optional)
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

### CI

The CI pipeline runs both `go vet` and `staticcheck` via `dominikh/staticcheck-action@v1`.

### Configuration

`.golangci.yml`:

```yaml
version: "2"
linters:
  default: standard
  disable:
    - depguard      # Disabled — not relevant for this project
  settings:
    gosec:
      excludes:
        - G115      # Integer overflow conversion (false positives in count vars)
  exclusions:
    generated: lax
    presets:
      - comments
      - common-false-positives
      - legacy
      - std-error-handling
```

## Building

### Development Build

```bash
go build -o dbai .
```

### Stripped Build (smaller binary)

```bash
go build -ldflags="-s -w" -o dbai .
```

### Cross-Compilation

```bash
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dbai-linux-amd64 .
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dbai-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dbai-windows-amd64.exe .
```

## CI/CD Pipeline

The GitHub Actions workflow (`.github/workflows/release.yml`) has 4 jobs:

### `lint`
- Runs `go vet ./...`
- Runs `staticcheck` (latest)
- Fast (<30s)

### `test`
- `go test -v -count=1 ./...`
- Runs in parallel with lint
- Fast (<30s)

### `build`
- Needs: lint + test
- Matrix: 5 platform/arch combos
  - linux/amd64, linux/arm64
  - darwin/amd64, darwin/arm64
  - windows/amd64 (excluded: windows/arm64)
- Strips debug symbols (`-ldflags="-s -w"`)
- Uploads each binary as a separate artifact (1-day retention)

### `release`
- Runs only on tag push (`v*`)
- Needs: lint + test + build
- Downloads all artifacts
- Creates `.tar.gz` archives with a clean binary name (`dbai` not `dbai-linux-amd64`)
- Generates SHA256 `checksums.txt`
- Publishes GitHub Release via `softprops/action-gh-release`
- Auto-generates release notes from commits

### Build Matrix Details

```yaml
matrix:
  goos: [linux, darwin, windows]
  goarch: [amd64, arm64]
  exclude:
    - goos: windows
      goarch: arm64
```

8 builds total (5 distinct targets):
- linux/amd64 — primary, 99% of use
- linux/arm64 — Raspberry Pi, ARM servers
- darwin/amd64 — Intel Mac
- darwin/arm64 — Apple Silicon Mac
- windows/amd64 — Windows

### Release Tagging

```bash
git tag v1.0.0
git push origin v1.0.0
```

### Environment Variables in CI

```yaml
env:
  GO_VERSION: "1.25"
```

## VCS Build Info

Go 1.18+ automatically embeds VCS information into the binary:

```
$ dbai version
dbai dev
  Go version: go1.25.0
  Platform: linux/amd64
  Commit: 4692dc5
```

The `version.go` command reads this via `debug.ReadBuildInfo()`:

```go
type buildInfo struct {
    version   string  // From bi.Main.Version (e.g., "v1.0.0" or "dev")
    goVersion string  // runtime.Version()
    os        string  // runtime.GOOS
    arch      string  // runtime.GOARCH
    revision  string  // vcs.revision setting (first 7 chars)
    buildTime string  // vcs.time setting (first 10 chars)
    dirty     bool    // vcs.modified setting
}
```

- No `-ldflags -X` injection needed
- Commit hash truncated to 7 characters
- Build time truncated to date (`YYYY-MM-DD`)
- Tagged releases show the version tag; untagged builds show `"dev"`

## Adding a New Database Driver

1. Create `internal/db/<name>.go`
2. Implement the `Database` interface (6 methods)
3. Implement `driverConnector` interface (2 methods)
4. Register in `driver.go`'s `init()` function
5. Add DSN detection tests in `driver_test.go`
6. Write driver-specific tests (introspection, query, explain)

```go
type myDriver struct{}

func (d *myDriver) Scheme() string { return "mydb" }

func (d *myDriver) Connect(dsn string) (Database, error) {
    // Open connection, ping, return driver struct
}

type myDB struct {
    conn *sql.DB
    dsn  string
}

func (m *myDB) Name() string { return fmt.Sprintf("MyDB (%s)", maskDSN(m.dsn)) }
func (m *myDB) Close() error { return m.conn.Close() }
func (m *myDB) Introspect(ctx context.Context) (*Schema, error) { ... }
func (m *myDB) Query(ctx context.Context, sql string) (*Rows, error) { ... }
func (m *myDB) Explain(ctx context.Context, query string) (string, error) { ... }
func (m *myDB) ExplainJSON(ctx context.Context, query string) (string, error) { ... }
```

## Adding a New Command

1. Create `cmd/<name>.go`
2. Define a `*cobra.Command` variable
3. Register with `rootCmd.AddCommand()` in an `init()` function
4. Add flags in the same `init()`
5. Add command tests in `cmd/cmd_test.go`
6. Add shell completion support (automatic via Cobra)

## Dependencies

| Dependency | Version | Purpose | License |
|------------|---------|---------|---------|
| `github.com/spf13/cobra` | v1.10.2 | CLI framework | Apache 2.0 |
| `github.com/spf13/pflag` | v1.0.9 | Flag parsing (Cobra dep) | BSD-3 |
| `github.com/lib/pq` | v1.12.3 | PostgreSQL driver | MIT |
| `github.com/go-sql-driver/mysql` | v1.10.0 | MySQL driver | MPL 2.0 |
| `modernc.org/sqlite` | v1.50.1 | SQLite driver (pure Go) | BSD-3 |
| `github.com/dustin/go-humanize` | v1.0.1 | Indirect (modernc dep) | MIT |
| `github.com/google/uuid` | v1.6.0 | Indirect (modernc dep) | BSD-3 |
| `github.com/mattn/go-isatty` | v0.0.20 | Indirect (modernc dep) | MIT |
| `github.com/ncruces/go-strftime` | v1.0.0 | Indirect (modernc dep) | MIT |
| `github.com/remyoudompheng/bigfft` | v0.0.0 | Indirect (modernc dep) | BSD-3 |

## Design Constraints

1. **Zero runtime deps** beyond Go standard library + database drivers
2. **No CGo** — cross-compilation must work without a C compiler
3. **Config in `~/.dbai/`** — follows XDG convention for CLI tools
4. **Secrets in env vars only** — API keys never written to disk by dbai
5. **Stderr for info, stdout for data** — following Unix pipeline philosophy
6. **Single binary** — no runtime, no interpreter, no VM needed

## Known Technical Debt

1. **PostgreSQL-only NL→SQL**: System prompt targets PostgreSQL dialect regardless of connected database
2. **TSV not exposed**: FormatTSV exists in the display package but has no CLI flag or config option
3. **Single active connection**: The global `var active Database` means only one connection at a time
4. **`SchemaForPrompt()` allocates**: Builds the schema string from scratch on every NL query
5. **Test `TestNumericAlignment`**: Marked as test but only logs lines — no assertions
6. **MySQL DBName**: Set to the full DSN string instead of extracting just the database name
7. **History depth hardcoded**: 5 recent NL queries included in context — not configurable
8. **No streaming config toggle**: Can only disable per-command with `--no-stream`, no config file option
