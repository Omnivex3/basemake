package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/DynamicKarabo/basemake/internal/ai"
	"github.com/DynamicKarabo/basemake/internal/config"
	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/DynamicKarabo/basemake/internal/display"
	"github.com/DynamicKarabo/basemake/internal/history"
)

// ── Types ──

type replState int

const (
	stateIdle     replState = iota
	stateThinking           // AI generating SQL or query running
)

type msgKind int

const (
	msgUser msgKind = iota
	msgBot
	msgResult
	msgError
	msgCmd
)

type message struct {
	kind    msgKind
	content string
}

// ── Bubbletea Messages ──

type queryResultMsg struct {
	content string
	err     error
}

type connResultMsg struct {
	name string
	err  error
}

type introspectResultMsg struct {
	content string
	err     error
}

// ── Model ──

type Model struct {
	conn     db.Database
	format   display.Format
	state    replState
	messages []message
	input    textinput.Model
	spinner  spinner.Model
	width    int

	version     string
	aiLabel     string
	thinkingMsg string // shown below spinner (e.g. "Generating SQL...")

	// Ctrl+C cancellation support
	queryCancel context.CancelFunc

	// Read-only guard
	readonly bool

	// Tab completion state
	tabPrefix  string   // the word being completed (reset on new word)
	tabMatches []string // matching table names
	tabIndex   int      // current cycle position

	// History navigation (up/down arrow)
	historyItems  []string // past queries, newest appended at end
	historyIdx    int      // -1 = not browsing (fresh input), 0..len-1 = browsing
	pendingInput  string   // saved input when entering browse mode

	// Dot command autocomplete
	autocompleteMatches []string // filtered dot commands matching current input
}

// ── Constructor ──

func NewModel(conn db.Database, format display.Format, version string, readonly bool) Model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 72

	// Contextual placeholder based on setup state
	if conn == nil {
		ti.Placeholder = "No database connected. Try: .connect postgres://user@localhost/mydb"
	} else if _, err := ai.SelectedProvider(); err != nil {
		ti.Placeholder = "AI queries need an API key. Run 'basemake init' to set one up"
	} else {
		ti.Placeholder = "Type .help for commands  ·  ask your question or enter SQL"
	}
	ti.Prompt = ""

	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(Red)
	s.Spinner = spinner.Dot

	aiLabel := aiProviderLabel()

	// Load session history from DB if available
	var historyItems []string
	if entries, err := history.List(50); err == nil {
		for _, e := range entries {
			if e.SQLGenerated != "" {
				historyItems = append(historyItems, e.SQLGenerated)
			} else if e.Question != "" {
				historyItems = append(historyItems, e.Question)
			}
		}
		// Reverse so newest is at the end (will be accessed from end)
		for i, j := 0, len(historyItems)-1; i < j; i, j = i+1, j-1 {
		historyItems[i], historyItems[j] = historyItems[j], historyItems[i]
		}
	}

	// Set initial cursor style (idle = white)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(White)

	return Model{
		conn:    conn,
		format:  format,
		state:   stateIdle,
		input:   ti,
		spinner: s,
		version: version,
		aiLabel: aiLabel,
		readonly: readonly,
		historyItems: historyItems,
		historyIdx:   -1,
		messages: []message{
			{kind: msgBot, content: fullStartupView(conn, aiLabel, version)},
		},
	}
}

// ── Init ──

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink, m.spinner.Tick}
	// If already connected at startup, auto-introspect and cache schema
	if m.conn != nil {
		cmds = append(cmds, m.syncIntrospectCmd())
	}
	return tea.Batch(cmds...)
}

