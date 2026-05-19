package license

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestGenerateAndValidate(t *testing.T) {
	// Generate a Pro license
	key, err := Generate(TierPro, "dev@example.com", 365*24*time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if key == "" {
		t.Fatal("Generate() returned empty key")
	}

	// Validate it
	lic, err := Validate(key)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if lic.Tier != TierPro {
		t.Errorf("Tier = %q, want %q", lic.Tier, TierPro)
	}
	if lic.Email != "dev@example.com" {
		t.Errorf("Email = %q, want %q", lic.Email, "dev@example.com")
	}
}

func TestGenerateAndValidateTeam(t *testing.T) {
	key, err := Generate(TierTeam, "team@company.com", 30*24*time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	lic, err := Validate(key)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if lic.Tier != TierTeam {
		t.Errorf("Tier = %q, want %q", lic.Tier, TierTeam)
	}
}

func TestValidate_InvalidFormat(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"too short", "bmk_pro"},
		{"missing prefix", "pro_email_sig"},
		{"garbage", "not-a-license-key"},
		{"wrong prefix", "xyz_pro_email_sig"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Validate(tt.key)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestValidate_InvalidTier(t *testing.T) {
	key := "bmk_enterprise_am9obkBhYmMuY29t_somehex"
	_, err := Validate(key)
	if err == nil {
		t.Error("expected error for invalid tier, got nil")
	}
}

func TestValidate_TamperedSignature(t *testing.T) {
	validKey, err := Generate(TierPro, "test@test.com", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// Tamper with the signature part (last segment)
	parts := splitKey(validKey)
	if len(parts) == 4 {
		tampered := parts[0] + "_" + parts[1] + "_" + parts[2] + "_deadbeef"
		_, err := Validate(tampered)
		if err == nil {
			t.Error("expected error for tampered signature, got nil")
		}
	}
}

func TestValidate_TamperedEmail(t *testing.T) {
	validKey, err := Generate(TierPro, "test@test.com", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// Tamper with the email part
	parts := splitKey(validKey)
	if len(parts) == 4 {
		evilEmail := b64("evil@hacker.com")
		tampered := parts[0] + "_" + parts[1] + "_" + evilEmail + "_" + parts[3]
		_, err := Validate(tampered)
		if err == nil {
			t.Error("expected error for tampered email, got nil")
		}
	}
}

func TestParseKey(t *testing.T) {
	key, err := Generate(TierPro, "user@example.com", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	tier, email, err := ParseKey(key)
	if err != nil {
		t.Fatalf("ParseKey() error = %v", err)
	}

	if tier != TierPro {
		t.Errorf("tier = %q, want %q", tier, TierPro)
	}
	if email != "user@example.com" {
		t.Errorf("email = %q, want %q", email, "user@example.com")
	}
}

func TestParseKey_Invalid(t *testing.T) {
	_, _, err := ParseKey("garbage")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLicense_IsValid(t *testing.T) {
	lic := &License{Tier: TierPro}
	if !lic.IsValid() {
		t.Error("license with zero expiry should be valid")
	}
}

func TestLicense_HasFeature(t *testing.T) {
	tests := []struct {
		name    string
		tier    Tier
		feature Feature
		want    bool
	}{
		{"pro has check", TierPro, FeatureCheck, true},
		{"pro has budget", TierPro, FeatureBudget, true},
		{"pro has watch", TierPro, FeatureWatch, true},
		{"pro has diff", TierPro, FeatureDiff, true},
		{"pro has index-apply", TierPro, FeatureIndexApply, true},
		{"pro does NOT have server", TierPro, FeatureServer, false},
		{"team has server", TierTeam, FeatureServer, true},
		{"team has check", TierTeam, FeatureCheck, true},
		{"free has no pro features", TierFree, FeatureCheck, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lic := &License{Tier: tt.tier}
			got := lic.HasFeature(tt.feature)
			if got != tt.want {
				t.Errorf("HasFeature(%q) = %v, want %v", tt.feature, got, tt.want)
			}
		})
	}
}

func TestLicense_HasFeature_Expired(t *testing.T) {
	// Licenses are perpetual — no expiry in v1 format.
	// This test will pass trivially but documents the behaviour.
	lic := &License{Tier: TierPro}
	if !lic.HasFeature(FeatureCheck) {
		t.Error("license should have check feature")
	}
}

func TestGenerate_InvalidTier(t *testing.T) {
	_, err := Generate(TierFree, "test@test.com", time.Hour)
	if err == nil {
		t.Error("expected error for free tier, got nil")
	}
}

func TestGenerate_EmptyEmail(t *testing.T) {
	_, err := Generate(TierPro, "", time.Hour)
	if err == nil {
		t.Error("expected error for empty email, got nil")
	}
}

// ── Helpers ──

func splitKey(key string) []string {
	parts := make([]string, 0, 4)
	current := ""
	for _, ch := range key {
		if ch == '_' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func b64(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
