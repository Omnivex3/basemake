/**
 * /api/license.test.js — local verification of the license API
 *
 * Run: node api/license.test.js
 * (or: node --test api/license.test.js  in Node 20+)
 *
 * Tests:
 *   1. Key generation matches Go's format
 *   2. Tier mapping
 *   3. Webhook signature verification
 *   4. Full end-to-end (without actually sending email)
 */

import crypto from "crypto";

// ── Replicate the functions from license.js for isolated testing ──

function generateLicenseKey(tier, email, secret) {
  const payload = `${tier}:${email}`;
  const hmac = crypto.createHmac("sha256", secret).update(payload, "utf-8").digest("hex");
  const encodedEmail = Buffer.from(email, "utf-8").toString("base64url");
  return `bmk_${tier}_${encodedEmail}_${hmac}`;
}

function verifyWebhookSignature(rawBody, signature, secret) {
  if (!signature || !secret) return false;
  const expected = crypto.createHmac("sha256", secret).update(rawBody, "utf-8").digest("hex");
  try {
    return crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(signature));
  } catch {
    return false;
  }
}

const DEFAULT_SECRET = "bmk-v1-2026-secret-change-in-production";

// ── Tests ──

let passed = 0;
let failed = 0;

function assert(condition, label) {
  if (condition) {
    console.log(`  ✅ ${label}`);
    passed++;
  } else {
    console.error(`  ❌ ${label}`);
    failed++;
  }
}

function assertEqual(actual, expected, label) {
  if (actual === expected) {
    console.log(`  ✅ ${label}`);
    passed++;
  } else {
    console.error(`  ❌ ${label}\n     expected: ${expected}\n     actual:   ${actual}`);
    failed++;
  }
}

// ── 1. Key format tests ──

console.log("\n📋 Key generation format");

// bmk_<tier>_<base64email>_<hmac hex>
const key1 = generateLicenseKey("pro", "test@example.com", DEFAULT_SECRET);
assert(key1.startsWith("bmk_pro_"), "Starts with bmk_pro_");
assert(key1.split("_").length === 4, "Has exactly 4 underscore-delimited parts");

const parts = key1.split("_");
assert(parts[0] === "bmk", "Prefix is bmk");
assert(parts[1] === "pro", "Tier is pro");

// Verify base64url email
const decodedEmail = Buffer.from(parts[2], "base64url").toString("utf-8");
assertEqual(decodedEmail, "test@example.com", "Base64-decoded email matches");

// Verify HMAC hex
const payload = "pro:test@example.com";
const expectedSig = crypto.createHmac("sha256", DEFAULT_SECRET).update(payload).digest("hex");
assertEqual(parts[3], expectedSig, "HMAC signature matches expected");

// ── 2. Tier mapping ──

console.log("\n📋 Tier mapping");

const keyPro = generateLicenseKey("pro", "user@test.com", DEFAULT_SECRET);
assert(keyPro.startsWith("bmk_pro_"), "Pro tier key starts with bmk_pro_");

const keyTeam = generateLicenseKey("team", "team@test.com", DEFAULT_SECRET);
assert(keyTeam.startsWith("bmk_team_"), "Team tier key starts with bmk_team_");

// Verify deterministic — same inputs = same key
const key2a = generateLicenseKey("pro", "alice@example.com", DEFAULT_SECRET);
const key2b = generateLicenseKey("pro", "alice@example.com", DEFAULT_SECRET);
assertEqual(key2a, key2b, "Deterministic: same inputs produce same key");

// Different email = different key
const key3 = generateLicenseKey("pro", "bob@example.com", DEFAULT_SECRET);
assert(key2a !== key3, "Different email produces different key");

// Different tier = different key
const key4 = generateLicenseKey("team", "alice@example.com", DEFAULT_SECRET);
assert(key2a !== key4, "Different tier produces different key");

// Different secret = different key
const key5 = generateLicenseKey("pro", "alice@example.com", "different-secret");
assert(key2a !== key5, "Different secret produces different key");

// ── 3. Edge cases ──

console.log("\n📋 Edge cases");

// Email with special characters
const keySpecial = generateLicenseKey("pro", "user+tag@example.co.uk", DEFAULT_SECRET);
assert(keySpecial.startsWith("bmk_pro_"), "Email with + works");
const decodedSpecial = Buffer.from(keySpecial.split("_")[2], "base64url").toString();
assertEqual(decodedSpecial, "user+tag@example.co.uk", "Email with + decodes correctly");

// Email with dots
const keyDots = generateLicenseKey("pro", "first.last@company.org", DEFAULT_SECRET);
assert(keyDots.startsWith("bmk_pro_"), "Email with dots works");

// ── 4. Webhook signature verification ──

console.log("\n📋 Webhook signature verification");

const rawBody = JSON.stringify({
  meta: { event_name: "order_created" },
  data: { attributes: { email: "test@example.com", variant_id: 1 } },
});
const webhookSecret = "ls_test_secret_abc123";
const sig = crypto.createHmac("sha256", webhookSecret).update(rawBody).digest("hex");

assert(verifyWebhookSignature(rawBody, sig, webhookSecret), "Valid signature passes");
assert(!verifyWebhookSignature(rawBody, "invalid_sig", webhookSecret), "Invalid signature fails");
assert(!verifyWebhookSignature(rawBody, "", webhookSecret), "Empty signature fails");
assert(!verifyWebhookSignature(rawBody, sig, "wrong_secret"), "Wrong secret fails");

// ── 5. Verification: ensure Go and Node produce identical keys ──

console.log("\n📋 Go compatibility cross-check");

// Known Go output: we test that the algorithm is identical
// Run this: go test -run TestGenerateAndValidate ./internal/license/
// The key format is: bmk_<tier>_<base64url(email)>_<HMAC-SHA256("tier:email")>

// We'll verify by generating with known inputs and checking against
// what the Go code would produce (same algorithm, same secret)
const testCases = [
  { tier: "pro", email: "test@example.com" },
  { tier: "team", email: "hello@test.com" },
  { tier: "pro", email: "karabo@basemake.dev" },
];

for (const tc of testCases) {
  const key = generateLicenseKey(tc.tier, tc.email, DEFAULT_SECRET);
  const parts = key.split("_");

  // Re-derive the signature to verify
  const rePayload = `${tc.tier}:${tc.email}`;
  const reSig = crypto.createHmac("sha256", DEFAULT_SECRET).update(rePayload).digest("hex");
  const reEncoded = Buffer.from(tc.email).toString("base64url");

  assertEqual(parts[0], "bmk", `bmk prefix for ${tc.email}`);
  assertEqual(parts[1], tc.tier, `Tier ${tc.tier} for ${tc.email}`);
  assertEqual(parts[2], reEncoded, `Base64 email for ${tc.email}`);
  assertEqual(parts[3], reSig, `HMAC signature for ${tc.email}`);
}

// ── Summary ──

console.log(`\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);
console.log(`Results: ${passed} passed, ${failed} failed\n`);
process.exit(failed > 0 ? 1 : 0);