// ── Update ──

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.input.Width = max(30, msg.Width-8)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.state == stateThinking {
				// Cancel in-flight query
				if m.queryCancel != nil {
					m.queryCancel()
					m.queryCancel = nil
				}
				m.state = stateIdle
				m.messages = append(m.messages, message{kind: msgCmd, content: "  ⏹️  Query cancelled"})
				m.updateCursorStyle()
				return m, nil
			}
			return m, tea.Quit

		case "escape":
			if m.state == stateThinking {
				if m.queryCancel != nil {
					m.queryCancel()
					m.queryCancel = nil
				}
				m.state = stateIdle
				m.messages = append(m.messages, message{kind: msgCmd, content: "  ⏹️  Query cancelled"})
				m.updateCursorStyle()
				return m, nil
			}
			// Exit browse mode if browsing history
			if m.historyIdx != -1 {
				m.input.SetValue(m.pendingInput)
				m.historyIdx = -1
				m.pendingInput = ""
				return m, nil
			}

		case "up":
			if m.state != stateIdle || len(m.historyItems) == 0 {
				break
			}
			// Move backward in history
			if m.historyIdx == -1 {
				m.pendingInput = m.input.Value()
				m.historyIdx = len(m.historyItems) - 1
			} else if m.historyIdx > 0 {
				m.historyIdx--
			} else {
				break // already at oldest
			}
			m.input.SetValue(m.historyItems[m.historyIdx])
			return m, nil

		case "down":
			if m.state != stateIdle || m.historyIdx == -1 {
				break
			}
			// Move forward in history
			if m.historyIdx < len(m.historyItems)-1 {
				m.historyIdx++
				m.input.SetValue(m.historyItems[m.historyIdx])
			} else {
				// Back to fresh input
				m.input.SetValue(m.pendingInput)
				m.historyIdx = -1
				m.pendingInput = ""
			}
			return m, nil

		case "enter":
			if m.state != stateIdle {
				break
			}
			m.autocompleteMatches = nil
			input := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if input == "" {
				break
			}
			// Reset tab completion state on submit
			m.tabPrefix = ""
			m.tabMatches = nil
			m.tabIndex = 0
			if strings.HasPrefix(input, ".") {
				model, cmd := m.handleDotCommand(input)
				return model, cmd
			}

			// Read-only guard: reject write queries
			if m.readonly && isWriteQuery(input) {
				m.messages = append(m.messages, message{kind: msgError, content: errorBubble("Write queries are blocked in read-only mode.")})
				m.messages = append(m.messages, message{kind: msgCmd, content: "  💡 Override with ! prefix (e.g. !DELETE FROM users) or restart without --readonly"})
				return m, nil
			}

			// Add to history for up/down navigation
			m.historyItems = append(m.historyItems, input)
			m.historyIdx = -1
			m.pendingInput = ""

			m.messages = append(m.messages, message{kind: msgUser, content: input})
			m.state = stateThinking
			m.updateCursorStyle()

			// Create cancellable context for this query
			ctx, cancel := context.WithCancel(context.Background())
			m.queryCancel = cancel

			if looksLikeSQL(input) {
				m.thinkingMsg = "Running query..."
				return m, func() tea.Msg {
					defer cancel()
					return m.execQueryWithCtx(ctx, input, false)
				}
			}
			m.thinkingMsg = "Generating SQL..."
			return m, func() tea.Msg {
				defer cancel()
				return m.startNLWithCtx(ctx, input)
			}

		case "tab":
			if m.state != stateIdle {
				break
			}
			m = m.handleTabCompletion()
			return m, nil
		}

	case queryResultMsg:
		m.state = stateIdle
		m.queryCancel = nil // query completed, cancel is no-op now
		m.updateCursorStyle()
		if msg.err != nil {
			m.messages = append(m.messages, message{kind: msgError, content: errorBubble(msg.err.Error())})
		} else {
			m.messages = append(m.messages, message{kind: msgResult, content: msg.content})
		}

	case connResultMsg:
		if msg.err != nil {
			m.messages = append(m.messages, message{kind: msgError, content: errorBubble(msg.err.Error())})
		} else {
			conn, err := db.ActiveConnection()
			if err == nil {
				m.conn = conn
			}
			m.messages = append(m.messages, message{kind: msgCmd, content: successBubble(msg.name)})
			// Auto-introspect and cache schema after connect so NL queries work immediately
			m.state = stateThinking
			m.updateCursorStyle()
			return m, m.syncIntrospectCmd()
		}

	case introspectResultMsg:
		m.state = stateIdle
		m.updateCursorStyle()
		if msg.err != nil {
			m.messages = append(m.messages, message{kind: msgError, content: errorBubble(msg.err.Error())})
		} else {
			m.messages = append(m.messages, message{kind: msgCmd, content: msg.content})
		}

	case spinner.TickMsg:
		if m.state != stateIdle {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Always feed non-control keys into the text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.updateAutocomplete()
	return m, cmd
}

// ── Dot Commands ──

func (m Model) handleDotCommand(input string) (tea.Model, tea.Cmd) {
	switch {
	case input == ".quit" || input == ".exit":
		return m, tea.Quit

	case input == ".help":
		m.messages = append(m.messages, message{kind: msgCmd, content: helpBox()})
		return m, nil

	case input == ".tables":
		return m, m.introspectCmd("tables")

	case input == ".schema":
		return m, m.introspectCmd("schema")

	case strings.HasPrefix(input, ".connect "):
		dsn := strings.TrimPrefix(input, ".connect ")
		return m, m.connectCmd(dsn)

	case input == ".refresh":
		return m, m.syncIntrospectCmd()

	case input == ".history":
		return m, m.historyCmd()

	case strings.HasPrefix(input, ".export "):
		filename := strings.TrimPrefix(input, ".export ")
		filename = strings.TrimSpace(filename)
		if filename == "" {
			m.messages = append(m.messages, message{kind: msgError, content: errorBubble("Usage: .export <filename> (.csv, .json, .md)")})
			return m, nil
		}
		return m, m.exportCmd(filename)

	case strings.HasPrefix(input, ".replay "):
		idxStr := strings.TrimPrefix(input, ".replay ")
		idxStr = strings.TrimSpace(idxStr)
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 1 {
			m.messages = append(m.messages, message{kind: msgError, content: errorBubble("Usage: .replay <N> — N is the # from .history (1 = most recent)")})
			return m, nil
		}
		return m, m.replayCmd(idx)

	case input == ".info":
		return m, m.infoCmd()

	case input == ".readonly":
		m.readonly = !m.readonly
		status := "OFF"
		if m.readonly {
			status = "ON"
		}
		m.messages = append(m.messages, message{kind: msgCmd, content: successBubble("Read-only mode: " + status)})
		m.updateCursorStyle()
		return m, nil

	case strings.HasPrefix(input, ".save "):
		name := strings.TrimPrefix(input, ".save ")
		name = strings.TrimSpace(name)
		if name == "" {
			m.messages = append(m.messages, message{kind: msgError, content: errorBubble("Usage: .save <name> — saves the last query from history")})
			return m, nil
		}
		return m, m.saveCmd(name)

	case strings.HasPrefix(input, ".run "):
		name := strings.TrimPrefix(input, ".run ")
		name = strings.TrimSpace(name)
		if name == "" {
			m.messages = append(m.messages, message{kind: msgError, content: errorBubble("Usage: .run <name> — runs a saved query")})
			return m, nil
		}
		return m, m.runCmd(name)

	case input == ".saved":
		return m, m.savedListCmd()

	default:
		// Check if it looks like a typo of a known command
		if best, dist := fuzzyMatchLevenshtein(input); dist <= 2 {
			m.messages = append(m.messages, message{kind: msgCmd, content: "  " + lipgloss.NewStyle().Foreground(Yellow).Render("💡 Did you mean") + " " + HelpCmdStyle.Render(best) + "?"})
		} else {
			m.messages = append(m.messages, message{kind: msgError, content: errorBubble("unknown command: " + input + "\n  Type .help for available commands")})
		}
		return m, nil
	}
}

