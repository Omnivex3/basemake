package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/DynamicKarabo/basemake/internal/ai"
	"github.com/DynamicKarabo/basemake/internal/config"
)

// ── Public API ──

// RunProviderSelector launches the interactive provider selection TUI.
// It blocks until the user completes or cancels, then saves the config.
// Returns nil on success, or an error if the user cancelled or something failed.
func RunProviderSelector() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	m := initialProviderModel(cfg)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return fmt.Errorf("provider selector error: %w", err)
	}

	fm := final.(providerModel)
	if fm.aborted {
		return nil // user cancelled, no error
	}

	// Save the config
	if err := fm.cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Print success message
	resultStyle := lipgloss.NewStyle().Foreground(Green).Bold(true)
	fmt.Fprintf(os.Stderr, "\n%s\n", resultStyle.Render("✓ Provider configured"))
	fmt.Fprintf(os.Stderr, "  AI Provider: %s\n", fm.cfg.AIProvider)
	model := resolvedModel(fm.cfg)
	fmt.Fprintf(os.Stderr, "  Model:       %s\n", model)
	fmt.Fprintf(os.Stderr, "  Test:        %s\n", fm.testResult)

	return nil
}

func resolvedModel(cfg *config.Config) string {
	switch cfg.AIProvider {
	case "openai":
		return cfg.OpenAIModel
	case "anthropic":
		return cfg.AnthropicModel
	case "ollama":
		return cfg.OllamaModel
	case "opencode":
		return cfg.OpenCodeModel
	}
	return "?"
}

// ── Provider & Model Definitions ──

type providerOption struct {
	id     string
	name   string
	desc   string
	models []modelOption
}

type modelOption struct {
	id   string
	name string
}

var providers = []providerOption{
	{
		id:   "openai",
		name: "OpenAI",
		desc: "GPT-4o, GPT-4o-mini — the classic",
		models: []modelOption{
			{id: "gpt-4o", name: "GPT-4o"},
			{id: "gpt-4o-mini", name: "GPT-4o-mini"},
			{id: "gpt-4-turbo", name: "GPT-4 Turbo"},
			{id: "gpt-4", name: "GPT-4"},
			{id: "__custom__", name: "Custom model..."},
		},
	},
	{
		id:   "anthropic",
		name: "Anthropic",
		desc: "Claude Sonnet 4, Haiku — smart & fast",
		models: []modelOption{
			{id: "claude-sonnet-4-20250514", name: "Claude Sonnet 4"},
			{id: "claude-3-5-sonnet-20241022", name: "Claude 3.5 Sonnet"},
			{id: "claude-3-haiku-20240307", name: "Claude 3 Haiku"},
			{id: "claude-3-opus-20240229", name: "Claude 3 Opus"},
			{id: "__custom__", name: "Custom model..."},
		},
	},
	{
		id:   "ollama",
		name: "Ollama",
		desc: "Local models — free, private, runs on your machine",
		models: []modelOption{
			{id: "llama3", name: "Llama 3 (8B)"},
			{id: "llama3:70b", name: "Llama 3 (70B)"},
			{id: "mistral", name: "Mistral (7B)"},
			{id: "codellama", name: "Code Llama (7B)"},
			{id: "mixtral:8x7b", name: "Mixtral (8x7B)"},
			{id: "__custom__", name: "Custom model..."},
		},
	},
	{
		id:   "opencode",
		name: "OpenCode",
		desc: "Open models, $10/mo subscription — deepseek-chat included",
		models: []modelOption{
			{id: "deepseek-chat", name: "DeepSeek V3 / deepseek-chat"},
			{id: "__custom__", name: "Custom model..."},
		},
	},
}

// ── Model ──

type selectorState int

const (
	stateSelectProvider selectorState = iota
	stateSelectModel
	stateEnterCustomModel
	stateTesting
	stateDone
)

type providerModel struct {
	cfg         *config.Config
	state       selectorState
	cursor      int
	modelCursor int

	// selected
	selectedProvider *providerOption
	selectedModel    string

	// testing
	testResult string
	testOK     bool

	// custom model input
	modelInput textinput.Model

	// ui
	spinner  spinner.Model
	quitting bool
	aborted  bool

	width, height int
}

