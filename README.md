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

## Full Documentation

Comprehensive documentation covering every command, flag, config option, driver, and internal detail lives in [`docs/`](docs/README.md).

| Document | What's Covered |
|----------|----------------|
| [Overview](docs/overview.md) | Architecture, design decisions, data flow, build pipeline |
| [Commands Reference](docs/commands.md) | All 6 commands: connect, query, analyze, repl, completion, version |
| [Configuration](docs/configuration.md) | Config file, env vars, defaults, precedence |
| [Database Drivers](docs/database-drivers.md) | PostgreSQL, MySQL, SQLite internals |
| [Output Formats](docs/output-formats.md) | Table, JSON, CSV, TSV — every formatting detail |
| [AI Integration](docs/ai-integration.md) | NL→SQL generation, model selection, API details |
| [Development Guide](docs/development.md) | Build, test, lint, CI/CD, adding drivers |

## License

MIT
