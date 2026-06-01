import { Button } from '@/components/ui/button'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'

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
    <div className="flex items-center gap-3 mb-6">
      <span className="h-px w-8 bg-[#ff3131]" />
      <span className="text-xs font-semibold tracking-[0.15em] uppercase text-[#ff3131]">{children}</span>
    </div>
  )
}

export default function Landing() {
  return (
    <div className="overflow-hidden">
      {/* ─── HERO ─── */}
      <section className="relative min-h-[90vh] flex items-center border-b border-white/[0.06]">
        {/* bg pattern */}
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,rgba(230,57,70,0.08)_0%,transparent_60%)]" />
        <div
          className="absolute inset-0 opacity-[0.015]"
          style={{
            backgroundImage: `radial-gradient(circle at 1px 1px, white 1px, transparent 0)`,
            backgroundSize: '40px 40px',
          }}
        />

        <div className="relative mx-auto max-w-6xl px-6 py-24 w-full">
          <motion.div initial="hidden" animate="visible" className="max-w-3xl">
            <motion.div variants={fadeUp} custom={0}>
              <span className="inline-flex items-center gap-2 rounded-full border border-[#ff3131]/30 bg-[#ff3131]/5 px-4 py-1.5 text-xs font-medium text-[#ff3131] mb-8">
                <span className="h-1.5 w-1.5 rounded-full bg-[#ff3131] animate-pulse" />
                v0.7.1 — just shipped
              </span>
            </motion.div>

            <motion.h1
              variants={fadeUp}
              custom={1}
              className="text-5xl sm:text-6xl lg:text-7xl font-bold tracking-tight leading-[1.05] mb-6"
            >
              Talk to your database{" "}
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-[#ff3131] to-[#ff6b6b]">
                in plain English
              </span>
              .
            </motion.h1>

            <motion.p
              variants={fadeUp}
              custom={2}
              className="text-lg sm:text-xl text-white/50 max-w-xl mb-10 leading-relaxed"
            >
              An AI-native CLI for PostgreSQL, MySQL, and SQLite. Queries,
              performance analysis, and schema insights — no data leaves your machine.
            </motion.p>

            <motion.div variants={fadeUp} custom={3} className="flex flex-wrap gap-4 mb-12">
              <Link to="/docs/quickstart">
                <Button size="lg" variant="destructive" className="bg-[#ff3131] hover:bg-[#d90c18] text-white shadow-lg shadow-[#ff3131]/20">
                  Get Started
                </Button>
              </Link>
              <Link to="/docs/quickstart">
                <Button size="lg" variant="outline" className="border-white/20 text-white/80 hover:bg-white/10 hover:text-white">View Docs</Button>
              </Link>
            </motion.div>

            <motion.div variants={fadeUp} custom={4} className="flex flex-wrap gap-x-8 gap-y-2 text-sm text-white/30">
              <span className="flex items-center gap-2">
                <span className="h-1 w-1 rounded-full bg-white/20" />
                Ships in 15 MB
              </span>
              <span className="flex items-center gap-2">
                <span className="h-1 w-1 rounded-full bg-white/20" />
                No data leaves your machine
              </span>
              <span className="flex items-center gap-2">
                <span className="h-1 w-1 rounded-full bg-white/20" />
                BYOK — use your own AI key
              </span>
            </motion.div>
          </motion.div>

          {/* Demo GIF */}
          <motion.div
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.8, delay: 0.5 }}
            className="absolute -right-20 top-1/2 -translate-y-1/2 hidden xl:block"
          >
            <img
              src="/basemake-demo.gif"
              alt="basemake demo — natural language query and analysis"
              className="w-[480px] rounded-xl border border-white/[0.08] shadow-2xl shadow-[#ff3131]/5"
            />
          </motion.div>
        </div>
      </section>
      {/* ─── LOGO CLOUD ─── */}
      <section className="border-b border-white/[0.06] py-16">
        <div className="mx-auto max-w-6xl px-6 text-center">
          <Label>Works with your database</Label>
          <div className="flex flex-wrap justify-center gap-x-12 gap-y-4 text-sm text-white/30">
            {["PostgreSQL", "MySQL", "SQLite", "MariaDB", "TimescaleDB", "CockroachDB", "ClickHouse"].map((db) => {
              const qualifier = db === "MariaDB" || db === "TimescaleDB" || db === "CockroachDB" ? " (beta)" : db === "ClickHouse" ? " (coming)" : "";
              return (
                <span key={db} className="font-mono tracking-tight hover:text-white/60 transition-colors cursor-default">
                  {db}<span className="text-white/15 text-[10px]">{qualifier}</span>
                </span>
              );
            })}
          </div>
        </div>
      </section>

      {/* ─── FEATURES ─── */}
      <section className="py-24 border-b border-white/[0.06]" id="features">
        <div className="mx-auto max-w-6xl px-6">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: "-100px" }}
            variants={stagger}
            className="mb-16"
          >
            <motion.div variants={fadeUp}>
              <Label>Everything you need</Label>
              <h2 className="text-3xl sm:text-4xl font-bold tracking-tight mb-4">
                Built for the way you work
              </h2>
              <p className="text-white/40 max-w-lg text-lg">
                From ad-hoc queries to production pipelines. One binary, no dependencies.
              </p>
            </motion.div>
          </motion.div>

          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: "-100px" }}
            variants={stagger}
            className="grid md:grid-cols-2 lg:grid-cols-3 gap-6"
          >
            {features.map((f, i) => {
              const Icon = f.icon
              return (
                <motion.div key={i} variants={fadeUp} custom={i}>
                  <TiltCard tiltDegree={5} glare={false}>
                    <div className="group relative rounded-2xl border border-white/[0.06] bg-white/[0.02] p-7 h-full transition-all duration-300 hover:border-[#ff3131]/20 hover:bg-white/[0.04]">
                      <div className="mb-5 flex h-10 w-10 items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.03] text-[#ff3131] group-hover:border-[#ff3131]/20 group-hover:bg-[#ff3131]/5 transition-colors">
                        <Icon className="h-5 w-5" />
                      </div>
                      <h3 className="text-base font-semibold mb-2">{f.title}</h3>
                      <p className="text-sm text-white/40 leading-relaxed">{f.desc}</p>
                    </div>
                  </TiltCard>
                </motion.div>
              )
            })}
          </motion.div>
        </div>
      </section>

      {/* ─── HOW IT WORKS ─── */}
      <section className="py-24 border-b border-white/[0.06]">
        <div className="mx-auto max-w-6xl px-6">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="mb-16"
          >
            <motion.div variants={fadeUp}>
              <Label>How it works</Label>
              <h2 className="text-3xl sm:text-4xl font-bold tracking-tight mb-4">
                Connect, ask, act.
              </h2>
              <p className="text-white/40 max-w-lg text-lg">
                No setup wizard. No training data. Three commands and you're productive.
              </p>
            </motion.div>
          </motion.div>

          <div className="grid md:grid-cols-3 gap-12">
            {[
              {
                step: "01",
                icon: TerminalIcon,
                title: "Connect",
                desc: "Point basemake at your database. It introspects the schema — tables, columns, indexes, types — and you're ready.",
                cmd: "$ basemake connect postgres://localhost/mydb",
              },
              {
                step: "02",
                icon: PromptIcon,
                title: "Ask",
                desc: "Type your question in plain English. basemake generates SQL, runs it, and shows you results — all in your terminal.",
                cmd: "$ basemake \"top 10 customers by revenue\"",
              },
              {
                step: "03",
                icon: BoltIcon,
                title: "Act",
                desc: "Apply indexes, write policies, gate your deploys. basemake works in CI, on your laptop, or in a Docker container.",
                cmd: "$ basemake check \"SELECT * FROM orders\" --threshold 200ms",
              },
            ].map((item, i) => {
              const Icon = item.icon
              return (
                <motion.div
                  key={i}
                  initial={{ opacity: 0, y: 30 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  viewport={{ once: true }}
                  transition={{ duration: 0.5, delay: i * 0.15 }}
                >
                  <TiltCard tiltDegree={4} glare={false}>
                    <div className="relative h-full rounded-2xl border border-white/[0.06] bg-white/[0.02] p-8">
                      <span className="absolute -top-3 -right-3 text-5xl font-bold text-white/[0.03] select-none">
                        {item.step}
                      </span>
                      <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.03] text-[#ff3131]">
                        <Icon className="h-5 w-5" />
                      </div>
                      <h3 className="text-lg font-semibold mb-2">{item.title}</h3>
                      <p className="text-sm text-white/40 leading-relaxed mb-5">{item.desc}</p>
                      <div className="rounded-lg bg-black/40 border border-white/[0.06] px-4 py-3">
                        <code className="text-xs text-white/30 font-mono">{item.cmd}</code>
                      </div>
                    </div>
                  </TiltCard>
                </motion.div>
              )
            })}
          </div>
        </div>
      </section>

      {/* ─── DESIGN ─── */}
      <section className="py-24 border-b border-white/[0.06]">
        <div className="mx-auto max-w-6xl px-6">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
          >
            <motion.div variants={fadeUp}>
              <h2 className="text-3xl sm:text-4xl font-bold tracking-tight mb-4">
                Design
              </h2>
              <p className="text-white/40 max-w-xl text-lg">
                Written in Go. Single binary, zero runtime dependencies. Interface-based drivers with a registry pattern. PlanCheck runs before every NL query with zero side effects. Observe module surfaces one signal per startup — or stays silent.
              </p>
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* ─── DEMOS — SEE IT IN ACTION ─── */}
      <section className="py-24 border-b border-white/[0.06]">
        <div className="mx-auto max-w-6xl px-6">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="mb-12"
          >
            <motion.div variants={fadeUp}>
              <Label>See it in action</Label>
              <h2 className="text-3xl sm:text-4xl font-bold tracking-tight mb-4">
                Your database, your terminal
              </h2>
              <p className="text-white/40 max-w-lg text-lg">
                No setup wizard. No GUI. Just your terminal, a database, and plain English.
              </p>
            </motion.div>
          </motion.div>

          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="grid md:grid-cols-3 gap-6"
          >
            {[
              {
                title: "Analyze",
                desc: "EXPLAIN ANALYZE for any query. Spot seq scans, missing indexes, and row estimate mismatches instantly.",
                gif: "/demos/demo-analyze.gif",
              },
              {
                title: "CI/CD Gate",
                desc: "basemake check exits 0/1/2. Plug it into your pipeline. Block slow queries before they hit production.",
                gif: "/demos/demo-check.gif",
              },
              {
                title: "Migration Safety",
                desc: "Check every migration against your profile history. Know which queries will break before you run the change.",
                gif: "/demos/demo-migrate.gif",
              },
              {
                title: "Query Profile",
                desc: "Built-in query history with plan change detection. See when a query gets slower and why.",
                gif: "/demos/demo-profile.gif",
              },
            ].map((demo, i) => (
              <motion.div key={i} variants={fadeUp} custom={i}>
                <div className="group rounded-2xl border border-white/[0.06] bg-white/[0.02] overflow-hidden transition-all duration-300 hover:border-[#ff3131]/20 hover:bg-white/[0.04]">
                  {/* Terminal window dots */}
                  <div className="flex items-center gap-1.5 px-5 pt-4 pb-2 border-b border-white/[0.04]">
                    {["#ff5f56", "#ffbd2e", "#28c840"].map((color) => (
                      <span key={color} className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: color }} />
                    ))}
                    <span className="ml-2 text-xs text-white/20 font-mono">{demo.title}</span>
                  </div>
                  {/* GIF */}
                  <div className="bg-black/20">
                    <img
                      src={demo.gif}
                      alt={demo.title}
                      className="w-full h-auto block"
                      loading="lazy"
                    />
                  </div>
                  {/* Description */}
                  <div className="px-5 py-4">
                    <p className="text-sm text-white/40 leading-relaxed">{demo.desc}</p>
                  </div>
                </div>
              </motion.div>
            ))}
          </motion.div>
        </div>
      </section>

      {/* ─── STATS ─── */}
      <section className="py-24 border-b border-white/[0.06]">
        <div className="mx-auto max-w-6xl px-6">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="text-center"
          >
            <motion.div variants={fadeUp}>
              <Label>Built different</Label>
            </motion.div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mt-8">
              {[
                { value: "15 MB", label: "single binary" },
                { value: "0", label: "dependencies" },
                { value: "3", label: "commands to production" },
                { value: "5+", label: "database dialects" },
              ].map((stat, i) => (
                <motion.div
                  key={i}
                  variants={fadeUp}
                  custom={i + 1}
                  className="flex flex-col items-center"
                >
                  <span className="text-4xl font-bold tracking-tight text-white">{stat.value}</span>
                  <span className="text-sm text-white/30 mt-1">{stat.label}</span>
                </motion.div>
              ))}
            </div>
          </motion.div>
        </div>
      </section>

      {/* ─── CTA ─── */}
      <section className="py-24">
        <div className="mx-auto max-w-6xl px-6 text-center">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
          >
            <motion.div variants={fadeUp}>
              <h2 className="text-4xl sm:text-5xl font-bold tracking-tight mb-4">
                Ready to ship faster?
              </h2>
              <p className="text-white/40 text-lg mb-10 max-w-md mx-auto">
                Download now. 15 MB binary, zero dependencies, one command to start.
              </p>
            </motion.div>
              <motion.p variants={fadeUp} custom={1} className="mt-6 text-xs text-white/20">
                macOS · Linux · Windows
              </motion.p>
          </motion.div>
        </div>
      </section>
    </div>
  )
}
