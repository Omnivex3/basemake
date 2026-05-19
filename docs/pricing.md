# basemake Pricing

> **All local. All private. All yours.**  
> Query, analyze, and optimize your databases — no data leaves your machine.

---

## Plans

### Free — $0
**For solo developers and exploration**

Everything you need to query databases with AI:

- ✅ Natural language → SQL (BYOK — use your own API key or Ollama)
- ✅ Query execution (PostgreSQL, MySQL, SQLite)
- ✅ Full interactive TUI (REPL with history, autocomplete, scroll)
- ✅ Chat mode (`.ask` — talk to your data naturally)
- ✅ `.explain` + `.analyze` — basic performance insights
- ✅ **Index recommendations** — see what indexes would help (read-only)
- ✅ Output formats: table, JSON, CSV
- ✅ Local query history
- ✅ Unlimited connections
- ✅ All data stays on your machine

**Best for:** Side projects, learning, personal use.

---

### Pro — $15/mo ($150/yr)
**For professional developers and teams shipping to production**

Everything in Free, plus:

- 🔓 **Index recommendations — apply mode**
  - One-command index creation: `basemake index apply <id>`
  - Safe two-step: dry-run → review → `CREATE INDEX CONCURRENTLY`
- 🔓 **`basemake check` — CI/CD gate**
  - Prevent slow queries from reaching production
  - Exits with code: `0` = pass, `1` = slow, `2` = dangerous
  - Enforce in GitHub Actions, GitLab CI, any pipeline
- 🔓 **`basemake budget` — performance policy as code**
  - Codify rules like "no sequential scans > 1,000 rows on `orders`"
  - Check budgets into version control
  - Enforced by `basemake check` automatically
- 🔓 **`basemake watch` — query monitoring**
  - Schedule recurring queries, alert on regressions
  - Detect slow-downs before your users do
- 🔓 **`basemake diff` — schema drift detection**
  - Compare two databases: dev vs staging, staging vs prod
  - Detect missing indexes, column type changes, dropped tables
- 🔓 **`basemake doctor` — advanced diagnostics**
  - Full system health check with actionable fixes
- 🔓 **Priority email support**

**Best for:** Individual developers, small teams, CI/CD pipelines.

---

### Team — $39/seat/mo
**For engineering teams that ship together**

Everything in Pro, plus:

- 👥 **`basemake server` — team sync**
  - Shared query history across the team
  - Shared budget policies (one PR updates everyone's gates)
  - See who ran what, when, and how slow it was
- 👥 **Shared AI proxy** (optional)
  - One corporate API key for the whole team
  - Automatic response caching — save 40–60% on AI costs
  - Central AI usage audit log
- 👥 **License management dashboard**
  - Add/remove seats, view usage, manage renewals
- 👥 **RBAC** — read-only enforcement server-side
  - Lock production queries to read-only at the server level
  - No more "who dropped the users table"
- 👥 **Audit log** — every query, every check, every apply
- 👥 **Slack/Teams integration** — alerts on budget violations
- 👥 **Priority support** — 4-hour SLA

**Best for:** Engineering teams, startups, mid-market companies.

---

### Enterprise — Custom
**For organizations with compliance, SSO, and on-prem requirements**

- 🔒 **On-prem server** — deploy basemake behind your firewall
- 🔒 **SSO/SAML** — Okta, Azure AD, Google Workspace
- 🔒 **Custom AI proxy** — bring your own model endpoint
- 🔒 **Audit export** — JSON/CSV export of all query history
- 🔒 **Custom contract** — annual billing, volume discounts
- 🔒 **Dedicated support** — 1-hour SLA, Slack channel
- 🔒 **Training** — team onboarding session

**Best for:** Enterprise companies, regulated industries.

---

## Pricing Summary

| | Free | Pro | Team | Enterprise |
|---|:---:|:---:|:---:|:---:|
| **Price** | $0 | $15/mo | $39/seat/mo | Custom |
| **NL→SQL** | ✅ BYOK | ✅ BYOK | ✅ BYOK | ✅ BYOK |
| **Query execution** | ✅ | ✅ | ✅ | ✅ |
| **Full TUI/REPL** | ✅ | ✅ | ✅ | ✅ |
| **Index recommendations** | ✅ Read-only | ✅ Apply mode | ✅ Apply mode | ✅ Apply mode |
| **`basemake check` CI gate** | — | ✅ | ✅ | ✅ |
| **`basemake budget` policies** | — | ✅ | ✅ | ✅ |
| **`basemake watch` monitoring** | — | ✅ | ✅ | ✅ |
| **`basemake diff` schema diff** | — | ✅ | ✅ | ✅ |
| **Team server + sync** | — | — | ✅ | ✅ |
| **Shared AI proxy + caching** | — | — | ✅ | ✅ |
| **RBAC server-side** | — | — | ✅ | ✅ |
| **Audit log** | — | — | ✅ | ✅ |
| **SSO/SAML** | — | — | — | ✅ |
| **On-prem deployment** | — | — | — | ✅ |
| **Support** | Community | Email (24h) | Slack (4h) | Dedicated (1h) |

---

## Comparison: What you actually save

### basemake Pro replaces:

| Tool | Cost | basemake covers it? |
|------|------|-------------------|
| DataGrip | €109/yr | ✅ SQL client |
| DataGrip AI Pro | €100/yr | ✅ NL→SQL |
| A CI performance check script | $Dev hours | ✅ `basemake check` |
| A monitoring tool (Datadog, New Relic) | $15+/host/mo | ✅ `basemake watch` |
| **Total** | **$224+/yr + monitoring** | **$150/yr** |

### basemake Team replaces:

| Tool | Cost | basemake covers it? |
|------|------|-------------------|
| DataGrip × 10 devs | €1,090/yr | ✅ SQL client for everyone |
| DataGrip AI Pro × 10 devs | €1,000/yr | ✅ NL→SQL for everyone |
| A shared AI proxy setup | $Dev hours + infra | ✅ Built-in |
| Query monitoring | $15+/host/mo | ✅ Built-in |
| **Total for 10 devs** | **$2,290+/yr + infra** | **$4,680/yr ($39/seat)** |

**10 devs using basemake Team = ~2× the cost of DataGrip alone. But you also get: CI/CD gates, performance policies, monitoring, schema diffing, shared AI caching (reduces API costs), and team audit. The package replaces 3–4 separate tools.**

---

## FAQ

**How does licensing work for a CLI tool?**
- **Free**: No license needed. Download and run.
- **Pro**: License key via `basemake config set license_key xxx`. Required for `check`, `budget`, `watch`, `diff`, and index apply. Local REPL/query stays free even without a key.
- **Team**: Server requires a team license to start. Client connects to server with seat-based auth.
- **CI/CD**: Set `BASEMAKE_LICENSE_KEY` as a CI secret. One license key per CI runner.

**Can I use basemake at work with the Free tier?**
Yes. Free includes the full TUI, NL→SQL, query execution, and read-only index recommendations. No data leaves your machine. If you need CI/CD gates, budgets, or monitoring, that's when you go Pro.

**What does "BYOK" mean?**
Bring Your Own Key. You use your own API key (OpenAI, Anthropic, OpenCode) or a local Ollama instance. basemake never charges for AI tokens — you pay your provider directly.

**Does the AI proxy in Team save money?**
Yes. The server caches AI responses for identical queries. Teams running the same reports see a 40–60% reduction in API costs. The first query pays full price, the next 9 devs get it from cache.

**What if I just want the TUI and don't care about CI/CD?**
Free tier is perfect for you. Unlimited REPL, NL→SQL, all the good stuff.

**How is this different from DataGrip?**
DataGrip is a GUI IDE for your database. basemake is a **terminal-native tool** that works in SSH, CI/CD, Docker, and any headless environment. Plus: index recommendations with actual pg_stats selectivity math, policy-as-code budgets, and CI/CD gates — none of which DataGrip has.

---

## Getting Started

```bash
# Free — just use it
basemake connect postgres://user@localhost/mydb
basemake "show me users who signed up last week"

# Pro — add a license key
basemake config set license_key bmk_pro_xxxx
basemake check "SELECT * FROM orders" --threshold 500ms

# Team — start a server
basemake server --license bmk_team_xxxx
```