// ── Cmds (goroutine-backed) ──

func (m Model) connectCmd(dsn string) tea.Cmd {
	return func() tea.Msg {
		conn, err := db.Connect(dsn)
		if err != nil {
			return connResultMsg{err: err}
		}
		// Save DSN so it auto-connects on next launch
		_ = db.SaveDSN(dsn)
		return connResultMsg{name: conn.Name()}
	}
}

func (m Model) introspectCmd(mode string) tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return introspectResultMsg{err: fmt.Errorf("no database connected — use .connect <dsn>")}
		}
		schema, err := m.conn.Introspect(context.Background())
		if err != nil {
			return introspectResultMsg{err: fmt.Errorf("introspect: %w", err)}
		}
		// Cache schema for NL queries
		_ = db.SaveSchema(schema)
		var b strings.Builder
		if mode == "tables" {
			b.WriteString(fmt.Sprintf("  📦 Found %d tables in %s:\n", len(schema.Tables), schema.DBName))
			for _, t := range schema.Tables {
				b.WriteString(fmt.Sprintf("    🗂  %s (%d columns)\n", t.Name, len(t.Columns)))
			}
		} else {
			for _, t := range schema.Tables {
				b.WriteString(fmt.Sprintf("  📦 %s (%d cols, %d indexes):\n", t.Name, len(t.Columns), len(t.Indexes)))
				for _, c := range t.Columns {
					pk := ""
					if c.IsPK {
						pk = " 🔑"
					}
					nullable := ""
					if c.IsNullable {
						nullable = " nullable"
					}
					b.WriteString(fmt.Sprintf("    ├─ %s %s%s%s\n", c.Name, c.Type, pk, nullable))
				}
			}
		}
		return introspectResultMsg{content: b.String()}
	}
}

// syncIntrospectCmd introspects the connected database, caches the schema,
// and returns a confirmation message. Used after .connect to prepare for NL queries.
func (m Model) syncIntrospectCmd() tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return introspectResultMsg{err: fmt.Errorf("no database connected")}
		}
		schema, err := m.conn.Introspect(context.Background())
		if err != nil {
			return introspectResultMsg{err: fmt.Errorf("introspect: %w", err)}
		}
		if err := db.SaveSchema(schema); err != nil {
			return introspectResultMsg{content: fmt.Sprintf("  📦 %d tables — schema read ✅ (cache write: %v)", len(schema.Tables), err)}
		}
		return introspectResultMsg{content: fmt.Sprintf("  📦 %d tables — schema cached ✅", len(schema.Tables))}
	}
}

func (m Model) historyCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := history.List(20)
		if err != nil {
			return introspectResultMsg{content: "  🤖 ⚠ " + err.Error()}
		}
		if len(entries) == 0 {
			return introspectResultMsg{content: "  🤖 No questions yet. Ask me something!"}
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("  🤖 Last %d questions:\n", len(entries)))
		for _, e := range entries {
			icon := "💬"
			if !e.WasNaturalLanguage {
				icon = "🔤"
			}
			timeStr := e.ExecutedAt.Format("15:04:05")
			q := e.Question
			if len(q) > 55 {
				q = q[:52] + "..."
			}
			b.WriteString(fmt.Sprintf("    %s [%s] %s\n", icon, timeStr, q))
		}
		return introspectResultMsg{content: b.String()}
	}
}

// ── Context-aware Query Methods ──

// execQueryWithCtx runs SQL with a cancellable context and saves result for .export.
func (m Model) execQueryWithCtx(ctx context.Context, sql string, isNL bool) tea.Msg {
	if m.conn == nil {
		return queryResultMsg{err: fmt.Errorf("no database connected — use .connect <dsn>")}
	}

	startTime := time.Now()
	rows, err := m.conn.Query(ctx, sql)
	if err != nil {
		return queryResultMsg{err: fmt.Errorf("query failed: %w", db.Friendly(err))}
	}
	defer rows.Close()

	elapsed := time.Since(startTime).Seconds() * 1000
	cols := rows.Columns()
	var resultRows [][]string
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))

	for rows.Next() {
		select {
		case <-ctx.Done():
			return queryResultMsg{err: fmt.Errorf("query cancelled")}
		default:
		}
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return queryResultMsg{err: fmt.Errorf("scan: %w", err)}
		}
		row := make([]string, len(cols))
		for i, v := range vals {
			switch val := v.(type) {
			case []byte:
				row[i] = string(val)
			case nil:
				row[i] = "NULL"
			default:
				row[i] = fmt.Sprint(val)
			}
		}
		resultRows = append(resultRows, row)
	}

	_ = history.Record(history.Entry{
		SQLGenerated:       sql,
		DatabaseName:       m.conn.Name(),
		ExecutedAt:         startTime,
		ExecutionTimeMs:    elapsed,
		RowCount:           len(resultRows),
		WasNaturalLanguage: isNL,
	})

	// Save for .export (via history re-execution in exportCmd)

	var b strings.Builder
	if isNL {
		b.WriteString(sqlPreview(sql))
		b.WriteString("\n\n")
	}
	res := display.Result{Columns: cols, Rows: resultRows}
	if err := display.Print(&b, res, m.format); err != nil {
		return queryResultMsg{err: fmt.Errorf("print: %w", err)}
	}
	plural := "rows"
	if len(resultRows) == 1 {
		plural = "row"
	}
	b.WriteString(fmt.Sprintf("\n\n  %s %d %s in %.0fms", Dot(false, false), len(resultRows), plural, elapsed))
	if len(cols) > 8 {
		b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("\n  💡 Wide results — try .export results.csv"))
	}

	return queryResultMsg{content: b.String()}
}

