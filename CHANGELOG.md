# dbai v0.2.0 — Multi-Provider, Streaming, History

This release fundamentally upgrades dbai's AI layer from a single-vendor blocking call to a pluggable, streaming, context-aware engine.

## Multi-Provider AI

dbai now supports **OpenAI** and **Anthropic** — bring your own key, pick your provider.

### Usage

```bash
# OpenAI (default)
export OPENAI_API_KEY=sk-...
dbai query "show me top users"

# Anthropic
export AI_PROVIDER=anthropic
export ANTHROPIC_API_KEY=sk-ant-...
dbai query "show me top users"
```

### Provider Selection Precedence

1. `AI_PROVIDER` environment variable
2. `ai_provider` field in `~/.dbai/config.json`
3. Default: `"openai"`

### Configuration

| Config Field | Default | Description |
|-------------|---------|-------------|
| `ai_provider` | `"openai"` | Provider: `"openai"` or `"anthropic"` |
| `openai_model` | `"gpt-4"` | OpenAI model to use |
| `anthropic_model` | `"claude-sonnet-4-20250514"` | Anthropic model to use |
| `anthropic_base_url` | `""` | Custom Anthropic API endpoint |

| Env Var | Required For |
|---------|-------------|
| `OPENAI_API_KEY` | OpenAI provider |
| `ANTHROPIC_API_KEY` | Anthropic provider |
| `OPENAI_MODEL` | Override OpenAI model |
| `ANTHROPIC_MODEL` | Override Anthropic model |
| `AI_PROVIDER` | Override provider selection |

### Provider Architecture

Common `Provider` interface with two methods:

```go
type Provider interface {
    Name() string
    GenerateSQL(ctx, system, question) (string, error)
    GenerateSQLStream(ctx, system, question) (<-chan string, error)
}
```

- **OpenAI**: `POST /v1/chat/completions` — `gpt-4` default
- **Anthropic**: `POST /v1/messages` — `claude-sonnet-4-20250514` default

Both support custom base URLs for proxies, Azure OpenAI, or local endpoints.

---

## Streaming Responses

NL→SQL generation now streams **token by token** to stderr as it arrives from the LLM. Instead of waiting seconds for a full response, you see SQL appear as it's being generated.

### Default Behavior

Streaming is **on by default** for both `dbai query` and `dbai repl`. The generated SQL appears in real-time, then the full text is validated and executed.

```
$ dbai query "show me users who ordered last month"
🤖 Generating SQL...

SELECT   ← appears instantly, then more tokens follow
  u.name,
  COUNT(o.id) as orders
FROM users u
JOIN orders o ON u.id = o.user_id
WHERE o.created_at > now() - interval '30 days'
GROUP BY u.id
ORDER BY orders DESC;

 id | name
----+-------
  1 | Alice
(1 row)
```

### Disable Streaming

```bash
dbai query "complex analytics" --no-stream
```

### Streaming Implementation

| Provider | SSE Event Format |
|----------|-----------------|
| OpenAI | `data: {"choices":[{"delta":{"content":"..."}}]}` → `data: [DONE]` |
| Anthropic | `event: content_block_delta` → `data: {"delta":{"text":"..."}}` → `event: message_stop` |

Both parsed via `bufio.Scanner` — text deltas are sent to a `<-chan string` and printed immediately.

---

## Query History (Local SQLite)

Every query execution is recorded in a local SQLite database with timing, row counts, and AI provider info. This powers **context compounding** — the AI learns from your past queries.

### Storage

- **Location**: `~/.dbai/history.db`
- **Engine**: SQLite (via `modernc.org/sqlite`, pure Go, no CGo)
- **WAL mode**: Enabled for concurrent reads

### Schema

```sql
CREATE TABLE query_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question TEXT NOT NULL,
    sql_generated TEXT NOT NULL,
    database_name TEXT DEFAULT '',
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms REAL DEFAULT 0,
    row_count INTEGER DEFAULT 0,
    was_natural_language INTEGER DEFAULT 0,
    provider_used TEXT DEFAULT '',
    model_used TEXT DEFAULT ''
);
```

### Context Compounding

When generating SQL from natural language, dbai prepends the **5 most recent NL query pairs** to the AI's system prompt. This means:

1. First query: "show me users" → `SELECT * FROM users`
2. Second query: "only their names" → AI sees the first Q&A pair, understands you're still in the same context, generates `SELECT name FROM users`

The effect compounds across a session — the AI builds understanding of your schema and query patterns over time.

### REPL History Command

```bash
dbai repl
dbai> .history
14:32:05 NL  [Anthropic]  show me users who ordered last month
14:28:12 SQL              SELECT * FROM orders LIMIT 5
```

### What's Recorded Per Query

- Original question or SQL input
- Generated/executed SQL
- Database name (masked)
- Execution time in milliseconds
- Row count returned
- Whether it was natural language or raw SQL
- AI provider used (if applicable)

---

## Summary

| Feature | Before | After |
|---------|--------|-------|
| AI Provider | OpenAI only | OpenAI + Anthropic (pluggable) |
| Response style | Blocking (wait for full response) | Streaming (token-by-token) |
| Query history | None | Local SQLite with 5-depth context |
| Vendor lock-in | Full (OpenAI API key) | None (bring your key, pick provider) |
| REPL commands | 4 (.help, .quit, .tables, .schema, .connect) | 6 (+ .history) |
| Config file fields | 4 | 7 |
| Environment variables | 2 | 6 |
