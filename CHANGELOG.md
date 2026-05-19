# Changelog

## v0.6.0 — Index Engine, Multi-Provider, TUI Overhaul (2025-05-19)

### 🔮 AI Provider System

basemake now supports **4 AI providers** with a pluggable, streaming, cost-aware engine.

#### Providers

| Provider | Auth | Default Model | Notes |
|----------|------|---------------|-------|
| OpenAI | `OPENAI_API_KEY` | `gpt-4` | Standard chat API |
| Anthropic | `ANTHROPIC_API_KEY` | `claude-sonnet-4-20250514` | Messages API |
| Ollama | None (local) | `llama3` | Runs on `localhost:11434/v1` |
| OpenCode | `OPENCODE_API_KEY` | `deepseek-chat` | $10/mo sub at opencode.ai |

#### Architecture

```go
type Provider interface {
    Name() string
    GenerateSQL(ctx, systemPrompt, question) (string, error)
    GenerateSQLStream(ctx, systemPrompt, question) (<-chan string, error)
}
```

- **`internal/ai/ai.go`** — Provider interface, `SelectedProvider()`, `QuestionToSQL()`, `QuestionToSQLStream()`, `PingProvider()`
- **`internal/ai/openai.go`** — OpenAI + OpenCode (OpenAI-compatible API)
- **`internal/ai/anthropic.go`** — Anthropic Messages API
- **`internal/ai/ollama.go`** — Ollama local endpoint

#### Key Discovery Chain

Every setting follows the same precedence:
1. Environment variable (`OPENAI_API_KEY`, `OPENAI_MODEL`, `AI_PROVIDER`, etc.)
2. Config file (`~/.basemake/config.json` field)
3. Hard-coded default

No secrets are stored on disk — only env vars.

#### New Functions

- **`PingProvider()`** — tests connectivity on demand, used by `basemake doctor` and the interactive provider selector
- **`EstimateCost(model, inputTokens, outputTokens)`** — returns human-readable cost estimate (e.g. `~$0.015`)
- **`EstimateTokens(text)`** — rough token count (~4 chars/token)
- **`ProviderInfo()`** — returns `"opencode/deepseek-chat"` label, replaces duplicate TUI label logic
- **`ModelPricing`** map — known pricing for 8 models (GPT-4, GPT-4o, Claude Sonnet 4, etc.)

#### Interactive Provider Selector

**`basemake config set-ai-provider`** — Bubbletea TUI with 4 screens:

```
◇ basemake — AI Provider Setup

Select an AI provider:
 ◉ OpenAI    GPT-4o, GPT-4o-mini — the classic
   Anthropic Claude Sonnet 4, Haiku — smart & fast
   Ollama    Local models — free, private, runs on your machine
   OpenCode  Open models, $10/mo subscription
```

1. Pick provider → curated model list
2. Pick model (or "Custom model..." for free-form)
3. Test connection via `PingProvider()`
4. Saves to `~/.basemake/config.json`

File: `internal/tui/provider_selector.go` (230 lines)

---

### 📊 Index Recommendation Engine

`internal/analyze/cardinality.go` — PostgreSQL statistics engine that powers index suggestions.

#### pg_stats Reader

Extracts per-column statistics from `pg_catalog.pg_stats`:
- **n_distinct** — distinct value count (-1 = unknown, fractional = sampled)
- **null_frac** — fraction of NULL values
- **avg_width** — average column width in bytes
- **correlation** — physical order correlation with the table
- **MCV (Most Common Values)** — top frequent values with frequencies
- **MCF (Most Common Frequencies)** — matching frequency array

#### Selectivity Math

| Operator | Selectivity Formula |
|----------|-------------------|
| `=` / `IN` | MCV frequency match, or `1/n_distinct` fallback |
| `>` / `<` / `>=` / `<=` | `(1 - correlated_position)` with MCV interpolation |
| `BETWEEN` | `upper_sel - lower_sel` using range selectivities |
| `LIKE` | `0.05` (fixed heuristic for prefix patterns) |

#### Index Trade-off Estimates

- **Selectivity** — estimated fraction of rows returned
- **Size vs speed** — qualitative trade-off (excellent/good/moderate/poor)
- **Confidence** — high (based on MCV), medium (based on n_distinct), speculative (n_distinct=-1)
- **n_distinct = -1** gracefully falls back to "speculative" confidence

#### Filter Parser

Parses SQL WHERE clauses extracted from plan nodes:
- Splits AND/OR predicates
- Extracts columns from all operator types (=, IN, >, <, >=, <=, <>, !=, IS, BETWEEN, LIKE)
- Strips table qualifiers (`users.id` → `id`) and type casts (`col::int` → `col`)
- Filters SQL keywords (CASE, WHEN, COALESCE, etc.)
- Rejects non-alphanumeric table names as SQL injection guard

