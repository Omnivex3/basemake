import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { motion, AnimatePresence } from 'framer-motion'
import { Link } from 'react-router-dom'
import {
  Check,
  X,
  ChevronDown,
  ArrowRight,
  Download,
  Sparkles,
  Users,
  Building2,
  Shield,
  Zap,
  MoveRight,
} from 'lucide-react'
import { useState } from 'react'

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
/*  Types                                                              */
/* ------------------------------------------------------------------ */
interface PlanFeature {
  name: string
  included: boolean | 'free' | 'pro' | 'team'
}

interface Plan {
  id: string
  name: string
  description: string
  monthlyPrice: number | null
  annualPrice: number | null
  yearlyPrice: number | null
  icon: React.ElementType
  badge?: string
  highlighted?: boolean
  features: PlanFeature[]
  bestFor: string
  cta: string
  ctaLink: string
}

interface FaqItem {
  q: string
  a: string
}

/* ------------------------------------------------------------------ */
/*  Data — Plans                                                       */
/* ------------------------------------------------------------------ */
const plans: Plan[] = [
  {
    id: 'free',
    name: 'Free',
    description: 'For solo developers and exploration',
    monthlyPrice: 0,
    annualPrice: 0,
    yearlyPrice: 0,
    icon: Zap,
    features: [
      { name: 'Natural language → SQL (BYOK)', included: true },
      { name: 'Query execution (PostgreSQL, MySQL, SQLite)', included: true },
      { name: 'Full interactive TUI/REPL', included: true },
      { name: 'Chat mode (.ask)', included: true },
      { name: '.explain + .analyze — basic insights', included: true },
      { name: 'Index recommendations (read-only)', included: true },
      { name: 'Output formats: table, JSON, CSV', included: true },
      { name: 'Local query history', included: true },
      { name: 'Unlimited connections', included: true },
      { name: 'All data stays on your machine', included: true },
      { name: 'Index recommendations — apply mode', included: false },
      { name: 'basemake check — CI/CD gate', included: false },
      { name: 'basemake budget — policy as code', included: false },
      { name: 'basemake watch — query monitoring', included: false },
      { name: 'basemake diff — schema drift', included: false },
      { name: 'basemake doctor — diagnostics', included: false },
      { name: 'Team server + sync', included: false },
      { name: 'Shared AI proxy + caching', included: false },
      { name: 'RBAC read-only enforcement', included: false },
      { name: 'Audit log', included: false },
      { name: 'Slack/Teams integration', included: false },
    ],
    bestFor: 'Side projects, learning, personal use',
    cta: 'Get Started',
    ctaLink: '/docs/quickstart',
  },
  {
    id: 'pro',
    name: 'Pro',
    description: 'For professional developers',
    monthlyPrice: 15,
    annualPrice: 150,
    yearlyPrice: 150,
    icon: Sparkles,
    badge: 'Most Popular',
    highlighted: true,
    features: [
      { name: 'Natural language → SQL (BYOK)', included: true },
      { name: 'Query execution (PostgreSQL, MySQL, SQLite)', included: true },
      { name: 'Full interactive TUI/REPL', included: true },
      { name: 'Chat mode (.ask)', included: true },
      { name: '.explain + .analyze — basic insights', included: true },
      { name: 'Index recommendations (read-only)', included: true },
      { name: 'Output formats: table, JSON, CSV', included: true },
      { name: 'Local query history', included: true },
      { name: 'Unlimited connections', included: true },
      { name: 'All data stays on your machine', included: true },
      { name: 'Index recommendations — apply mode', included: true },
      { name: 'basemake check — CI/CD gate', included: true },
      { name: 'basemake budget — policy as code', included: true },
      { name: 'basemake watch — query monitoring', included: true },
      { name: 'basemake diff — schema drift', included: true },
      { name: 'basemake doctor — diagnostics', included: true },
      { name: 'Team server + sync', included: false },
      { name: 'Shared AI proxy + caching', included: false },
      { name: 'RBAC read-only enforcement', included: false },
      { name: 'Audit log', included: false },
      { name: 'Slack/Teams integration', included: false },
    ],
    bestFor: 'Individual developers, small teams, CI/CD pipelines',
    cta: 'Get Pro',
    ctaLink: '#',
  },
  {
    id: 'team',
    name: 'Team',
    description: 'For engineering teams that ship together',
    monthlyPrice: 39,
    annualPrice: 39,
    yearlyPrice: 39,
    icon: Users,
    features: [
      { name: 'Natural language → SQL (BYOK)', included: true },
      { name: 'Query execution (PostgreSQL, MySQL, SQLite)', included: true },
      { name: 'Full interactive TUI/REPL', included: true },
      { name: 'Chat mode (.ask)', included: true },
      { name: '.explain + .analyze — basic insights', included: true },
      { name: 'Index recommendations (read-only)', included: true },
      { name: 'Output formats: table, JSON, CSV', included: true },
      { name: 'Local query history', included: true },
      { name: 'Unlimited connections', included: true },
      { name: 'All data stays on your machine', included: true },
      { name: 'Index recommendations — apply mode', included: true },
      { name: 'basemake check — CI/CD gate', included: true },
      { name: 'basemake budget — policy as code', included: true },
      { name: 'basemake watch — query monitoring', included: true },
      { name: 'basemake diff — schema drift', included: true },
      { name: 'basemake doctor — diagnostics', included: true },
      { name: 'Team server + sync', included: true },
      { name: 'Shared AI proxy + caching', included: true },
      { name: 'RBAC read-only enforcement', included: true },
      { name: 'Audit log', included: true },
      { name: 'Slack/Teams integration', included: true },
    ],
    bestFor: 'Engineering teams, startups, mid-market companies',
    cta: 'Get Team',
    ctaLink: '#',
  },
]

