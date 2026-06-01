import { motion } from 'framer-motion'

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

const featureGroups = [
  {
    title: "Query",
    items: [
      "Natural language → SQL generation with configurable AI providers (OpenAI, Anthropic, Ollama, OpenCode)",
      "Raw SQL execution — type or pipe queries directly",
      "Interactive REPL with tab completion, history, bookmarks",
      "Query profiling with execution plan storage across runs",
      "PlanCheck — compares current plan against history before executing",
    ],
  },
  {
    title: "Analysis",
    items: [
      "EXPLAIN ANALYZE with structured issue detection (seq scans, estimate mismatches, expensive filters)",
      "Index recommendations with column ordering and reasoning",
      "Schema diffing between two databases (live, cached, or file)",
      "Cardinality estimates per table for guardrail enforcement",
    ],
  },
  {
    title: "CI/CD & Governance",
    items: [
      "basemake check — CI gate with exit codes 0/1/2/3 for query performance",
      "basemake budget — performance policy as code in .basemake/budgets.json",
      "basemake watch — scheduled query monitoring with regression alerts",
      "Server mode for team sync, shared AI proxy, and audit logging",
    ],
  },
  {
    title: "Databases",
    items: [
      "PostgreSQL — full EXPLAIN JSON, plan parsing, pg_class estimates",
      "MySQL — EXPLAIN ANALYZE, schema introspection via information_schema",
      "SQLite — WAL mode, PRAGMA-based introspection, pure Go driver",
    ],
  },
  {
    title: "Safety",
    items: [
      "SELECT * guardrail — graduated enforcement by table size (rewrite, warn, or block)",
      "Schema-aware prompt truncation for LLM context window limits",
      "FK context injection for accurate multi-table JOIN generation",
      "Read-only mode per-session",
      "API keys via environment variables only — never written to disk",
    ],
  },
  {
    title: "Infrastructure",
    items: [
      "Single static Go binary — no runtime dependencies",
      "Cross-platform: Linux amd64/arm64, macOS amd64/arm64",
      "Docker multi-arch images on GHCR",
      "CI/CD via GitHub Actions with self-hosted runner support",
      "Pre-commit hooks, golangci-lint, race detection testing",
    ],
  },
]

export default function FeaturesPage() {
  return (
    <div className="overflow-hidden">
      <section className="relative border-b border-white/[0.06] py-24">
        <div className="mx-auto max-w-6xl px-6">
          <motion.div initial="hidden" animate="visible" variants={stagger}>
            <motion.div variants={fadeUp}>
              <h1 className="text-4xl sm:text-5xl font-bold tracking-tight leading-[1.05] mb-6">
                Feature Reference
              </h1>
              <p className="text-lg text-white/50 max-w-xl mb-10 leading-relaxed">
                Everything basemake does. No tiers, no gates — just a single binary.
              </p>
            </motion.div>
          </motion.div>
        </div>
      </section>

      <section className="py-24">
        <div className="mx-auto max-w-4xl px-6">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: "-100px" }}
            variants={stagger}
            className="space-y-16"
          >
            {featureGroups.map((group, gi) => (
              <motion.div key={gi} variants={fadeUp}>
                <h2 className="text-2xl font-bold tracking-tight mb-6 text-white/90">
                  {group.title}
                </h2>
                <ul className="space-y-3">
                  {group.items.map((item, ii) => (
                    <li key={ii} className="flex items-start gap-3 text-sm text-white/50 leading-relaxed">
                      <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#ff3131]/60" />
                      {item}
                    </li>
                  ))}
                </ul>
              </motion.div>
            ))}
          </motion.div>
        </div>
      </section>
    </div>
  )
}