#### Output

```
💡 Index Suggestions

  Table: orders
    ┌── idx_orders_created_at on (created_at)
    │   Selectivity: 3.8% — very selective
    │   Confidence:  high (based on MCV data)
    │   Trade-off:   excellent (small index, big query win)
    ├── Status: pending
```

---

### 🗄 Recommendation Persistence

`internal/analyze/rec_store.go` — durable index recommendation storage.

#### Storage Format

`~/.basemake/recommendations.json`:
```json
{
  "recommendations": [
    {
      "table": "orders",
      "columns": ["created_at"],
      "type": "single-column",
      "reason": "Sequential scan on orders (8000 rows, 3.2ms)",
      "selectivity": 0.038,
      "confidence": "high",
      "created_at": "2026-05-19T12:00:00Z",
      "status": "pending"
    }
  ]
}
```

#### Status Transitions

```
pending ──apply──▶ applied
pending ──dismiss─▶ dismissed
```

#### Merge Dedup

- Identical table+columns+type → updates reason/timestamp
- Partial match (same table, overlapping cols) → updates reason
- No match → appends new entry

#### Staleness Detection

- `rec_store.StaleReport(days)` returns recommendations older than N days
- TUI shows startup alert: `⏰ 3 index recommendations are 7+ days old`

#### CLI

```
basemake index list              # Show all recommendations
basemake index list --stale 7    # Show stale recommendations
basemake index list --status applied
basemake index apply <id>       # Show SQL → confirm → CREATE INDEX CONCURRENTLY
basemake index dismiss <id>     # Dismiss without applying
```

File: `cmd/index.go` (170 lines)

---

### 🎮 TUI Enhancements

#### REPL Features

| Feature | Dot Command | Description |
|---------|-------------|-------------|
| Chat mode | `.ask` | Free-form chat with AI (no SQL required) |
| Exit chat | `.sql` / `Esc` | Return to query mode |
| Explain | `.explain` | Show execution plan for last query |
| Analyze | `.analyze` | Full EXPLAIN ANALYZE + index suggestions |
| Export | `.export <file>` | Save last result as CSV, JSON, or Markdown |
| Replay | `.replay <N>` | Re-run any query from history |
| Save/Run | `.save <name>` / `.run <name>` | Bookmark and replay named queries |
| Saved list | `.saved` | List all saved queries |
| Toggle readonly | `.readonly` | Toggle write protection mid-session |
| Info | `.info` | Dashboard with DB, provider, version, mode |
| Refresh | `.refresh` | Re-introspect without disconnecting |

#### Chat Mode (`.ask`)

- Type naturally, AI answers without SQL
- `.sql` or `Esc` returns to query mode
- All dot commands work inside chat mode
- Persistent `inChat` flag survives thinking state

#### Viewport

- Scrollable with PgUp/PgDn/Home/End
- Mouse wheel support (cell motion mouse mode)
- Auto-scroll to bottom on new messages
- Smart scroll — won't jerk when reading history
- Startup logo locked to top during animation
- Text select works (no AltScreen, no mouse capture)

#### Status Bar

```
basemake v0.6.0  │  ● PostgreSQL (***@localhost:5433)  │  OPENCODE/DEEPSEEK-CHAT  │  💬 CHAT  │  💡 INDEX
```

- Persistent tmux-style bar at bottom of terminal
- Green dot when connected, white when disconnected
- Provider/model label (refreshed on connect)
- Mode tags: `💬 CHAT`, `🔒 read-only`, `💡 INDEX`
- Now refreshes correctly after `.connect` (fixed in v0.6.0)

#### Micro-interactions

- **Splash animation** — logo reveals line-by-line on startup (60ms per frame)
- **Shake** — input prompt flashes red briefly when write blocked in read-only mode
- **Sparklines** — visual indicators in query results
- **Dot ghost autocomplete** — fish-style inline suggestion for dot commands
- **Tab completion** — cycle through matching table/column names
- **History navigation** — up/down arrows with preview restore

#### Startup Flow

1. Logo animation plays (line-by-line reveal)
2. Status line shows: version, AI provider, connection state
3. Input placeholder adapts to state:
   - `"No database connected. Try: .connect postgres://user@localhost/mydb"`
   - `"Type .help for commands  ·  ask your question or enter SQL"`
4. Auto-connects to last-used database if DSN is cached
5. Auto-introspects schema for NL queries
6. Stale recommendation alert: `⏰ 3 index recommendations are 7+ days old`

---

### 🔧 DevOps & Tooling

#### Makefile

```
make lint     — go vet + staticcheck
make test     — go test -count=1 ./...
make build    — go build -ldflags="-s -w" -o basemake
make install  — go install
```

#### Pre-commit Hook (`.githooks/pre-commit`)

