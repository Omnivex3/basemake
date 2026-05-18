# dbai — AI-powered database CLI

Query, analyze, and optimize your databases with natural language.

```
$ dbai connect postgres://user:pass@localhost:5432/mydb
✓ Connected to PostgreSQL
  Schema loaded: 23 tables, 147 columns, 12 indexes

$ dbai query "show me users who ordered last month"
  SELECT u.name, COUNT(o.id) as orders
  FROM users u JOIN orders o ON u.id = o.user_id
  WHERE o.created_at > now() - interval '30 days'
  GROUP BY u.id ORDER BY orders DESC;

→ 2 columns
name    orders
Alice   12
Bob     7

✓ 2 rows
```

## Install

```bash
# Via Go
go install github.com/DynamicKarabo/dbai@latest

# Or download binary
curl -sfL https://github.com/DynamicKarabo/dbai/releases/latest/download/dbai-linux-amd64.tar.gz | tar xz
sudo mv dbai /usr/local/bin/
```

## Usage

```bash
# Connect and cache schema
dbai connect postgres://user:pass@localhost:5432/mydb

# Ask questions in plain English
export OPENAI_API_KEY="sk-..."
dbai query "top 10 products by revenue this month"

# Run raw SQL
dbai query "SELECT * FROM users LIMIT 5"

# Analyze query performance
dbai analyze "SELECT * FROM orders WHERE created_at > now() - interval '30 days'"

# Output as JSON
dbai query "SELECT count(*) FROM users" --json
```

## Supported Databases

- **PostgreSQL** — full support (introspect, query, explain)
- **MySQL** — full support (introspect, query, explain)
- MariaDB — via MySQL driver
- More coming...

## Configuration

| Env var | Purpose |
|---------|---------|
| `OPENAI_API_KEY` | AI query generation |
| `DBAI_DSN` | Default connection string |

## How It Works

1. **`dbai connect`** introspects your schema and caches it locally (~/.dbai/schema.json)
2. **`dbai query`** sends schema context + your question to an LLM, gets back SQL, executes it
3. **`dbai analyze`** runs EXPLAIN ANALYZE and surfaces performance insights

## License

MIT
