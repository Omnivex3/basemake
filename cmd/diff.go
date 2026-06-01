package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/DynamicKarabo/basemake/internal/diff"
	"github.com/DynamicKarabo/basemake/internal/license"
	"github.com/spf13/cobra"
)

var (
	diffFrom     string
	diffTo       string
	diffJSON     bool
	diffFileFrom string
	diffFileTo   string
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show schema differences between two databases",
	Long: `Compare database schemas and show what changed.

Detects added/removed tables, columns, indexes, and type changes.
Useful for catching schema drift between environments.

  basemake diff                                    # Compare active connection vs cached schema
  basemake diff --from "postgres://..." --to "postgres://..."  # Compare two live databases
  basemake diff --from-file schema_a.json --to-file schema_b.json  # Compare saved schemas
  basemake diff --json                             # Output as JSON`,

	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !requireLicense(license.FeatureDiff) {
			return fmt.Errorf("license required for diff feature")
		}
		var fromSchema, toSchema *db.Schema
		var fromName, toName string

		// Mode 1: Compare two saved schema files
		if diffFileFrom != "" && diffFileTo != "" {
			s1, s2, err := loadSchemasFromFiles(diffFileFrom, diffFileTo)
			if err != nil {
				return fmt.Errorf("load schema files: %w", err)
			}
			fromSchema, toSchema = s1, s2
			fromName = diffFileFrom
			toName = diffFileTo
		} else if diffFrom != "" && diffTo != "" {
			// Mode 2: Compare two live databases
			schema1, err := introspectDSN(diffFrom)
			if err != nil {
				return fmt.Errorf("introspect --from: %w", err)
			}
			schema2, err := introspectDSN(diffTo)
			if err != nil {
				return fmt.Errorf("introspect --to: %w", err)
			}
			fromSchema = schema1
			toSchema = schema2
			fromName = maskShort(diffFrom)
			toName = maskShort(diffTo)
		} else {
			// Mode 3: Compare active connection vs cached schema
			conn, err := db.ActiveConnection()
			if err != nil {
				dsn, loadErr := db.LoadDSN()
				if loadErr != nil {
					return fmt.Errorf("no active connection — run 'basemake connect' first: %w", err)
				}
				conn, err = db.Connect(dsn)
				if err != nil {
					return fmt.Errorf("reconnect: %w", err)
				}
			}
			defer conn.Close()

			liveSchema, err := conn.Introspect(cmd.Context())
			if err != nil {
				return fmt.Errorf("introspect live database: %w", err)
			}

			cachedSchema, err := db.LoadSchema()
			if err != nil {
				// No cached schema — compare live vs live (or just show the live schema)
				_ = liveSchema.Save()
				fmt.Println("No cached schema found. Saved current schema as baseline.")
				fmt.Println("Run 'basemake diff' again after making changes to see the diff.")
				return nil
			}

			fromSchema = cachedSchema
			toSchema = liveSchema
			fromName = "cached"
			toName = "live"
		}

		report := diff.SchemaDiff(fromSchema, toSchema, fromName, toName)

		if diffJSON {
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal diff: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Print(diff.FormatPlain(report))
		return nil
	},
}

// introspectDSN connects to a DSN and returns its schema.
func introspectDSN(dsn string) (*db.Schema, error) {
	conn, err := db.Connect(dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	return conn.Introspect(context.TODO())
}

// loadSchemasFromFiles loads two schema JSON files from disk.
func loadSchemasFromFiles(fromPath, toPath string) (*db.Schema, *db.Schema, error) {
	from, err := loadSchemaFile(fromPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load %s: %w", fromPath, err)
	}
	to, err := loadSchemaFile(toPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load %s: %w", toPath, err)
	}
	return from, to, nil
}

func loadSchemaFile(path string) (*db.Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var s db.Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &s, nil
}

// maskShort shortens a DSN for display (user:pass@host:port/db → db@host:port)
func maskShort(dsn string) string {
	// Mask credentials in DSN while preserving host/db identity.
	// Handles both URL-style (postgres://user:pass@host/db) and
	// MySQL-style (user:pass@tcp(host:port)/db) formats.
	if dsn == "" {
		return ""
	}
	// Mask everything before the first '@', leave host/db visible
	if idx := strings.Index(dsn, "@"); idx >= 0 {
		return "***@" + dsn[idx+1:]
	}
	// SQLite file path or uncredentialed DSN — return as-is
	return dsn
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().StringVar(&diffFrom, "from", "", "Source DSN (dev)")
	diffCmd.Flags().StringVar(&diffTo, "to", "", "Target DSN (staging)")
	diffCmd.Flags().StringVar(&diffFileFrom, "from-file", "", "Path to source schema JSON file")
	diffCmd.Flags().StringVar(&diffFileTo, "to-file", "", "Path to target schema JSON file")
	diffCmd.Flags().BoolVar(&diffJSON, "json", false, "Output as JSON")
}