// startNLWithCtx runs an NL query with a cancellable context and saves result for .export.
func (m Model) startNLWithCtx(ctx context.Context, question string) tea.Msg {
	if m.conn == nil {
		return queryResultMsg{err: fmt.Errorf("no database connected — use .connect <dsn>")}
	}

	schema, err := db.LoadSchema()
	if err != nil {
		return queryResultMsg{err: fmt.Errorf("no schema cache — run 'basemake connect' first: %w", err)}
	}

	provider, _ := ai.SelectedProvider()
	providerName := ""
	if provider != nil {
		providerName = provider.Name()
	}

	dialect := m.conn.Dialect()
	prompt := history.BuildPromptWithHistory(schema.SchemaForPrompt(), 5, dialect)

	ch, err := ai.QuestionToSQLStream(ctx, prompt, question)
	if err != nil {
		return queryResultMsg{err: fmt.Errorf("AI error: %w", err)}
	}

	var sql strings.Builder
	for token := range ch {
		sql.WriteString(token)
	}
	sqlStr := strings.TrimSpace(sql.String())

	// Auto-add LIMIT for SELECT queries to prevent massive result sets
	sqlStr = autoLimit(sqlStr)

	// Validate AI-generated SQL — retry once if invalid
	retryCount := 0
	maxRetries := 1
	for {
		if _, err := m.conn.ExplainNoAnalyze(ctx, sqlStr); err != nil {
			if retryCount < maxRetries {
				retryCount++
				// Feed error back to AI for self-correction
				retryPrompt := prompt + "\n\nThe previous SQL was invalid. Error: " + err.Error() + "\nPlease fix the SQL query for this question: " + question
				ch, err := ai.QuestionToSQLStream(ctx, retryPrompt, question)
				if err != nil {
					return queryResultMsg{err: fmt.Errorf("AI error on retry: %w", err)}
				}
				var retrySQL strings.Builder
				for token := range ch {
					retrySQL.WriteString(token)
				}
				sqlStr = strings.TrimSpace(retrySQL.String())
				sqlStr = autoLimit(sqlStr)
				continue
			}
			return queryResultMsg{err: fmt.Errorf("generated SQL is still invalid after retry:\n  %s\n  %w", sqlStr, db.Friendly(err))}
		}
		break
	}

	startTime := time.Now()
	rows, err := m.conn.Query(ctx, sqlStr)
	if err != nil {
		return queryResultMsg{err: fmt.Errorf("query failed: %w", db.Friendly(err))}
	}
	defer rows.Close()

	elapsed := time.Since(startTime).Seconds() * 1000
	cols := rows.Columns()
	var resultRows [][]string
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))

	for rows.Next() {
		select {
		case <-ctx.Done():
			return queryResultMsg{err: fmt.Errorf("query cancelled")}
		default:
		}
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return queryResultMsg{err: fmt.Errorf("scan: %w", err)}
		}
		row := make([]string, len(cols))
		for i, v := range vals {
			switch val := v.(type) {
			case []byte:
				row[i] = string(val)
			case nil:
				row[i] = "NULL"
			default:
				row[i] = fmt.Sprint(val)
			}
		}
		resultRows = append(resultRows, row)
	}

	_ = history.Record(history.Entry{
		Question:           question,
		SQLGenerated:       sqlStr,
		DatabaseName:       m.conn.Name(),
		ExecutedAt:         startTime,
		ExecutionTimeMs:    elapsed,
		RowCount:           len(resultRows),
		WasNaturalLanguage: true,
		ProviderUsed:       providerName,
	})

	// Save for .export (via history re-execution in exportCmd)

	var b strings.Builder
	b.WriteString(sqlPreview(sqlStr))
	b.WriteString("\n\n")
	res := display.Result{Columns: cols, Rows: resultRows}
	if err := display.Print(&b, res, m.format); err != nil {
		return queryResultMsg{err: fmt.Errorf("print: %w", err)}
	}
	plural := "rows"
	if len(resultRows) == 1 {
		plural = "row"
	}
	b.WriteString(fmt.Sprintf("\n\n  %s %d %s in %.0fms", Dot(false, false), len(resultRows), plural, elapsed))
	if len(cols) > 8 {
		b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("\n  💡 Wide results — try .export results.csv"))
	}

	return queryResultMsg{content: b.String()}
}

// ── New Dot Command Methods ──

