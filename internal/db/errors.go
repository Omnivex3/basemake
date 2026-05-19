package db

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNoConnection is returned when no database is connected.
	ErrNoConnection = errors.New("no active database connection — run 'basemake connect' first")
	// ErrUnsupported is returned when the DSN scheme isn't recognized.
	ErrUnsupported = errors.New("unsupported database driver")
)

// Friendly wraps a database error with a human-readable message and a fix suggestion.
// Returns the original error if it doesn't match any known pattern.
func Friendly(err error) error {
	if err == nil {
		return nil
	}

	msg := err.Error()

	// Unwrap common patterns
	switch {
	// Connection issues
	case strings.Contains(msg, "connection refused"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ Could not connect to database",
			Suggestion: "Is the database server running?\n  → PostgreSQL: brew services start postgresql (or: sudo systemctl start postgresql)\n  → MySQL: brew services start mysql (or: sudo systemctl start mysql)\n  → Docker: docker start <container-name>",
		}
	case strings.Contains(msg, "no connection"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ Not connected to a database",
			Suggestion: "Run `basemake connect` first, or `basemake init` to get set up.",
		}

	// Authentication
	case strings.Contains(msg, "password authentication failed") || strings.Contains(msg, "access denied") || strings.Contains(msg, "authentication failed"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ Authentication failed",
			Suggestion: "Check your username and password.\n  → Update: basemake connect postgres://user:correctpass@host/db\n  → Or use a .env file (BASEMAKE_DB_PASSWORD) so passwords stay out of shell history.",
		}
	case strings.Contains(msg, "role") && strings.Contains(msg, "does not exist"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ Database user not found",
			Suggestion: "Create the user or use an existing one.\n  → CREATE ROLE myuser WITH LOGIN PASSWORD 'pass';\n  → Or: basemake connect postgres://postgres@localhost/db (use default superuser)",
		}

	// Database not found
	case strings.Contains(msg, "database") && strings.Contains(msg, "does not exist"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ Database not found",
			Suggestion: "Check the database name in your connection string.\n  → List databases: \\l (psql) or SHOW DATABASES; (MySQL)\n  → Create it: createdb mydatabase",
		}

	// Network issues
	case strings.Contains(msg, "dial tcp"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ Could not reach the database host",
			Suggestion: "Check the hostname and port in your DSN.\n  → Is the host correct? Try: ping <hostname>\n  → Is the port correct? Defaults: PostgreSQL=5432, MySQL=3306\n  → Is a firewall blocking it?",
		}
	case strings.Contains(msg, "i/o timeout"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ Connection timed out",
			Suggestion: "The database is not responding.\n  → Is it running on the expected host?\n  → Network issue? Try: nc -zv <host> <port>\n  → Increase timeout? Add ?connect_timeout=10 to DSN",
		}
	case strings.Contains(msg, "no route to host"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ No route to database host",
			Suggestion: "The IP/hostname can't be reached from your network.\n  → Check VPN or network configuration\n  → Is the host on a private network?",
		}

	// SSL/TLS issues
	case strings.Contains(msg, "ssl") || strings.Contains(msg, "tls"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ SSL connection error",
			Suggestion: "Add sslmode to your DSN:\n  → sslmode=disable (no SSL — local dev only)\n  → sslmode=require (SSL required)\n  → postgres://user@localhost/db?sslmode=disable",
		}

	// Schema introspection
	case strings.Contains(msg, "no cached schema"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ Schema not cached",
			Suggestion: "Reconnect to reload the schema.\n  → basemake connect postgres://...\n  → Or: .connect in the REPL",
		}

	// SQL execution
	case strings.Contains(msg, "syntax error"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ SQL syntax error",
			Suggestion: "The AI generated invalid SQL.\n  → Try rephrasing your question\n  → Check for typos in table/column names\n  → Try: basemake query \"...\" --dry-run to preview SQL first",
		}
	case strings.Contains(msg, "relation") && strings.Contains(msg, "does not exist"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ Table not found",
			Suggestion: "The table doesn't exist in the current database.\n  → Check available tables: basemake query \".tables\"\n  → Or: .tables in the REPL",
		}

	// Generic fallback
	case strings.Contains(msg, "unsupported database"):
		return &friendlyError{
			Original:   err,
			Message:    "✗ Unsupported database type",
			Suggestion: "Basemake supports PostgreSQL, MySQL, and SQLite.\n  → Use: postgres://user@host/db\n  → Use: mysql://user@tcp(host:3306)/db\n  → Use: sqlite:/path/to/db.db",
		}
	}

	return err
}

// FriendlyMsg returns just the human-readable message for an error.
// Returns the original error text if no friendly mapping exists.
func FriendlyMsg(err error) string {
	var fe *friendlyError
	if errors.As(err, &fe) {
		return fe.Message
	}
	return err.Error()
}

// FriendlyWithSuggestion returns the full message + suggestion for an error.
func FriendlyWithSuggestion(err error) string {
	var fe *friendlyError
	if errors.As(err, &fe) {
		return fe.Message + "\n  " + fe.Suggestion
	}
	return err.Error()
}

type friendlyError struct {
	Original   error
	Message    string
	Suggestion string
}

func (e *friendlyError) Error() string {
	return e.Message + "\n  " + e.Suggestion
}

func (e *friendlyError) Unwrap() error {
	return e.Original
}

// FormatFloat formats a float value for display.
func FormatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}
