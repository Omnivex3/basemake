/**
 * /api/license.js — Lemon Squeezy webhook → license key generator
 *
 * Flow:
 *   1. Lemon Squeezy sends webhook on order/subscription creation
 *   2. Verify HMAC-SHA256 signature (LS signs all webhooks)
 *   3. Extract email + variant_id → determine tier (pro/team)
 *   4. Generate HMAC license key (same algorithm as Go internal/license)
 *   5. Email the key to the customer via Resend
 *   6. Return 200
 *
 * Env vars required on Vercel:
 *   BASEMAKE_LICENSE_SECRET     — same secret compiled into the CLI binary
 *   LEMON_SQUEEZY_WEBHOOK_SECRET — from Lemon Squeezy store settings
 *   RESEND_API_KEY              — from resend.com
 *
 * Lemon Squeezy variant → tier mapping:
 *   Set these as env vars: LS_VARIANT_PRO, LS_VARIANT_TEAM
 *   Defaults: variant_pro=1, variant_team=2
 */

import crypto from "crypto";
import { Resend } from "resend";

// ── Tier mapping ──

const VARIANT_PRO = process.env.LS_VARIANT_PRO || "1";
const VARIANT_TEAM = process.env.LS_VARIANT_TEAM || "2";

function tierFromVariant(variantId) {
  if (variantId === VARIANT_TEAM) return "team";
  return "pro"; // default — also covers VARIANT_PRO
}

// ── License key generation (mirrors Go internal/license.Generate) ──

/**
 * Generate a basemake license key identical to the Go implementation.
 *
 * Go source:
 *   payload = fmt.Sprintf("%s:%s", tier, email)
 *   mac = HMAC-SHA256(secret, payload)
 *   sig = hex(mac)
 *   encodedEmail = base64.RawURLEncoding(email)
 *   key = "bmk_<tier>_<encodedEmail>_<sig>"
 */
function generateLicenseKey(tier, email, secret) {
  const payload = `${tier}:${email}`;
  const hmac = crypto.createHmac("sha256", secret).update(payload, "utf-8").digest("hex");
  const encodedEmail = Buffer.from(email, "utf-8").toString("base64url");
  return `bmk_${tier}_${encodedEmail}_${hmac}`;
}

// ── Webhook signature verification ──

/**
 * Lemon Squeezy signs webhooks with HMAC-SHA256.
 * Signature is in the x-signature header.
 * The raw request body is the signing payload.
 */
function verifyWebhookSignature(rawBody, signature, secret) {
  if (!signature || !secret) return false;
  const expected = crypto.createHmac("sha256", secret).update(rawBody, "utf-8").digest("hex");
  // Constant-time comparison
  return crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(signature));
}

// ── Email templates ──

function buildEmailHtml({ email, licenseKey, tier }) {
  const tierLabel = tier === "team" ? "Team" : "Pro";
  const activateCmd =
    tier === "team"
      ? `basemake config set key ${licenseKey}`
      : `basemake config set key ${licenseKey}`;

  return `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; color: #111; line-height: 1.6; max-width: 560px; margin: 0 auto; padding: 32px 24px; }
    .logo { font-size: 20px; font-weight: 700; margin-bottom: 24px; }
    .logo b { color: #FC0E22; }
    h1 { font-size: 24px; font-weight: 700; margin: 0 0 8px; }
    p { margin: 0 0 16px; color: #444; }
    .key-box { background: #f5f5f5; border: 1px solid #e0e0e0; border-radius: 8px; padding: 16px; font-family: 'SF Mono', 'Menlo', monospace; font-size: 13px; word-break: break-all; margin: 16px 0; }
    .cmd-box { background: #111; color: #e0e0e0; border-radius: 8px; padding: 12px 16px; font-family: 'SF Mono', 'Menlo', monospace; font-size: 14px; margin: 12px 0; }
    .btn { display: inline-block; background: #111; color: #fff; text-decoration: none; padding: 10px 20px; border-radius: 6px; font-size: 14px; font-weight: 500; margin: 8px 0; }
    hr { border: none; border-top: 1px solid #e0e0e0; margin: 24px 0; }
    .footer { font-size: 12px; color: #888; }
  </style>
</head>
<body>
  <div class="logo"><b>base</b>make</div>

  <h1>You're in, ${tierLabel}.</h1>
  <p>Here's your license key and how to activate it.</p>

  <div class="key-box">${licenseKey}</div>

  <p style="font-weight: 600; margin-bottom: 4px;">Activate with one command:</p>
  <div class="cmd-box">${activateCmd}</div>

  <p>Or copy it manually:<br>
  <code>basemake config set key ${licenseKey}</code></p>

  <a href="https://basemake.dev/docs/quickstart" class="btn">Quickstart Guide →</a>

  <hr>

  <div class="footer">
    <p>Questions? Reply to this email.</p>
    <p>basemake — database tools that don't suck.</p>
  </div>
</body>
</html>`;
}