func (m Model) exportCmd(filename string) tea.Cmd {
	return func() tea.Msg {
		// Load last entry from history to get the SQL and results
		entries, err := history.List(1)
		if err != nil {
			return introspectResultMsg{content: "  ⚠ No history: " + err.Error()}
		}
		if len(entries) == 0 {
			return introspectResultMsg{content: "  ⚠ No queries in history yet"}
		}

		entry := entries[0]

		// Re-execute the query to get fresh results
		if m.conn == nil {
			return introspectResultMsg{content: "  ⚠ No database connected"}
		}

		rows, err := m.conn.Query(context.Background(), entry.SQLGenerated)
		if err != nil {
			return introspectResultMsg{content: "  ⚠ Re-execute failed: " + err.Error()}
		}
		defer rows.Close()

		cols := rows.Columns()
		var resultRows [][]string
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for rows.Next() {
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return introspectResultMsg{content: "  ⚠ Scan failed: " + err.Error()}
			}
			row := make([]string, len(cols))
			for i, v := range vals {
				switch val := v.(type) {
				case []byte:
					row[i] = string(val)
				case nil:
					row[i] = "NULL"
				default:
					row[i] = fmt.Sprint(val)
				}
			}
			resultRows = append(resultRows, row)
		}

		// Determine format from file extension
		var exportFormat display.Format
		switch {
		case strings.HasSuffix(filename, ".json"):
			exportFormat = display.FormatJSON
		case strings.HasSuffix(filename, ".csv"):
			exportFormat = display.FormatCSV
		case strings.HasSuffix(filename, ".md"):
			exportFormat = display.FormatTable
		default:
			filename += ".csv"
			exportFormat = display.FormatCSV
		}

		f, err := os.Create(filename)
		if err != nil {
			return introspectResultMsg{content: "  ⚠ Cannot create file: " + err.Error()}
		}
		defer f.Close()

		plural := "rows"
		if len(resultRows) == 1 {
			plural = "row"
		}
		msg := fmt.Sprintf("(%d %s)", len(resultRows), plural)
		res := display.Result{Columns: cols, Rows: resultRows, Message: msg}
		if err := display.Print(f, res, exportFormat); err != nil {
			return introspectResultMsg{content: "  ⚠ Write failed: " + err.Error()}
		}

		return introspectResultMsg{content: fmt.Sprintf("  💾 Exported %d %s to %s", len(resultRows), plural, filename)}
	}
}

func (m Model) replayCmd(idx int) tea.Cmd {
	return func() tea.Msg {
		entries, err := history.List(idx)
		if err != nil {
			return queryResultMsg{err: fmt.Errorf("history: %w", err)}
		}
		if idx > len(entries) {
			return queryResultMsg{err: fmt.Errorf("only %d entries in history", len(entries))}
		}

		entry := entries[idx-1] // .history shows most recent first, idx 1 = most recent
		sql := entry.SQLGenerated
		if sql == "" {
			return queryResultMsg{err: fmt.Errorf("entry %d has no SQL — was it a command?", idx)}
		}

		// Show what we're re-running
		_ = history.Record(history.Entry{
			Question:           ".replay " + strconv.Itoa(idx) + ": " + entry.Question,
			SQLGenerated:       sql,
			DatabaseName:       m.conn.Name(),
			WasNaturalLanguage: entry.WasNaturalLanguage,
		})

		return m.execQueryWithCtx(context.Background(), sql, entry.WasNaturalLanguage)
	}
}

func (m Model) infoCmd() tea.Cmd {
	return func() tea.Msg {
		var b strings.Builder
		b.WriteString("  ════════════════════ Status ════════════════════\n")

		if m.conn != nil {
			b.WriteString(fmt.Sprintf("  Database:   %s\n", m.conn.Name()))
			b.WriteString(fmt.Sprintf("  Dialect:    %s\n", m.conn.Dialect()))
		} else {
			b.WriteString("  Database:   ❌ Not connected\n")
		}

		provider, err := ai.SelectedProvider()
		if err != nil {
			b.WriteString(fmt.Sprintf("  AI:         ❌ %s\n", err.Error()))
		} else {
			b.WriteString(fmt.Sprintf("  AI:         ✅ %s\n", provider.Name()))
		}

		ro := "OFF"
		if m.readonly {
			ro = "ON"
		}
		b.WriteString(fmt.Sprintf("  Read-only:  %s\n", ro))
		b.WriteString(fmt.Sprintf("  Format:     %s\n", formatLabel(m.format)))

		b.WriteString(fmt.Sprintf("  Version:    %s\n", m.version))

		b.WriteString("  ═══════════════════════════════════════════\n")

		return introspectResultMsg{content: b.String()}
	}
}

func formatLabel(f display.Format) string {
	switch f {
	case display.FormatJSON:
		return "json"
	case display.FormatCSV:
		return "csv"
	default:
		return "table"
	}
}

// ── Tab Completion ──

// handleTabCompletion completes the current word by cycling through matching table names.
// Returns updated model with input modified or completion state reset.
func (m Model) handleTabCompletion() Model {
	input := m.input.Value()
	if input == "" {
		return m
	}

	// Find the current word (last whitespace-delimited token)
	trimmed := strings.TrimRight(input, " 	")
	var currentWord string
	lastSpace := strings.LastIndex(trimmed, " ")
	if lastSpace >= 0 {
		currentWord = trimmed[lastSpace+1:]
	} else {
		currentWord = trimmed
	}

	if currentWord == "" {
		return m
	}

	// If the word changed since last tab, reset completion state
	if currentWord != m.tabPrefix {
		schema, err := db.LoadSchema()
		if err != nil {
			return m
		}
		m.tabPrefix = currentWord
		m.tabIndex = 0
		m.tabMatches = nil

		prefix := strings.ToLower(currentWord)
		for _, t := range schema.Tables {
			if strings.HasPrefix(strings.ToLower(t.Name), prefix) {
				m.tabMatches = append(m.tabMatches, t.Name)
			}
		}
		// Also complete column names with table prefix (e.g. "orders.tot")
		for _, t := range schema.Tables {
			for _, c := range t.Columns {
				fullName := t.Name + "." + c.Name
				if strings.HasPrefix(strings.ToLower(fullName), prefix) {
					m.tabMatches = append(m.tabMatches, fullName)
				}
			}
		}
	}

	if len(m.tabMatches) == 0 {
		return m
	}

	// Replace the current word with the match
	match := m.tabMatches[m.tabIndex]
	newInput := trimmed[:lastSpace+1] + match

	m.input.SetValue(newInput)
	m.tabIndex = (m.tabIndex + 1) % len(m.tabMatches)

	return m
}

