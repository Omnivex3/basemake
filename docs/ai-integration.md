# AI Integration

The AI integration allows dbai to translate natural language questions into SQL queries using configurable LLM providers.

## Supported Providers

| Provider  | API Key Env Var       | Default Model                       | Config Field       |
|-----------|----------------------|--------------------------------------|--------------------|
| OpenAI    | `OPENAI_API_KEY`     | `gpt-4`                              | `openai_model`     |
| Anthropic | `ANTHROPIC_API_KEY`  | `claude-sonnet-4-20250514`           | `anthropic_model`  |

## Provider Selection

Provider selection follows this precedence:

1. `AI_PROVIDER` environment variable (`"openai"` or `"anthropic"`)
2. `ai_provider` field in `~/.dbai/config.json`
3. Default: `"openai"`

Selected via `ai.SelectedProvider()`:

```go
func SelectedProvider() (Provider, error) {
    // Check AI_PROVIDER env → config.ai_provider → "openai"
    // For openai: check OPENAI_API_KEY → OPENAI_MODEL/env → config.openai_model → "gpt-4"
    // For anthropic: check ANTHROPIC_API_KEY → ANTHROPIC_MODEL/env → config.anthropic_model → "claude-sonnet-4-20250514"
}
```

If the required API key for the selected provider is not set, `ErrNoKey` is returned, and the command falls back to a placeholder SQL (`SELECT 1`).

## Architecture

All AI functionality lives in the `internal/ai` package with three files:

```
User Question
    │
    ▼
QuestionToSQL(schemaPrompt, question)
    │
    ├── OPENAI_API_KEY not set?
    │   └── Return placeholder SQL: "SELECT 1;"
    │
    ├── Build system prompt with schema context
    │   └── "You are a SQL expert..." + schema definition
    │
    ├── Determine model
    │   └── OPENAI_MODEL env → config.openai_model → "gpt-4"
    │
    ├── callOpenAI(ctx, apiKey, systemPrompt, question)
    │   │
    │   └── POST https://api.openai.com/v1/chat/completions
    │       ├── Model: resolved model name
    │       ├── Messages: [{role: "system", content: ...}, {role: "user", content: ...}]
    │       ├── Temperature: 0.1
    │       └── Headers: Authorization: Bearer <key>, Content-Type: application/json
    │
    │   ← Response: {choices: [{message: {content: "SELECT ..."}}]}
    │
    ├── Clean response
    │   └── Trim spaces, strip ```sql and ``` markers
    │
    └── Return cleaned SQL string
```

## API Details

### Request

```go
type openAIRequest struct {
    Model    string          `json:"model"`
    Messages []openAIMessage `json:"messages"`
    Temp     float64         `json:"temperature"`
}

type openAIMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}
```

### Response

```go
type openAIResponse struct {
    Choices []struct {
        Message struct {
            Content string `json:"content"`
        } `json:"message"`
    } `json:"choices"`
    Error *struct {
        Message string `json:"message"`
    } `json:"error,omitempty"`
}
```

### Endpoint

```
POST https://api.openai.com/v1/chat/completions
```

### Error Handling

1. HTTP errors → wrapped as `"http call: <error>"`
2. Non-200 responses with `error` field → `"openai error: <message>"`
3. Empty choices array → `"no choices in response"`
4. JSON parse failures → `"parse response: <error>"`
5. Request creation failures → `"create request: <error>"`
6. All wrapped at the top level → `"ai call: <error>"`

## System Prompt

The system prompt template passed to the AI:

```
You are a SQL expert. Given the following database schema, convert the user's
natural language question into a SQL query.

Rules:
- Generate PostgreSQL-compatible SQL
- Return ONLY the SQL query — no markdown, no backticks, no explanations
- Use proper formatting with newlines
- If the question is ambiguous, make a reasonable assumption and add a comment explaining it

Schema:
<schema_from_SchemaForPrompt()>
```

Key behaviors baked into the prompt:
- **PostgreSQL dialect** only — queries generated for PG syntax even if connected to MySQL/SQLite. This is a known limitation.
- **No markdown** — the model is instructed not to wrap SQL in markdown code blocks, but the cleaner strips them anyway as a safety net
- **Reasonable assumptions** — ambiguous questions get a comment explaining the assumption
- **Temperature 0.1** — low randomness for deterministic SQL generation

## Schema Prompt Format

The `SchemaForPrompt()` method on the `Schema` struct produces this format:

```
Database: mydb

Tables:
  users:
    - id integer [PK]
    - name text nullable
    - email text nullable
    Indexes:
      - users_pkey on (id) (unique)
      - idx_users_email on (email)
  orders:
    - id integer [PK]
    - user_id integer nullable
    - total real
```

This compact representation fits within GPT-4's context window while providing full schema context including:
- Table names and column names
- Data types (database-specific)
- Primary key markers
- Nullability
- Indexed columns with uniqueness info

## Model Selection

```go
func callOpenAI(ctx context.Context, apiKey, system, user string) (string, error) {
    model := os.Getenv("OPENAI_MODEL")
    if model == "" {
        cfg, err := config.Load()
        if err == nil && cfg.OpenAIModel != "" {
            model = cfg.OpenAIModel
        }
    }
    if model == "" {
        model = "gpt-4"
    }
    // ...
}
```

