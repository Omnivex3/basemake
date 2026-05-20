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
    <code className="rounded-md border border-white/[0.06] bg-white/[0.04] px-1.5 py-0.5 text-sm font-mono text-[#ff3131]">
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

function Tip({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-6 rounded-xl border border-emerald-500/20 bg-emerald-500/5 px-5 py-4 text-sm text-emerald-300/80">
      {children}
    </div>
  )
}

/* ================================================================== */
/*  PAGE — Quickstart                                                  */
/* ================================================================== */
export default function Quickstart() {
  return (
    <div className="pb-24">
      {/* Header */}
      <div className="mb-10">
        <Badge variant="outline" className="mb-3 border-[#ff3131]/30 text-[#ff3131] text-xs tracking-wide uppercase">
          Getting Started
        </Badge>
        <h1 className="text-4xl font-bold tracking-tight text-white sm:text-5xl">
          Quickstart
        </h1>
        <p className="mt-3 text-lg text-white/50">
          Get basemake installed and running your first query in under two minutes.
        </p>
      </div>

      <Separator className="mb-10 bg-white/[0.04]" />

      {/* 1. Install */}
      <H2 id="install">1. Install</H2>

      <P>
        basemake ships as a single static binary — no runtime dependencies, no Node.js, no
        Python runtime. Download the latest release for your platform from the{' '}
        <Link to="https://github.com/DynamicKarabo/basemake/releases" className="text-[#ff3131] hover:underline">
          releases page
        </Link>
        , make it executable, and move it into your PATH.
      </P>

      <CodeBlock lang="bash">
        {`# Download (Linux / macOS)
curl -LO https://github.com/DynamicKarabo/basemake/releases/latest/download/basemake-$(uname -s)-$(uname -m).tar.gz
tar xzf basemake-*.tar.gz

# Make it executable and install
chmod +x basemake
sudo mv basemake /usr/local/bin/

# Verify
basemake --version`}
      </CodeBlock>

      <Tip>
        No sudo? Move it to <Code>~/.local/bin</Code> instead and make sure that directory
        is in your <Code>$PATH</Code>.
      </Tip>

      <H3>Homebrew (macOS / Linux)</H3>
      <P>If you prefer Homebrew:</P>
      <CodeBlock lang="bash">brew install basemake</CodeBlock>

      {/* 2. Init */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="init">2. Initialize</H2>

      <P>
        Run <Code>basemake init</Code> to start the setup wizard. It will:
      </P>

      <UL>
        <LI><strong className="text-white">Detect your database</strong> — scans common
        connection strings from environment variables and existing config files.</LI>
        <LI><strong className="text-white">Pick an AI provider</strong> — prompts you to
        choose between OpenAI, Anthropic, OpenCode, or a local Ollama model. You can
        bring your own API key or use the built-in defaults.</LI>
        <LI><strong className="text-white">Run a test query</strong> — executes a sample
        question against your database to confirm everything works end-to-end.</LI>
      </UL>

      <CodeBlock lang="bash">
{`basemake init

# ── basemake setup wizard ──────────────────────
# 
#   ✓ Found PostgreSQL at localhost:5432
#   ✓ Detected 28 tables in public schema
#   ? AI provider: [OpenAI / Anthropic / OpenCode / Ollama]
#   ? API key: ********
#   ✓ Test query passed (137ms)
#   ✓ Config written to ~/.config/basemake/config.yaml
#   ✓ Ready to go!
# ───────────────────────────────────────────────`}
      </CodeBlock>

      {/* 3. Connect */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="connect">3. Connect to a Database</H2>

      <P>
        If you skipped <Code>basemake init</Code> or want to connect to a different database,
        use the <Code>connect</Code> command with a connection string:
      </P>

      <CodeBlock lang="bash">
        {`basemake connect postgres://user@localhost:5432/mydb
basemake connect mysql://user:pass@host:3306/mydb
basemake connect sqlite:///path/to/db.sqlite`}
      </CodeBlock>

      <P>
        basemake supports PostgreSQL, MySQL, and SQLite. Wire-compatible databases like
        and ClickHouse. See the full list in{' '}
        <Link to="/docs/configuration" className="text-[#ff3131] hover:underline">Configuration</Link>.
      </P>

      {/* 4. First Query */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="first-query">4. Run Your First Query</H2>

      <P>
        Ask questions in plain English. basemake translates your natural language into
        optimized SQL and returns the results as a formatted table.
      </P>

      <CodeBlock lang="bash">
{`basemake "show me the top 10 customers by revenue this month"

# ┌──────────────┬────────┐
# │ name         │ rev    │
# ├──────────────┼────────┤
# │ Acme Corp    │ $142k  │
# │ Globex Inc   │ $98k   │
# │ Initech      │ $87k   │
# │ ...          │ ...    │
# └──────────────┴────────┘
# ✓ 10 rows · 23ms · 4.2s AI`}
      </CodeBlock>

      <P>
        You can pipe queries from other commands too:
      </P>

      <CodeBlock lang="bash">
{`echo "list all tables with their row counts" | basemake --format=csv`}
      </CodeBlock>

      {/* 5. REPL */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="repl">5. Enter the REPL</H2>

      <P>
        Run <Code>basemake</Code> with no arguments to open the interactive shell. The REPL
        gives you a conversational interface — ask follow-up questions, refine queries,
        and explore your schema without leaving the terminal.
      </P>

      <CodeBlock lang="bash">
{`basemake

  ╭──────────────────────────────────────────╮
  │  basemake  v1.0.0                        │
  │  Connected: postgres://localhost:5432/mydb│
  │  Type .help for commands                 │
  ╰──────────────────────────────────────────╯

  > show me orders that haven't shipped in 7 days
  >   …
  > add a status filter for 'pending'
  >   …
  > explain the query plan for the last one
  >   …
  > .exit`}
      </CodeBlock>

      <P>
        Inside the REPL you can use dot-commands like <Code>.connect</Code>,{' '}
        <Code>.tables</Code>, <Code>.schema</Code>, and <Code>.help</Code>. See the{' '}
        <Link to="/docs/commands" className="text-[#ff3131] hover:underline">Commands Reference</Link> for the full list.
      </P>

      {/* Next steps */}
      <Separator className="my-10 bg-white/[0.04]" />

      <Card className="border-white/[0.06] bg-white/[0.02]">
        <CardHeader>
          <CardTitle className="text-white text-lg">Next Steps</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-white/60">
          <p>
            🚀 <strong className="text-white">Set up CI/CD</strong> —{' '}
            <Link to="/docs/ci-cd" className="text-[#ff3131] hover:underline">
              Integrate basemake check into your pipeline
            </Link>
          </p>
          <p>
            🔑 <strong className="text-white">Configure AI providers</strong> —{' '}
            <Link to="/docs/ai-providers" className="text-[#ff3131] hover:underline">
              Bring your own API key or use local models
            </Link>
          </p>
          <p>
            👥 <strong className="text-white">Team Server</strong> —{' '}
            <Link to="/docs/team-server" className="text-[#ff3131] hover:underline">
              Share queries and AI credits with your team
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
