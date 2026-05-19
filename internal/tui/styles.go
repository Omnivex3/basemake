package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Colour Palette (from logo SVG) ──
// Primary: #FC0E22 (red)
// Background/text: #020303 (near black)
// Secondary: white for contrast
var (
	Red      = lipgloss.Color("#FC0E22")
	DarkBg   = lipgloss.Color("#020303")
	DarkCard = lipgloss.Color("#0A0A0A")
	White    = lipgloss.Color("#FFFFFF")
	DimText  = lipgloss.Color("#888888")
	Muted    = lipgloss.Color("#555555")
	Text     = lipgloss.Color("#E0E0E0")
	Green    = lipgloss.Color("#22C55E")
	RedDot   = lipgloss.Color("#FC0E22")
	Yellow   = lipgloss.Color("#F59E0B")
	Orange   = lipgloss.Color("#F97316")
)

// ── Base Styles ──

// BoxStyle — thin red border, near-black fill.
var BoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Red).
	Padding(0, 1)

// SubBoxStyle — thinner inner box for results.
var SubBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Red).
	Padding(0, 1)

// ── Status Line (under logo) ──

func StatusLine(version, provider, model, dbName string, connected bool) string {
	v := lipgloss.NewStyle().Foreground(DimText).Render("v" + version)

	providerStr := strings.ToUpper(provider)
	if model != "" {
		providerStr += "/" + model
	}
	p := lipgloss.NewStyle().Foreground(White).Render(providerStr)

	dot := Dot(connected, true)
	db := lipgloss.NewStyle().Foreground(White).Render(dbName)

	parts := []string{lipgloss.NewStyle().Foreground(Red).Render("◆") + " " + v}
	if connected {
		parts = append(parts, dot+" "+db)
	}
	parts = append(parts, p)

	return strings.Join(parts, "  │  ")
}

// Dot returns a coloured circle.
func Dot(connected, isRed bool) string {
	if connected {
		return lipgloss.NewStyle().Foreground(Green).Render("●")
	}
	if isRed {
		return lipgloss.NewStyle().Foreground(Red).Render("●")
	}
	return lipgloss.NewStyle().Foreground(White).Render("●")
}

// ── Logo Colouring ──

// ColoriseLogo renders the ASCII art in #FC0E22 red.
func ColoriseLogo(raw string) string {
	redStyle := lipgloss.NewStyle().Foreground(Red)
	lines := strings.Split(raw, "\n")
	var out []string
	for _, line := range lines {
		coloured := ""
		for _, ch := range line {
			if ch == '█' || ch == '▒' {
				coloured += redStyle.Render(string(ch))
			} else {
				coloured += string(ch)
			}
		}
		out = append(out, coloured)
	}
	return strings.Join(out, "\n")
}

// ── Startup Screen ──

func StartupScreen(logo, version, provider, model, dbName string, connected bool) string {
	var b strings.Builder

	// Coloured logo
	b.WriteString(ColoriseLogo(logo))
	b.WriteString("\n")

	// Version line (no divider yet — spec layout)
	b.WriteString(lipgloss.NewStyle().Foreground(White).Render("basemake " + version))
	b.WriteString("\n")

	// Divider
	div := lipgloss.NewStyle().Foreground(DimText).Render(strings.Repeat("─", 50))
	b.WriteString(div)
	b.WriteString("\n")

	// Status block
	aiLabel := provider
	if model != "" {
		aiLabel += " (" + model + ")"
	}
	dbLabel := dbName
	if dbLabel == "" {
		dbLabel = "Not connected"
	}

	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("AI Provider: "))
	b.WriteString(lipgloss.NewStyle().Foreground(White).Render(aiLabel))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("Database:    "))
	b.WriteString(lipgloss.NewStyle().Foreground(White).Render(dbLabel))
	b.WriteString("\n")

	// Divider
	b.WriteString(div)
	b.WriteString("\n")

	// Description
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(Text).Render("basemake connects to your database, learns your schema,"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(Text).Render("and lets you ask questions in plain English."))
	b.WriteString("\n")
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("  basemake connect postgres://user:***@localhost/mydb"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render(`  basemake "show me users who signed up last week"`))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("  basemake check queries/report.sql --threshold 500ms"))

	return b.String()
}

// ── Text Styles ──

var TitleStyle = lipgloss.NewStyle().
	Foreground(Red).
	Bold(true).
	Padding(0, 1)

var SubtitleStyle = lipgloss.NewStyle().
	Foreground(DimText).
	Padding(0, 1)

var UserPromptStyle = lipgloss.NewStyle().
	Foreground(Red).
	Bold(true)

var BotNameStyle = lipgloss.NewStyle().
	Foreground(Red).
	Bold(true)

var ThinkingStyle = lipgloss.NewStyle().
	Foreground(White).
	Italic(true)

var ErrorStyle = lipgloss.NewStyle().
	Foreground(Red)

var SuccessStyle = lipgloss.NewStyle().
	Foreground(Green)

var ResultMetaStyle = lipgloss.NewStyle().
	Foreground(DimText).
	Italic(true)

var HelpHeaderStyle = lipgloss.NewStyle().
	Foreground(Red).
	Bold(true)

var HelpCmdStyle = lipgloss.NewStyle().
	Foreground(White).
	Bold(true)

var HelpDescStyle = lipgloss.NewStyle().
	Foreground(Text)

var HelpHintStyle = lipgloss.NewStyle().
	Foreground(DimText).
	Italic(true)
