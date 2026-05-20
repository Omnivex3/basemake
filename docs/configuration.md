# Configuration

`basemake uses a layered configuration system with clear precedence rules.

## Configuration Layers (Highest to Lowest Priority)

1. **CLI flags** — `--json`, `--csv`, `--dry-run`, `--explain`, `--all`, `--format`
2. **Environment variables** — `OPENAI_API_KEY`, `OPENAI_MODEL`, `BASEMAKE_DSN`
3. **Config file** — `~/.basemake/config.json`
4. **Global defaults** — Hardcoded fallbacks

## Config File

Location: `~/.basemake/config.json`

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
| `default_dsn` | string | `""` | Default connection string — used by `basemake query`, `basemake analyze`, and `basemake repl` when no active connection exists |
| `output_format` | string | `"table"` | Default output format. Valid values: `"table"`, `"json"`, `"csv"`. |
| `ai_provider` | string | `"openai"` | AI provider for NL→SQL. Valid values: `"openai"`, `"anthropic"`. |
| `openai_model` | string | `"gpt-4"` | OpenAI model for NL→SQL generation |
| `openai_base_url` | string | `""` | Custom OpenAI API base URL (for proxies, Azure OpenAI, or local LLM servers) |
| `anthropic_model` | string | `"claude-sonnet-4-20250514"` | Anthropic model for NL→SQL generation |
| `anthropic_base_url` | string | `""` | Custom Anthropic API base URL |

### Config File Lifecycle

- **Created**: Automatically on first `config.Load()` — no, actually the config file is NOT auto-created. It's only written when you manually create it. `Load()` returns defaults if the file doesn't exist.
- **Read**: Every command invocation loads the file from disk
- **Written**: Only programmatically via `config.Save()` — there's no `basemake config set` command (you edit the JSON directly)
- **Deleted**: Remove `~/.basemake/config.json` to reset to defaults

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

- `Load()` reads `~/.basemake/config.json`, returns `DefaultConfig()` if file doesn't exist
- `DefaultConfig()` returns: `{OutputFormat: "table", OpenAIModel: "gpt-4"}`
- `Save()` creates `~/.basemake/` with 0755, writes with 0600, uses `json.MarshalIndent`
- Config directory: `$HOME/.basemake/`

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENAI_API_KEY` | For OpenAI NL queries | `""` | OpenAI API key for natural language → SQL generation |
| `OPENAI_MODEL` | No | `"gpt-4"` | Overrides the OpenAI model. Takes precedence over `openai_model` in config file. |
| `ANTHROPIC_API_KEY` | For Anthropic NL queries | `""` | Anthropic API key. Required when `ai_provider` is `"anthropic"`. |
| `ANTHROPIC_MODEL` | No | `"claude-sonnet-4-20250514"` | Overrides the Anthropic model. Takes precedence over `anthropic_model` in config. |
| `AI_PROVIDER` | No | `"openai"` | AI provider: `"openai"` or `"anthropic"`. Takes precedence over `ai_provider` config. |
| `BASEMAKE_DSN` | No | `""` | Default connection string fallback. |

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

Before the config file existed, basemake used `~/.basemake/dsn.txt` to persist the last-used DSN. This is still supported as a fallback:

- Location: `~/.basemake/dsn.txt`
- Format: Raw DSN string (one line)
- Written by: `db connect` command
- Read by: `db.LoadDSN()` — used when `ActiveConnection()` returns `ErrNoConnection`
- Priority: Below `BASEMAKE_DSN` env var and `default_dsn` config field

## Schema Cache

- Location: `~/.basemake/schema.json`
- Written by: `basemake connect` (on successful introspection)
- Read by: `basemake query` (for NL→SQL), `basemake analyze --all`
- Format: Full JSON schema dump (see Schema types in db.go)
- Cleared by: Running `basemake connect` to a different database
- Not auto-expired: Schema is assumed stable between connections

## Known Limitations

- No `basemake config set` command — you edit the JSON directly
3. **No streaming config** — streaming is always on by default and can only be disabled per-command with `--no-stream`. No config-level toggle.
4. **No per-profile configs** — switching between databases requires re-editing the file
5. **Output format config doesn't support TSV** (only code-level constant)
