# basemake Documentation

Welcome to the comprehensive documentation for **basemake** — the AI-powered database CLI.

## Quick Navigation

| Document | What You'll Find |
|----------|------------------|
| [Overview](overview.md) | Architecture, design decisions, data flow, build pipeline |
| [Commands](commands.md) | Full reference for all 14 commands with flags and behavior |
| [Configuration](configuration.md) | Config file format, env vars, defaults, precedence rules |
| [Database Drivers](database-drivers.md) | PostgreSQL, MySQL, SQLite implementation details |
| [Output Formats](output-formats.md) | Table, JSON, CSV, TSV formatting rules |
| [AI Integration](ai-integration.md) | NL→SQL generation, OpenAI API, model selection |
| [Guardrails](guardrails.md) | SELECT * protection, schema truncation, FK context |
| [Payment Flow](payment-flow.md) | Lemon Squeezy webhook, license key generation, Vercel deploy |
| [Operations](operations.md) | Self-hosted runner, CI/CD pipeline, pre-commit hooks |
| [Development](development.md) | Building, testing, CI/CD, adding drivers/commands |

## Quick Reference

```
Usage:
  basemake [command]

Understand:
  connect     Connect and introspect a database
  analyze     Analyze query performance with EXPLAIN ANALYZE
  diff        Show schema differences between two databases
  history     Show team query log (via server)

Act:
  query       Ask a natural language question about your data
  repl        Interactive SQL shell with AI assistance
  check       CI gate — check query performance, exit with code

Govern:
  budget      Database performance policy as code
  watch       Monitor a query on a schedule, alert on regression
  server      Start the basemake team daemon
  sync        Sync data with the team server

Infrastructure:
  config      Manage persistent configuration
  completion  Generate shell completion scripts
  version     Print version information

Flags:
  -h, --help          Show help for any command

Query Flags:
  --json              Output as JSON
  --csv               Output as CSV
  --dry-run           Preview SQL without executing
  --explain           Show execution plan

Analyze Flag:
  --all               Analyze all cached tables

Environment Variables:
  AI_PROVIDER         Provider: openai, anthropic, ollama
  OPENAI_API_KEY      API key for OpenAI
  ANTHROPIC_API_KEY   API key for Anthropic

Config File:
  ~/.basemake/config.json

Schema Cache:
  ~/.basemake/schema.json

Server Data:
  ~/.basemake/server/basemake.db
```

## Stats

| Metric | Value |
|--------|-------|
| Commands | 14 (3 categories: Understand, Act, Govern) |
| Database drivers | 3 (PostgreSQL, MySQL, SQLite) |
| Guardrails | 3 (SELECT *, schema truncation, FK context) |
| Output formats | 3 (table, JSON, CSV) |
| Test functions | 31+ |
| CI build targets | 1 (push), 5 (tags) — self-hosted runner |
| Go version | 1.26 |
| Lines of Go | ~6,700 |
| Payment | Lemon Squeezy → Vercel → Resend |
| External deps | 4 (cobra, lib/pq, go-sql-driver/mysql, modernc.org/sqlite) |
