package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/DynamicKarabo/basemake/internal/ai"
)

// ── Schema cache ──

// TotalColumns returns the total number of columns across all tables.
func (s *Schema) TotalColumns() int {
	count := 0
	for _, t := range s.Tables {
		count += len(t.Columns)
	}
	return count
}

// TotalIndexes returns the total number of indexes across all tables.
func (s *Schema) TotalIndexes() int {
	count := 0
	for _, t := range s.Tables {
		count += len(t.Indexes)
	}
	return count
}

// Save persists the schema to the local JSON cache (convenience wrapper).
func (s *Schema) Save() error {
	return SaveSchema(s)
}

// SaveSchema persists the schema to a local JSON cache.
func SaveSchema(s *Schema) error {
	dir := cacheDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	if err := os.WriteFile(cachePath(), data, 0600); err != nil {
		return fmt.Errorf("write schema cache: %w", err)
	}
	return nil
}

// LoadSchema reads the cached schema from disk.
func LoadSchema() (*Schema, error) {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return nil, fmt.Errorf("no cached schema — run 'basemake connect' first: %w", err)
	}

	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}
	return &s, nil
}

// ClearSchemaCache removes the cached schema file from disk.
func ClearSchemaCache() error {
	if err := os.Remove(cachePath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear schema cache: %w", err)
	}
	return nil
}

func cacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".basemake")
}

func cachePath() string {
	return filepath.Join(cacheDir(), "schema.json")
}

// ── Prompt serialization ──

// SchemaForPrompt returns a compact schema description for AI prompts.
// If question is non-empty, it uses two-stage filtering:
//
//	Stage 1 — keyword matching against table/column names
//	Stage 2 — FK expansion to include join neighbors
//
// The output is hard-capped at ~2000 tokens (~8000 chars).
// Truncated tables are summarized as a single line.
// When question is empty, tables are included in order up to the budget.
func (s *Schema) SchemaForPrompt(question string) string {
	maxTokens := 2000
	maxChars := maxTokens * 4 // ~4 chars per token

	// Build the header — always include
	var out strings.Builder
	out.WriteString(fmt.Sprintf("Database: %s\n\nTables:\n", s.DBName))
	headerLen := len(out.String())

	// Determine table order and priorities
	type rankedTable struct {
		index int  // index into s.Tables
		score int  // relevance score (0 = not matched)
		fk    bool // included via FK expansion
	}

	ranked := make([]rankedTable, len(s.Tables))
	keywords := tokenize(question)

	for i := range s.Tables {
		ranked[i].index = i
		if len(keywords) > 0 {
			ranked[i].score = tableRelevance(&s.Tables[i], keywords)
		}
	}

	if len(keywords) > 0 {
		// Stage 2: FK expansion — include FK neighbors of matched tables
		matched := make(map[int]bool) // indices of tables with score > 0
		for _, r := range ranked {
			if r.score > 0 {
				matched[r.index] = true
			}
		}
		// One-hop FK expansion
		for _, r := range ranked {
			if r.score > 0 {
				t := &s.Tables[r.index]
				for _, fk := range t.ForeignKeys {
					if idx := tableIndexByName(s.Tables, fk.RefTable); idx >= 0 && !matched[idx] {
						matched[idx] = true
						ranked[idx].fk = true
					}
				}
				// Reverse FK: tables that reference this one
				for j := range s.Tables {
					if matched[j] {
						continue
					}
					for _, fk := range s.Tables[j].ForeignKeys {
						if fk.RefTable == t.Name {
							matched[j] = true
							ranked[j].fk = true
							break
						}
					}
				}
			}
		}

		// Sort: by score desc, then FK neighbors, then name
		sort.Slice(ranked, func(a, b int) bool {
			sa, sb := ranked[a].score, ranked[b].score
			if sa != sb {
				return sa > sb
			}
			// FK neighbors before unmatched
			if ranked[a].fk != ranked[b].fk {
				return ranked[a].fk
			}
			return s.Tables[ranked[a].index].Name < s.Tables[ranked[b].index].Name
		})
	}
	// When no question, tables stay in original order (ranked[i].score = 0, no sort)

	// Render tables up to budget
	remaining := maxChars - headerLen
	rendered := 0
	totalTables := len(s.Tables)

	for i, r := range ranked {
		block := renderTableBlock(&s.Tables[r.index])
		blockLen := len(block)

		// If this table doesn't fit within budget, stop
		if remaining-blockLen < 0 {
			// Count remaining tables that haven't been rendered
			remainingTables := totalTables - i
			if remainingTables > 0 {
				note := fmt.Sprintf("    ... and %d additional table(s) omitted. Ask about a specific table for details.\n", remainingTables)
				if maxChars-headerLen-len(out.String())+headerLen >= len(note) {
					out.WriteString(note)
				}
			}
			break
		}

		out.WriteString(block)
		remaining -= blockLen
		rendered++
	}

	// If question was empty and we truncated, still add a note
	if len(keywords) == 0 && rendered < totalTables {
		out.WriteString(fmt.Sprintf("    ... and %d additional table(s) omitted. Ask about a specific table for details.\n", totalTables-rendered))
	}

	return out.String()
}

