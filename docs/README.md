# dbai Documentation

Welcome to the comprehensive documentation for **dbai** — the AI-powered database CLI.

## Quick Navigation

| Document | What You'll Find |
|----------|------------------|
| [Overview](overview.md) | Architecture, design decisions, data flow, build pipeline |
| [Commands](commands.md) | Full reference for all 6 commands with every flag and behavior |
| [Configuration](configuration.md) | Config file format, env vars, defaults, precedence rules |
| [Database Drivers](database-drivers.md) | PostgreSQL, MySQL, SQLite implementation details |
| [Output Formats](output-formats.md) | Table, JSON, CSV, TSV formatting rules |
| [AI Integration](ai-integration.md) | NL→SQL generation, OpenAI API, model selection |
| [Development](development.md) | Building, testing, CI/CD, adding drivers/commands |

## Quick Reference

```
Usage:
  dbai [command]

Available Commands:
  connect     Connect and introspect a database
  query       Ask a natural language question about your data
  analyze     Analyze query performance with EXPLAIN ANALYZE
  repl        Interactive SQL shell with AI assistance
  completion  Generate shell completion scripts
  version     Print version information

Flags:
  -h, --help   Show help for any command

Query Flags:
  --json              Output as JSON
  --csv               Output as CSV
  --dry-run           Preview SQL without executing
  --explain           Show execution plan

Analyze Flag:
  --all               Analyze all cached tables

REPL Flag:
  --format <format>   Output format (table, json, csv)

Environment Variables:
  OPENAI_API_KEY      Required for NL→SQL queries
  OPENAI_MODEL        Override AI model (default: gpt-4)
  DBAI_DSN            Default connection string

Config File:
  ~/.dbai/config.json

Schema Cache:
  ~/.dbai/schema.json
```

## Stats

| Metric | Value |
|--------|-------|
| Commands | 6 |
| Database drivers | 3 (PostgreSQL, MySQL, SQLite) |
| Output formats | 4 (table, JSON, CSV, TSV) |
| Test functions | 29+ |
| CI build targets | 5 (linux/mac/windows × amd64/arm64) |
| Go version | 1.25 |
| Lines of Go | ~2,500 |
| External deps | 3 (cobra, lib/pq, go-sql-driver/mysql, modernc.org/sqlite) |
