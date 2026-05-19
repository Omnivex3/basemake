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

function Warn({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-6 rounded-xl border border-amber-500/20 bg-amber-500/5 px-5 py-4 text-sm text-amber-300/80">
      {children}
    </div>
  )
}

/* ================================================================== */
/*  PAGE — AI Providers                                                */
/* ================================================================== */
export default function AIProviders() {
  return (
    <div className="pb-24">
      {/* Header */}
      <div className="mb-10">
        <Badge variant="outline" className="mb-3 border-[#e63946]/30 text-[#e63946] text-xs tracking-wide uppercase">
          Configuration
        </Badge>
        <h1 className="text-4xl font-bold tracking-tight text-white sm:text-5xl">
          AI Providers
        </h1>
        <p className="mt-3 text-lg text-white/50">
          basemake uses a bring-your-own-key (BYOK) model — you choose which AI provider
          powers your queries, and your data never leaves your infrastructure.
        </p>
      </div>

      <Separator className="mb-10 bg-white/[0.04]" />

      {/* BYOK Overview */}
      <Card className="mb-10 border-white/[0.06] bg-white/[0.02]">
        <CardHeader>
          <CardTitle className="text-white text-lg flex items-center gap-2">
            🔑 Bring Your Own Key
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-white/60">
          <p>
            basemake does not charge per-token or bundle AI credits. You bring your own
            API key from the provider of your choice, and basemake uses it to translate
            natural language into SQL. This means:
          </p>
          <UL>
            <LI><strong className="text-white">No vendor lock-in</strong> — switch
            providers with a single config command.</LI>
            <LI><strong className="text-white">Use existing credits</strong> — if you
            already have OpenAI or Anthropic credits, they work here.</LI>
            <LI><strong className="text-white">Run entirely offline</strong> — with
            Ollama, every query stays on your machine.</LI>
            <LI><strong className="text-white">Team savings</strong> — the{' '}
            <Link to="/docs/team-server" className="text-[#e63946] hover:underline">Team Server</Link>{' '}
            proxies AI requests through a shared cache, cutting costs by 40–60%.</LI>
          </UL>
        </CardContent>
      </Card>

      {/* Provider: OpenAI */}
      <H2 id="openai">OpenAI</H2>
      <P>
        Use OpenAI models including GPT-4o, GPT-4o-mini, and o-series reasoning models.
        Set your API key and model via the config CLI.
      </P>
      <CodeBlock lang="bash">{`basemake config set ai_provider openai
basemake config set ai_model gpt-4o
basemake config set ai_api_key sk-...`}</CodeBlock>
      <Tip>
        You can also set the <Code>OPENAI_API_KEY</Code> environment variable — basemake
        picks it up automatically during <Code>basemake init</Code>.
      </Tip>

      {/* Provider: Anthropic */}
      <H2 id="anthropic">Anthropic</H2>
      <P>
        Access Claude 4 Sonnet, Claude 3.5 Sonnet, and other Anthropic models. Known for
        strong SQL generation and multi-step reasoning.
      </P>
      <CodeBlock lang="bash">{`basemake config set ai_provider anthropic
basemake config set ai_model claude-sonnet-4-20250514
basemake config set ai_api_key sk-ant-...`}</CodeBlock>
      <P>
        Recommended models: <Code>claude-sonnet-4-20250514</Code> (best overall),{' '}
        <Code>claude-3-5-sonnet-20241022</Code>, <Code>claude-3-haiku-20240307</Code> (fast/low-cost).
      </P>

      {/* Provider: OpenCode */}
      <H2 id="opencode">OpenCode</H2>
      <P>
        Connect to OpenCode's inference endpoint for code-specialized models. A great
        middle ground between fully local and paid cloud APIs.
      </P>
      <CodeBlock lang="bash">{`basemake config set ai_provider opencode
basemake config set ai_model opencode/gpt-4o
basemake config set ai_api_key oc-...`}</CodeBlock>

      {/* Provider: Ollama */}
      <H2 id="ollama">Ollama (Local)</H2>
      <P>
        Run models entirely on your machine using Ollama. No data ever leaves your
        laptop — every query is processed locally. Perfect for sensitive data environments
        and offline use.
      </P>
      <CodeBlock lang="bash">{`# Install Ollama first: https://ollama.com
ollama pull llama3.2:latest

# Then configure basemake
basemake config set ai_provider ollama
basemake config set ai_model llama3.2:latest
basemake config set ai_api_key ""  # not needed for local`}</CodeBlock>

      <Warn>
        Local models have smaller context windows and may produce less accurate SQL for
        complex schemas. We recommend a model with at least 7B parameters for production
        use.
      </Warn>

      <P>
        Ollama endpoints: basemake connects to <Code>http://localhost:11434</Code> by
        default. Override with <Code>basemake config set ollama_host http://192.168.1.50:11434</Code>.
      </P>

      {/* Model Selection Guide */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="model-selection">Model Selection Guide</H2>

      <P>
        Different models excel at different tasks. Here's a quick guide:
      </P>

      <div className="mb-6 overflow-x-auto rounded-xl border border-white/[0.06]">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">
                Use Case
              </th>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">
                Recommended Model
              </th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Complex queries, large schemas</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Claude 4 Sonnet or GPT-4o</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Fast iteration / ad-hoc</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">GPT-4o-mini or Claude 3 Haiku</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Offline / sensitive data</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Llama 3.2 (via Ollama)</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">CI/CD pipelines</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">GPT-4o-mini (fast + cheap)</td>
            </tr>
          </tbody>
        </table>
      </div>

      {/* Switching providers */}
      <H2 id="switching">Switching Providers</H2>
      <P>
        You can switch providers at any time. basemake stores each provider's configuration
        independently, so switching back doesn't require re-entry.
      </P>
      <CodeBlock lang="bash">{`basemake config set ai_provider anthropic
# API key and model for Anthropic are already saved

# Switch back to OpenAI
basemake config set ai_provider openai`}</CodeBlock>

      <P>
        See the{' '}
        <Link to="/docs/configuration" className="text-[#e63946] hover:underline">
          Configuration page
        </Link>{' '}
        for all available settings.
      </P>
    </div>
  )
}
