# Payment & Licensing Flow

> **How customers go from paying on Lemon Squeezy to running `basemake config set key`**

---

## Architecture Overview

```
Customer
   │
   ▼
Lemon Squeezy (Checkout)
   │
   │  POST /api/license  (webhook: order_created / subscription_created)
   │  + HMAC-SHA256 signature in x-signature header
   ▼
Vercel Serverless Function  (/api/license.js)
   │
   │  1. Verify webhook HMAC-SHA256 signature
   │  2. Extract email + variant_id
   │  3. Map variant_id → tier (pro / team)
   │  4. Generate license key: bmk_<tier>_<base64url(email)>_<HMAC hex>
   │  5. Send email via Resend
   │  6. Return 200 to Lemon Squeezy
   ▼
Customer's Inbox ← Resend ←─ "Your basemake Pro license key"
   │
   ▼
  basemake config set key bmk_pro_dXNlckBleGFtcGxlLmNvbQ_abc123...
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Symmetric HMAC (not asymmetric)** | Simpler to implement across Go + Node; secret compiled into CLI binary and set as Vercel env var. v2 will migrate to Ed25519. |
| **No expiry in key** | Keys are perpetual; expiry managed server-side by re-issuance. The CLI validates the HMAC signature only — no online check required. |
| **Webhook returns 200 on errors** | Prevents Lemon Squeezy from retrying webhooks endlessly. The license key is logged and can be manually re-sent. |
| **Email fallback in response** | If Resend fails, the key is returned in the webhook response body so Lemon Squeezy order metadata can capture it. |

---

## License Key Format

```
bmk_<tier>_<base64url(email)>_<HMAC-SHA256 hex>
```

| Part | Encoding | Example |
|------|----------|---------|
| `bmk` | Literal | `bmk` |
| `tier` | Literal | `pro` or `team` |
| `email` | Base64 URL‑safe (no padding) | `dXNlckBleGFtcGxlLmNvbQ` |
| `signature` | HMAC-SHA256 hex | `a1b2c3d4e5f6...` |

### Generation Algorithm (identical in Go and Node.js)

```
payload   = tier + ":" + email               // e.g. "pro:user@example.com"
sig       = HMAC-SHA256(secret, payload)     // hex-encoded
encoded   = base64url(email)                  // raw URL-safe, no padding
key       = "bmk_" + tier + "_" + encoded + "_" + sig
```

### Validation Algorithm (Go CLI side)

```
parts = Split(key, "_")                      // expects exactly 4 parts
Check: parts[0] == "bmk"
Check: parts[1] ∈ {"pro", "team"}
Decode: email = base64url(parts[2])
Check: HMAC-SHA256(secret, parts[1] + ":" + email) == parts[3]
```

### Why This Format

- **Self-contained**: No server round-trip needed for validation. The CLI verifies offline.
- **Tamper-proof**: HMAC prevents forging or modifying tier/email without the secret.
- **User-identifiable**: The embedded base64 email lets support see who a key belongs to.
- **Tier-scoped**: Different prefixes (`bmk_pro_` vs `bmk_team_`) make keys visually distinguishable.

---

## HMAC Secret Management

The same secret **must** exist in two places:

### 1. Compiled into the CLI Binary

```go
// internal/license/license.go
var hmacSecret = []byte("bmk-v1-2026-secret-change-in-production")
```

Set at build time via `-ldflags`:

```bash
go build -ldflags "-X github.com/DynamicKarabo/basemake/internal/license.hmacSecret=$BASEMAKE_LICENSE_SECRET" -o basemake
```

### 2. Vercel Environment Variable

```bash
vercel env add BASEMAKE_LICENSE_SECRET
```

Set to the **same** value as the build-time secret. The Node.js handler reads it from `process.env.BASEMAKE_LICENSE_SECRET`.

### Production Checklist

- Generate a **unique, random** secret for production — do not use the default dev value.
- Rotate the secret at least annually.
- Old binaries will fail validation after rotation — users must upgrade.
- The secret is sensitive: never commit it to Git, never log it.

---

## Pricing Tiers

| Tier | Price | License Key Required | Unlocked Features |
|------|-------|---------------------|-------------------|
| **Free** | $0 | No | Diagnosis only: `.explain`, `.analyze` (read‑only), full TUI, NL→SQL, query execution |
| **Pro** | $15/mo ($150/yr) | `bmk_pro_...` | `basemake check` (CI/CD gate), `basemake budget` (policy as code), `basemake watch` (monitoring), `basemake diff` (schema drift), `basemake index apply` |
| **Team** | $39/seat/mo | `bmk_team_...` | Everything in Pro + `basemake server` (team sync, shared AI proxy, RBAC, audit log, Slack/Teams integration) |
| **Enterprise** | Custom | Custom | On-prem deployment, SSO/SAML, custom AI proxy, dedicated support |

### Feature-to-Tier Mapping (Go)

```go
var tierFeatures = map[Tier][]Feature{
    TierPro: {
        FeatureCheck,      // basemake check
        FeatureBudget,     // basemake budget
        FeatureWatch,      // basemake watch
        FeatureDiff,       // basemake diff
        FeatureIndexApply, // basemake index apply
    },
    TierTeam: {
        FeatureCheck,
        FeatureBudget,
        FeatureWatch,
        FeatureDiff,
        FeatureIndexApply,
        FeatureServer,    // basemake server (team sync)
    },
}
```

---

## Vercel Environment Variables

These must be set in the Vercel project dashboard (or via `vercel env add`):

| Variable | Purpose | Required |
|----------|---------|----------|
| `BASEMAKE_LICENSE_SECRET` | HMAC secret for signing license keys — must match binary build-time secret | ✅ |
| `RESEND_API_KEY` | API key from [resend.com](https://resend.com) for sending license emails | ✅ |
| `LEMON_SQUEEZY_WEBHOOK_SECRET` | Webhook signing secret from Lemon Squeezy Store Settings | ✅ |
| `LS_VARIANT_PRO` | Lemon Squeezy variant ID for the Pro tier (default `"1"`) | Optional |
| `LS_VARIANT_TEAM` | Lemon Squeezy variant ID for the Team tier (default `"2"`) | Optional |

### Variant → Tier Mapping (Node.js)

```javascript
const VARIANT_PRO = process.env.LS_VARIANT_PRO || "1";
const VARIANT_TEAM = process.env.LS_VARIANT_TEAM || "2";