// ── Saved Queries ──

// savedQuery represents a named, saved query.
type savedQuery struct {
	Name     string `json:"name"`
	Question string `json:"question"`
	SQL      string `json:"sql"`
}

func savedQueriesPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".basemake")
	return filepath.Join(dir, "saved-queries.json")
}

func loadSavedQueries() ([]savedQuery, error) {
	path := savedQueriesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var queries []savedQuery
	if err := json.Unmarshal(data, &queries); err != nil {
		return nil, err
	}
	return queries, nil
}

func saveQueryToDisk(q savedQuery) error {
	queries, _ := loadSavedQueries()
	// Replace if name already exists
	found := false
	for i, existing := range queries {
		if existing.Name == q.Name {
			queries[i] = q
			found = true
			break
		}
	}
	if !found {
		queries = append(queries, q)
	}
	data, err := json.MarshalIndent(queries, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(savedQueriesPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(savedQueriesPath(), data, 0644)
}

func (m Model) saveCmd(name string) tea.Cmd {
	return func() tea.Msg {
		entries, err := history.List(1)
		if err != nil || len(entries) == 0 {
			return introspectResultMsg{content: "  ⚠ Nothing to save — run a query first"}
		}
		entry := entries[0]
		if entry.SQLGenerated == "" {
			return introspectResultMsg{content: "  ⚠ Nothing to save — most recent entry has no SQL"}
		}
		q := savedQuery{
			Name:     name,
			Question: entry.Question,
			SQL:      entry.SQLGenerated,
		}
		if err := saveQueryToDisk(q); err != nil {
			return introspectResultMsg{content: "  ⚠ Failed to save: " + err.Error()}
		}
		return introspectResultMsg{content: fmt.Sprintf("  💾 Saved as \"%s\"", name)}
	}
}

func (m Model) runCmd(name string) tea.Cmd {
	return func() tea.Msg {
		queries, err := loadSavedQueries()
		if err != nil {
			return introspectResultMsg{content: "  ⚠ Failed to load saved queries: " + err.Error()}
		}
		for _, q := range queries {
			if q.Name == name {
				// Re-run the saved SQL
				return m.execQueryWithCtx(context.Background(), q.SQL, q.Question != "")
			}
		}
		return introspectResultMsg{content: fmt.Sprintf("  ⚠ No saved query named \"%s\" — use .saved to list them", name)}
	}
}

func (m Model) savedListCmd() tea.Cmd {
	return func() tea.Msg {
		queries, err := loadSavedQueries()
		if err != nil {
			return introspectResultMsg{content: "  ⚠ " + err.Error()}
		}
		if len(queries) == 0 {
			return introspectResultMsg{content: "  📋 No saved queries yet. Use .save <name> to save one."}
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("  📋 %d saved queries:\n\n", len(queries)))
		for _, q := range queries {
			question := q.Question
			if question == "" {
				question = "(direct SQL)"
			}
			if len(question) > 50 {
				question = question[:47] + "..."
			}
			sql := q.SQL
			if len(sql) > 60 {
				sql = sql[:57] + "..."
			}
			b.WriteString(fmt.Sprintf("    %s  %s\n      ─╴%s\n\n", q.Name, question, sql))
		}
		return introspectResultMsg{content: b.String()}
	}
}

// ── View ──

func (m Model) View() string {
	var b strings.Builder

	for _, msg := range m.messages {
		switch msg.kind {
		case msgUser:
			b.WriteString(UserPromptStyle.Render("  You > "))
			b.WriteString(lipgloss.NewStyle().Foreground(Text).Render(msg.content))
			b.WriteString("\n")
		case msgBot:
			b.WriteString(msg.content)
			b.WriteString("\n")
		case msgResult:
			b.WriteString(SubBoxStyle.Render(msg.content))
			b.WriteString("\n")
		case msgCmd:
			b.WriteString(msg.content)
			b.WriteString("\n")
		case msgError:
			b.WriteString(msg.content)
			b.WriteString("\n")
		}
	}

	if m.state == stateThinking {
		b.WriteString("\n  " + m.spinner.View() + " " + ThinkingStyle.Render(m.thinkingMsg) + "\n")
	}

	b.WriteString("\n" + lipgloss.NewStyle().Foreground(DimText).Render(strings.Repeat("─", min(60, max(20, m.input.Width+4)))) + "\n")

	prompt := UserPromptStyle.Render("  You > ")
	b.WriteString(prompt + m.input.View())
	b.WriteString("\n")

	// Dot command autocomplete suggestions
	if len(m.autocompleteMatches) > 0 {
		for _, match := range m.autocompleteMatches {
			b.WriteString("  " + match + "\n")
		}
	}

	return b.String()
}

// ── Helpers ──

func looksLikeSQL(s string) bool {
	trimmed := strings.TrimSpace(s)
	upper := strings.ToUpper(trimmed)
	keywords := []string{"SELECT", "WITH", "EXPLAIN", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "TRUNCATE"}
	for _, kw := range keywords {
		if len(upper) >= len(kw) && upper[:len(kw)] == kw {
			return true
		}
	}
	return false
}

// autoLimit appends LIMIT 100 to SELECT queries that don't already have a LIMIT clause.
// This prevents AI-generated queries from returning millions of rows and freezing the terminal.
func autoLimit(sql string) string {
	trimmed := strings.TrimSpace(sql)
	upper := strings.ToUpper(trimmed)
	// Only auto-limit SELECT queries
	if !strings.HasPrefix(upper, "SELECT") {
		return sql
	}
	// Don't add LIMIT if one already exists or it's an aggregate
	if strings.Contains(upper, "LIMIT") {
		return sql
	}
	return sql + "\nLIMIT 100"
}

func aiProviderLabel() string {
	cfg, _ := config.Load()
	provider := os.Getenv("AI_PROVIDER")
	if provider == "" {
		provider = cfg.AIProvider
	}
	if provider == "" {
		provider = "openai"
	}

	model := ""
	switch provider {
	case "openai":
		model = os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = cfg.OpenAIModel
		}
		if model == "" {
			model = "gpt-4"
		}
	case "opencode":
		model = os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = os.Getenv("OPENCODE_MODEL")
		}
		if model == "" {
			model = cfg.OpenAIModel
		}
		if model == "" {
			model = "deepseek-chat"
		}
	case "anthropic":
		model = os.Getenv("ANTHROPIC_MODEL")
		if model == "" {
			model = cfg.AnthropicModel
		}
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
	case "ollama":
		model = os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = cfg.OllamaModel
		}
		if model == "" {
			model = "llama3"
		}
	}

	if model != "" {
		return strings.ToUpper(provider) + "/" + model
	}
	return strings.ToUpper(provider)
}