// renderTableBlock renders a single table's full detail (columns, FKs, indexes).
func renderTableBlock(t *TableInfo) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s:\n", t.Name))
	for _, c := range t.Columns {
		pk := ""
		if c.IsPK {
			pk = " [PK]"
		}
		nullable := ""
		if c.IsNullable {
			nullable = " nullable"
		}
		b.WriteString(fmt.Sprintf("    - %s %s%s%s\n", c.Name, c.Type, pk, nullable))
	}
	if len(t.ForeignKeys) > 0 {
		b.WriteString("    Foreign Keys:\n")
		for _, fk := range t.ForeignKeys {
			b.WriteString(fmt.Sprintf("      - %s → %s.%s\n", fk.Column, fk.RefTable, fk.RefColumn))
		}
	}
	if len(t.Indexes) > 0 {
		b.WriteString("    Indexes:\n")
		for _, idx := range t.Indexes {
			u := ""
			if idx.Unique {
				u = " (unique)"
			}
			b.WriteString(fmt.Sprintf("      - %s on (%s)%s\n", idx.Name, strings.Join(idx.Cols, ", "), u))
		}
	}
	return b.String()
}

// tableIndexByName finds a table by name, returns -1 if not found.
func tableIndexByName(tables []TableInfo, name string) int {
	for i := range tables {
		if tables[i].Name == name {
			return i
		}
	}
	return -1
}

// ── Tokenization ──

// stopWords are common English/SQL words that add no relevance signal.
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "of": true, "in": true, "on": true,
	"at": true, "to": true, "for": true, "with": true, "by": true, "from": true,
	"show": true, "get": true, "me": true, "give": true, "list": true, "find": true,
	"all": true, "each": true, "every": true, "any": true,
	"that": true, "this": true, "these": true, "those": true,
	"is": true, "are": true, "was": true, "were": true, "has": true, "have": true, "had": true,
	"do": true, "does": true, "did": true, "will": true, "would": true, "can": true, "could": true,
	"should": true, "may": true, "might": true, "shall": true,
	"and": true, "or": true, "but": true, "not": true, "no": true, "yes": true,
	"how": true, "what": true, "when": true, "where": true, "which": true, "who": true, "whom": true, "why": true,
	"i": true, "we": true, "you": true, "they": true, "he": true, "she": true, "it": true,
	"my": true, "our": true, "your": true, "their": true, "its": true,
	"up": true, "down": true, "out": true, "off": true, "over": true, "under": true,
	"more": true, "most": true, "much": true, "many": true, "some": true,
	"into": true, "onto": true, "than": true, "then": true, "also": true, "very": true,
	"just": true, "only": true, "now": true, "here": true, "there": true,
}

// tokenize splits a question into meaningful keywords.
func tokenize(question string) []string {
	if question == "" {
		return nil
	}

	// Split on non-alphanumeric characters
	words := strings.FieldsFunc(strings.ToLower(question), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var keywords []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		if len(w) < 2 || stopWords[w] {
			continue
		}
		keywords = append(keywords, w)
	}
	return keywords
}

// tableRelevance scores a table against query keywords.
// Table name matches score 2, column name matches score 1.
func tableRelevance(t *TableInfo, keywords []string) int {
	score := 0
	tableLower := strings.ToLower(t.Name)

	for _, kw := range keywords {
		if strings.Contains(tableLower, kw) || strings.Contains(kw, tableLower) {
			score += 2
			continue
		}
		for _, c := range t.Columns {
			colLower := strings.ToLower(c.Name)
			if strings.Contains(colLower, kw) || strings.Contains(kw, colLower) {
				score++
				break // one point per keyword per table
			}
		}
	}
	return score
}

// EstimateTokens returns a rough token count for a string, matching ai.EstimateTokens.
func (s *Schema) EstimatePromptTokens(question string) int {
	return ai.EstimateTokens(s.SchemaForPrompt(question))
}
