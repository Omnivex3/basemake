# basemake — Complete Feature Reference

> **The terminal-native database tool. Query, analyze, optimize — in English or SQL.**

---

## 1. Core Query Engine

### Natural Language → SQL
Describe what you need in plain English. basemake generates production-ready SQL with the correct dialect, joins, filtering, and aggregation.

```bash
basemake "show me users who signed up last week grouped by plan type"
```

### Direct SQL Execution
Type or pipe raw SQL — works identically to psql, mysql, sqlite3.

```bash
echo "SELECT * FROM orders WHERE total > 100" | basemake
basemake "SELECT date_trunc('month', created_at) AS month, COUNT(*) FROM users GROUP BY 1"
```

### Pipe Mode
Pipe queries in, get formatted output out. Perfect for scripts and pipelines.

```bash
cat query.sql | basemake --explain
cat query.sql | basemake -f json
```

### Query History
Every query is recorded locally. Browse, search, and replay with `.history` and `.replay <N>`.

---

## 2. Database Support

### PostgreSQL
- `postgres://` / `postgresql://` DSN
- Full schema introspection: tables, columns, types, PKs, indexes, nullability, defaults
- `information_schema` + `pg_index` / `pg_class` / `pg_attribute` queries
- `EXPLAIN ANALYZE` wrapped in transaction + rollback (safe — never executes DML)
- `EXPLAIN (FORMAT JSON)` for plan analysis
- Connection masking: `***@host:port/dbname`

### MySQL
- `mysql://` DSN
- Schema introspection via `information_schema`
- `EXPLAIN ANALYZE` wrapped in `BeginTx` + `defer tx.Rollback()`
- All standard query operations

### SQLite
- `sqlite:/path` / `sqlite:///path` DSN
- Pure Go driver (`modernc.org/sqlite`) — no CGo, no shared library
- WAL mode for concurrent read performance
- Cross-compilation friendly

### Multi-Dialect Interface
All drivers implement a shared `Database` interface:

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

Driver auto-detection from DSN prefix. Same tool, same interface, any database.

### Schema Caching
Introspected schemas are cached to `~/.basemake/schema.json` so the AI has context without re-scanning on every query. Cache refreshes on `.refresh` or on reconnect.

---

## 3. AI Integration

### Multi-Provider
Bring your own key. Supported providers:

| Provider | Env Variable | Default Model | Config Field |
|----------|-------------|---------------|--------------|
| OpenAI | `OPENAI_API_KEY` | `gpt-4` | `openai_model` |
| Anthropic | `ANTHROPIC_API_KEY` | `claude-sonnet-4-20250514` | `anthropic_model` |
| Ollama | — (local) | — | — |

### Provider Selection
Precedence: `AI_PROVIDER` env → config `ai_provider` → default `"openai"`

### Schema-Aware Generation
AI receives the full cached schema (tables, columns, types, relationships) as context, producing accurate, join-correct SQL.

### BYOK (Bring Your Own Key)
No AI token markup. You use your own API keys or a local Ollama instance. basemake never charges for AI inference.

---

## 4. Interactive TUI / REPL

### Full Terminal UI
Built with Charm Bubbletea. Features:
- **Message viewport** with scroll (PgUp/PgDn, Home/End)
- **Inline ghost autocomplete** for dot commands (fish-style)
- **Autocomplete dropdown** — filtered command list below input with `▸` highlight
- **Spinner + thinking indicator** during AI generation
- **Real-time SQL preview** — see the SQL being generated token by token
- **Status bar** — connection status, AI provider, read-only indicator, chat mode, index alerts
- **Color-coded cursor** — green = connected, white = disconnected, red = thinking
- **Smart auto-scroll** — follows new messages when at bottom, preserves position when reading history

### Dot Commands

| Command | Description |
|---------|-------------|
| `.help` | Show help |
| `.quit` / `.exit` | Exit basemake |
| `.connect <dsn>` | Connect to a database |
| `.disconnect` | Disconnect from current database |
| `.ask` | Free-form chat mode (no SQL generation) |
| `.sql` | Return to query mode (from `.ask`) |
| `.tables` | List tables in current database |
| `.schema` | Show full database schema |
| `.explain` | Explain last query result in plain English |
| `.analyze` | Run EXPLAIN ANALYZE + index suggestions on last query |
| `.info` | Show connection and AI status |
| `.history` | Show past questions |
| `.replay <N>` | Re-run query from history |
| `.refresh` | Re-introspect and cache schema |
| `.readonly` | Toggle write protection on/off |
| `.save <name>` | Save last query as a bookmark |
| `.run <name>` | Run a saved query |
| `.saved` | List all saved queries |
| `.export <.csv\|.json\|.md>` | Export last result to file |

