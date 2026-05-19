import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import { TabsProvider } from '@/components/ui/tabs-context'
import { TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs-barrel'
import {
  Database,
  MessageSquare,
  BarChart3,
  Terminal,
  GitBranch,
  Zap,
  GitCompare,
  ArrowRight,
  Download,
  BookOpen,
  Sparkles,
  Quote,
  ChevronRight,
} from 'lucide-react'

/* ------------------------------------------------------------------ */
/*  Animation variants                                                 */
/* ------------------------------------------------------------------ */
function easeOut(): [number, number, number, number] {
  return [0.25, 0.1, 0.25, 1]
}

const fadeUp: import('framer-motion').Variants = {
  hidden: { opacity: 0, y: 32 },
  visible: (i = 0) => ({
    opacity: 1,
    y: 0,
    transition: { duration: 0.6, delay: i * 0.12, ease: easeOut() },
  }),
}

const stagger: import('framer-motion').Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.12, delayChildren: 0.15 },
  },
}

const scaleIn: import('framer-motion').Variants = {
  hidden: { opacity: 0, scale: 0.92 },
  visible: {
    opacity: 1,
    scale: 1,
    transition: { duration: 0.5, ease: easeOut() },
  },
}

/* ------------------------------------------------------------------ */
/*  Section label helper                                               */
/* ------------------------------------------------------------------ */
function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <motion.div variants={fadeUp} className="mb-6 flex items-center gap-3">
      <span className="h-px w-8 bg-[#e63946]" />
      <span className="text-[11px] font-semibold tracking-[0.2em] text-[#e63946] uppercase">
        {children}
      </span>
    </motion.div>
  )
}

/* ------------------------------------------------------------------ */
/*  Data                                                               */
/* ------------------------------------------------------------------ */
const steps = [
  {
    number: '01',
    icon: Database,
    title: 'Connect',
    description:
      'Point basemake at any database — PostgreSQL, MySQL, SQLite, and more. Auto-detects your schema in seconds.',
  },
  {
    number: '02',
    icon: MessageSquare,
    title: 'Ask',
    description:
      'Type questions in plain English. basemake translates your intent into optimized SQL, no prompt engineering required.',
  },
  {
    number: '03',
    icon: BarChart3,
    title: 'Analyze',
    description:
      'Get performance insights alongside every query. Spot slow indexes, missing constraints, and optimization opportunities.',
  },
]

const features = [
  {
    icon: Terminal,
    title: 'Natural Language → SQL',
    description:
      'Describe what you need in plain English and get production-ready SQL. No more context-switching between docs and the terminal.',
  },
  {
    icon: Database,
    title: 'Multi-Dialect',
    description:
      'Works across PostgreSQL, MySQL, SQLite, MariaDB, and more. The same natural language query generates the right dialect every time.',
  },
  {
    icon: GitBranch,
    title: 'CI/CD Gate',
    description:
      'Integrate basemake into your pipeline. Gate deploys on SQL quality checks, schema drift detection, and migration safety.',
  },
  {
    icon: Zap,
    title: 'Index Recommendations',
    description:
      'Basemake analyzes your query patterns and suggests indexes that speed up your most frequent access paths — automatically.',
  },
  {
    icon: GitCompare,
    title: 'Schema Diffing',
    description:
      'Compare schemas across environments in seconds. Spot missing columns, type mismatches, and drift before it hits production.',
  },
  {
    icon: BarChart3,
    title: 'Query Monitoring',
    description:
      'Track query performance over time. Identify regressions, hot queries, and opportunities to tune your database. Built-in, not bolted on.',
  },
]

const testimonials = [
  {
    quote:
      'basemake turned our team into SQL pros. Non-technical stakeholders write their own queries now — our data team got hours back every week.',
    name: 'Alex Chen',
    role: 'Engineering Lead, Data Platform',
  },
  {
    quote:
      'The CI/CD gate alone saved us from three schema-related incidents in the first month. This tool pays for itself in incident response time.',
    name: 'Sarah Mitchell',
    role: 'Staff Engineer',
  },
  {
    quote:
      'I\'ve tried every "AI for SQL" tool out there. basemake is the first one that actually understands my database schema and generates correct, efficient queries.',
    name: 'Marcus Rivera',
    role: 'Principal Backend Engineer',
  },
]

const dbBadges = ['PostgreSQL', 'MySQL', 'SQLite', 'MariaDB', 'TimescaleDB', 'CockroachDB', 'ClickHouse']

