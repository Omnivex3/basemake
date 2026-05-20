import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { ChevronDown } from 'lucide-react'

/* ------------------------------------------------------------------ */
/*  Prose helpers                                                      */
/* ------------------------------------------------------------------ */

function P({ children }: { children: React.ReactNode }) {
  return (
    <p className="mb-4 leading-relaxed text-muted-foreground">
      {children}
    </p>
  )
}

function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded-md border border-border/[0.06] bg-muted/30 px-1.5 py-0.5 text-sm font-mono text-[#ff3131]">
      {children}
    </code>
  )
}

function CodeBlock({ children, lang = 'bash' }: { children: string; lang?: string }) {
  return (
    <div className="group relative mb-6 overflow-hidden rounded-xl border border-border/[0.06] bg-black/60 backdrop-blur-sm">
      <div className="flex items-center gap-2 border-b border-border/[0.06] px-4 py-2.5">
        <div className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-[#ff3131]/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-yellow-500/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-green-500/70" />
        </div>
        <span className="ml-2 text-[11px] text-muted-foreground/80 font-mono">{lang}</span>
      </div>
      <pre className="overflow-x-auto p-5 text-sm leading-relaxed">
        <code className="font-mono text-foreground/80 [word-spacing:0.15em]">
          {children}
        </code>
      </pre>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/*  Accordion / FAQ item                                              */
/* ------------------------------------------------------------------ */

interface FAQItemProps {
  question: string
  children: React.ReactNode
  defaultOpen?: boolean
}

function FAQItem({ question, children, defaultOpen = false }: FAQItemProps) {
  const [open, setOpen] = useState(defaultOpen)

  return (
    <div className="group rounded-xl border border-border/[0.06] bg-muted/30 transition-colors hover:border-border/[0.10]">
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between px-6 py-5 text-left"
      >
        <span className="pr-4 text-base font-medium text-foreground/90">
          {question}
        </span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-muted-foreground transition-transform duration-200 ${
            open ? 'rotate-180' : ''
          }`}
        />
      </button>
      {open && (
        <div className="border-t border-border/[0.06] px-6 py-5 text-sm leading-relaxed text-muted-foreground">
          {children}
        </div>
      )}
    </div>
  )
}

/* ================================================================== */
/*  PAGE — FAQ                                                         */
/* ================================================================== */
export default function FAQ() {
  return (
    <div className="pb-24">
      {/* Header */}
      <div className="mb-10">
        <Badge variant="outline" className="mb-3 border-[#ff3131]/30 text-[#ff3131] text-xs tracking-wide uppercase">
          Help
        </Badge>
        <h1 className="text-4xl font-bold tracking-tight text-foreground sm:text-5xl">
          Frequently Asked Questions
        </h1>
        <p className="mt-3 text-lg text-muted-foreground">
          Everything you need to know about basemake.
        </p>
      </div>

      <Separator className="mb-10 bg-muted/30" />

      <div className="space-y-3">
        <FAQItem question="What is basemake?">
          <P>
            basemake is a <strong className="text-foreground">local-first CLI tool</strong>{' '}
            that converts natural language questions into optimized SQL. Instead of
            writing complex JOINs or memorizing table schemas, you describe what you
            need in plain English and basemake handles the translation, execution, and
            performance analysis.
          </P>
          <P>
            It runs entirely on your machine — your data never leaves your database. You
            bring your own AI provider (or use a local model via Ollama), and basemake
            orchestrates the rest.
          </P>
          <P>
            Beyond query generation, basemake includes CI/CD gates, schema diffing,
            index recommendations, query monitoring, and a team server for
            collaboration.
          </P>
        </FAQItem>

        <FAQItem question="How is the free tier different from Pro?">
          <P>
            The <strong className="text-foreground">Free tier</strong> gives you full access to
            basemake's core CLI — REPL mode, natural language queries, and local
            configuration. It's designed for individual developers and personal projects.
          </P>
          <P>
            <strong className="text-foreground">Pro</strong> adds CI/CD integration with{' '}
            <Code>basemake check</Code>, custom check policies, budget profiles, index
            recommendations, and schema diffing. It's designed for professional developers
            who need to gate SQL quality in their pipelines.
          </P>
          <P>
            <strong className="text-foreground">Team</strong> adds the Team Server — shared AI
            proxy and caching (40-60% cost savings), RBAC, audit logging, and Slack/Teams
            integrations. It's designed for organizations.
          </P>
          <P>
            See the{' '}
            <Link to="/docs/licensing" className="text-[#ff3131] hover:underline">
              Licensing page
            </Link>{' '}
            for pricing details.
          </P>
        </FAQItem>

        <FAQItem question="How does Bring Your Own Key (BYOK) work?">
          <P>
            basemake does not charge per-token or bundle AI credits. You configure
            basemake with your own API key from OpenAI, Anthropic, OpenCode, or a local
            Ollama instance. basemake uses your key to translate natural language into
            SQL.
          </P>
          <CodeBlock lang="bash">{`basemake config set ai_provider openai
basemake config set ai_model gpt-4o
basemake config set ai_api_key sk-...`}</CodeBlock>
          <P>
            This means you use your existing API credits, you can switch providers at
            any time, and with Ollama you can run everything offline with zero API costs.
            See the{' '}
            <Link to="/docs/ai-providers" className="text-[#ff3131] hover:underline">
              AI Providers page
            </Link>{' '}
            for details.
          </P>
        </FAQItem>

        <FAQItem question="Can I use basemake at work?">
          <P>
            Yes. basemake is designed for professional use. Key compliance features:
          </P>
          <ul className="mb-4 space-y-2">
            {[
              'All processing is local — your data never leaves your database or your machine.',
              'For Team Server deployments, audit logs capture every query and configuration change.',
              'RBAC enforces read-only access for non-admin team members.',
              'No telemetry or phone-home — basemake does not collect usage data.',
              'License keys use offline HMAC verification — no network calls to validate.',
            ].map((item) => (
              <li key={item} className="flex items-start gap-2">
                <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-500/60" />
                <span>{item}</span>
              </li>
            ))}
          </ul>
          <P>
            The Team Server includes role-based access control and full audit logging
            for compliance requirements. See the{' '}
            <Link to="/docs/team-server" className="text-[#ff3131] hover:underline">
              Team Server page
            </Link>{' '}
            for details.
          </P>
        </FAQItem>

        <FAQItem question="Does the AI proxy actually save money?">
          <P>
            Yes. Teams using the{' '}
            <Link to="/docs/team-server" className="text-[#ff3131] hover:underline">
              Team Server
            </Link>{' '}
            typically see <strong className="text-foreground">40–60% reduction</strong> in AI
            API costs. Here's how:
          </P>
          <ul className="mb-4 space-y-2">
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
              <span><strong className="text-foreground">Semantic caching:</strong> When one person asks "show me monthly revenue," the result is cached. If a colleague asks "what's our monthly revenue trend?" they get the cached result — no new API call.</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
              <span><strong className="text-foreground">Single API key:</strong> One key for the whole team means you benefit from provider tier discounts and consolidated billing.</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
              <span><strong className="text-foreground">Rate limiting:</strong> Prevents a single user from accidentally running up the bill.</span>
            </li>
          </ul>
          <P>
            The cache is most effective for teams where multiple people ask about the
            same data — data teams, analytics teams, and engineering teams all see
            strong cache hit rates.
          </P>
        </FAQItem>

        <FAQItem question="How is this different from DataGrip or other database tools?">
          <P>
            Traditional database tools like DataGrip, DBeaver, and TablePlus are
            <strong className="text-foreground"> GUI-based editors</strong> that require you
            to write SQL by hand. basemake takes a different approach:
          </P>
          <ul className="mb-4 space-y-2">
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
              <span><strong className="text-foreground">Natural language first:</strong> Describe what you want in English — basemake generates the SQL. You don't need to know table names, column types, or JOIN syntax.</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
              <span><strong className="text-foreground">CLI-native:</strong> Designed for the terminal, not a GUI. Perfect for SSH sessions, CI/CD pipelines, and developer workflows.</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
              <span><strong className="text-foreground">Built-in performance analysis:</strong> Every query comes with execution insights — no separate EXPLAIN step needed.</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
              <span><strong className="text-foreground">CI/CD gates:</strong> basemake check slots into your pipeline to catch bad SQL before it ships.</span>
            </li>
          </ul>
          <P>
            That said, you can absolutely use basemake alongside your existing database
            tools — they solve different problems.
          </P>
        </FAQItem>

        <FAQItem question="What databases are supported?">
          <P>
            basemake currently supports:
          </P>
          <div className="mb-4 grid grid-cols-2 gap-2 sm:grid-cols-3">
            {[
              'PostgreSQL',
              'MySQL',
              'MariaDB',
              'SQLite',
              'TimescaleDB (beta)',
              'CockroachDB (beta)',
              'ClickHouse',
            ].map((db) => (
              <div
                key={db}
                className="rounded-lg border border-border/[0.06] bg-muted/30 px-3 py-2 text-center text-sm text-muted-foreground"
              >
                {db}
              </div>
            ))}
          </div>
          <P>
            basemake auto-detects each database's dialect, so the same natural language
            question generates correct SQL for your specific database. Need support for
            another database?{' '}
            <Link to="https://github.com/DynamicKarabo/basemake/issues" className="text-[#ff3131] hover:underline">
              Open an issue
            </Link>
            .
          </P>
        </FAQItem>

        <FAQItem question="Is it really all local?">
          <P>
            Yes — with one caveat: the AI translation step.
          </P>
          <P>
            <strong className="text-foreground">Your data stays local:</strong> basemake sends
            your database <strong className="text-foreground">schema metadata</strong> (table
            names, column names, types, relationships) to the AI provider. It does
            <strong className="text-foreground"> not</strong> send row data, query results, or
            any actual database content to the AI.
          </P>
          <P>
            For maximum privacy:
          </P>
          <ul className="mb-4 space-y-2">
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-500/60" />
              <span><strong className="text-foreground">Use Ollama</strong> — run a local LLM
              on your machine. Nothing leaves your laptop.</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-500/60" />
              <span><strong className="text-foreground">Team Server</strong> — the server
              proxies AI requests through a shared cache on your infrastructure.</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-500/60" />
              <span><strong className="text-foreground">No telemetry</strong> — basemake has
              no phone-home, no analytics, no usage tracking.</span>
            </li>
          </ul>
          <P>
            See the{' '}
            <Link to="/docs/ai-providers" className="text-[#ff3131] hover:underline">
              AI Providers page
            </Link>{' '}
            for detailed privacy information per provider.
          </P>
        </FAQItem>

        <FAQItem question="How do I get support?">
          <P>
            basemake offers several support channels:
          </P>
          <ul className="mb-4 space-y-2">
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
              <span><strong className="text-foreground">GitHub Issues</strong> — report bugs
              and request features at{' '}
              <Link to="https://github.com/DynamicKarabo/basemake/issues" className="text-[#ff3131] hover:underline">
                github.com/DynamicKarabo/basemake
              </Link></span>
            </li>
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
              <span><strong className="text-foreground">Documentation</strong> — this site
              covers Quickstart, Commands, Configuration, and more.</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
              <span><strong className="text-foreground">Pro / Team</strong> — priority email
              support with guaranteed response times.</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
              <span><strong className="text-foreground">basemake doctor</strong> — built-in
              diagnostics that can help you troubleshoot common issues.</span>
            </li>
          </ul>
          <CodeBlock lang="bash">basemake doctor --verbose</CodeBlock>
        </FAQItem>
      </div>

      {/* Still have questions */}
      <Separator className="my-10 bg-muted/30" />

      <Card className="border-border/[0.06] bg-muted/30">
        <CardHeader>
          <CardTitle className="text-foreground text-lg">Still have questions?</CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">
          <P>
            Check out the other documentation pages or{' '}
            <Link to="https://github.com/DynamicKarabo/basemake/issues" className="text-[#ff3131] hover:underline">
              open a GitHub issue
            </Link>
            . We're actively building basemake and love hearing from users.
          </P>
        </CardContent>
      </Card>
    </div>
  )
}
