# AI Integration

The AI integration allows dbai to translate natural language questions into SQL queries using the OpenAI Chat Completions API.

## Architecture

All AI functionality lives in `internal/ai/ai.go` — a single file with no external dependencies beyond the standard library, OpenAI HTTP API, and the local config package.

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

## Known Limitations

1. **PostgreSQL-only dialect** — even when connected to MySQL or SQLite, the generated SQL targets PostgreSQL. This can produce incompatible SQL for MySQL-specific features or SQLite's limited SQL syntax.
2. **Single-turn generation** — no multi-turn refinement. If the SQL is wrong, the user must rephrase.
3. **No streaming** — the full response is buffered and returned at once. No token-by-token streaming.
4. **Hardcoded model** — no automatic model selection based on query complexity. The config/env var model is used for everything.
5. **No prompt caching** — the schema prompt is sent on every NL query. For large schemas (100+ tables), this can be slow and expensive.
6. **Single provider** — OpenAI only. No Anthropic, local Ollama, or other LLM provider support (though `openai_base_url` config field exists for API-compatible proxies).
7. **`openai_base_url` is defined but unused** — the config field exists in the Config struct but `callOpenAI()` always uses `https://api.openai.com/v1/chat/completions`. This is a dormant feature.