- `gofmt -d .` — fail on unformatted code
- `go vet ./...` — catch suspicious constructs
- `staticcheck ./...` — lint
- `go test ./...` — run full suite
- Skip with `--no-verify`

#### Docker Compose

`docker-compose.yml` — monitoring stack for production deployments:
- basemake server
- Prometheus (metrics collection)
- Grafana (visualization)

#### Dev Setup

- `.golangci.yml` — linter configuration
- `.goreleaser.yaml` — GoReleaser for cross-platform builds
- CI/CD: GitHub Actions with 5-platform matrix (linux amd64/arm64, darwin amd64/arm64, windows amd64)

---

### 🧪 Test Suite

76+ tests across all packages.

#### AI Package (`internal/ai/ai_test.go`)

- `EstimateCost` — known model, unknown model, free model, small/large token counts
- `EstimateTokens` — empty string, short, long, unicode

#### Recommendations (`internal/analyze/recommendations_test.go`)

- `FormatSuggestions` — empty, single, multiple, speculative, medium, partial confidence
- `buildStatsQuery` — single column, multi-column, all tables, SQL injection rejection
- `CollectTablesFromIssues` — deduplication, empty input

#### Filter Extraction

- All 9 operator types: LIKE, IN, BETWEEN, deep nesting, <>, IS NOT NULL, no-operator, mixed, type casts
- Real-world filters from stressdb dataset

#### RecStore (`internal/analyze/rec_store_test.go`)

- Save/Load round-trip
- Merge: dedup exact/partial, update reason/timestamp, no overlap = append
- Apply/Dismiss status transitions
- StaleReport by days

#### Cardinality Engine

- ColumnStats parsing from pg_stats
- Selectivity calculations per operator
- Confidence levels

Current: `go vet ./...` ✅ | `go test ./...` ✅ | `go build ./...` ✅

---

### 🐛 Bug Fixes

| Bug | Fix |
|-----|-----|
| TUI status bar not refreshing after `.connect` | Removed `animFrame` guard, always re-render `fullStartupView()`, added `refreshViewport()`, update `aiLabel` on connect |
| Dot commands broken in chat mode | Enter handler routes `.` prefix through `handleDotCommand()` before chat processing |
| Merged duplicate `init()` in `cmd/root.go` | Consolidated into single function |
| History lazy init race | Wrapped SQLite init in `sync.Mutex` via `ensureDB()` |
| MySQL EXPLAIN ANALYZE DML execution | Wrapped in `BeginTx` + `defer tx.Rollback()` |
| Server goroutine leak | Added `done chan struct{}` + `Shutdown()` method |
| AltScreen blocking text selection | Removed AltScreen, inlined viewport rendering |
| Mouse capture blocking copy/paste | Switched to cell motion mouse mode, removed mouse capture |
| Logo clipped on startup | Adjusted viewport reserved height, locked to top during animation |
| Keyboard not responding on launch | Key events now flow to text input after model init |
| Staticcheck lint (dead code, unused struct, redundant fmt.Sprintf) | Removed or resolved |
| Nil conn checks in server paths | Added nil guards before conn calls |

---

### 📁 Files Changed (since v0.5.0)

**New:**
- `internal/tui/provider_selector.go` — Interactive provider selector TUI (230 lines)
- `internal/tui/repl.go` — Bubbletea REPL (2167 lines)
- `internal/tui/styles.go` — Lipgloss styles, status line, startup screen (197 lines)
- `internal/ai/ai_test.go` — Cost tracking tests
- `internal/analyze/cardinality.go` — pg_stats engine + selectivity math
- `internal/analyze/cardinality_test.go` — Cardinality engine tests
- `internal/analyze/recommendations.go` — Index suggestion formatting
- `internal/analyze/recommendations_test.go` — 564 lines of tests
- `internal/analyze/suggest_test.go` — Index suggestion display tests
- `cmd/index.go` — `basemake index` CLI
- `cmd/doctor.go` — `basemake doctor` diagnostics
- `cmd/init.go` — `basemake init` wizard
- `cmd/pg_helper.go` — PostgreSQL helper utilities
- `cmd/use.go` — `basemake use` connection switching
- `Makefile`, `.githooks/pre-commit`, `docker-compose.yml`

**Modified:**
- `internal/ai/ai.go` — Provider interface, OpenCode, cost tracking, health checks
- `internal/ai/openai.go` — OpenCode provider wrapping
- `internal/config/config.go` — OpenCode fields
- `cmd/config.go` — OpenCode config, `set-ai-provider` command
- `cmd/analyze.go` — Index suggestion integration
- `cmd/repl.go` — Complete rewrite with Bubbletea TUI
- `internal/analyze/plan.go` — Plan parser improvements
- `internal/db/` — MySQL rollback fix, connection management
- `internal/history/` — Race fix, lazy init
- `internal/server/` — Goroutine leak fix
