import { Button } from '@/components/ui/button'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import {
  BrainIcon, BoltIcon, DatabaseIcon, ShieldIcon,
  CompareIcon, EyeIcon,
  TerminalIcon, PromptIcon,
} from '@/components/icons'

const fadeUp = {
  hidden: { opacity: 0, y: 24 },
  visible: (i: number = 0) => ({
    opacity: 1, y: 0,
    transition: { duration: 0.5, delay: i * 0.1, ease: [0.25, 0.1, 0.25, 1] as const },
  }),
}

const stagger = {
  visible: { transition: { staggerChildren: 0.08 } },
}

function Label({ children }: { children: string }) {
  return (
    <div className="flex items-center gap-3 mb-5">
      <span className="h-px w-6 bg-[#FC0E22]" />
      <span className="text-[11px] font-semibold tracking-[0.18em] uppercase text-[#FC0E22]">{children}</span>
    </div>
  )
}

function FeatureCard({
  icon: Icon, tag, title, desc,
}: {
  icon: React.ElementType; tag: string; title: string; desc: string
}) {
  return (
    <motion.div variants={fadeUp} className="group relative">
      <div className="relative h-full rounded-2xl border border-border/60 bg-white p-7 transition-all duration-300 hover:border-[#FC0E22]/20 hover:shadow-lg hover:shadow-[#FC0E22]/5 hover:-translate-y-0.5">
        <span className="inline-flex items-center rounded-full bg-muted px-3 py-1 text-[11px] font-semibold tracking-wide text-muted-foreground uppercase mb-5">
          {tag}
        </span>
        <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-[#FC0E22]/8 text-[#FC0E22] ring-1 ring-[#FC0E22]/15 group-hover:bg-[#FC0E22]/12 group-hover:ring-[#FC0E22]/25 transition-all">
          <Icon className="h-5 w-5" />
        </div>
        <h3 className="text-base font-semibold mb-2 text-foreground">{title}</h3>
        <p className="text-sm text-muted-foreground leading-relaxed">{desc}</p>
      </div>
    </motion.div>
  )
}

const features = [
  {
    icon: BrainIcon, BoltIcon, tag: "QUERY",
    title: "Natural Language → SQL",
    desc: "Describe what you need in plain English. basemake generates production-ready SQL — with the right dialect, joins, and filtering — every time.",
  },
  {
    icon: DatabaseIcon, tag: "DIALECTS",
    title: "Multi-Dialect",
    desc: "PostgreSQL, MySQL, SQLite. Plus MariaDB and TimescaleDB via wire-compatible drivers. One tool, same interface.",
  },
  {
    icon: ShieldIcon, tag: "CI/CD",
    title: "Pipeline Gate",
    desc: "`basemake check` exits 0, 1, or 2. Plug it into your pipeline. Block slow queries, detect dangerous patterns, enforce budgets.",
  },
  {
    icon: BoltIcon, tag: "PERFORMANCE",
    title: "Index Recommendations",
    desc: "Not just \"add an index.\" It tells you which columns, in what order, and why. Apply with one command or review the diff first.",
  },
  {
    icon: CompareIcon, tag: "SCHEMA",
    title: "Schema Diffing",
    desc: "Compare dev, staging, and prod in seconds. Catch drift before it becomes an incident. Works offline, no server needed.",
  },
  {
    icon: EyeIcon, tag: "OBSERVABILITY",
    title: "Query Monitoring",
    desc: "Schedule recurring checks. Get alerted when a query slows down. Track regressions over time — built in, no Datadog bill.",
  },
]

const howItWorks = [
  {
    step: "01", icon: TerminalIcon,
    title: "Connect",
    desc: "Point basemake at your database. It introspects the schema — tables, columns, indexes, types — and you're ready.",
    cmd: "$ basemake connect postgres://localhost/mydb",
  },
  {
    step: "02", icon: PromptIcon,
    title: "Ask",
    desc: "Type your question in plain English. basemake generates SQL, runs it, and shows you results — all in your terminal.",
    cmd: "$ basemake \"top 10 customers by revenue\"",
  },
  {
    step: "03", icon: BoltIcon,
    title: "Act",
    desc: "Apply indexes, write policies, gate your deploys. basemake works in CI, on your laptop, or in a Docker container.",
    cmd: "$ basemake check \"SELECT * FROM orders\" --threshold 200ms",
  },
]

export default function Features() {
  return (
    <div className="overflow-hidden">
      {/* Hero */}
      <section className="relative isolate pt-20 pb-16 md:pb-24">
        <div className="pointer-events-none absolute -top-40 left-1/2 -z-10 h-[600px] w-[800px] -translate-x-1/2 rounded-full bg-[#FC0E22]/3 blur-[120px]" />
        <div className="mx-auto max-w-7xl px-6 text-center">
          <motion.div initial="hidden" animate="visible" variants={stagger}>
            <motion.div variants={fadeUp}>
              <Label>Everything you need</Label>
              <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight leading-[1.05] mb-6 text-foreground">
                Full-stack database{' '}
                <span className="text-transparent bg-clip-text bg-gradient-to-r from-[#FC0E22] to-[#FC0E22]/70">
                  CLI.
                </span>
              </h1>
              <p className="text-lg text-muted-foreground max-w-xl mx-auto mb-10 leading-relaxed">
                From ad-hoc queries to production pipelines. One binary, no dependencies.
                basemake replaces 3-4 separate tools with a single 15 MB binary.
              </p>
              <div className="flex flex-wrap justify-center gap-4">
                <Link to="/docs/quickstart">
                  <Button size="lg" className="rounded-full bg-foreground text-primary-foreground hover:bg-foreground/90 shadow-lg shadow-foreground/10 px-8 h-12 text-base font-semibold">
                    Get Started
                  </Button>
                </Link>
                <Link to="/pricing">
                  <Button size="lg" variant="outline" className="rounded-full border-border text-muted-foreground hover:bg-muted hover:text-foreground px-8 h-12 text-base font-semibold">
                    See Pricing
                  </Button>
                </Link>
              </div>
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Feature grid */}
      <section className="border-t border-border/50 py-24 md:py-32">
        <div className="mx-auto max-w-6xl px-6">
          <motion.div
            initial="hidden" whileInView="visible"
            viewport={{ once: true, margin: "-100px" }}
            variants={stagger}
            className="grid md:grid-cols-2 lg:grid-cols-3 gap-5"
          >
            {features.map((f, i) => (
              <FeatureCard key={i} icon={f.icon} tag={f.tag} title={f.title} desc={f.desc} />
            ))}
          </motion.div>
        </div>
      </section>

      {/* How it works */}
      <section className="border-t border-border/50 bg-muted/30 py-24 md:py-32">
        <div className="mx-auto max-w-6xl px-6">
          <motion.div
            initial="hidden" whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="mb-16 text-center"
          >
            <motion.div variants={fadeUp}>
              <Label>How it works</Label>
              <h2 className="text-3xl sm:text-4xl font-bold tracking-tight text-foreground mb-4">
                Connect, ask, act.
              </h2>
              <p className="text-muted-foreground max-w-lg text-lg mx-auto">
                No setup wizard. No training data. Three commands and you're productive.
              </p>
            </motion.div>
          </motion.div>

          <div className="grid md:grid-cols-3 gap-8">
            {howItWorks.map((item, i) => {
              const Icon = item.icon
              return (
                <motion.div
                  key={i}
                  initial={{ opacity: 0, y: 30 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  viewport={{ once: true }}
                  transition={{ duration: 0.5, delay: i * 0.15 }}
                >
                  <div className="relative h-full rounded-2xl border border-border/60 bg-white p-8 hover:shadow-lg hover:shadow-[#FC0E22]/5 transition-all duration-300">
                    <span className="absolute -top-3 -right-3 text-5xl font-bold text-muted-foreground/5 select-none">{item.step}</span>
                    <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-[#FC0E22]/8 text-[#FC0E22] ring-1 ring-[#FC0E22]/15">
                      <Icon className="h-5 w-5" />
                    </div>
                    <h3 className="text-lg font-semibold text-foreground mb-2">{item.title}</h3>
                    <p className="text-sm text-muted-foreground leading-relaxed mb-5">{item.desc}</p>
                    <div className="rounded-xl bg-muted/50 border border-border/50 px-4 py-3">
                      <code className="text-xs text-muted-foreground/70 font-mono">{item.cmd}</code>
                    </div>
                  </div>
                </motion.div>
              )
            })}
          </div>
        </div>
      </section>

      {/* Stats */}
      <section className="border-t border-border/50 py-20">
        <div className="mx-auto max-w-6xl px-6">
          <motion.div
            initial="hidden" whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="text-center"
          >
            <motion.div variants={fadeUp}><Label>Built different</Label></motion.div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mt-8">
              {[
                { value: "15 MB", label: "single binary" },
                { value: "0", label: "dependencies" },
                { value: "3", label: "commands to production" },
                { value: "5+", label: "database dialects" },
              ].map((stat, i) => (
                <motion.div key={i} variants={fadeUp} custom={i + 1} className="flex flex-col items-center">
                  <span className="text-4xl font-bold tracking-tight text-foreground">{stat.value}</span>
                  <span className="text-sm text-muted-foreground mt-1">{stat.label}</span>
                </motion.div>
              ))}
            </div>
          </motion.div>
        </div>
      </section>

      {/* CTA */}
      <section className="border-t border-border/50 bg-muted/30 py-24">
        <div className="mx-auto max-w-6xl px-6 text-center">
          <motion.div initial="hidden" whileInView="visible" viewport={{ once: true }} variants={stagger}>
            <motion.div variants={fadeUp}>
              <h2 className="text-4xl sm:text-5xl font-bold tracking-tight text-foreground mb-4">
                Ready to ship faster?
              </h2>
              <p className="text-muted-foreground text-lg mb-10 max-w-md mx-auto">
                Download now. 15 MB binary, zero dependencies, one command to start.
              </p>
            </motion.div>
            <motion.div variants={fadeUp} custom={1} className="flex flex-wrap justify-center gap-4">
              <Link to="/docs/quickstart">
                <Button size="lg" className="rounded-full bg-foreground text-primary-foreground hover:bg-foreground/90 shadow-lg shadow-foreground/10 px-8 h-12 text-base font-semibold">
                  Get Started
                </Button>
              </Link>
              <Link to="/pricing">
                <Button size="lg" variant="outline" className="rounded-full border-border text-muted-foreground hover:bg-muted hover:text-foreground px-8 h-12 text-base font-semibold">
                  See Pricing
                </Button>
              </Link>
            </motion.div>
          </motion.div>
        </div>
      </section>
    </div>
  )
}