Resolution order:
1. `OPENAI_MODEL` environment variable
2. `openai_model` in `~/.dbai/config.json`
3. Hardcoded default: `"gpt-4"`

## Response Cleaning

The AI response goes through a cleanup process:

```go
sql := strings.TrimSpace(resp)
sql = strings.TrimPrefix(sql, "```sql")
sql = strings.TrimPrefix(sql, "```")
sql = strings.TrimSuffix(sql, "```")
sql = strings.TrimSpace(sql)
```

This handles:
- Leading/trailing whitespace
- Markdown code fence ` ```sql ` prefix
- Markdown code fence ` ``` ` prefix (if language omitted)
- Trailing ` ``` ` fence

Even though the system prompt tells the AI not to use markdown, this cleanup ensures robustness against non-compliant responses.

## No-Key Behavior

When `OPENAI_API_KEY` is not set, `QuestionToSQL()` doesn't return an error — it returns a benign placeholder:

```go
func QuestionToSQL(ctx context.Context, schemaPrompt, question string) (string, error) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        return "-- Set OPENAI_API_KEY for AI-powered queries\n" +
               "-- Schema loaded. Run: export OPENAI_API_KEY=\"sk-...\"\n" +
               "SELECT 1;", nil
    }
    // ...
}
```

This lets `dbai query` work without an API key (it executes `SELECT 1` and displays the result). The instructional comments in the SQL are visible in dry-run mode.

## Query Validation

After AI generates SQL, it's validated via `EXPLAIN` before execution:

```go
if isNL && !queryExplain {
    if err := validateSQL(cmd.Context(), conn, sql); err != nil {
        return fmt.Errorf("generated SQL is invalid — try rephrasing your question:\n  %s\n  %v", sql, err)
    }
}
```

This provides a clear feedback loop: if the AI hallucinated bad SQL, the user sees the error and can rephrase their question. The validation only runs for NL-generated queries (not raw SQL input) and is skipped in `--explain` mode (since EXPLAIN would be called twice).

## Streaming Responses

By default, NL→SQL generation streams tokens to stderr as they arrive from the LLM. This makes the tool feel instant — the user sees SQL appearing token by token instead of waiting for a full response.

Streaming is enabled by default for both `dbai query` and `dbai repl`. Disable with `--no-stream`.

### Streaming Implementation

Both providers implement `GenerateSQLStream()` which returns a `<-chan string`:

**OpenAI streaming:**
```
data: {"id":"...","object":"chat.completion.chunk","choices":[{"delta":{"content":"SELECT"}}]}
data: {"id":"...","choices":[{"delta":{"content":" *"}}]}
...
data: [DONE]
```

**Anthropic streaming:**
```
event: content_block_delta
data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"SELECT"}}
...
event: message_stop
data: {"type":"message_stop"}
```

In both cases:
- SSE (Server-Sent Events) are parsed line-by-line via `bufio.Scanner`
- Each text delta is sent to the channel immediately
- The channel is closed when generation completes
- The command layer collects full SQL from the channel while printing tokens

### No-Key Fallback

When no API key is set and streaming is requested, a one-element channel is returned with the placeholder SQL, then closed. No streaming occurs — the placeholder appears instantly.

## Context Compounding via Query History

When generating SQL, dbai prepends recent natural language queries and their generated SQL to the system prompt. This creates a compounding context effect — the AI learns from past Q&A patterns.

### How It Works

The `history.BuildPromptWithHistory()` function constructs the system prompt:

```
You are a SQL expert. ...

Recent queries you've helped with:
- Question: show me users who ordered
  SQL: SELECT u.name, COUNT(o.id) ...
- Question: top products by revenue
  SQL: SELECT p.name, SUM(o.total) ...

Schema:
  Database: mydb
  Tables:
    ...
```

### Configuration

- History depth: 5 recent NL→SQL pairs (hardcoded in both `cmd/query.go` and `cmd/repl.go`)
- History stored in `~/.dbai/history.db` (SQLite)
- Only NATURAL LANGUAGE queries are included in the context (raw SQL inputs are excluded)
- Set `AI_PROVIDER` env var or `ai_provider` config to switch between OpenAI and Anthropic

## Known Limitations

1. **PostgreSQL-only dialect** — even when connected to MySQL or SQLite, the generated SQL targets PostgreSQL. This can produce incompatible SQL for MySQL-specific features or SQLite's limited SQL syntax.
2. **Single-turn generation** — no multi-turn refinement. If the SQL is wrong, the user must rephrase.
3. **No prompt caching** — the schema prompt + history context is sent on every NL query. For large schemas (100+ tables), this can be slow and expensive.
4. **History depth is hardcoded** — the 5 most recent NL queries are included in context. No config option to adjust this yet.
5. **`openai_base_url` is unused by streaming** — both `GenerateSQL` and `GenerateSQLStream` use the base URL from the provider struct, which IS set from the config field. This now works correctly.
