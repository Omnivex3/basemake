import { Link } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'

/* ------------------------------------------------------------------ */
/*  Prose helpers                                                      */
/* ------------------------------------------------------------------ */

function H1({ children }: { children: React.ReactNode }) {
  return (
    <h1 className="text-4xl font-bold tracking-tight text-white sm:text-5xl">
      {children}
    </h1>
  )
}

function H2({ id, children }: { id?: string; children: React.ReactNode }) {
  return (
    <h2
      id={id}
      className="mt-12 mb-4 scroll-mt-24 text-2xl font-bold tracking-tight text-white"
    >
      {children}
    </h2>
  )
}

function H3({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="mt-8 mb-3 text-xl font-semibold tracking-tight text-white">
      {children}
    </h3>
  )
}

function P({ children }: { children: React.ReactNode }) {
  return (
    <p className="mb-4 leading-relaxed text-white/60">
      {children}
    </p>
  )
}

function Code({ children, className = '' }: { children: React.ReactNode; className?: string }) {
  return (
    <code className={`rounded-md border border-white/[0.06] bg-white/[0.04] px-1.5 py-0.5 text-sm font-mono text-[#ff3131] ${className}`}>
      {children}
    </code>
  )
}

function CodeBlock({ children, lang = 'bash' }: { children: string; lang?: string }) {
  return (
    <div className="group relative mb-6 overflow-hidden rounded-xl border border-white/[0.06] bg-black/60 backdrop-blur-sm">
      <div className="flex items-center gap-2 border-b border-white/[0.06] px-4 py-2.5">
        <div className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-[#ff3131]/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-yellow-500/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-green-500/70" />
        </div>
        <span className="ml-2 text-[11px] text-white/30 font-mono">{lang}</span>
      </div>
      <pre className="overflow-x-auto p-5 text-sm leading-relaxed">
        <code className="font-mono text-white/80 [word-spacing:0.15em]">
          {children}
        </code>
      </pre>
    </div>
  )
}

function UL({ children }: { children: React.ReactNode }) {
  return (
    <ul className="mb-6 space-y-2 text-white/60">
      {children}
    </ul>
  )
}

function LI({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex items-start gap-2 leading-relaxed">
      <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
      <span>{children}</span>
    </li>
  )
}

function Table({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-6 overflow-x-auto rounded-xl border border-white/[0.06]">
      <table className="w-full text-sm">
        {children}
      </table>
    </div>
  )
}

function Th({ children }: { children: React.ReactNode }) {
  return (
    <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">
      {children}
    </th>
  )
}

function Td({ children }: { children: React.ReactNode }) {
  return (
    <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 last:border-0">
      {children}
    </td>
  )
}

/* ------------------------------------------------------------------ */
/*  Command entry component                                            */
/* ------------------------------------------------------------------ */
function CmdEntry({
  name,
  description,
  syntax,
  example,
}: {
  name: string
  description: string
  syntax?: string
  example?: string
}) {
  return (
    <div className="mb-8">
      <div className="mb-2 flex items-center gap-3">
        <Code className="!bg-[#ff3131]/10 !border-[#ff3131]/20 !text-white text-base font-semibold !px-3 !py-1">
          {name}
        </Code>
      </div>
      <P>{description}</P>
      {syntax && (
        <>
          <H3>Syntax</H3>
          <CodeBlock lang="bash">{syntax}</CodeBlock>
        </>
      )}
      {example && (
        <>
          <H3>Example</H3>
          <CodeBlock lang="bash">{example}</CodeBlock>
        </>
      )}
    </div>
  )
}

/* ================================================================== */
/*  PAGE — Commands Reference                                          */
/* ================================================================== */
export default function Commands() {
  return (
    <div className="pb-24">
      {/* Header */}
      <div className="mb-10">
        <Badge variant="outline" className="mb-3 border-[#ff3131]/30 text-[#ff3131] text-xs tracking-wide uppercase">
          Reference
        </Badge>
        <H1>Commands Reference</H1>
        <p className="mt-3 text-lg text-white/50">
          Every command in the basemake CLI, with syntax and examples.
        </p>
      </div>

      <Separator className="mb-10 bg-white/[0.04]" />

      {/* Overview table */}
      <Card className="mb-10 border-white/[0.06] bg-white/[0.02]">
        <CardHeader>
          <CardTitle className="text-white text-lg">Quick Reference</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <thead>
              <tr>
                <Th>Command</Th>
                <Th>Description</Th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <Td><Code>basemake</Code></Td>
                <Td>REPL mode — interactive shell</Td>
              </tr>
              <tr>
                <Td><Code>basemake init</Code></Td>
                <Td>Setup wizard</Td>
              </tr>
              <tr>
                <Td><Code>basemake connect</Code></Td>
                <Td>Connect to a database</Td>
              </tr>
              <tr>
                <Td><Code>basemake "query"</Code></Td>
                <Td>One-liner natural-language query</Td>
              </tr>
              <tr>
                <Td><Code>basemake check</Code></Td>
                <Td>CI/CD gate — SQL quality checks</Td>
              </tr>
              <tr>
                <Td><Code>basemake budget</Code></Td>
                <Td>Performance policy management</Td>
              </tr>
              <tr>
                <Td><Code>basemake watch</Code></Td>
                <Td>Query monitoring</Td>
              </tr>
              <tr>
                <Td><Code>basemake diff</Code></Td>
                <Td>Schema drift detection</Td>
              </tr>
              <tr>
                <Td><Code>basemake doctor</Code></Td>
                <Td>Diagnostics and health checks</Td>
              </tr>
              <tr>
                <Td><Code>basemake index</Code></Td>
                <Td>Index recommendations</Td>
              </tr>
              <tr>
                <Td><Code>basemake config</Code></Td>
                <Td>Configuration management</Td>
              </tr>
              <tr>
                <Td><Code>basemake server</Code></Td>
                <Td>Team server mode</Td>
              </tr>
            </tbody>
          </Table>
        </CardContent>
      </Card>

      {/* ============================================================ */}
      {/*  Commands (alphabetical)                                      */}
      {/* ============================================================ */}

      <H2 id="basemake">basemake (REPL Mode)</H2>
      <P>
        With no arguments, basemake launches an interactive REPL. Type questions in
        plain English, get back SQL results in real time. The REPL preserves conversation
        context so you can ask follow-up questions.
      </P>
      <P>
        Dot-commands available inside the REPL: <Code>.connect</Code>,{' '}
        <Code>.tables</Code>, <Code>.schema</Code>, <Code>.help</Code>, <Code>.exit</Code>.
      </P>
      <CodeBlock lang="bash">{`basemake

  > show me orders from last week
  >   …
  > filter to only completed orders
  >   …
  > .exit`}</CodeBlock>

      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="init">basemake init</H2>
      <P>
        The setup wizard. Detects your database connection, prompts for an AI provider,
        runs a test query, and writes a configuration file. Safe to re-run at any time.
      </P>
      <CodeBlock lang="bash">{`basemake init
basemake init --force   # overwrite existing config`}</CodeBlock>

      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="connect">basemake connect</H2>
      <P>
        Connect to a database using a connection string. Supports PostgreSQL, MySQL,
        MariaDB, and SQLite. Wire-compatible databases like TimescaleDB and CockroachDB
      </P>
      <CodeBlock lang="bash">{`basemake connect postgres://user:pass@host:5432/dbname
basemake connect "postgres://user@localhost:5432/mydb?sslmode=require"`}</CodeBlock>

      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="query">basemake "query"</H2>
      <P>
        Pass a natural-language query as a single argument. basemake translates it to
        SQL, executes it, and prints the results. Supports piping via stdin.
      </P>
      <CodeBlock lang="bash">{`basemake "show me the top 10 customers by revenue"
basemake "list all tables with row counts" --format=csv
cat query.txt | basemake --format=json`}</CodeBlock>
      <P>
        Flags: <Code>--format</Code> (table, csv, json), <Code>--explain</Code> (show
        generated SQL), <Code>--no-execute</Code> (generate SQL without running it).
      </P>

      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="check">basemake check</H2>
      <P>
        The CI/CD gate. Analyzes SQL files or queries for performance issues,
        security concerns, and schema compatibility. Returns exit codes suitable for
        pipeline gating.
      </P>
      <CodeBlock lang="bash">{`basemake check migrations/001_add_users.sql
basemake check --dir ./migrations --budget=strict
basemake check --stdin < query.sql`}</CodeBlock>
      <P>
        Exit codes: <Code>0</Code> = pass, <Code>1</Code> = slow queries detected,{' '}
        <Code>2</Code> = dangerous patterns found.
      </P>

      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="budget">basemake budget</H2>
      <P>
        Define and manage performance budgets. Set thresholds for query latency, row
        scans, join depth, and more. Violations are surfaced during <Code>check</Code>.
      </P>
      <CodeBlock lang="bash">{`basemake budget set --max-latency=500ms
basemake budget set --max-rows=10000
basemake budget list
basemake budget apply --profile=strict`}</CodeBlock>

      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="watch">basemake watch</H2>
      <P>
        Monitor query performance in real time. Shows active queries, historical
        latency trends, and identifies hot or regressed queries.
      </P>
      <CodeBlock lang="bash">{`basemake watch
basemake watch --interval=5s
basemake watch --alert-latency=1s`}</CodeBlock>

      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="diff">basemake diff</H2>
      <P>
        Compare schemas across environments. Detect missing columns, type mismatches,
        index differences, and schema drift before it reaches production.
      </P>
      <CodeBlock lang="bash">{`basemake diff --from=staging --to=production
basemake diff --file=schema_a.sql --file=schema_b.sql`}</CodeBlock>

      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="doctor">basemake doctor</H2>
      <P>
        Run diagnostics on your basemake installation. Checks configuration validity,
        database connectivity, AI provider reachability, and license status.
      </P>
      <CodeBlock lang="bash">{`basemake doctor
basemake doctor --verbose`}</CodeBlock>

      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="index">basemake index</H2>
      <P>
        Analyze query patterns and recommend indexes to speed up your most frequent
        access paths. Recommendations include estimated improvement.
      </P>
      <CodeBlock lang="bash">{`basemake index
basemake index --apply    # generate and apply recommended indexes
basemake index --review   # interactive review before applying`}</CodeBlock>

      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="config">basemake config</H2>
      <P>
        View and modify basemake configuration. Supports get, set, list, and unset
        operations on individual keys.
      </P>
      <CodeBlock lang="bash">{`basemake config list
basemake config set ai_provider openai
basemake config set ai_model gpt-4o
basemake config get ai_provider`}</CodeBlock>

      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="server">basemake server</H2>
      <P>
        Start basemake in team server mode. Provides a shared query proxy, AI cache,
        RBAC enforcement, and audit logging for teams.
      </P>
      <CodeBlock lang="bash">{`basemake server --port=8080
basemake server --config=team.yaml
basemake server --license=bmk_team_xxxx`}</CodeBlock>
    </div>
  )
}