func initialProviderModel(cfg *config.Config) providerModel {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(Red)
	s.Spinner = spinner.Dot

	ti := textinput.New()
	ti.Placeholder = "gpt-4o"
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 40

	return providerModel{
		cfg:        cfg,
		state:      stateSelectProvider,
		spinner:    s,
		modelInput: ti,
	}
}

func (m providerModel) Init() tea.Cmd {
	return nil
}

// ── Update ──

func (m providerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.quitting {
			return m, tea.Quit
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			if m.state == stateSelectProvider {
				m.aborted = true
				return m, tea.Quit
			}
			// Go back one step
			switch m.state {
			case stateSelectModel:
				m.state = stateSelectProvider
				m.cursor = 0
				m.selectedProvider = nil
			case stateEnterCustomModel:
				m.state = stateSelectModel
			default:
				m.aborted = true
				return m, tea.Quit
			}
			return m, nil

		case "q", "Q":
			if m.state == stateSelectProvider || m.state == stateDone {
				m.aborted = true
				return m, tea.Quit
			}
			return m, nil
		}

		switch m.state {
		case stateSelectProvider:
			return m.updateProviderSelect(msg)
		case stateSelectModel:
			return m.updateModelSelect(msg)
		case stateEnterCustomModel:
			return m.updateCustomModel(msg)
		case stateTesting:
			// No key handling during test
			return m, nil
		case stateDone:
			if msg.String() == "enter" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

	case testCompleteMsg:
		m.state = stateDone
		m.testResult = msg.result
		m.testOK = msg.success
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// ── Provider Selection ──

func (m providerModel) updateProviderSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(providers)-1 {
			m.cursor++
		}
	case "enter":
		m.selectedProvider = &providers[m.cursor]
		m.modelCursor = 0
		m.state = stateSelectModel
	}

	return m, nil
}

// ── Model Selection ──

func (m providerModel) updateModelSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	models := m.selectedProvider.models

	switch msg.String() {
	case "up", "k":
		if m.modelCursor > 0 {
			m.modelCursor--
		}
	case "down", "j":
		if m.modelCursor < len(models)-1 {
			m.modelCursor++
		}
	case "enter":
		sel := models[m.modelCursor]
		if sel.id == "__custom__" {
			m.modelInput.SetValue("")
			m.state = stateEnterCustomModel
			return m, textinput.Blink
		}
		m.selectedModel = sel.id
		return m, m.startTest()
	}

	return m, nil
}

// ── Custom Model Input ──

func (m providerModel) updateCustomModel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.selectedModel = strings.TrimSpace(m.modelInput.Value())
		if m.selectedModel == "" {
			return m, nil
		}
		return m, m.startTest()
	}

	var cmd tea.Cmd
	m.modelInput, cmd = m.modelInput.Update(msg)
	return m, cmd
}

// ── Testing ──

type testCompleteMsg struct {
	result  string
	success bool
}

func (m *providerModel) startTest() tea.Cmd {
	m.state = stateTesting
	m.testResult = ""
	m.testOK = false

	// Write the selected values to config temporarily
	writeSelection(m.cfg, m.selectedProvider.id, m.selectedModel)

	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			err := ai.PingProvider()
			if err != nil {
				return testCompleteMsg{
					result:  fmt.Sprintf("✗ %v", err),
					success: false,
				}
			}
			return testCompleteMsg{
				result:  "✓ Connected successfully",
				success: true,
			}
		},
	)
}

func writeSelection(cfg *config.Config, providerID, model string) {
	cfg.AIProvider = providerID

	// Clear all model fields, then set the right one
	cfg.OpenAIModel = ""
	cfg.AnthropicModel = ""
	cfg.OllamaModel = ""
	cfg.OpenCodeModel = ""

	switch providerID {
	case "openai":
		cfg.OpenAIModel = model
	case "anthropic":
		cfg.AnthropicModel = model
	case "ollama":
		cfg.OllamaModel = model
	case "opencode":
		cfg.OpenCodeModel = model
	}
}

// ── View ──

