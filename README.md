# dbai — AI-powered database CLI

[![Release](https://img.shields.io/github/v/release/DynamicKarabo/dbai?style=flat&label=release)](https://github.com/DynamicKarabo/dbai/releases)
[![CI](https://github.com/DynamicKarabo/dbai/actions/workflows/release.yml/badge.svg)](https://github.com/DynamicKarabo/dbai/actions/workflows/release.yml)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://go.dev)
[![GitHub Downloads](https://img.shields.io/github/downloads/DynamicKarabo/dbai/total?style=flat&label=downloads)](https://github.com/DynamicKarabo/dbai/releases)

Query, analyze, and optimize your databases with natural language.

```
$ dbai connect postgres://user:***@localhost:5432/mydb
✓ Connected to PostgreSQL
  Schema loaded: 23 tables, 147 columns, 12 indexes

$ dbai query "show me users who ordered last month"
  SELECT u.name, COUNT(o.id) as orders
  FROM users u JOIN orders o ON u.id = o.user_id
  WHERE o.created_at > now() - interval '30 days'
  GROUP BY u.id ORDER BY orders DESC;

 id | name
----+-------
  1 | Alice
  2 | Bob
(2 rows)

$ dbai analyze "SELECT * FROM orders WHERE created_at > now()"
Execution Time: 12.50 ms

Issues:
🟡 Seq scan on orders (8000 rows) → Consider adding an index
🟡 Row estimate mismatch → Update statistics
```

## Install

### Binary (Linux)

```bash
curl -sfL https://github.com/DynamicKarabo/dbai/releases/latest/download/dbai-linux-amd64.tar.gz | tar xz
sudo mv dbai /usr/local/bin/
```

### Via Go

```bash
go install github.com/DynamicKarabo/dbai@latest
```

### Shell completion

```bash
eval "$(dbai completion bash)"       # bash
eval "$(dbai completion zsh)"        # zsh
dbai completion fish | source        # fish
```

## Commands

| Command | Description |
|---------|-------------|
| `dbai connect <dsn>` | Connect and introspect schema |
| `dbai query <sql\|question>` | Execute SQL or ask in plain English |
| `dbai analyze <query>` | Run EXPLAIN ANALYZE with performance insights |
| `dbai analyze --all` | Analyze all cached tables |
| `dbai repl` | Interactive shell with AI assistance |
| `dbai version` | Print version information |
| `dbai completion <shell>` | Generate shell completion scripts |

### Query flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Show generated SQL without executing |
| `--explain` | Show execution plan alongside results |
| `--json` | Output as JSON array |
| `--csv` | Output as CSV |

### REPL commands

| Command | Description |
|---------|-------------|
| `.help` | Show available commands |
| `.quit` | Exit the REPL |
| `.tables` | List all tables |
| `.schema` | Show full schema |
| `.connect <dsn>` | Connect to a different database |

## Supported Databases

| Database | Introspect | Query | Explain |
|----------|:----------:|:-----:|:-------:|
| PostgreSQL | ✅ | ✅ | ✅ JSON |
| SQLite | ✅ | ✅ | ✅ |
| MySQL | ✅ | ✅ | ✅ text |
| MariaDB | via MySQL driver | | |
| CockroachDB | via PG driver (use `postgresql://`) | |

## Configuration

Config is stored in `~/.dbai/config.json` and auto-loaded on each command.

```json
{
  "default_dsn": "postgres://user:***@localhost/mydb",
  "output_format": "table",
  "openai_model": "gpt-4"
}
```

| Env var | Purpose |
|---------|---------|
| `OPENAI_API_KEY` | AI query generation (required for NL) |
| `OPENAI_MODEL` | Override the AI model (default: gpt-4) |
| `DBAI_DSN` | Default connection string (fallback) |

## How It Works

1. **`dbai connect`** introspects your schema and caches it locally
2. **`dbai query`** sends schema context + your question to an LLM, gets back SQL, executes it
3. **`dbai analyze`** runs EXPLAIN ANALYZE in JSON format (PostgreSQL), parses the plan tree, and surfaces performance issues
4. **`dbai repl`** provides an interactive shell with history and dot-commands

## License

MIT