/* ------------------------------------------------------------------ */
/*  Data — Enterprise                                                  */
/* ------------------------------------------------------------------ */
const enterprise = {
  icon: Building2,
  title: 'Enterprise',
  tagline: 'For organizations with compliance, SSO, and on-prem requirements',
  features: [
    'On-prem server — deploy behind your firewall',
    'SSO/SAML — Okta, Azure AD, Google Workspace',
    'Custom AI proxy — bring your own model endpoint',
    'Audit export — JSON/CSV of all query history',
    'Custom contract — annual billing, volume discounts',
    'Dedicated support — 1-hour SLA, Slack channel',
    'Training — team onboarding session',
  ],
  cta: 'Contact Sales',
}

/* ------------------------------------------------------------------ */
/*  Data — Savings comparison                                         */
/* ------------------------------------------------------------------ */
const proSavings = [
  { tool: 'DataGrip', cost: '€109/yr', covered: 'SQL client' },
  { tool: 'DataGrip AI Pro', cost: '€100/yr', covered: 'NL→SQL' },
  { tool: 'CI performance check script', cost: 'Dev hours', covered: 'basemake check' },
  { tool: 'Monitoring (Datadog, New Relic)', cost: '$15+/host/mo', covered: 'basemake watch' },
]

const teamSavings = [
  { tool: 'DataGrip × 10 devs', cost: '€1,090/yr', covered: 'SQL client for everyone' },
  { tool: 'DataGrip AI Pro × 10', cost: '€1,000/yr', covered: 'NL→SQL for everyone' },
  { tool: 'Shared AI proxy setup', cost: 'Dev hours + infra', covered: 'Built-in' },
  { tool: 'Query monitoring', cost: '$15+/host/mo', covered: 'Built-in' },
]

/* ------------------------------------------------------------------ */
/*  Data — FAQ                                                         */
/* ------------------------------------------------------------------ */
const faqItems: FaqItem[] = [
  {
    q: 'How does licensing work for a CLI tool?',
    a: '**Free**: No license needed. Download and run.\n**Pro**: License key via `basemake config set license_key xxx`. Required for `check`, `budget`, `watch`, `diff`, and index apply. Local REPL/query stays free even without a key.\n**Team**: Server requires a team license to start. Client connects to server with seat-based auth.\n**CI/CD**: Set `BASEMAKE_LICENSE_KEY` as a CI secret. One license key per CI runner.',
  },
  {
    q: 'Can I use basemake at work with the Free tier?',
    a: 'Yes. Free includes the full TUI, NL→SQL, query execution, and read-only index recommendations. No data leaves your machine. If you need CI/CD gates, budgets, or monitoring, that\'s when you go Pro.',
  },
  {
    q: 'What does "BYOK" mean?',
    a: 'Bring Your Own Key. You use your own API key (OpenAI, Anthropic, OpenCode) or a local Ollama instance. basemake never charges for AI tokens — you pay your provider directly.',
  },
  {
    q: 'Does the AI proxy in Team save money?',
    a: 'Yes. The server caches AI responses for identical queries. Teams running the same reports see a 40–60% reduction in API costs. The first query pays full price, the next 9 devs get it from cache.',
  },
  {
    q: 'What if I just want the TUI and don\'t care about CI/CD?',
    a: 'Free tier is perfect for you. Unlimited REPL, NL→SQL, all the good stuff.',
  },
  {
    q: 'How is this different from DataGrip?',
    a: 'DataGrip is a GUI IDE for your database. basemake is a **terminal-native tool** that works in SSH, CI/CD, Docker, and any headless environment. Plus: index recommendations with actual pg_stats selectivity math, policy-as-code budgets, and CI/CD gates — none of which DataGrip has.',
  },
]