// ── Render Components ──

func fullStartupView(conn db.Database, aiLabel, version string) string {
	// Extract provider and model
	provider := aiLabel
	model := ""
	if idx := strings.Index(aiLabel, "/"); idx >= 0 {
		provider = aiLabel[:idx]
		model = aiLabel[idx+1:]
	}

	dbName := ""
	connected := false
	if conn != nil {
		dbName = conn.Name()
		connected = true
	}

	// Use the spec startup screen layout
	screen := StartupScreen(logoASCII, version, provider, model, dbName, connected)

	// Add contextual hint
	var hint string
	if !connected {
		hint = "\n\n  💡 Not connected to a database.\n     Use .connect postgres://user@localhost/mydb or run basemake init"
	} else if _, err := ai.SelectedProvider(); err != nil {
		hint = "\n\n  💡 AI queries need an API key.\n     Run 'basemake init' to set one up, or use raw SQL queries"
	}

	// Wrap in the TUI welcome
	return BoxStyle.Render(screen + hint)
}

func helpBox() string {
	lines := []string{
		HelpHeaderStyle.Render("  ⚡ Commands"),
		"",
		"    " + HelpCmdStyle.Render(".help") + "    " + HelpDescStyle.Render("Show this help") + "  " + HelpHintStyle.Render("→ you're looking at it"),
		"    " + HelpCmdStyle.Render(".quit") + "    " + HelpDescStyle.Render("Exit basemake"),
		"    " + HelpCmdStyle.Render(".tables") + "  " + HelpDescStyle.Render("List tables in the current database"),
		"    " + HelpCmdStyle.Render(".schema") + "  " + HelpDescStyle.Render("Show full database schema"),
		"    " + HelpCmdStyle.Render(".connect") + " " + HelpDescStyle.Render("<dsn> — Connect to a database"),
		"    " + HelpCmdStyle.Render(".refresh") + " " + HelpDescStyle.Render("Re-introspect and cache schema"),
		"    " + HelpCmdStyle.Render(".history") + " " + HelpDescStyle.Render("Show past questions"),
		"    " + HelpCmdStyle.Render(".replay") + "  " + HelpDescStyle.Render("<N> — Re-run query N from history"),
		"    " + HelpCmdStyle.Render(".export") + "  " + HelpDescStyle.Render("<file> — Save last result (.csv, .json, .md)"),
		"    " + HelpCmdStyle.Render(".info") + "    " + HelpDescStyle.Render("Show connection and AI status"),
		"    " + HelpCmdStyle.Render(".readonly") + " " + HelpDescStyle.Render("Toggle write protection on/off"),
		"    " + HelpCmdStyle.Render(".save") + "    " + HelpDescStyle.Render("<name> — Save last query as a bookmark"),
		"    " + HelpCmdStyle.Render(".run") + "     " + HelpDescStyle.Render("<name> — Run a saved query"),
		"    " + HelpCmdStyle.Render(".saved") + "   " + HelpDescStyle.Render("List all saved queries"),
		"",
		HelpHeaderStyle.Render("  ⌨️  Keyboard"),
		"",
		"    " + HelpDescStyle.Render("Enter") + "      " + HelpHintStyle.Render("Run query or send message"),
		"    " + HelpDescStyle.Render("Tab") + "       " + HelpHintStyle.Render("Complete table/column names"),
		"    " + HelpDescStyle.Render("Esc/Ctrl+C") + "  " + HelpHintStyle.Render("Cancel running query"),
		"    " + HelpDescStyle.Render("Ctrl+C") + "   " + HelpHintStyle.Render("Exit (when idle)"),
		"",
		HelpHeaderStyle.Render("  💬 Queries"),
		"",
		"    " + HelpDescStyle.Render("Type a question in plain English,"),
		"    " + HelpDescStyle.Render("or write raw SQL directly."),
		"",
		"    " + HelpHintStyle.Render("Examples:") + " " + HelpDescStyle.Render("\"show me users who signed up last week\""),
		"    " + HelpDescStyle.Render("          \"SELECT * FROM orders LIMIT 5\""),
	}
	return BoxStyle.Render(strings.Join(lines, "\n"))
}