function buildEmailText({ email, licenseKey, tier }) {
  const activateCmd = `basemake config set key ${licenseKey}`;
  return `You're in, ${tier}.\n\nHere's your license key:\n${licenseKey}\n\nActivate with one command:\n${activateCmd}\n\nOr visit: https://basemake.dev/docs/quickstart\n\n— basemake`;
}

// ── Main handler ──

export default async function handler(req, res) {
  // Only accept POST
  if (req.method !== "POST") {
    return res.status(405).json({ error: "Method not allowed" });
  }

  // Only accept JSON content type
  const contentType = req.headers["content-type"] || "";
  if (!contentType.includes("application/json")) {
    return res.status(400).json({ error: "Content-Type must be application/json" });
  }

  try {
    // Get raw body for signature verification
    const rawBody = typeof req.body === "string" ? req.body : JSON.stringify(req.body);
    const signature = req.headers["x-signature"];

    // Verify webhook signature
    const webhookSecret = process.env.LEMON_SQUEEZY_WEBHOOK_SECRET;
    if (webhookSecret && !verifyWebhookSignature(rawBody, signature, webhookSecret)) {
      console.warn("Invalid webhook signature");
      return res.status(401).json({ error: "Invalid signature" });
    }

    const event = typeof req.body === "string" ? JSON.parse(req.body) : req.body;

    // We handle: order_created, subscription_created
    const eventName = event.meta?.event_name || event.type || "";
    if (!eventName.includes("order_created") && !eventName.includes("subscription_created")) {
      // Acknowledge other events silently
      return res.status(200).json({ received: true });
    }

    const data = event.data;
    const attributes = data?.attributes || {};

    // Extract customer info
    const email =
      attributes.user_email ||
      attributes.email ||
      attributes.customer_email ||
      data.attributes?.customer?.email ||
      "";

    if (!email) {
      console.error("No email in webhook payload", JSON.stringify(attributes).slice(0, 500));
      return res.status(200).json({ error: "No email address" }); // 200 to avoid LS retry spam
    }

    // Determine tier from variant/product
    const variantId = String(attributes.variant_id || attributes.variantId || "");
    const tier = tierFromVariant(variantId);
    const licenseSecret = process.env.BASEMAKE_LICENSE_SECRET;

    if (!licenseSecret) {
      console.error("BASEMAKE_LICENSE_SECRET not set");
      return res.status(500).json({ error: "Server misconfiguration" });
    }

    // Generate key
    const licenseKey = generateLicenseKey(tier, email, licenseSecret);

    console.log(`Generated ${tier} key for ${email}: bmk_${tier}_${Buffer.from(email).toString("base64url").slice(0, 12)}...`);

    // Send email via Resend
    const resendApiKey = process.env.RESEND_API_KEY;
    if (resendApiKey) {
      try {
        const resend = new Resend(resendApiKey);
        await resend.emails.send({
          from: "basemake <keys@basemake.dev>",
          to: email,
          subject: `Your basemake ${tier === "team" ? "Team" : "Pro"} license key`,
          html: buildEmailHtml({ email, licenseKey, tier }),
          text: buildEmailText({ email, licenseKey, tier }),
        });
        console.log(`Email sent to ${email}`);
      } catch (emailErr) {
        // Don't fail the request if email fails — log and respond with key anyway
        console.error("Failed to send email:", emailErr.message);
        // Return the key in the response as fallback
        return res.status(200).json({
          received: true,
          license_key: licenseKey,
          email,
          tier,
          email_delivered: false,
        });
      }
    }

    return res.status(200).json({
      received: true,
      license_key: licenseKey,
      email,
      tier,
      email_delivered: true,
    });
  } catch (err) {
    console.error("Webhook handler error:", err);
    return res.status(200).json({ error: err.message }); // 200 to avoid LS retry
  }
}
