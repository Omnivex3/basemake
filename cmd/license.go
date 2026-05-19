package cmd

import (
	"fmt"
	"os"

	"github.com/DynamicKarabo/basemake/internal/config"
	"github.com/DynamicKarabo/basemake/internal/license"
	"github.com/spf13/cobra"
)

var licenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Show license status and activate a license key",
	Long: `Display current license status or activate a Pro/Team license.

Without arguments, shows the current license status.
With a license key argument, validates and activates it.

Examples:
  basemake license                          # Show current license status
  basemake license bmk_pro_xxx_yyy          # Activate a license key
  basemake config set license_key <key>     # Alternative: set via config`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// If a key was provided, validate and save it
		if len(args) == 1 {
			key := args[0]
			lic, err := license.Validate(key)
			if err != nil {
				return fmt.Errorf("invalid license key: %w", err)
			}
			cfg.LicenseKey = key
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Fprintf(os.Stderr, "✓ License activated: %s — %s\n", lic.Tier, lic.Email)
			return nil
		}

		// Show current license status
		if cfg.LicenseKey == "" {
			fmt.Fprintf(os.Stderr, "License: Free tier\n")
			fmt.Fprintf(os.Stderr, "\n  Upgrade to Pro for CI/CD gates, budgets, monitoring, and more.\n")
			fmt.Fprintf(os.Stderr, "  Visit https://basemake.dev/pricing\n")
			return nil
		}

		lic, err := license.Validate(cfg.LicenseKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "License: Invalid or expired (%v)\n", err)
			fmt.Fprintf(os.Stderr, "\n  Run 'basemake config set license_key <key>' with a valid key.\n")
			return nil
		}

		fmt.Fprintf(os.Stderr, "License: %s tier — activated for %s\n", lic.Tier, lic.Email)
		fmt.Fprintf(os.Stderr, "\n  Unlocked features:\n")
		for _, feature := range []license.Feature{
			license.FeatureCheck,
			license.FeatureBudget,
			license.FeatureWatch,
			license.FeatureDiff,
			license.FeatureIndexApply,
			license.FeatureServer,
		} {
			mark := " "
			if lic.HasFeature(feature) {
				mark = "✓"
			}
			fmt.Fprintf(os.Stderr, "    %s %s\n", mark, feature)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(licenseCmd)
}

// requireLicense is a helper that checks if the current license allows a feature.
// Prints an error and returns true if access is denied.
func requireLicense(feature license.Feature) bool {
	cfg, err := config.Load()
	if err != nil || cfg.LicenseKey == "" {
		fmt.Fprintf(os.Stderr, "Error: %s requires a Pro license.\n", feature)
		fmt.Fprintf(os.Stderr, "  Run 'basemake config set license_key <key>' or 'basemake license <key>'\n")
		fmt.Fprintf(os.Stderr, "  Get a license at https://basemake.dev/pricing\n")
		return false
	}

	lic, err := license.Validate(cfg.LicenseKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid license key (%v).\n", err)
		fmt.Fprintf(os.Stderr, "  Run 'basemake config set license_key <key>' with a valid key.\n")
		return false
	}

	if !lic.HasFeature(feature) {
		fmt.Fprintf(os.Stderr, "Error: %s requires %s tier.\n", feature, lic.Tier)
		fmt.Fprintf(os.Stderr, "  Your current license is %s, which does not include this feature.\n", lic.Tier)
		return false
	}

	return true
}