func errorBubble(msg string) string {
	return lipgloss.NewStyle().
		Foreground(Red).
		Render("  ⚠ " + msg)
}

func successBubble(msg string) string {
	return "  " + lipgloss.NewStyle().Foreground(Green).Render("✅") + " " + lipgloss.NewStyle().Foreground(Text).Render(msg)
}

// ── Dot Command Autocomplete ──

type dotCmd struct {
	cmd  string
	desc string
}

var dotCommands = []dotCmd{
	{".help", "Show this help"},
	{".quit", "Exit basemake"},
	{".exit", "Exit basemake"},
	{".tables", "List tables in the current database"},
	{".schema", "Show full database schema"},
	{".connect", "Connect to a database: .connect <dsn>"},
	{".refresh", "Re-introspect and cache schema"},
	{".history", "Show past questions"},
	{".replay", "Re-run query from history: .replay <N>"},
	{".export", "Save last result: .export <.csv|.json|.md>"},
	{".info", "Show connection and AI status"},
	{".readonly", "Toggle write protection on/off"},
	{".save", "Save last query as a bookmark: .save <name>"},
	{".run", "Run a saved query: .run <name>"},
	{".saved", "List all saved queries"},
}

// updateAutocomplete filters dot commands matching the current input prefix
func (m *Model) updateAutocomplete() {
	val := strings.TrimSpace(m.input.Value())
	m.autocompleteMatches = nil
	if !strings.HasPrefix(val, ".") || len(val) < 2 || m.state != stateIdle {
		return
	}
	for _, dc := range dotCommands {
		if strings.HasPrefix(dc.cmd, val) {
			m.autocompleteMatches = append(m.autocompleteMatches, dc.cmd+"  "+lipgloss.NewStyle().Foreground(DimText).Render(dc.desc))
		}
	}
}

// fuzzyMatchLevenshtein returns the closest matching command and its distance
func fuzzyMatchLevenshtein(input string) (string, int) {
	bestCmd := ""
	bestDist := 3 // only suggest if distance <= 2
	for _, dc := range dotCommands {
		dist := levenshtein(input, dc.cmd)
		if dist < bestDist {
			bestDist = dist
			bestCmd = dc.cmd
		}
	}
	return bestCmd, bestDist
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	// Optimize: use shorter string as columns
	if len(a) > len(b) {
		a, b = b, a
	}
	la, lb := len(a), len(b)
	row := make([]int, la+1)
	for i := range row {
		row[i] = i
	}
	for i := 1; i <= lb; i++ {
		prev := i
		for j := 1; j <= la; j++ {
			cur := row[j-1]
			if b[i-1] != a[j-1] {
				cur = min(prev, min(row[j], row[j-1])) + 1
			}
			row[j-1] = prev
			prev = cur
		}
		row[la] = prev
	}
	return row[la]
}

// updateCursorStyle changes the cursor color based on current state
func (m *Model) updateCursorStyle() {
	switch m.state {
	case stateThinking:
		m.input.Cursor.Style = lipgloss.NewStyle().Foreground(Red)
		m.input.Cursor.Blink = true
	default:
		if m.readonly {
			m.input.Cursor.Style = lipgloss.NewStyle().Foreground(Orange)
		} else if m.conn != nil {
			m.input.Cursor.Style = lipgloss.NewStyle().Foreground(Green)
		} else {
			m.input.Cursor.Style = lipgloss.NewStyle().Foreground(White)
		}
		m.input.Cursor.Blink = true
	}
}

func sqlPreview(sql string) string {
	if len(sql) > 80 {
		sql = sql[:77] + "..."
	}
	return lipgloss.NewStyle().
		Foreground(DimText).
		Italic(true).
		Render("  ─╴" + sql)
}

// isWriteQuery checks if SQL is a write operation (for read-only guard).
func isWriteQuery(s string) bool {
	trimmed := strings.TrimSpace(s)
	upper := strings.ToUpper(trimmed)
	writeKeywords := []string{"INSERT ", "UPDATE ", "DELETE ", "DROP ", "ALTER ", "CREATE ", "TRUNCATE ", "MERGE "}
	for _, kw := range writeKeywords {
		if len(upper) >= len(kw) && upper[:len(kw)] == kw {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Same as cmd/root.go's BannerASCII.
const logoASCII = `█████                                                          █████              
▒▒███                                                          ▒▒███               
 ▒███████   ██████    █████   ██████  █████████████    ██████   ▒███ █████  ██████ 
 ▒███▒▒███ ▒▒▒▒▒███  ███▒▒   ███▒▒███▒▒███▒▒███▒▒███  ▒▒▒▒▒███  ▒███▒▒███  ███▒▒███
 ▒███ ▒███  ███████ ▒▒█████ ▒███████  ▒███ ▒███ ▒███   ███████  ▒██████▒  ▒███████ 
 ▒███ ▒███ ███▒▒███  ▒▒▒▒███▒███▒▒▒   ▒███ ▒███ ▒███  ███▒▒███  ▒███▒▒███ ▒███▒▒▒  
 ████████ ▒▒████████ ██████ ▒▒██████  █████▒███ █████▒▒████████ ████ █████▒▒██████ 
▒▒▒▒▒▒▒▒   ▒▒▒▒▒▒▒▒ ▒▒▒▒▒▒   ▒▒▒▒▒▒  ▒▒▒▒▒ ▒▒▒ ▒▒▒▒▒  ▒▒▒▒▒▒▒▒ ▒▒▒▒ ▒▒▒▒▒  ▒▒▒▒▒▒`
