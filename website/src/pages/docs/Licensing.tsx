import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export default function Licensing() {
  return (
    <div className="prose prose-neutral dark:prose-invert max-w-3xl mx-auto">
      <h1>License Verification</h1>

      <p>
        basemake uses offline-verifiable HMAC-SHA256 signed license keys to gate
        certain features. The verification code lives in <code>internal/license/license.go</code>.
      </p>

      <h2>Key Format</h2>

      <pre>{`bmk_<tier>_<base64(email)>_<hmac-hex-signature>`}</pre>

      <table>
        <thead>
          <tr><th>Part</th><th>Description</th></tr>
        </thead>
        <tbody>
          <tr><td><code>bmk</code></td><td>Literal prefix</td></tr>
          <tr><td><code>tier</code></td><td>Feature tier (<code>pro</code> or <code>team</code>)</td></tr>
          <tr><td><code>email</code></td><td>Base64 URL-safe encoded email</td></tr>
          <tr><td><code>signature</code></td><td>HMAC-SHA256 hex digest of <code>tier:email</code></td></tr>
        </tbody>
      </table>

      <h2>Verification</h2>

      <p>
        Keys are validated entirely offline — no phone-home, no server call. The CLI
        parses the key, recomputes the HMAC with a compiled-in secret, and compares signatures.
        If the key is tampered with (tier upgrade, email change), the signature won't match.
      </p>

      <p>
        The HMAC secret is a Go <code>var</code> that can be overridden at build time via
        <code>-ldflags</code>:
      </p>

      <pre>{`go build -ldflags "-X github.com/karabo-labs/basemake/internal/license.hmacSecret=$SECRET"`}</pre>

      <h2>API Endpoint</h2>

      <p>
        A Vercel serverless function at <code>website/api/license.js</code> handles
        Lemon Squeezy webhooks and generates license keys on purchase. It shares the
        same HMAC algorithm as the Go CLI, ensuring cross-compatibility.
      </p>

      <p>
        The function expects <code>BASEMAKE_LICENSE_SECRET</code> and
        <code>LEMON_SQUEEZY_WEBHOOK_SECRET</code> environment variables.
      </p>
    </div>
  )
}
