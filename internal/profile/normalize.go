package profile

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// NormalizeSQL normalizes a SQL query for fingerprinting:
// - lowercase, collapse whitespace
// - replace string/numeric literals with ?
// - remove comments, normalize IN lists
func NormalizeSQL(sql string) string {
	s := sql

	// Strip comments (single-line)
	commentRE := regexp.MustCompile(`--.*$`)
	s = commentRE.ReplaceAllString(s, "")

	// Block comments
	blockRE := regexp.MustCompile(`(?s)/\*.*?\*/`)
	s = blockRE.ReplaceAllString(s, "")

	// Lowercase
	s = strings.ToLower(s)

	// Collapse whitespace
	wsRE := regexp.MustCompile(`\s+`)
	s = wsRE.ReplaceAllString(s, " ")

	// Replace single-quoted strings
	stringRE := regexp.MustCompile(`'[^']*'`)
	s = stringRE.ReplaceAllString(s, "?")

	// Replace numeric literals (integers and decimals, including negative)
	numRE := regexp.MustCompile(`\b-?\d+(?:\.\d+)?\b`)
	s = numRE.ReplaceAllString(s, "?")

	// Normalize IN lists: (?, ?, ?) -> (?)
	inListRE := regexp.MustCompile(`\(\?(?:\s*,\s*\?)+\)`)
	s = inListRE.ReplaceAllString(s, "(?)")

	// Normalize repeated whitespace from replacements
	s = wsRE.ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}

// QueryHash returns a stable SHA-256 fingerprint of a normalized SQL query.
// First 16 bytes = 32 hex chars — collision probability is negligible for
// the profile dataset size (thousands of unique queries per database).
func QueryHash(normalizedSQL string) string {
	h := sha256.Sum256([]byte(normalizedSQL))
	return fmt.Sprintf("%x", h[:16])
}