/* ================================================================== */
/*  Savings Table Component                                            */
/* ================================================================== */
function SavingsTable({
  title,
  description,
  totalLabel,
  totalCost,
  basemakeCost,
  items,
  cols,
}: {
  title: string
  description?: string
  totalLabel: string
  totalCost: string
  basemakeCost: string
  items: { tool: string; cost: string; covered: string }[]
  cols: string
}) {
  const Icon = cols === 'pro' ? Zap : Users
  return (
    <motion.div variants={scaleIn} className="overflow-hidden rounded-2xl border border-white/[0.06] bg-white/[0.02]">
      <div className="border-b border-white/[0.06] bg-white/[0.03] px-6 py-5">
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20">
            <Icon className="h-4 w-4" />
          </div>
          <div>
            <h4 className="text-base font-semibold text-white">{title}</h4>
            {description && (
              <p className="text-sm text-white/40">{description}</p>
            )}
          </div>
        </div>
      </div>

      <div className="divide-y divide-white/[0.04]">
        {items.map((item) => (
          <div
            key={item.tool}
            className="flex items-center justify-between gap-4 px-6 py-3.5 text-sm"
          >
            <span className="text-white/60">{item.tool}</span>
            <span className="text-white/30">{item.cost}</span>
            <span className="ml-auto whitespace-nowrap rounded-full bg-emerald-500/8 px-3 py-0.5 text-xs font-medium text-emerald-400">
              ✓ {item.covered}
            </span>
          </div>
        ))}
      </div>

      <div className="border-t border-white/[0.06] bg-white/[0.03] px-6 py-4">
        <div className="flex items-center justify-between gap-4">
          <span className="text-sm font-medium text-white/80">{totalLabel}</span>
          <div className="flex items-center gap-3">
            <span className="text-sm text-white/30 line-through">{totalCost}</span>
            <MoveRight className="h-4 w-4 text-emerald-400" />
            <span className="text-base font-bold text-emerald-400">{basemakeCost}</span>
          </div>
        </div>
      </div>
    </motion.div>
  )
}