const codeExamples = {
  'one-liner': `basemake "show me the top 10 users by revenue this month"`,
  'pipe-mode': `cat query.sql | basemake --format=table`,
  repl: `.connect postgres://localhost:5432/mydb
> show me orders that haven't shipped in 7 days
> add a status filter for 'pending'
> explain the query plan`,
}

/* ------------------------------------------------------------------ */
/*  Code Block component                                               */
/* ------------------------------------------------------------------ */
function CodeBlock({ children, lang = 'bash' }: { children: string; lang?: string }) {
  return (
    <div className="group relative overflow-hidden rounded-xl border border-white/[0.06] bg-black/60 backdrop-blur-sm">
      {/* Title bar */}
      <div className="flex items-center gap-2 border-b border-white/[0.06] px-4 py-2.5">
        <div className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-red-500/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-yellow-500/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-green-500/70" />
        </div>
        <span className="ml-2 text-[11px] text-white/30 font-mono">{lang}</span>
      </div>
      {/* Code */}
      <pre className="overflow-x-auto p-5 text-sm leading-relaxed">
        <code className="font-mono text-white/80 [word-spacing:0.15em]">
          {children}
        </code>
      </pre>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/*  Terminal Hero mockup                                               */
/* ------------------------------------------------------------------ */
function TerminalMockup() {
  return (
    <div className="relative mx-auto w-full max-w-lg overflow-hidden rounded-xl border border-white/[0.08] bg-black/50 shadow-2xl shadow-[#e63946]/5 backdrop-blur-sm">
      {/* Title bar */}
      <div className="flex items-center gap-2 border-b border-white/[0.06] px-4 py-3">
        <div className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-red-500/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-yellow-500/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-green-500/70" />
        </div>
        <span className="ml-2 text-xs text-white/30 font-mono">basemake — ~/project</span>
      </div>
      {/* Terminal output */}
      <div className="space-y-2 p-5 font-mono text-sm leading-relaxed">
        <p className="flex items-start gap-2">
          <span className="mt-px shrink-0 text-[#e63946]">❯</span>
          <span className="text-white/60">basemake</span>
          <span className="text-white/90">
            "show me the top 5 customers by lifetime value"
          </span>
        </p>
        <div className="ml-5 space-y-0.5 border-l-2 border-[#e63946]/30 pl-4">
          <p className="text-emerald-400/90">
            SELECT c.name, SUM(o.total) AS ltv
          </p>
          <p className="text-emerald-400/90">
            FROM customers c JOIN orders o ON c.id = o.customer_id
          </p>
          <p className="text-emerald-400/90">
            GROUP BY c.name ORDER BY ltv DESC LIMIT 5;
          </p>
        </div>
        <div className="ml-5 mt-1 flex items-center gap-2 text-xs">
          <span className="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-0.5 text-emerald-400">
            ✓ 8 rows
          </span>
          <span className="text-white/30">in 12ms</span>
        </div>
        <div className="mt-3 flex items-start gap-2">
          <span className="mt-px shrink-0 text-[#e63946]">❯</span>
          <span className="inline-flex h-5 w-2 animate-pulse rounded-full bg-white/40" />
        </div>
      </div>
    </div>
  )
}

/* ================================================================== */
/*  PAGE — Landing                                                     */
/* ================================================================== */
export default function Landing() {
  return (
    <div className="overflow-hidden">
      {/* ============================================================ */}
      {/*  1. HERO                                                      */}
      {/* ============================================================ */}
      <section className="relative isolate overflow-hidden">
        {/* Background grid pattern */}
        <div
          className="pointer-events-none absolute inset-0 -z-10"
          style={{
            backgroundImage: [
              'linear-gradient(rgba(255,255,255,0.025) 1px, transparent 1px)',
              'linear-gradient(90deg, rgba(255,255,255,0.025) 1px, transparent 1px)',
            ].join(', '),
            backgroundSize: '56px 56px',
          }}
        />
        {/* Glow overlay */}
        <div className="pointer-events-none absolute -top-40 left-1/2 -z-10 h-[600px] w-[800px] -translate-x-1/2 rounded-full bg-[#e63946]/5 blur-[120px]" />
        <div className="pointer-events-none absolute -bottom-40 right-0 -z-10 h-[400px] w-[600px] rounded-full bg-[#e63946]/3 blur-[100px]" />

        <div className="mx-auto max-w-7xl px-6 pb-24 pt-20 md:pt-28 lg:pb-32">
          <motion.div
            initial="hidden"
            animate="visible"
            variants={stagger}
            className="grid items-center gap-16 lg:grid-cols-2"
          >
            {/* Left column — text */}
            <div className="max-w-xl">
              <SectionLabel>All Local. All Private.</SectionLabel>

              <motion.h1
                variants={fadeUp}
                className="mt-2 text-4xl font-bold leading-tight tracking-tight text-white sm:text-5xl lg:text-6xl"
              >
                Talk to your database{' '}
                <span className="text-[#e63946]">in plain English.</span>
              </motion.h1>

              <motion.p
                variants={fadeUp}
                className="mt-6 text-base leading-relaxed text-white/50 sm:text-lg"
              >
                basemake is a local-first CLI tool that converts natural language into
                optimized SQL. No data leaves your machine. No API keys required. Just
                your database and your questions.
              </motion.p>

              <motion.div
                variants={fadeUp}
                className="mt-8 flex flex-wrap items-center gap-4"
              >
                <Button
                  size="lg"
                  className="bg-[#e63946] text-white shadow-lg shadow-[#e63946]/25 hover:bg-[#e63946]/90 hover:shadow-[#e63946]/35"
                >
                  <Download className="mr-1.5 h-4 w-4" />
                  Download Now
                </Button>
                <Link to="/docs/quickstart">
                  <Button
                    size="lg"
                    variant="outline"
                    className="border-white/10 text-white/70 hover:bg-white/5 hover:text-white"
                  >
                    <BookOpen className="mr-1.5 h-4 w-4" />
                    View Docs
                  </Button>
                </Link>
              </motion.div>

              <motion.div
                variants={fadeUp}
                className="mt-6 flex items-center gap-4 text-xs text-white/30"
              >
                <span className="flex items-center gap-1.5">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                  Ships in 15 MB
                </span>
                <span className="h-3 w-px bg-white/10" />
                <span className="flex items-center gap-1.5">
                  <Sparkles className="h-3 w-3 text-amber-400/60" />
                  No API key needed
                </span>
                <span className="h-3 w-px bg-white/10" />
                <span className="flex items-center gap-1.5">
                  <Database className="h-3 w-3 text-white/40" />
                  5+ dialects
                </span>
              </motion.div>
            </div>

            {/* Right column — terminal mockup */}
            <motion.div
              variants={fadeUp}
              className="hidden lg:block"
            >
              <TerminalMockup />
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* ============================================================ */}
      {/*  2. LOGO CLOUD                                                */}
      {/* ============================================================ */}
      <section className="border-y border-white/[0.04] bg-white/[0.01] py-12">
        <div className="mx-auto max-w-7xl px-6">
          <motion.p
            initial={{ opacity: 0, y: 8 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="mb-8 text-center text-[11px] font-semibold tracking-[0.2em] text-white/30 uppercase"
          >
            Works with your database
          </motion.p>

          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="flex flex-wrap items-center justify-center gap-x-10 gap-y-4"
          >
            {dbBadges.map((name) => (
              <motion.span
                key={name}
                variants={scaleIn}
                className="select-none rounded-full border border-white/[0.06] bg-white/[0.03] px-5 py-2 text-sm font-medium text-white/50 transition-colors hover:border-[#e63946]/30 hover:text-white/80"
              >
                {name}
              </motion.span>
            ))}
          </motion.div>
        </div>
      </section>

      {/* ============================================================ */}
      {/*  3. HOW IT WORKS                                               */}
      {/* ============================================================ */}
      <section className="py-24 md:py-32">
        <div className="mx-auto max-w-7xl px-6">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="mb-16 text-center"
          >
            <SectionLabel>How It Works</SectionLabel>
            <motion.h2
              variants={fadeUp}
              className="text-3xl font-bold tracking-tight text-white sm:text-4xl"
            >
              From question to query in three steps
            </motion.h2>
            <motion.p
              variants={fadeUp}
              className="mx-auto mt-4 max-w-2xl text-white/40"
            >
              No complex setup. No training data. Just connect, ask, and get answers.
            </motion.p>
          </motion.div>

          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="grid gap-8 md:grid-cols-3"
          >
            {steps.map((step, i) => {
              const Icon = step.icon
              return (
                <motion.div
                  key={step.number}
                  variants={fadeUp}
                  custom={i}
                  className="group relative rounded-2xl border border-white/[0.06] bg-white/[0.02] p-8 transition-colors hover:border-[#e63946]/20"
                >
                  {/* Number backdrop */}
                  <span className="pointer-events-none absolute -right-4 -top-4 select-none text-[100px] font-black leading-none text-white/[0.02]">
                    {step.number}
                  </span>

                  <div className="relative">
                    <div className="mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-[#e63946]/10 text-[#e63946] ring-1 ring-[#e63946]/20">
                      <Icon className="h-6 w-6" />
                    </div>
                    <h3 className="mb-2 text-lg font-semibold text-white">
                      {step.title}
                    </h3>
                    <p className="text-sm leading-relaxed text-white/40">
                      {step.description}
                    </p>
                  </div>

                  {/* Connector line (desktop) */}
                  {i < steps.length - 1 && (
                    <div className="absolute -right-4 top-1/3 hidden -translate-y-1/2 md:block">
                      <ChevronRight className="h-5 w-5 text-white/15" />
                    </div>
                  )}
                </motion.div>
              )
            })}
          </motion.div>
        </div>
      </section>

      <Separator className="mx-auto max-w-7xl bg-white/[0.04]" />

      {/* ============================================================ */}
      {/*  4. FEATURES GRID                                              */}
      {/* ============================================================ */}
      <section className="py-24 md:py-32" id="features">
        <div className="mx-auto max-w-7xl px-6">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="mb-16 text-center"
          >
            <SectionLabel>Everything you need</SectionLabel>
            <motion.h2
              variants={fadeUp}
              className="text-3xl font-bold tracking-tight text-white sm:text-4xl"
            >
              Built for the way you work
            </motion.h2>
            <motion.p
              variants={fadeUp}
              className="mx-auto mt-4 max-w-2xl text-white/40"
            >
              From ad-hoc queries to production pipelines — basemake fits into every
              workflow.
            </motion.p>
          </motion.div>

          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3"
          >
            {features.map((feature) => {
              const Icon = feature.icon
              return (
                <motion.div key={feature.title} variants={scaleIn}>
                  <Card className="h-full border-white/[0.06] bg-white/[0.02] transition-colors hover:border-[#e63946]/15">
                    <CardContent className="p-6">
                      <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-[#e63946]/10 text-[#e63946] ring-1 ring-[#e63946]/20">
                        <Icon className="h-5 w-5" />
                      </div>
                      <h3 className="mb-2 text-base font-semibold text-white">
                        {feature.title}
                      </h3>
                      <p className="text-sm leading-relaxed text-white/40">
                        {feature.description}
                      </p>
                    </CardContent>
                  </Card>
                </motion.div>
              )
            })}
          </motion.div>
        </div>
      </section>

      <Separator className="mx-auto max-w-7xl bg-white/[0.04]" />

      {/* ============================================================ */}
      {/*  5. CODE SNIPPETS — TABS                                       */}
      {/* ============================================================ */}
      <section className="py-24 md:py-32">
        <div className="mx-auto max-w-4xl px-6">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="mb-14 text-center"
          >
            <SectionLabel>Three ways to use</SectionLabel>
            <motion.h2
              variants={fadeUp}
              className="text-3xl font-bold tracking-tight text-white sm:text-4xl"
            >
              Your workflow, your way
            </motion.h2>
            <motion.p
              variants={fadeUp}
              className="mx-auto mt-4 max-w-xl text-white/40"
            >
              Drop in anywhere in your pipeline — one-liner, pipe, or interactive REPL.
            </motion.p>
          </motion.div>

          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={fadeUp}
          >
            <TabsProvider>
              <div className="flex justify-center">
                <TabsList className="mb-8 inline-flex gap-1 rounded-xl border border-white/[0.06] bg-white/[0.02] p-1.5">
                  <TabsTrigger
                    value="one-liner"
                    className="rounded-lg px-5 py-2 text-sm font-medium text-white/40 transition-all data-[state=active]:bg-[#e63946] data-[state=active]:text-white data-[state=active]:shadow-sm"
                  >
                    One-liner
                  </TabsTrigger>
                  <TabsTrigger
                    value="pipe-mode"
                    className="rounded-lg px-5 py-2 text-sm font-medium text-white/40 transition-all data-[state=active]:bg-[#e63946] data-[state=active]:text-white data-[state=active]:shadow-sm"
                  >
                    Pipe mode
                  </TabsTrigger>
                  <TabsTrigger
                    value="repl"
                    className="rounded-lg px-5 py-2 text-sm font-medium text-white/40 transition-all data-[state=active]:bg-[#e63946] data-[state=active]:text-white data-[state=active]:shadow-sm"
                  >
                    REPL
                  </TabsTrigger>
                </TabsList>
              </div>

              <TabsContent value="one-liner">
                <CodeBlock lang="bash">{codeExamples['one-liner']}</CodeBlock>
              </TabsContent>
              <TabsContent value="pipe-mode">
                <CodeBlock lang="bash">{codeExamples['pipe-mode']}</CodeBlock>
              </TabsContent>
              <TabsContent value="repl">
                <CodeBlock lang="sql">{codeExamples.repl}</CodeBlock>
              </TabsContent>
            </TabsProvider>
          </motion.div>
        </div>
      </section>

      {/* ============================================================ */}
      {/*  6. TESTIMONIALS                                               */}
      {/* ============================================================ */}
      <section className="border-t border-white/[0.04] bg-white/[0.01] py-24 md:py-32">
        <div className="mx-auto max-w-7xl px-6">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="mb-16 text-center"
          >
            <SectionLabel>Trusted by engineers</SectionLabel>
            <motion.h2
              variants={fadeUp}
              className="text-3xl font-bold tracking-tight text-white sm:text-4xl"
            >
              What your peers are saying
            </motion.h2>
            <motion.p
              variants={fadeUp}
              className="mx-auto mt-4 max-w-xl text-white/40"
            >
              Real feedback from real developers who ship with basemake.
            </motion.p>
          </motion.div>

          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="grid gap-6 md:grid-cols-3"
          >
            {testimonials.map((t) => (
              <motion.div key={t.name} variants={scaleIn}>
                <Card className="h-full border-white/[0.06] bg-white/[0.02] transition-colors hover:border-[#e63946]/15">
                  <CardContent className="flex flex-col gap-4 p-6">
                    <Quote className="h-6 w-6 text-[#e63946]/40" />
                    <p className="text-sm leading-relaxed text-white/60">
                      &ldquo;{t.quote}&rdquo;
                    </p>
                    <div className="mt-auto flex items-center gap-3 pt-2">
                      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-[#e63946]/15 text-xs font-bold text-[#e63946]">
                        {t.name
                          .split(' ')
                          .map((n) => n[0])
                          .join('')}
                      </div>
                      <div>
                        <p className="text-sm font-medium text-white">{t.name}</p>
                        <p className="text-xs text-white/30">{t.role}</p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </motion.div>
        </div>
      </section>

      {/* ============================================================ */}
      {/*  7. CTA BANNER                                                 */}
      {/* ============================================================ */}
      <section className="relative isolate overflow-hidden py-24 md:py-32">
        {/* Background effects */}
        <div
          className="pointer-events-none absolute inset-0 -z-10"
          style={{
            backgroundImage: [
              'linear-gradient(rgba(230,57,70,0.03) 1px, transparent 1px)',
              'linear-gradient(90deg, rgba(230,57,70,0.03) 1px, transparent 1px)',
            ].join(', '),
            backgroundSize: '40px 40px',
          }}
        />
        <div className="pointer-events-none absolute left-1/2 top-1/2 -z-10 h-[500px] w-[500px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-[#e63946]/8 blur-[120px]" />

        <motion.div
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true }}
          variants={stagger}
          className="mx-auto max-w-3xl px-6 text-center"
        >
          <motion.h2
            variants={fadeUp}
            className="text-3xl font-bold tracking-tight text-white sm:text-5xl"
          >
            Ready to ship faster?
          </motion.h2>
          <motion.p
            variants={fadeUp}
            className="mt-4 text-lg text-white/50"
          >
            Download now. Ships in <strong className="text-white/80">15 MB</strong>.
            No registration. No API keys. No data leaves your machine.
          </motion.p>

          <motion.div
            variants={fadeUp}
            className="mt-10 flex flex-wrap items-center justify-center gap-4"
          >
            <Button
              size="lg"
              className="bg-[#e63946] text-white shadow-lg shadow-[#e63946]/25 hover:bg-[#e63946]/90 hover:shadow-[#e63946]/35"
            >
              <Download className="mr-1.5 h-4 w-4" />
              Download basemake
            </Button>
            <Link to="/pricing">
              <Button
                size="lg"
                variant="outline"
                className="border-white/10 text-white/70 hover:bg-white/5 hover:text-white"
              >
                See Pricing
                <ArrowRight className="ml-1.5 h-4 w-4" />
              </Button>
            </Link>
          </motion.div>

          <motion.p
            variants={fadeUp}
            className="mt-6 text-xs text-white/20"
          >
            macOS · Linux · Windows &nbsp;·&nbsp; Homebrew &amp; direct download
          </motion.p>
        </motion.div>
      </section>
    </div>
  )
}
