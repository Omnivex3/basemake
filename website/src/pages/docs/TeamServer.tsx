import { Link } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'

/* ------------------------------------------------------------------ */
/*  Prose helpers                                                      */
/* ------------------------------------------------------------------ */

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

function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded-md border border-white/[0.06] bg-white/[0.04] px-1.5 py-0.5 text-sm font-mono text-[#e63946]">
      {children}
    </code>
  )
}

function CodeBlock({ children, lang = 'bash' }: { children: string; lang?: string }) {
  return (
    <div className="group relative mb-6 overflow-hidden rounded-xl border border-white/[0.06] bg-black/60 backdrop-blur-sm">
      <div className="flex items-center gap-2 border-b border-white/[0.06] px-4 py-2.5">
        <div className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-red-500/70" />
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
      <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#e63946]/60" />
      <span>{children}</span>
    </li>
  )
}

function Tip({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-6 rounded-xl border border-emerald-500/20 bg-emerald-500/5 px-5 py-4 text-sm text-emerald-300/80">
      {children}
    </div>
  )
}

/* ================================================================== */
/*  PAGE — Team Server                                                 */
/* ================================================================== */
export default function TeamServer() {
  return (
    <div className="pb-24">
      {/* Header */}
      <div className="mb-10">
        <Badge variant="outline" className="mb-3 border-[#e63946]/30 text-[#e63946] text-xs tracking-wide uppercase">
          Enterprise
        </Badge>
        <h1 className="text-4xl font-bold tracking-tight text-white sm:text-5xl">
          Team Server
        </h1>
        <p className="mt-3 text-lg text-white/50">
          Run basemake as a shared service for your team. Cut AI costs, enforce
          governance, and keep a central query history.
        </p>
      </div>

      <Separator className="mb-10 bg-white/[0.04]" />

      {/* Overview */}
      <Card className="mb-10 border-white/[0.06] bg-white/[0.02]">
        <CardHeader>
          <CardTitle className="text-white text-lg">At a Glance</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {[
            { label: 'Shared Cache', value: '40–60% cost savings' },
            { label: 'Audit Trail', value: 'Every query logged' },
            { label: 'RBAC', value: 'Read-only enforcement' },
            { label: 'Integrations', value: 'Slack & Teams' },
          ].map((stat) => (
            <div key={stat.label} className="rounded-lg border border-white/[0.06] bg-white/[0.03] px-4 py-3 text-center">
              <p className="text-xs font-semibold tracking-wide text-white/40 uppercase">{stat.label}</p>
              <p className="mt-1 text-lg font-semibold text-white">{stat.value}</p>
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Starting the Server */}
      <H2 id="starting">Starting the Server</H2>

      <P>
        Launch the Team Server with a single command. It runs as a lightweight HTTP
        service that team members connect to instead of running basemake directly.
      </P>

      <CodeBlock lang="bash">{`# Start on default port (8080)
basemake server

# Custom port and config
basemake server --port=9090 --config=team.yaml

# With license (required for Team features)
basemake server --license=bmk_team_xxxxxxxxxxxxxxxx`}</CodeBlock>

      <P>
        Once running, team members configure their local basemake to point at the server:
      </P>

      <CodeBlock lang="bash">{`# On each team member's machine
basemake config set server_url http://team-server:8080
basemake config set server_token <shared-secret>`}</CodeBlock>

      <Tip>
        The Team Server is stateless — you can run multiple instances behind a load
        balancer. The shared state lives in your database, not the server process.
      </Tip>

      {/* Shared Query History */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="query-history">Shared Query History</H2>

      <P>
        Every query run through the Team Server is recorded in a central query log.
        Team members can:
      </P>

      <UL>
        <LI><strong className="text-white">Browse recent queries</strong> — see what
        others have been working on.</LI>
        <LI><strong className="text-white">Replay past queries</strong> — reuse and
        adapt successful queries instead of starting from scratch.</LI>
        <LI><strong className="text-white">Search history</strong> — find queries by
        user, table, or natural language description.</LI>
        <LI><strong className="text-white">Flag favorites</strong> — bookmark queries
        for team-wide reference.</LI>
      </UL>

      <CodeBlock lang="bash">{`# Browse recent team queries
basemake history

# Search query history
basemake history --search "monthly revenue"

# View a specific query
basemake history --id=q-abc123`}</CodeBlock>

      {/* AI Proxy & Caching */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="ai-proxy">Shared AI Proxy & Caching</H2>

      <P>
        The Team Server proxies all AI requests through a shared cache. When one team
        member asks a question, the result is cached so that identical or similar
        questions from other team members are served instantly — no API call needed.
      </P>

      <div className="mb-6 overflow-x-auto rounded-xl border border-white/[0.06]">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Feature</th>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Benefit</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Semantic caching</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Similar questions return cached results (not just exact matches)</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Single API key</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">One API key for the whole team — no individual sign-ups</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Rate limiting</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Prevents runaway API costs from a single user</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Cost tracking</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Per-user and per-team AI usage dashboards</td>
            </tr>
          </tbody>
        </table>
      </div>

      <P>
        Typical teams see <strong>40–60% reduction</strong> in AI API costs after
        deploying the Team Server, since common questions are cached after the first ask.
      </P>

      {/* RBAC */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="rbac">RBAC: Read-Only Enforcement</H2>

      <P>
        Control what each team member can do with granular role-based access control.
        The Team Server enforces permissions at the proxy level — no database-level
        configuration needed.
      </P>

      <div className="mb-6 overflow-x-auto rounded-xl border border-white/[0.06]">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Role</th>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Permissions</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-semibold text-white">Admin</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Full access — queries, config, user management, audit logs</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-semibold text-white">Member</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Read-write queries, view history, personal config</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-semibold text-white">Read-Only</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">SELECT queries only — INSERT/UPDATE/DELETE/DROP are blocked by the proxy</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-semibold text-white">Viewer</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Browse query history only — cannot run new queries</td>
            </tr>
          </tbody>
        </table>
      </div>

      <CodeBlock lang="bash">{`# Add a team member
basemake server invite user@company.com --role=member

# Change a role
basemake server set-role user@company.com --role=read-only

# List team members
basemake server members`}</CodeBlock>

      {/* Audit Logging */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="audit-logging">Audit Logging</H2>

      <P>
        Every action on the Team Server is logged for compliance and troubleshooting.
        The audit log captures:
      </P>

      <UL>
        <LI><strong className="text-white">Query execution</strong> — who ran what,
        when, against which database, with the full SQL and generated SQL.</LI>
        <LI><strong className="text-white">Configuration changes</strong> — who modified
        server settings, budget profiles, or role assignments.</LI>
        <LI><strong className="text-white">Access events</strong> — login attempts,
        token rotations, and permission denials.</LI>
        <LI><strong className="text-white">AI usage</strong> — tokens consumed, cache
        hit rates, and cost per user.</LI>
      </UL>

      <CodeBlock lang="bash">{`# View the audit log
basemake server audit

# Filter by user
basemake server audit --user=alice@company.com

# Filter by action type
basemake server audit --type=query --since=7d

# Export as JSON for SIEM ingestion
basemake server audit --format=json --since=30d > audit-export.json`}</CodeBlock>

      <Tip>
        Audit logs are stored in your database and are never sent to basemake's
        servers. You own your data completely.
      </Tip>

      {/* Slack / Teams Integration */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="integrations">Slack & Teams Integrations</H2>

      <P>
        Connect the Team Server to your team's messaging platform. Team members can ask
        questions and get results directly from Slack or Microsoft Teams — no CLI required.
      </P>

      <CodeBlock lang="bash">{`# Configure Slack integration
basemake server integrations add slack \
  --token=xoxb-... \
  --channel=#data-questions

# Configure Teams integration
basemake server integrations add teams \
  --webhook=https://your-company.webhook.office.com/...`}</CodeBlock>

      <P>
        Once configured, team members can:
      </P>

      <UL>
        <LI>Ask questions in natural language from Slack/Teams.</LI>
        <LI>Receive formatted results inline with the message.</LI>
        <LI>Get notified when slow queries are detected.</LI>
        <LI>Receive weekly AI usage and cost summaries.</LI>
      </UL>

      {/* Deployment */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="deployment">Deployment Options</H2>

      <P>
        The Team Server is a single binary — deploy it anywhere:
      </P>

      <UL>
        <LI><strong className="text-white">Docker</strong> —{' '}
        <Code>docker run ghcr.io/dynamickarabo/basemake-server</Code></LI>
        <LI><strong className="text-white">Systemd service</strong> — run as a
        background service on any Linux VM.</LI>
        <LI><strong className="text-white">Kubernetes</strong> — deploy as a Deployment
        with a ConfigMap for configuration.</LI>
        <LI><strong className="text-white">Railway / Fly.io</strong> — one-click deploy
        on supported platforms.</LI>
      </UL>

      <CodeBlock lang="yaml">{`# docker-compose.yml
version: '3.8'
services:
  basemake-server:
    image: ghcr.io/dynamickarabo/basemake-server:latest
    ports:
      - "8080:8080"
    environment:
      - BASEMAKE_LICENSE_KEY=bmk_team_xxxx
      - DATABASE_URL=postgres://.../basemake
    volumes:
      - ./team.yaml:/etc/basemake/config.yaml:ro`}</CodeBlock>
    </div>
  )
}
