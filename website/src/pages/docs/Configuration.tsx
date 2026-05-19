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

/* ================================================================== */
/*  PAGE — Configuration                                               */
/* ================================================================== */
export default function Configuration() {
  return (
    <div className="pb-24">
      {/* Header */}
      <div className="mb-10">
        <Badge variant="outline" className="mb-3 border-[#e63946]/30 text-[#e63946] text-xs tracking-wide uppercase">
          Reference
        </Badge>
        <h1 className="text-4xl font-bold tracking-tight text-white sm:text-5xl">
          Configuration
        </h1>
        <p className="mt-3 text-lg text-white/50">
          Configure basemake via the CLI, config file, or environment variables.
        </p>
      </div>

      <Separator className="mb-10 bg-white/[0.04]" />

      {/* Config CLI */}
      <H2 id="config-cli">basemake config CLI</H2>

      <P>
        The <Code>basemake config</Code> command is the primary way to view and modify
        settings. Changes take effect immediately — no restart required.
      </P>

      <CodeBlock lang="bash">{`# List all settings
basemake config list

# Get a specific value
basemake config get ai_provider

# Set a value
basemake config set ai_provider openai

# Unset a value (revert to default)
basemake config unset ai_api_key`}</CodeBlock>

      {/* Config File */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="config-file">Config File Location</H2>

      <P>
        basemake stores its configuration in a YAML file. The default location depends on
        your operating system:
      </P>

      <div className="mb-6 overflow-x-auto rounded-xl border border-white/[0.06]">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Platform</th>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Path</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Linux</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-mono text-xs">~/.config/basemake/config.yaml</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">macOS</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-mono text-xs">~/Library/Application Support/basemake/config.yaml</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Windows</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-mono text-xs">%APPDATA%\basemake\config.yaml</td>
            </tr>
          </tbody>
        </table>
      </div>

      <P>
        Override the path with the <Code>--config</Code> flag or the{' '}
        <Code>BASEMAKE_CONFIG</Code> environment variable:
      </P>
      <CodeBlock lang="bash">{`basemake --config /path/to/custom.yaml init
export BASEMAKE_CONFIG=/path/to/custom.yaml`}</CodeBlock>

      {/* Environment Variables */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="env-vars">Environment Variables</H2>

      <P>
        All configuration keys can be set via environment variables. Environment variables
        take precedence over values in the config file.
      </P>

      <div className="mb-6 overflow-x-auto rounded-xl border border-white/[0.06]">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Variable</th>
              <th className="border-b border-white/[0.06] bg-white/[0.03] px-4 py-3 text-left font-semibold text-white/70">Description</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-mono text-xs">BASEMAKE_LICENSE_KEY</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">License key for Pro/Team</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-mono text-xs">BASEMAKE_CONFIG</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Config file path override</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-mono text-xs">OPENAI_API_KEY</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">OpenAI API key (auto-detected)</td>
            </tr>
            <tr>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60 font-mono text-xs">ANTHROPIC_API_KEY</td>
              <td className="border-b border-white/[0.04] px-4 py-3 text-white/60">Anthropic API key (auto-detected)</td>
            </tr>
          </tbody>
        </table>
      </div>

      {/* Key Settings */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="key-settings">Key Settings Reference</H2>

      <Card className="mb-6 border-white/[0.06] bg-white/[0.02]">
        <CardContent className="pt-6">
          <dl className="space-y-6 text-sm">
            <div>
              <dt className="mb-1 font-semibold text-white">license_key</dt>
              <dd className="text-white/60">
                Your basemake license key. Required for Pro features and Team Server.
                Format: <Code>bmk_pro_xxx</Code> or <Code>bmk_team_xxx</Code>.
                Set via <Code>basemake config set license_key bmk_pro_xxx</Code>.
              </dd>
            </div>
            <Separator className="bg-white/[0.04]" />
            <div>
              <dt className="mb-1 font-semibold text-white">ai_provider</dt>
              <dd className="text-white/60">
                The AI provider to use. One of: <Code>openai</Code>,{' '}
                <Code>anthropic</Code>, <Code>opencode</Code>, <Code>ollama</Code>.
                Default: detected during <Code>basemake init</Code>.
              </dd>
            </div>
            <Separator className="bg-white/[0.04]" />
            <div>
              <dt className="mb-1 font-semibold text-white">ai_model</dt>
              <dd className="text-white/60">
                The model identifier for the selected provider. Examples:{' '}
                <Code>gpt-4o</Code>, <Code>claude-sonnet-4-20250514</Code>,{' '}
                <Code>llama3.2:latest</Code>.
              </dd>
            </div>
            <Separator className="bg-white/[0.04]" />
            <div>
              <dt className="mb-1 font-semibold text-white">ai_api_key</dt>
              <dd className="text-white/60">
                Your API key for the selected provider. Stored in the config file.
                Can also be set via provider-specific environment variables.
              </dd>
            </div>
            <Separator className="bg-white/[0.04]" />
            <div>
              <dt className="mb-1 font-semibold text-white">default_connection</dt>
              <dd className="text-white/60">
                The connection string for your default database. Set automatically by{' '}
                <Code>basemake connect</Code>.
              </dd>
            </div>
            <Separator className="bg-white/[0.04]" />
            <div>
              <dt className="mb-1 font-semibold text-white">ollama_host</dt>
              <dd className="text-white/60">
                Ollama server URL. Default: <Code>http://localhost:11434</Code>.
                Override to use a remote Ollama instance.
              </dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      {/* Connection Management */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="connections">Connection Management</H2>

      <P>
        basemake can manage multiple database connections. Switch between them without
        re-entering credentials.
      </P>

      <CodeBlock lang="bash">{`# Add a named connection
basemake connect --name=staging postgres://user@staging:5432/db
basemake connect --name=production postgres://user@prod:5432/db

# List connections
basemake config list | grep connection

# Switch active connection
basemake connect --name=production

# Remove a connection
basemake config unset connection.production`}</CodeBlock>

      <Tip>
        Connection names are stored in the config file under a <Code>connections</Code>{' '}
        key. The <Code>default_connection</Code> setting controls which one is active.
      </Tip>

      {/* Sample config file */}
      <Separator className="my-10 bg-white/[0.04]" />

      <H2 id="example-config">Example Config File</H2>

      <P>A typical <Code>config.yaml</Code> looks like this:</P>

      <CodeBlock lang="yaml">{`license_key: ""
ai_provider: openai
ai_model: gpt-4o
ai_api_key: sk-...
default_connection: postgres://user@localhost:5432/mydb

connections:
  staging:
    url: postgres://user@staging:5432/db
  production:
    url: postgres://user@prod:5432/db

ollama_host: http://localhost:11434`}</CodeBlock>
    </div>
  )
}