### Chat Mode (`.ask`)
Free-form conversation with the AI about your database. No SQL generated — just natural language analysis, explanations, and insights.

### Did-You-Mean
Levenshtein-based fuzzy matching on unknown dot commands — if you type `.hel` it suggests `.help`.

### Read-Only Mode
Toggle write protection with `.readonly`. Flash red prompt when write commands are blocked. Server-side RBAC enforcement on Team tier.

---

## 5. Performance Analysis

### EXPLAIN + EXPLAIN ANALYZE
Safe execution for all drivers. PostgreSQL wraps in a transaction with rollback. MySQL uses `BeginTx` + rollback. Never executes DML during analysis.

### Index Recommendations
Not just "add an index." basemake analyzes query patterns and suggests:
- **Which columns** to index (inclusion vs key columns)
- **What order** for composite indexes (selectivity-based)
- **Why** — estimated improvement based on `pg_stats` cardinality analysis
- **Two-step apply** — dry-run → review → `CREATE INDEX CONCURRENTLY`

### Budget Engine (Policy as Code)
Define performance policies in `.basemake/budgets.json`:

```json
{
  "version": 1,
  "rules": [
    {
      "type": "table",
      "table": "orders",
      "max_seq_rows": 1000
    }
  ]
}
```

Rule types: `table` (per-table constraints), `migration` (schema change safety), `query` (pattern-based). Enforced by `basemake check`.

### Cardinality Analysis
Reads `pg_stats` / equivalent to estimate selectivity of WHERE clauses, JOIN conditions, and composite index ordering.

### Plan JSON Analysis
Parses `EXPLAIN (FORMAT JSON)` output. Detects:
- Sequential scans on large tables
- Missing or unused indexes
- Expensive joins (nested loop vs hash vs merge)
- Sort operations on unindexed columns

---

## 6. Schema Management

### Diff (`basemake diff`)
Compare two databases in seconds:
- Dev vs staging, staging vs prod
- Detects: missing tables, dropped columns, type changes, missing indexes
- Schema drift detection without a server

### Index Management (`basemake index`)
```
basemake index                  List pending recommendations
basemake index apply <id>       Create an index (dry-run first)
basemake index dismiss <id>     Dismiss a recommendation
basemake index apply --force    Skip dry-run
```

### Doctor (`basemake doctor`)
Full system health check with actionable fixes:
- Connection status
- Schema cache health
- AI provider availability
- Config file integrity
- License status

### Schema Introspection
Full metadata scan on connect: tables, columns, data types, primary keys, foreign keys, nullability, defaults, indexes, and estimated row counts.

---

## 7. CI/CD Integration

### `basemake check`
CI/CD gate that exits with a machine-readable code:

| Exit Code | Meaning |
|-----------|---------|
| 0 | Pass — query is safe |
| 1 | Slow — exceeds threshold or cost |
| 2 | Dangerous — could cause outage |

```bash
basemake check "SELECT * FROM orders" --threshold 500ms
basemake check --file query.sql --budget .basemake/budgets.json
```

### CI Usage
```yaml
# GitHub Actions
- name: Check query performance
  run: basemake check --file migrations/001.sql --threshold 100ms
  env:
    BASEMAKE_LICENSE_KEY: ${{ secrets.BASEMAKE_LICENSE_KEY }}
```

Features:
- Exit-code based (standard CI practice)
- Enforce budget policies automatically
- Check threshold: max execution time, max rows scanned
- Works in GitHub Actions, GitLab CI, Jenkins, any pipeline

---

## 8. Monitoring & Observability

### Watch (`basemake watch`)
Schedule recurring queries and alert on regressions:

```bash
basemake watch --query "SELECT COUNT(*) FROM orders" --interval 5m --threshold 2000
```

- Detect slow-downs before users do
- Track query performance over time
- Webhook notifications on budget violations

### Webhook Subscriptions
Server mode supports event-driven webhooks:
- Budget violation alerts → Slack, Teams
- Schema drift detection → notification channel
- CI pipeline results

### Status Dashboard
`.info` command in the TUI shows:
- Connection status & database name
- AI provider & model
- License tier
- Read-only status
- Version

---

## 9. Team Features (Team tier)

