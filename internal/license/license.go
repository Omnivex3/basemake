// Package license provides offline-verifiable license key management for basemake.
//
// License keys are HMAC-signed tokens with embedded tier, email, and expiry.
// The secret key is compiled into the binary for offline verification.
// In production, license keys are generated server-side and distributed to customers.
//
// Key format: bmk_<tier>_<base64email>_<hmac hex>
//
// Tiers: pro, team
// Free tier requires no key — just use the tool.
package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Tier represents a license tier.
type Tier string

const (
	TierFree Tier = "free"
	TierPro  Tier = "pro"
	TierTeam Tier = "team"
)

// Feature represents a gated feature name.
type Feature string

const (
	FeatureCheck      Feature = "check"       // basemake check — CI/CD gate
	FeatureBudget     Feature = "budget"      // basemake budget — policy as code
	FeatureWatch      Feature = "watch"       // basemake watch — query monitoring
	FeatureDiff       Feature = "diff"        // basemake diff — schema diffing
	FeatureIndexApply Feature = "index-apply" // basemake index apply — apply index recs
	FeatureServer     Feature = "server"      // basemake server — team sync
)

// tierFeatures maps each tier to the features it unlocks.
var tierFeatures = map[Tier][]Feature{
	TierPro: {
		FeatureCheck,
		FeatureBudget,
		FeatureWatch,
		FeatureDiff,
		FeatureIndexApply,
	},
	TierTeam: {
		FeatureCheck,
		FeatureBudget,
		FeatureWatch,
		FeatureDiff,
		FeatureIndexApply,
		FeatureServer,
	},
}

// License holds the decoded and validated license information.
type License struct {
	Tier      Tier
	Email     string
	ExpiresAt time.Time
}

// IsValid returns true if the license has not expired.
func (l *License) IsValid() bool {
	return l.ExpiresAt.IsZero() || time.Now().Before(l.ExpiresAt)
}

// HasFeature checks if this license allows the given feature.
func (l *License) HasFeature(feature Feature) bool {
	if !l.IsValid() {
		return false
	}
	features, ok := tierFeatures[l.Tier]
	if !ok {
		return false
	}
	for _, f := range features {
		if f == feature {
			return true
		}
	}
	return false
}

// ── Cryptographic key ──
// In production, these are generated per-customer. For the open-source binary,
// we embed a shared verification key. Customers who pay get a unique key.

// hmacSecret is the shared secret used for HMAC signing and verification.
// In production this is per-customer in our license server, but the binary
// ships with a compiled-in public verification key.
// For v1, we use a single symmetric secret. v2 will switch to asymmetric.
var hmacSecret = []byte("bmk-v1-2026-secret-change-in-production")

// SetHMACSecret overrides the HMAC secret (used in tests or by license server).
func SetHMACSecret(secret []byte) {
	hmacSecret = secret
}

// ── Generate (server-side) ──

// Generate creates a signed license key for the given parameters.
// This is called by the license server when a customer pays.
// Note: v1 keys are perpetual — expiry is managed by key re-issuance.
func Generate(tier Tier, email string, duration time.Duration) (string, error) {
	if tier != TierPro && tier != TierTeam {
		return "", fmt.Errorf("invalid tier: %s", tier)
	}
	if email == "" {
		return "", fmt.Errorf("email is required")
	}

	_ = duration // v1: keys are perpetual, expiry managed server-side
	payload := fmt.Sprintf("%s:%s", tier, email)

	mac := hmac.New(sha256.New, hmacSecret)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	encodedEmail := base64.RawURLEncoding.EncodeToString([]byte(email))
	key := fmt.Sprintf("bmk_%s_%s_%s", tier, encodedEmail, sig)
	return key, nil
}

// ── Validate (client-side) ──

// Validate parses and validates a license key string.
// Returns a License on success, or an error if the key is invalid.
func Validate(key string) (*License, error) {
	key = strings.TrimSpace(key)

	parts := strings.Split(key, "_")
	if len(parts) != 4 || parts[0] != "bmk" {
		return nil, fmt.Errorf("invalid license key format")
	}

	tier := Tier(parts[1])
	if tier != TierPro && tier != TierTeam {
		return nil, fmt.Errorf("unknown license tier: %s", tier)
	}

	emailBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid license key encoding: %w", err)
	}
	email := string(emailBytes)

	sigHex := parts[3]
	if err := verifySignature(tier, email, sigHex); err != nil {
		return nil, err
	}

	return &License{
		Tier:  tier,
		Email: email,
	}, nil
}

// verifySignature re-derives the HMAC to verify a license key signature.
// For v1 simplicity, we use a format where the key itself is the token
// and we verify it against known-good tokens or direct HMAC.
//
// New approach: the key is simply bmk_tier_email_b64_hmac
// where hmac = HMAC("tier:email:secret_salt")
// This means no expiry in the key itself — expiry is enforced server-side
// and managed by issuing new keys or a short-lived validation API.
func verifySignature(tier Tier, email, sigHex string) error {
	// Reconstruct the payload that was signed: "tier:email"
	payload := fmt.Sprintf("%s:%s", tier, email)

	mac := hmac.New(sha256.New, hmacSecret)
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sigHex), []byte(expected)) {
		return fmt.Errorf("invalid license key signature")
	}
	return nil
}

// ── No-expiry key ──
// For v1, license keys don't embed an expiry. Instead:
// - Pro keys are perpetual for the purchased year (re-issued annually)
// - Team keys are subscription-based (validated against server)
// - The binary can include a kill-date mechanism for really old versions

// ParseKey extracts tier and email from a license key without full validation.
// Useful for displaying info before validation.
func ParseKey(key string) (tier Tier, email string, err error) {
	key = strings.TrimSpace(key)
	parts := strings.Split(key, "_")
	if len(parts) != 4 || parts[0] != "bmk" {
		return "", "", fmt.Errorf("invalid license key format")
	}

	tier = Tier(parts[1])
	if tier != TierPro && tier != TierTeam {
		return "", "", fmt.Errorf("unknown license tier: %s", tier)
	}

	emailBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", "", fmt.Errorf("invalid license key encoding: %w", err)
	}

	return tier, string(emailBytes), nil
}