function tierFromVariant(variantId) {
    if (variantId === VARIANT_TEAM) return "team";
    return "pro";
}
```

---

## Webhook Endpoint

**URL:** `https://website-eight-plum-77.vercel.app/api/license`

**Method:** `POST`

**Content-Type:** `application/json`

### What It Handles

- `order_created` — one-time purchases (Pro yearly, etc.)
- `subscription_created` — recurring subscriptions (Team monthly)

All other event types are acknowledged with `200 { received: true }` and ignored.

### Request Flow

1. **Signature Verification** — The raw request body is HMAC-SHA256 signed with the Lemon Squeezy webhook secret. The signature is in the `x-signature` header. If verification fails, returns `401`.

2. **Email Extraction** — Tries multiple fields in order: `user_email`, `email`, `customer_email`, `attributes.customer.email`.

3. **Tier Resolution** — Maps `variant_id` via `tierFromVariant()` using the configured env vars.

4. **Key Generation** — Calls `generateLicenseKey(tier, email, secret)` — identical algorithm to Go's `license.Generate()`.

5. **Email Delivery** — Sends via Resend with HTML + plaintext templates. If email fails, the key is returned in the JSON response body so it can be retrieved from Lemon Squeezy's webhook logs.

6. **Response** — Always returns `200` (even on errors) to prevent Lemon Squeezy retries.

### Response Format

```json
{
    "received": true,
    "license_key": "bmk_pro_dXNlckBleGFtcGxlLmNvbQ_a1b2c3...",
    "email": "user@example.com",
    "tier": "pro",
    "email_delivered": true
}
```

---

## Email Template

Sent by the Vercel function via Resend. The template includes:

- **License key** displayed in a monospace box
- **Activation command**: `basemake config set key <license_key>`
- **Quickstart link**: `https://basemake.dev/docs/quickstart`
- Plaintext fallback for text-only mail clients

**From address:** `basemake <keys@basemake.dev>`

**Subject:** `Your basemake Pro license key` / `Your basemake Team license key`

---

## Setup Steps

### 1. Create Lemon Squeezy Products

1. Log in to [Lemon Squeezy](https://lemonsqueezy.com).
2. Create two products (or one with multiple variants):
   - **basemake Pro** — $15/month (variant ID becomes `LS_VARIANT_PRO`)
   - **basemake Team** — $39/seat/month (variant ID becomes `LS_VARIANT_TEAM`)
3. Note the variant IDs from the Lemon Squeezy dashboard.

### 2. Set Vercel Environment Variables

```bash
vercel env add BASEMAKE_LICENSE_SECRET
vercel env add RESEND_API_KEY
vercel env add LEMON_SQUEEZY_WEBHOOK_SECRET
vercel env add LS_VARIANT_PRO
vercel env add LS_VARIANT_TEAM
```

Or set them in the Vercel Dashboard → Project → Settings → Environment Variables.

### 3. Configure the Webhook in Lemon Squeezy

1. Go to **Store Settings → Webhooks** in Lemon Squeezy.
2. Add endpoint: `https://website-eight-plum-77.vercel.app/api/license`
3. Select events: **`order_created`** and **`subscription_created`** (at minimum).
4. Lemon Squeezy will generate a webhook signing secret — copy it to `LEMON_SQUEEZY_WEBHOOK_SECRET`.
5. Click **Save**.

### 4. Deploy

```bash
cd website
vercel --prod
```

### 5. Test

1. Place a test order in Lemon Squeezy (use a test card — Lemon Squeezy provides test card numbers).
2. Check the Vercel function logs for the generated key.
3. Verify the customer receives the email.
4. Test that `basemake config set key <key>` works with the CLI.

---

## File Reference

| File | Language | Purpose |
|------|----------|---------|
| `website/api/license.js` | Node.js | Vercel serverless function for webhook → key generation → email |
| `website/api/license.test.js` | Node.js | 31 tests covering key format, tier mapping, edge cases, webhook verification, Go-Node cross-compatibility |
| `internal/license/license.go` | Go | License key generation (`Generate`) and client-side validation (`Validate`) |
| `internal/license/license_test.go` | Go | Unit tests for Go license logic |
| `cmd/license.go` | Go | CLI command: `basemake license` (show status / activate key) |

### Test Coverage (license.test.js)

The test file runs 31 assertions across these groups:

1. **Key format** — prefix, underscore-delimited parts (4), HMAC hex correctness
2. **Tier mapping** — `bmk_pro_` vs `bmk_team_` prefix, deterministic output
3. **Edge cases** — emails with `+` (subaddressing), dots, special characters
4. **Webhook verification** — valid signature passes, invalid/empty/wrong-secret fails
5. **Go compatibility** — cross-check against known inputs confirming Node produces identical keys to the Go implementation

### Go-Node Cross-Compatibility

Both implementations use the same algorithm:

- Payload: `tier + ":" + email`
- HMAC: SHA-256, hex-encoded
- Email encoding: `base64.RawURLEncoding` (Go) = `Buffer.from(email).toString("base64url")` (Node)

A key generated by the Vercel Node.js handler can be validated by the Go CLI binary, and vice versa — **as long as the same HMAC secret is used in both places.**

---

## Security Considerations

- **Symmetric secret**: Anyone who extracts the secret from the CLI binary can forge license keys. This is acceptable for v1 because:
  - The binary is open-source — the default dev secret is already public.
  - Production binaries use a different secret set at build time.
  - v2 will move to Ed25519 asymmetric signatures.
- **No online validation**: The CLI never phones home. Revocation requires re-issuing keys with a new secret and forcing a CLI update.
- **Webhook signing**: Lemon Squeezy signs all webhooks. Always verify the signature — do not trust unauthenticated requests.
- **HTTPS**: The Vercel endpoint uses HTTPS automatically. No additional TLS configuration needed.