### Server (`basemake server`)
```bash
basemake server --license bmk_team_xxxx --port 8080
```
- Shared query history across the team
- Shared budget policies (one PR updates everyone's gates)
- See who ran what, when, and how slow

### Shared AI Proxy
- One corporate API key for the whole team
- Automatic response caching — save 40–60% on AI costs
- Central AI usage audit log

### RBAC (Server-Side)
- Lock production queries to read-only at the server level
- Role-based access control

### Audit Log
Every query, every check, every apply — timestamped and attributed.

### Slack / Teams Integration
Webhook alerts on budget violations, slow queries, and schema drift.

---

## 10. Output Formats

| Format | Flag | Use Case |
|--------|------|----------|
| Table (default) | — | Human-readable terminal output |
| JSON | `-f json` | API consumption, `jq` pipelines |
| CSV | `-f csv` | Spreadsheets, data export |
| TSV | `-f tsv` | Unix tooling (`cut`, `awk`) |

### `.export` Command
Export last query result to a file:
```bash
.export results.csv
.export report.json
.export analysis.md
```

---

## 11. Security

### Privacy-First Architecture
- **All data stays on your machine** — no telemetry, no cloud sync
- AI queries go directly from CLI → your configured provider (OpenAI / Anthropic / Ollama)
- No basemake cloud intermediary

### Connection Masking
DSNs are masked in logs and UI — `***@host:port/dbname`

### Secrets Handling
- API keys: environment variables only (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`)
- Database passwords: embedded in DSN (standard practice)
- Config stores: default DSN, output format, AI model — never API keys

### Auto-Reconnect
If no active connection, basemake falls back to:
1. `BASEMAKE_DSN` environment variable
2. `default_dsn` from `~/.basemake/config.json`
3. Legacy DSN cache (`~/.basemake/dsn.txt`)

### Read-Only Mode
Toggle per-session with `.readonly`. Server-side enforcement on Team tier.

---

## 12. License & Tier System

### Offline Verification
HMAC-SHA256 signed license keys with compiled-in secret. No phone-home, no internet required.

Format: `bmk_<tier>_<base64(email)>_<hmac-signature>`

### Feature Tiers

| Feature | Free | Pro ($15/mo) | Team ($39/seat/mo) |
|---------|:----:|:------------:|:------------------:|
| Full TUI / REPL | ✅ | ✅ | ✅ |
| NL→SQL (BYOK) | ✅ | ✅ | ✅ |
| Query execution | ✅ | ✅ | ✅ |
| EXPLAIN / analyze | ✅ | ✅ | ✅ |
| Index recommendations (list) | ✅ | ✅ | ✅ |
| Index apply / dismiss | ❌ | ✅ | ✅ |
| `basemake check` (CI/CD gate) | ❌ | ✅ | ✅ |
| `basemake budget` (policy engine) | ❌ | ✅ | ✅ |
| `basemake watch` (monitoring) | ❌ | ✅ | ✅ |
| `basemake diff` (schema diff) | ❌ | ✅ | ✅ |
| `basemake doctor` (diagnostics) | ❌ | ✅ | ✅ |
| Team server + sync | ❌ | ❌ | ✅ |
| Shared AI proxy + caching | ❌ | ❌ | ✅ |
| RBAC server-side | ❌ | ❌ | ✅ |
| Audit log | ❌ | ❌ | ✅ |
| Slack/Teams integration | ❌ | ❌ | ✅ |

Activation:
```bash
basemake config set license_key bmk_pro_xxxx
# or set as env var for CI:
BASEMAKE_LICENSE_KEY=bmk_pro_xxxx basemake check --query "SELECT * FROM orders"
```

---

## 13. Platform & Distribution

### Single Static Binary
One download, zero runtime dependencies. No Python, no Node, no JVM.

| Platform | Support |
|----------|---------|
| Linux (amd64) | ✅ |
| Linux (arm64) | ✅ |
| macOS (amd64) | ✅ |
| macOS (arm64) | ✅ |
| Windows (amd64) | ✅ |

### Docker
Multi-arch Docker image on GHCR:
```bash
docker pull ghcr.io/dynamickarabo/basemake:latest
```

### Installation
- Direct download from basemake.dev
- Homebrew (planned)
- Docker

### Stress-Tested
30+ benchmark scenarios covering:
- Large result sets (10K+ rows)
- Parallel queries (50 concurrent)
- Memory usage tracking
- Schema analysis (50+ tables)
- Empty result sets, NULL queries, edge cases
- Server event throughput

---

## 14. Website

Deployed at [basemake.dev](https://website-eight-plum-77.vercel.app):
- Landing page with feature overview
- Interactive demo
- Pricing page with plan comparison
- Documentation hub (commands, config, drivers, AI integration)
- Download links

Built with Vite + React + Tailwind CSS. Dark theme. Deployed on Vercel.

---

## Roadmap (Planned)

- [ ] Homebrew formula for macOS
- [ ] SSO/SAML (Okta, Azure AD, Google Workspace)
- [ ] On-prem enterprise deployment
- [ ] AI proxy response caching dashboard
- [ ] Webhook-based Slack/Teams integration
- [ ] More AI providers (Gemini, Groq, DeepSeek)
- [ ] Windows native installer (MSI)
