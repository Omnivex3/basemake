# Configuration

dbai uses a layered configuration system with clear precedence rules.

## Configuration Layers (Highest to Lowest Priority)

1. **CLI flags** — `--json`, `--csv`, `--dry-run`, `--explain`, `--all`, `--format`
2. **Environment variables** — `OPENAI_API_KEY`, `OPENAI_MODEL`, `DBAI_DSN`
3. **Config file** — `~/.dbai/config.json`
4. **Global defaults** — Hardcoded fallbacks

## Config File

Location: `~/.dbai/config.json`

### Format

```json
{
  "default_dsn": "postgres://user:pass@localhost:5432/mydb",
  "output_format": "table",
  "openai_model": "gpt-4",
  "openai_base_url": ""
}
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `default_dsn` | string | `""` | Default connection string — used by `dbai query`, `dbai analyze`, and `dbai repl` when no active connection exists |
| `output_format` | string | `"table"` | Default output format. Valid values: `"table"`, `"json"`, `"csv"`. Note: TSV is code-level only (FormatTSV constant), not exposed via config. |
| `openai_model` | string | `"gpt-4"` | OpenAI model for NL→SQL generation |
| `openai_base_url` | string | `""` | Custom OpenAI API base URL (for proxies, Azure OpenAI, or local LLM servers) |

### Config File Lifecycle

- **Created**: Automatically on first `config.Load()` — no, actually the config file is NOT auto-created. It's only written when you manually create it. `Load()` returns defaults if the file doesn't exist.
- **Read**: Every command invocation loads the file from disk
- **Written**: Only programmatically via `config.Save()` — there's no `dbai config set` command (you edit the JSON directly)
- **Deleted**: Remove `~/.dbai/config.json` to reset to defaults

### Code Details

The `config` package lives at `internal/config/config.go`:

```go
type Config struct {
    DefaultDSN    string `json:"default_dsn,omitempty"`
    OutputFormat  string `json:"output_format,omitempty"`
    OpenAIModel   string `json:"openai_model,omitempty"`
    OpenAIBaseURL string `json:"openai_base_url,omitempty"`
}
```

- `Load()` reads `~/.dbai/config.json`, returns `DefaultConfig()` if file doesn't exist
- `DefaultConfig()` returns: `{OutputFormat: "table", OpenAIModel: "gpt-4"}`
- `Save()` creates `~/.dbai/` with 0755, writes with 0644, uses `json.MarshalIndent`
- Config directory: `$HOME/.dbai/`

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENAI_API_KEY` | For NL queries | `""` | OpenAI API key for natural language → SQL generation. Without this, NL questions return a placeholder SQL telling you to set it. |
| `OPENAI_MODEL` | No | `"gpt-4"` | Overrides the AI model. Takes precedence over `openai_model` in config file. |
| `DBAI_DSN` | No | `""` | Default connection string fallback. Used when there's no active connection and no `default_dsn` in config. |

### OPENAI_API_KEY — No Key Behavior

When `OPENAI_API_KEY` is not set and a natural language question is asked:

```
🤖 Generating SQL from: show me users
-- Set OPENAI_API_KEY for AI-powered queries
-- Schema loaded. Run: export OPENAI_API_KEY="sk-..."
SELECT 1;
```

The generated SQL is `SELECT 1` and it executes (returning 1 row). The error message is embedded as SQL comments.

### OPENAI_API_KEY — Source Precedence

```
os.Getenv("OPENAI_API_KEY")  ← only source, no file fallback
```

The API key is NEVER stored in the config file. Environment variable only.

## AI Model Selection

Model resolution order (for NL→SQL generation):

1. `os.Getenv("OPENAI_MODEL")` → if non-empty, use it
2. `config.Load().OpenAIModel` → if non-empty, use it
3. Fallback → `"gpt-4"`

## Output Format Selection

Applied in `cmd/query.go` and `cmd/repl.go`:

```
1. --json flag        → FormatJSON
2. --csv flag         → FormatCSV
3. config.OutputFormat == "json"  → FormatJSON
4. config.OutputFormat == "csv"   → FormatCSV
5. default            → FormatTable
```

## DSN Persistence (Legacy)

Before the config file existed, dbai used `~/.dbai/dsn.txt` to persist the last-used DSN. This is still supported as a fallback:

- Location: `~/.dbai/dsn.txt`
- Format: Raw DSN string (one line)
- Written by: `db connect` command
- Read by: `db.LoadDSN()` — used when `ActiveConnection()` returns `ErrNoConnection`
- Priority: Below `DBAI_DSN` env var and `default_dsn` config field

## Schema Cache

- Location: `~/.dbai/schema.json`
- Written by: `dbai connect` (on successful introspection)
- Read by: `dbai query` (for NL→SQL), `dbai analyze --all`
- Format: Full JSON schema dump (see Schema types in db.go)
- Cleared by: Running `dbai connect` to a different database
- Not auto-expired: Schema is assumed stable between connections

## Known Limitations

- No `dbai config set` command — you edit the JSON directly
- No config validation on load (malformed JSON returns defaults and no error)
- No per-profile configs (switching between databases requires re-editing the file)
- Output format config doesn't support TSV (only code-level constant)
