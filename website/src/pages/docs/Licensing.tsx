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
      className="mt-12 mb-4 scroll-mt-24 text-2xl font-bold tracking-tight text-foreground"
    >
      {children}
    </h2>
  )
}

function H3({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="mt-8 mb-3 text-xl font-semibold tracking-tight text-foreground">
      {children}
    </h3>
  )
}

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

function UL({ children }: { children: React.ReactNode }) {
  return (
    <ul className="mb-6 space-y-2 text-muted-foreground">
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

/* ------------------------------------------------------------------ */
/*  Tier card                                                          */
/* ------------------------------------------------------------------ */
function TierCard({
  name,
  price,
  description,
  features,
  highlight = false,
}: {
  name: string
  price: string
  description: string
  features: string[]
  highlight?: boolean
}) {
  return (
    <Card
      className={
        highlight
          ? 'border-[#ff3131]/30 bg-[#ff3131]/5'
          : 'border-border/[0.06] bg-muted/30'
      }
    >
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-foreground text-xl">{name}</CardTitle>
          <span className="text-lg font-semibold text-muted-foreground">{price}</span>
        </div>
        <p className="text-sm text-muted-foreground">{description}</p>
      </CardHeader>
      <CardContent>
        <UL>
          {features.map((f) => (
            <LI key={f}>{f}</LI>
          ))}
        </UL>
      </CardContent>
    </Card>
  )
}

/* ================================================================== */
/*  PAGE — Licensing                                                   */
/* ================================================================== */
export default function Licensing() {
  return (
    <div className="pb-24">
      {/* Header */}
      <div className="mb-10">
        <Badge variant="outline" className="mb-3 border-[#ff3131]/30 text-[#ff3131] text-xs tracking-wide uppercase">
          Plans
        </Badge>
        <h1 className="text-4xl font-bold tracking-tight text-foreground sm:text-5xl">
          Licensing
        </h1>
        <p className="mt-3 text-lg text-muted-foreground">
          basemake is free for individual use. Upgrade to Pro or Team for advanced
          features, CI/CD gates, and team collaboration.
        </p>
      </div>

      <Separator className="mb-10 bg-muted/30" />

      {/* Tiers */}
      <div className="mb-12 grid gap-6 md:grid-cols-3">
        <TierCard
          name="Free"
          price="$0"
          description="Everything you need locally."
          features={[
            'All core CLI features',
            'Any AI provider (BYOK)',
            'REPL mode',
            'Query monitoring',
            'Local configuration',
          ]}
        />
        <TierCard
          name="Pro"
          price="$15/mo"
          description="For professional use and CI/CD."
          highlight
          features={[
            'Everything in Free',
            'basemake check in CI/CD',
            'Custom check policies',
            'Budget profiles',
            'Index recommendations',
            'Schema diffing',
            'Priority support',
          ]}
        />
        <TierCard
          name="Team"
          price="$29/user/mo"
          description="For teams and organizations."
          features={[
            'Everything in Pro',
            'Team Server mode',
            'Shared AI proxy & cache',
            'RBAC readonly enforcement',
            'Audit logging',
            'Slack / Teams integrations',
            'Dedicated support',
          ]}
        />
      </div>

      {/* License Keys */}
      <H2 id="license-keys">License Keys</H2>

      <P>
        Pro and Team plans require a license key. License keys are verified locally —
        basemake never phones home.
      </P>

      <div className="mb-6 overflow-x-auto rounded-xl border border-border/[0.06]">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="border-b border-border/[0.06] bg-muted/30 px-4 py-3 text-left font-semibold text-muted-foreground">Plan</th>
              <th className="border-b border-border/[0.06] bg-muted/30 px-4 py-3 text-left font-semibold text-muted-foreground">Key Format</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="border-b border-border/[0.04] px-4 py-3 text-muted-foreground">Pro</td>
              <td className="border-b border-border/[0.04] px-4 py-3 text-muted-foreground font-mono text-xs">bmk_pro_xxxxxxxxxxxxxxxx</td>
            </tr>
            <tr>
              <td className="border-b border-border/[0.04] px-4 py-3 text-muted-foreground">Team</td>
              <td className="border-b border-border/[0.04] px-4 py-3 text-muted-foreground font-mono text-xs">bmk_team_xxxxxxxxxxxxxxxx</td>
            </tr>
          </tbody>
        </table>
      </div>

      <P>To activate your license:</P>

      <CodeBlock lang="bash">{`basemake config set license_key bmk_pro_xxxxxxxxxxxxxxxx
# or via environment variable
export BASEMAKE_LICENSE_KEY=bmk_pro_xxxxxxxxxxxxxxxx`}</CodeBlock>

      <Tip>
        License keys work across all your machines. The same key can be used on your
        laptop, CI runners, and team server.
      </Tip>

      {/* HMAC Verification */}
      <Separator className="my-10 bg-muted/30" />

      <H2 id="hmac-verification">HMAC Verification</H2>

      <P>
        License keys are cryptographically signed using <strong>HMAC-SHA256</strong>.
        Each key contains an embedded payload with the plan tier, expiration date, and
        a signature that basemake verifies locally.
      </P>

      <P>
        Key structure (decoded):
      </P>

      <CodeBlock lang="text">{`bmk_pro_<base64url(payload + signature)>

Payload (JSON):
{
  "tier":    "pro",
  "sub":     "user@example.com",
  "exp":     1767225600,
  "iat":     1735689600
}

Signature: HMAC-SHA256(public_salt, payload)
}`}</CodeBlock>

      <P>
        This means:
      </P>
      <UL>
        <LI><strong className="text-foreground">No phone-home required</strong> — verification
        is fully offline.</LI>
        <LI><strong className="text-foreground">Tamper-proof</strong> — modified keys fail
        HMAC verification immediately.</LI>
        <LI><strong className="text-foreground">Time-bound</strong> — expired keys are rejected
        with a clear message.</LI>
      </UL>

      {/* Grace Period */}
      <Separator className="my-10 bg-muted/30" />

      <H2 id="grace-period">License Expiry & Grace Period</H2>

      <P>
        When a Pro or Team license expires, basemake enters a <strong>14-day grace
        period</strong>. During this time:
      </P>

      <UL>
        <LI>All Pro/Team features continue to work normally.</LI>
        <LI>A warning is printed on each invocation with the remaining grace days.</LI>
        <LI>You can renew your license at any time to remove the warning.</LI>
      </UL>

      <P>
        After the grace period, Pro/Team features are disabled and basemake reverts to
        Free mode. No data is lost — renewing your license restores full functionality.
      </P>

      <Warn>
        The grace period is designed to prevent service interruption. If your license
        expires while you're mid-project, you won't lose access to your data or queries.
      </Warn>

      {/* CI/CD */}
      <Separator className="my-10 bg-muted/30" />

      <H2 id="ci-cd-licensing">Licensing for CI/CD</H2>

      <P>
        CI/CD pipelines using <Code>basemake check</Code> require a Pro or Team license.
        Set the license key as an environment variable in your CI provider:
      </P>

      <CodeBlock lang="bash">{`# GitHub Actions / GitLab CI / CircleCI
export BASEMAKE_LICENSE_KEY=bmk_pro_xxxxxxxxxxxxxxxx

# Or pass it directly
basemake check --dir ./migrations --license=bmk_pro_xxxxxxxxxxxxxxxx`}</CodeBlock>

      <P>
        See the{' '}
        <Link to="/docs/ci-cd" className="text-[#ff3131] hover:underline">
          CI/CD Integration page
        </Link>{' '}
        for complete setup instructions.
      </P>

      {/* FAQ */}
      <Separator className="my-10 bg-muted/30" />

      <Card className="border-border/[0.06] bg-muted/30">
        <CardHeader>
          <CardTitle className="text-foreground text-lg">Licensing FAQ</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4 text-sm text-muted-foreground">
          <div>
            <p className="mb-1 font-semibold text-foreground">Can I use my license on multiple machines?</p>
            <p>Yes. Your license key works on any machine you control — laptop, desktop,
            CI runners, and servers.</p>
          </div>
          <Separator className="bg-muted/30" />
          <div>
            <p className="mb-1 font-semibold text-foreground">What happens if my payment fails?</p>
            <p>You enter the 14-day grace period. We'll send you reminder emails. If the
            issue isn't resolved, basemake drops to Free mode after 14 days.</p>
          </div>
          <Separator className="bg-muted/30" />
          <div>
            <p className="mb-1 font-semibold text-foreground">Do you offer student or open-source discounts?</p>
            <p>Yes. Contact us at <Code>license@basemake.dev</Code> with your
            situation — we offer free Pro licenses for qualifying students and
            open-source projects.</p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