func (m providerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header
	b.WriteString(lipgloss.NewStyle().
		Foreground(Red).
		Bold(true).
		Render("◇ basemake — AI Provider Setup"))
	b.WriteString("\n\n")

	switch m.state {
	case stateSelectProvider:
		b.WriteString(m.renderProviderList())
	case stateSelectModel:
		b.WriteString(m.renderModelList())
	case stateEnterCustomModel:
		b.WriteString(m.renderCustomModel())
	case stateTesting:
		b.WriteString(m.renderTesting())
	case stateDone:
		b.WriteString(m.renderDone())
	}

	// Footer help
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m providerModel) renderProviderList() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("Select an AI provider:"))
	b.WriteString("\n\n")

	for i, p := range providers {
		cursor := "  "
		if i == m.cursor {
			cursor = " ◉ "
		}

		nameStyle := lipgloss.NewStyle().Foreground(White).Bold(i == m.cursor)
		if i == m.cursor {
			nameStyle = nameStyle.Foreground(Red)
		}
		descStyle := lipgloss.NewStyle().Foreground(DimText)

		line := fmt.Sprintf("%s %s", cursor, nameStyle.Render(p.name))
		line += fmt.Sprintf("  %s", descStyle.Render(p.desc))
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func (m providerModel) renderModelList() string {
	var b strings.Builder
	p := m.selectedProvider

	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render(
		fmt.Sprintf("Select a model for %s:", p.name)))
	b.WriteString("\n\n")

	for i, mo := range p.models {
		cursor := "  "
		if i == m.modelCursor {
			cursor = " ◉ "
		}

		style := lipgloss.NewStyle().Foreground(White)
		if i == m.modelCursor {
			style = lipgloss.NewStyle().Foreground(Red).Bold(true)
		}

		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(mo.name)))
	}

	return b.String()
}

func (m providerModel) renderCustomModel() string {
	var b strings.Builder
	p := m.selectedProvider

	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render(
		fmt.Sprintf("Enter model name for %s:", p.name)))
	b.WriteString("\n\n")
	b.WriteString(m.modelInput.View())
	b.WriteString("\n")

	return b.String()
}

func (m providerModel) renderTesting() string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render(
		"Testing connection..."))
	b.WriteString("\n\n")
	b.WriteString(m.spinner.View())

	return b.String()
}

func (m providerModel) renderDone() string {
	var b strings.Builder

	if m.testOK {
		b.WriteString(lipgloss.NewStyle().Foreground(Green).Bold(true).Render("✓ All set!"))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(Yellow).Bold(true).Render("✓ Saved (test: " + m.testResult + ")"))
	}
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(White).Render(
		fmt.Sprintf("Provider:  %s", m.cfg.AIProvider)))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(White).Render(
		fmt.Sprintf("Model:     %s", resolvedModel(m.cfg))))
	b.WriteString("\n")

	if m.testResult != "" {
		testStyle := lipgloss.NewStyle().Foreground(Green)
		if !m.testOK {
			testStyle = lipgloss.NewStyle().Foreground(Orange)
		}
		b.WriteString(testStyle.Render(m.testResult))
		b.WriteString("\n")
	}

	return b.String()
}

func (m providerModel) renderHelp() string {
	var b strings.Builder

	helpStyle := lipgloss.NewStyle().Foreground(DimText).Italic(true)
	keyStyle := lipgloss.NewStyle().Foreground(White).Bold(true)

	switch m.state {
	case stateSelectProvider:
		b.WriteString(helpStyle.Render(
			fmt.Sprintf("%s  %s  %s  to move • %s  select • %s  quit",
				keyStyle.Render("↑/↓"),
				keyStyle.Render("j/k"),
				keyStyle.Render("↑/↓"),
				keyStyle.Render("enter"),
				keyStyle.Render("esc/ctrl+c"),
			)))
	case stateSelectModel:
		b.WriteString(helpStyle.Render(
			fmt.Sprintf("%s  %s  to move • %s  select • %s  back",
				keyStyle.Render("↑/↓"),
				keyStyle.Render("j/k"),
				keyStyle.Render("enter"),
				keyStyle.Render("esc"),
			)))
	case stateEnterCustomModel:
		b.WriteString(helpStyle.Render(
			fmt.Sprintf("%s to confirm • %s  back",
				keyStyle.Render("enter"),
				keyStyle.Render("esc"),
			)))
	case stateTesting:
		b.WriteString(helpStyle.Render("Testing provider connectivity..."))
	case stateDone:
		b.WriteString(helpStyle.Render(
			fmt.Sprintf("%s  to exit", keyStyle.Render("enter"))))
	}

	return b.String()
}