/* ================================================================== */
/*  FAQ Item Component                                                 */
/* ================================================================== */
function FaqAccordion({ items }: { items: FaqItem[] }) {
  const [openIndex, setOpenIndex] = useState<number | null>(0)

  return (
    <div className="mx-auto max-w-3xl space-y-3">
      {items.map((item, i) => {
        const isOpen = openIndex === i
        return (
          <motion.div
            key={i}
            variants={scaleIn}
            className="overflow-hidden rounded-xl border border-white/[0.06] bg-white/[0.02] transition-colors hover:border-white/[0.1]"
          >
            <button
              onClick={() => setOpenIndex(isOpen ? null : i)}
              className="flex w-full items-center justify-between gap-4 px-6 py-4 text-left text-sm font-medium text-white/80 transition-colors hover:text-white"
            >
              <span>{item.q}</span>
              <ChevronDown
                className={`h-4 w-4 shrink-0 text-white/30 transition-transform duration-300 ${
                  isOpen ? 'rotate-180' : ''
                }`}
              />
            </button>
            <AnimatePresence initial={false}>
              {isOpen && (
                <motion.div
                  key="content"
                  initial={{ height: 0, opacity: 0 }}
                  animate={{ height: 'auto', opacity: 1 }}
                  exit={{ height: 0, opacity: 0 }}
                  transition={{ duration: 0.3, ease: easeOut() }}
                  className="overflow-hidden"
                >
                  <div className="border-t border-white/[0.04] px-6 py-4 text-sm leading-relaxed text-white/50 whitespace-pre-line">
                    {item.a}
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </motion.div>
        )
      })}
    </div>
  )
}

/* ================================================================== */
/*  Pricing Card Component                                             */
/* ================================================================== */
function PricingCard({ plan, annual }: { plan: Plan; annual: boolean }) {
  const Icon = plan.icon

  const displayPrice =
    plan.id === 'free'
      ? '$0'
      : annual && plan.annualPrice !== null
        ? `$${plan.annualPrice}`
        : `$${plan.monthlyPrice}`

  const displayPeriod =
    plan.id === 'free'
      ? 'forever'
      : annual && plan.annualPrice !== null
        ? '/yr'
        : '/mo'

  const monthlyEquivalent =
    plan.id !== 'free' && annual && plan.yearlyPrice
      ? `$${(plan.yearlyPrice / 12).toFixed(0)}/mo billed annually`
      : null

  return (
    <motion.div
      variants={scaleIn}
      className={`relative flex flex-col rounded-2xl border transition-all duration-300 ${
        plan.highlighted
          ? 'border-[#e63946]/40 bg-[#e63946]/[0.03] shadow-xl shadow-[#e63946]/10 scale-[1.02] md:scale-105 z-10'
          : 'border-white/[0.06] bg-white/[0.02] hover:border-white/[0.12]'
      }`}
    >
      {/* Badge */}
      {plan.badge && (
        <div className="absolute -top-3 left-1/2 -translate-x-1/2">
          <Badge variant="destructive" className="bg-[#e63946] text-white border-none px-4 py-1 text-[11px] font-semibold tracking-wide uppercase shadow-lg shadow-[#e63946]/30">
            {plan.badge}
          </Badge>
        </div>
      )}

      {/* Header */}
      <div className={`px-6 pt-8 pb-6 ${plan.highlighted ? 'pt-10' : ''}`}>
        <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-[#e63946]/10 text-[#e63946] ring-1 ring-[#e63946]/20">
          <Icon className="h-5 w-5" />
        </div>
        <CardTitle className="text-xl font-bold text-white">{plan.name}</CardTitle>
        <CardDescription className="mt-1 text-sm text-white/40">
          {plan.description}
        </CardDescription>

        {/* Price */}
        <div className="mt-6 flex items-baseline gap-1">
          <span className="text-4xl font-bold tracking-tight text-white">
            {displayPrice}
          </span>
          <span className="text-sm text-white/30">{displayPeriod}</span>
        </div>
        {monthlyEquivalent && (
          <p className="mt-1 text-xs text-emerald-400/80">{monthlyEquivalent}</p>
        )}
      </div>

      {/* Features */}
      <CardContent className="flex-1 px-6 pb-6">
        <ul className="space-y-3">
          {plan.features.slice(0, 11).map((feature) => (
            <li key={feature.name} className="flex items-start gap-3 text-sm">
              {feature.included ? (
                <Check className="mt-0.5 h-4 w-4 shrink-0 text-emerald-400" />
              ) : (
                <X className="mt-0.5 h-4 w-4 shrink-0 text-white/15" />
              )}
              <span
                className={
                  feature.included ? 'text-white/70' : 'text-white/20'
                }
              >
                {feature.name}
              </span>
            </li>
          ))}
        </ul>

        {/* Toggle extra features indicator */}
        {plan.features.length > 11 && (
          <details className="group mt-4">
            <summary className="flex cursor-pointer items-center gap-2 text-xs font-medium text-white/30 hover:text-white/50 transition-colors">
              <ChevronDown className="h-3.5 w-3.5 transition-transform group-open:rotate-180" />
              Show all features
            </summary>
            <ul className="mt-3 space-y-3">
              {plan.features.slice(11).map((feature) => (
                <li key={feature.name} className="flex items-start gap-3 text-sm">
                  {feature.included ? (
                    <Check className="mt-0.5 h-4 w-4 shrink-0 text-emerald-400" />
                  ) : (
                    <X className="mt-0.5 h-4 w-4 shrink-0 text-white/15" />
                  )}
                  <span
                    className={
                      feature.included ? 'text-white/70' : 'text-white/20'
                    }
                  >
                    {feature.name}
                  </span>
                </li>
              ))}
            </ul>
          </details>
        )}
      </CardContent>

      {/* Footer */}
      <CardFooter className="px-6 pb-8 pt-0">
        <Link
          to={plan.ctaLink}
          className={`inline-flex h-12 w-full items-center justify-center gap-2 rounded-lg px-8 text-base font-semibold transition-all ${
            plan.highlighted
              ? 'bg-[#e63946] text-white shadow-lg shadow-[#e63946]/25 hover:bg-[#e63946]/90 hover:shadow-[#e63946]/35'
              : 'border border-white/10 text-white/70 hover:bg-white/5 hover:text-white'
          }`}
        >
          {plan.cta}
          <ArrowRight className="ml-1.5 h-4 w-4" />
        </Link>
        <p className="mt-3 text-center text-xs text-white/20">{plan.bestFor}</p>
      </CardFooter>
    </motion.div>
  )
}

/* ================================================================== */
/*  Enterprise CTA Component                                           */
/* ================================================================== */
function EnterpriseCTA() {
  return (
    <motion.div
      variants={scaleIn}
      className="relative isolate overflow-hidden rounded-2xl border border-white/[0.06] bg-gradient-to-br from-white/[0.03] to-white/[0.01]"
    >
      {/* Grid pattern */}
      <div
        className="pointer-events-none absolute inset-0 -z-10"
        style={{
          backgroundImage: [
            'linear-gradient(rgba(255,255,255,0.025) 1px, transparent 1px)',
            'linear-gradient(90deg, rgba(255,255,255,0.025) 1px, transparent 1px)',
          ].join(', '),
          backgroundSize: '40px 40px',
        }}
      />
      <div className="pointer-events-none absolute right-0 top-0 -z-10 h-72 w-72 rounded-full bg-[#e63946]/5 blur-[100px]" />

      <div className="flex flex-col items-center gap-8 px-8 py-12 text-center lg:flex-row lg:text-left">
        <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-[#e63946]/10 text-[#e63946] ring-1 ring-[#e63946]/20">
          <Building2 className="h-7 w-7" />
        </div>

        <div className="flex-1">
          <h3 className="text-2xl font-bold text-white">{enterprise.title}</h3>
          <p className="mt-1 text-sm text-white/40">{enterprise.tagline}</p>

          <div className="mt-6 flex flex-wrap justify-center gap-3 lg:justify-start">
            {enterprise.features.slice(0, 4).map((feat) => (
              <span
                key={feat}
                className="inline-flex items-center gap-1.5 rounded-full border border-white/[0.06] bg-white/[0.03] px-3 py-1 text-xs text-white/50"
              >
                <Shield className="h-3 w-3 text-[#e63946]/60" />
                {feat}
              </span>
            ))}
          </div>
        </div>

        <Button
          size="lg"
          variant="destructive"
          className="shrink-0 bg-[#e63946] text-white shadow-lg shadow-[#e63946]/25 hover:bg-[#e63946]/90 hover:shadow-[#e63946]/35"
        >
          {enterprise.cta}
          <ArrowRight className="ml-1.5 h-4 w-4" />
        </Button>
      </div>

      {/* Additional features row */}
      <div className="border-t border-white/[0.04] px-8 py-4">
        <div className="flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-xs text-white/30 lg:justify-start">
          {enterprise.features.slice(4).map((feat) => (
            <span key={feat} className="flex items-center gap-1.5">
              <Check className="h-3 w-3 text-emerald-400/60" />
              {feat}
            </span>
          ))}
        </div>
      </div>
    </motion.div>
  )
}

/* ================================================================== */
/*  PAGE — Pricing                                                     */
/* ================================================================== */
export default function Pricing() {
  const [annual, setAnnual] = useState(false)

  return (
    <div className="overflow-hidden">
      {/* ============================================================ */}
      {/*  1. HERO                                                      */}
      {/* ============================================================ */}
      <section className="relative isolate overflow-hidden">
        {/* Background grid */}
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
        {/* Glow */}
        <div className="pointer-events-none absolute -top-40 left-1/2 -z-10 h-[600px] w-[800px] -translate-x-1/2 rounded-full bg-[#e63946]/5 blur-[120px]" />
        <div className="pointer-events-none absolute -bottom-40 right-0 -z-10 h-[400px] w-[600px] rounded-full bg-[#e63946]/3 blur-[100px]" />

        <div className="mx-auto max-w-7xl px-6 pb-16 pt-20 md:pt-28 lg:pb-20">
          <motion.div
            initial="hidden"
            animate="visible"
            variants={stagger}
            className="mx-auto max-w-2xl text-center"
          >
            <SectionLabel>Simple Pricing</SectionLabel>

            <motion.h1
              variants={fadeUp}
              className="mt-2 text-4xl font-bold leading-tight tracking-tight text-white sm:text-5xl lg:text-6xl"
            >
              Pricing that scales with{' '}
              <span className="text-[#e63946]">your team.</span>
            </motion.h1>

            <motion.p
              variants={fadeUp}
              className="mt-4 text-base leading-relaxed text-white/50 sm:text-lg"
            >
              Start free. Upgrade when you need CI/CD gates, monitoring, or
              team sync. No hidden fees, no surprise bills.
            </motion.p>

            {/* Billing toggle */}
            <motion.div
              variants={fadeUp}
              className="mt-10 flex items-center justify-center gap-4"
            >
              <span
                className={`text-sm font-medium transition-colors ${
                  !annual ? 'text-white' : 'text-white/30'
                }`}
              >
                Monthly
              </span>
              <button
                onClick={() => setAnnual(!annual)}
                className={`relative inline-flex h-7 w-12 shrink-0 cursor-pointer items-center rounded-full border transition-colors ${
                  annual
                    ? 'border-[#e63946]/40 bg-[#e63946]/20'
                    : 'border-white/[0.12] bg-white/[0.04]'
                }`}
                role="switch"
                aria-checked={annual}
              >
                <span
                  className={`inline-block h-5 w-5 transform rounded-full bg-white shadow-sm transition-transform duration-300 ${
                    annual ? 'translate-x-6' : 'translate-x-1'
                  }`}
                />
              </button>
              <div className="flex items-center gap-2">
                <span
                  className={`text-sm font-medium transition-colors ${
                    annual ? 'text-white' : 'text-white/30'
                  }`}
                >
                  Annual
                </span>
                <Badge className="bg-emerald-500/10 text-emerald-400 border-none text-[10px] font-semibold uppercase tracking-wider">
                  Save 17%
                </Badge>
              </div>
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* ============================================================ */}
      {/*  2. PRICING CARDS                                             */}
      {/* ============================================================ */}
      <section className="pb-24 md:pb-32">
        <div className="mx-auto max-w-7xl px-6">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="grid items-start gap-8 md:grid-cols-3"
          >
            {/* Free & Team — regular cards */}
            {plans
              .filter((p) => !p.highlighted)
              .map((plan) => (
                <PricingCard key={plan.id} plan={plan} annual={annual} />
              ))}

            {/* Pro — highlighted card */}
            {plans
              .filter((p) => p.highlighted)
              .map((plan) => (
                <PricingCard key={plan.id} plan={plan} annual={annual} />
              ))}
          </motion.div>

          {/* Enterprise CTA */}
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="mt-8"
          >
            <EnterpriseCTA />
          </motion.div>
        </div>
      </section>

      {/* ============================================================ */}
      {/*  3. WHAT YOU ACTUALLY SAVE                                    */}
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
            <SectionLabel>Compare Costs</SectionLabel>
            <motion.h2
              variants={fadeUp}
              className="text-3xl font-bold tracking-tight text-white sm:text-4xl"
            >
              What you{' '}
              <span className="text-emerald-400">actually save.</span>
            </motion.h2>
            <motion.p
              variants={fadeUp}
              className="mx-auto mt-4 max-w-2xl text-white/40"
            >
              basemake replaces 3–4 separate tools. Here's what the math
              looks like.
            </motion.p>
          </motion.div>

          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="grid gap-8 md:grid-cols-2"
          >
            <SavingsTable
              title="basemake Pro replaces"
              totalLabel="Total for 1 dev"
              totalCost="$224+/yr + monitoring"
              basemakeCost="$150/yr"
              items={proSavings}
              cols="pro"
            />

            <SavingsTable
              title="basemake Team replaces"
              description="For a 10-person team"
              totalLabel="Total for 10 devs"
              totalCost="$2,290+/yr + infra"
              basemakeCost="$4,680/yr ($39/seat)"
              items={teamSavings}
              cols="team"
            />
          </motion.div>

          <motion.p
            variants={fadeUp}
            className="mx-auto mt-8 max-w-2xl text-center text-sm leading-relaxed text-white/30"
          >
            10 devs using basemake Team = ~2× the cost of DataGrip alone. But
            you also get: CI/CD gates, performance policies, monitoring, schema
            diffing, shared AI caching (reduces API costs), and team audit. The
            package replaces <strong className="text-white/50">3–4 separate tools.</strong>
          </motion.p>
        </div>
      </section>

      {/* ============================================================ */}
      {/*  4. FEATURE COMPARISON                                        */}
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
            <SectionLabel>Feature Comparison</SectionLabel>
            <motion.h2
              variants={fadeUp}
              className="text-3xl font-bold tracking-tight text-white sm:text-4xl"
            >
              Everything at a glance
            </motion.h2>
            <motion.p
              variants={fadeUp}
              className="mx-auto mt-4 max-w-xl text-white/40"
            >
              See exactly what each plan includes, side by side.
            </motion.p>
          </motion.div>

          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={stagger}
            className="overflow-hidden rounded-2xl border border-white/[0.06]"
          >
            {/* Table header */}
            <div className="hidden md:grid md:grid-cols-[1fr_120px_120px_120px] border-b border-white/[0.06] bg-white/[0.03]">
              <div className="px-6 py-4 text-xs font-semibold tracking-[0.1em] text-white/30 uppercase">
                Feature
              </div>
              {['Free', 'Pro', 'Team'].map((name) => (
                <div
                  key={name}
                  className="px-4 py-4 text-center text-xs font-semibold tracking-[0.1em] text-white/30 uppercase"
                >
                  {name}
                </div>
              ))}
            </div>

            {/* Table rows */}
            {[
              { label: 'Natural language → SQL', free: true, pro: true, team: true },
              { label: 'Query execution', free: true, pro: true, team: true },
              { label: 'Full TUI/REPL', free: true, pro: true, team: true },
              { label: 'Chat mode (.ask)', free: true, pro: true, team: true },
              { label: '.explain + .analyze', free: true, pro: true, team: true },
              { label: 'Index recommendations', free: 'Read-only', pro: 'Apply', team: 'Apply' },
              { label: 'Output formats (table, JSON, CSV)', free: true, pro: true, team: true },
              { label: 'Local query history', free: true, pro: true, team: true },
              { label: 'Unlimited connections', free: true, pro: true, team: true },
              { label: 'All data stays on your machine', free: true, pro: true, team: true },
              { label: 'Index recommendations — apply mode', free: false, pro: true, team: true },
              { label: 'basemake check — CI/CD gate', free: false, pro: true, team: true },
              { label: 'basemake budget — policies', free: false, pro: true, team: true },
              { label: 'basemake watch — monitoring', free: false, pro: true, team: true },
              { label: 'basemake diff — schema drift', free: false, pro: true, team: true },
              { label: 'basemake doctor — diagnostics', free: false, pro: true, team: true },
              { label: 'Team server + sync', free: false, pro: false, team: true },
              { label: 'Shared AI proxy + caching', free: false, pro: false, team: true },
              { label: 'RBAC server-side', free: false, pro: false, team: true },
              { label: 'Audit log', free: false, pro: false, team: true },
              { label: 'Slack/Teams integration', free: false, pro: false, team: true },
              { label: 'SSO/SAML', free: false, pro: false, team: false },
              { label: 'On-prem deployment', free: false, pro: false, team: false },
            ].map((row, i) => (
              <div
                key={row.label}
                className={`hidden md:grid md:grid-cols-[1fr_120px_120px_120px] items-center transition-colors ${
                  i % 2 === 0
                    ? 'bg-white/[0.01]'
                    : 'bg-transparent'
                } hover:bg-white/[0.03]`}
              >
                <div className="px-6 py-3.5 text-sm text-white/60">{row.label}</div>
                {(['free', 'pro', 'team'] as const).map((tier) => {
                  const val = row[tier]
                  return (
                    <div key={tier} className="flex justify-center px-4 py-3.5">
                      {val === true ? (
                        <Check className="h-4 w-4 text-emerald-400" />
                      ) : val === false ? (
                        <X className="h-4 w-4 text-white/15" />
                      ) : (
                        <span className="text-xs font-medium text-white/50">
                          {val}
                        </span>
                      )}
                    </div>
                  )
                })}
              </div>
            ))}

            {/* Mobile feature rows */}
            <div className="divide-y divide-white/[0.04] md:hidden">
              {[
                { label: 'Natural language → SQL', free: true, pro: true, team: true },
                { label: 'Query execution', free: true, pro: true, team: true },
                { label: 'Full TUI/REPL', free: true, pro: true, team: true },
                { label: 'Chat mode (.ask)', free: true, pro: true, team: true },
                { label: '.explain + .analyze', free: true, pro: true, team: true },
                { label: 'Index recommendations', free: 'Read-only', pro: 'Apply', team: 'Apply' },
                { label: 'Output formats', free: true, pro: true, team: true },
                { label: 'Unlimited connections', free: true, pro: true, team: true },
                { label: 'Data stays on your machine', free: true, pro: true, team: true },
                { label: 'Index apply mode', free: false, pro: true, team: true },
                { label: 'CI/CD gate', free: false, pro: true, team: true },
                { label: 'Budget policies', free: false, pro: true, team: true },
                { label: 'Query monitoring', free: false, pro: true, team: true },
                { label: 'Schema drift detection', free: false, pro: true, team: true },
                { label: 'Advanced diagnostics', free: false, pro: true, team: true },
                { label: 'Team server + sync', free: false, pro: false, team: true },
                { label: 'Shared AI proxy + caching', free: false, pro: false, team: true },
                { label: 'RBAC + audit log', free: false, pro: false, team: true },
                { label: 'Slack/Teams integration', free: false, pro: false, team: true },
                { label: 'SSO/SAML', free: false, pro: false, team: false },
                { label: 'On-prem deployment', free: false, pro: false, team: false },
              ].map((row, i) => (
                <div
                  key={row.label}
                  className={`px-6 py-4 ${
                    i % 2 === 0 ? 'bg-white/[0.01]' : ''
                  }`}
                >
                  <div className="text-sm font-medium text-white/70">
                    {row.label}
                  </div>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {(['free', 'pro', 'team'] as const).map((tier) => {
                      const val = row[tier]
                      const tierName = tier.charAt(0).toUpperCase() + tier.slice(1)
                      return (
                        <span
                          key={tier}
                          className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-[11px] font-medium ${
                            val === true
                              ? 'bg-emerald-500/10 text-emerald-400'
                              : val === false
                                ? 'bg-white/[0.03] text-white/20'
                                : 'bg-amber-500/10 text-amber-300'
                          }`}
                        >
                          {val === true ? (
                            <Check className="h-3 w-3" />
                          ) : val === false ? (
                            <X className="h-3 w-3" />
                          ) : (
                            <span className="h-3 w-3 flex items-center justify-center text-[10px] font-bold">~</span>
                          )}
                          {tierName}
                        </span>
                      )
                    })}
                  </div>
                </div>
              ))}
            </div>
          </motion.div>
        </div>
      </section>

      {/* ============================================================ */}
      {/*  5. FAQ                                                       */}
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
            <SectionLabel>Got Questions?</SectionLabel>
            <motion.h2
              variants={fadeUp}
              className="text-3xl font-bold tracking-tight text-white sm:text-4xl"
            >
              Frequently asked questions
            </motion.h2>
            <motion.p
              variants={fadeUp}
              className="mx-auto mt-4 max-w-xl text-white/40"
            >
              Everything you need to know about basemake pricing and
              licensing.
            </motion.p>
          </motion.div>

          <FaqAccordion items={faqItems} />
        </div>
      </section>

      {/* ============================================================ */}
      {/*  6. CTA BANNER                                                */}
      {/* ============================================================ */}
      <section className="relative isolate overflow-hidden py-24 md:py-32">
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
            Start for free. Upgrade when you need it.
          </motion.h2>
          <motion.p
            variants={fadeUp}
            className="mx-auto mt-4 max-w-xl text-lg text-white/50"
          >
            No credit card required. No time limit on the Free tier. Your
            data never leaves your machine.
          </motion.p>

          <motion.div
            variants={fadeUp}
            className="mt-10 flex flex-wrap items-center justify-center gap-4"
          >
            <Link to="/docs/quickstart">
              <Button
                size="lg"
                variant="outline"
                className="border-white/10 text-white/70 hover:bg-white/5 hover:text-white"
              >
                View Quickstart
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
