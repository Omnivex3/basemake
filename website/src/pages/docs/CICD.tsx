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

function Tip({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-6 rounded-xl border border-emerald-500/20 bg-emerald-500/5 px-5 py-4 text-sm text-emerald-300/80">
      {children}
    </div>
  )
}

function Warn({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-6 rounded-xl border border-amber-500/20 bg-amber-500/5 px-5 py-4 text-sm text-amber-300/80">
      {children}
    </div>
  )
}

/* ================================================================== */
/*  PAGE — CI/CD Integration                                           */
/* ================================================================== */
export default function CICD() {
  return (
    <div className="pb-24">
      {/* Header */}
      <div className="mb-10">
        <Badge variant="outline" className="mb-3 border-[#ff3131]/30 text-[#ff3131] text-xs tracking-wide uppercase">
          Integration
        </Badge>
        <h1 className="text-4xl font-bold tracking-tight text-white sm:text-5xl">
          CI/CD Integration
        </h1>
        <p className="mt-3 text-lg text-white/50">
          Gate your deployments on SQL quality with <Code>basemake check</Code>. Catch
          slow queries, dangerous patterns, and schema drift before they reach production.
        </p>
      </div>

      <Separator className="mb-10 bg-white/[0.04]" />

      {/* Overview */}
      <Card className="mb-10 border-white/[0.06] bg-white/[0.02]">
        <CardHeader>
          <CardTitle className="text-white text-lg">How It Works</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-white/60">
          <p>
            <Code>basemake check</Code> analyzes SQL files and returns a standardized
            exit code that your CI pipeline can act on:
          </p>
          <div className="overflow-x-auto rounded-xl border border-white/[0.06]">
            <table className="w-full text-sm">
              <thead>
                <tr>
                  <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Exit Code</th>
                  <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Meaning</th>
                  <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td className="border-b border-white/[0.04] px-4 py-3"><Code className="text-emerald-400">0</Code></td>
                  <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Pass</td>
                  <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">All queries meet quality thresholds</td>
                </tr>
                <tr>
                  <td className="border-b border-white/[0.04] px-4 py-3"><Code className="text-amber-400">1</Code></td>
                  <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Slow</td>
                  <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Queries exceed latency or cost budget</td>
                </tr>
                <tr>
                  <td className="border-b border-white/[0.04] px-4 py-3"><Code className="text-[#ff3131]">2</Code></td>
                  <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Dangerous</td>
                  <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Dangerous patterns (full table scans, Cartesian joins, etc.)</td>
                </tr>
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* GitHub Actions */}
      <H2 id="github-actions">GitHub Actions</H2>

      <P>
        Add <Code>basemake check</Code> to your GitHub Actions workflow to validate SQL
        migrations on every pull request.
      </P>

      <CodeBlock lang="yaml">{`name: SQL Quality Gate
on:
  pull_request:
    paths:
      - 'migrations/**/*.sql'
      - '**/*.sql'

jobs:
  basemake-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install basemake
        run: |
          curl -LO https://github.com/DynamicKarabo/basemake/releases/latest/download/basemake-Linux-x86_64.tar.gz
          tar xzf basemake-Linux-x86_64.tar.gz
          sudo mv basemake /usr/local/bin/

      - name: Run SQL quality check
        env:
          BASEMAKE_LICENSE_KEY: \${{ secrets.BASEMAKE_LICENSE_KEY }}
        run: |
          basemake check --dir ./migrations --budget=ci`}</CodeBlock>

      <Tip>
        Use <Code>--budget=ci</Code> for stricter thresholds in CI (e.g., max 200ms
        estimated latency, no full table scans). Your dev budget can be more permissive.
      </Tip>

      {/* GitLab CI */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="gitlab-ci">GitLab CI</H2>

      <P>
        The same pattern works in GitLab CI. Here's a <Code>.gitlab-ci.yml</Code> job:
      </P>

      <CodeBlock lang="yaml">{`basemake-check:
  stage: test
  image: alpine:latest
  before_script:
    - apk add curl tar
    - curl -LO https://github.com/DynamicKarabo/basemake/releases/latest/download/basemake-Linux-x86_64.tar.gz
    - tar xzf basemake-Linux-x86_64.tar.gz
    - mv basemake /usr/local/bin/
  script:
    - basemake check --dir ./migrations --budget=ci
  variables:
    BASEMAKE_LICENSE_KEY: \$BASEMAKE_LICENSE_KEY
  rules:
    - changes:
        - migrations/**/*.sql`}</CodeBlock>

      {/* Budget Policies in CI */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="budget-policies">Budget Policies in CI</H2>

      <P>
        Define named budget profiles that apply different thresholds in different contexts.
      </P>

      <CodeBlock lang="bash">{`# Create a CI-specific budget profile
basemake budget create ci \
  --max-latency=200ms \
  --max-rows=5000 \
  --block-full-table-scan \
  --block-cartesian-join

# Create a more permissive dev profile
basemake budget create dev \
  --max-latency=1s \
  --max-rows=50000

# Use a profile
basemake check --dir ./migrations --budget=ci`}</CodeBlock>

      <P>
        Budget profiles are stored in your config file and can be shared across your team
        via the Team Server.
      </P>

      {/* Environment Variables */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="env-vars">Environment Variables for CI</H2>

      <P>
        In CI environments, use environment variables instead of a config file to keep
        credentials out of your repository.
      </P>

      <div className="mb-6 overflow-x-auto rounded-xl border border-white/[0.06]">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Variable</th>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Required</th>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Description</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-mono text-xs">BASEMAKE_LICENSE_KEY</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">For Pro/Team features</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Your license key</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-mono text-xs">OPENAI_API_KEY</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">For AI-powered checks</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Provider API key</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-mono text-xs">DATABASE_URL</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">For schema-aware checks</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Test database connection</td>
            </tr>
          </tbody>
        </table>
      </div>

      <Warn>
        <Code>basemake check</Code> works in offline mode (without a database connection)
        for static analysis, but connecting to a test database unlocks schema-aware checks
        that catch more issues.
      </Warn>

      {/* Advanced: Custom Policies */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="custom-policies">Custom Check Policies</H2>

      <P>
        Define custom check rules via a <Code>.basemakerc.yaml</Code> file in your
        repository root:
      </P>

      <CodeBlock lang="yaml">{`# .basemakerc.yaml
checks:
  - rule: no-drop-table
    severity: error
    message: "DROP TABLE is not allowed in migrations"

  - rule: max-columns
    severity: warning
    args:
      limit: 50
    message: "Tables should have at most 50 columns"

  - rule: naming-convention
    severity: error
    args:
      pattern: "^[a-z_]+$"
    message: "Use snake_case for table and column names"`}</CodeBlock>

      <P>
        basemake picks up <Code>.basemakerc.yaml</Code> automatically when it's present
        in the working directory. Custom rules apply to both local runs and CI.
      </P>
    </div>
  )
}
