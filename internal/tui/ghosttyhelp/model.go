package ghosttyhelp

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type shortcut struct {
	key  string
	desc string
}

// Model renders a list of Ghostty -> tmux shortcuts.
type Model struct {
	width  int
	height int
}

var shortcuts = []shortcut{
	{"Cmd+H/J/K/L", "Navigate panes"},
	{"Cmd+[ / ]", "Prev/next window"},
	{"Cmd+T", "New window"},
	{"Cmd+W", "Close window"},
	{"Cmd+1…9", "Jump to window"},
	{"Cmd+Shift+W", "Kill session"},
	{"Cmd+Shift+H/J/K/L", "Resize panes"},
	{"Cmd+Backspace", "Clear line"},
	{"Cmd+Shift+P", "Command palette"},
	{"Cmd+I", "Toggle this help"},
}

// NewModel creates a help view with the predefined shortcuts.
func NewModel() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Padding(0, 1).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("⌨️  Ghostty → tmux"))
	b.WriteString("\n\n")

	// Shortcuts
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("114")).
		Bold(true).
		Width(22)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	for _, s := range shortcuts {
		b.WriteString(keyStyle.Render(s.key))
		b.WriteString(descStyle.Render(s.desc))
		b.WriteString("\n")
	}

	// Footer note
	b.WriteString("\n")
	noteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Italic(true)
	b.WriteString(noteStyle.Render("Cmd sends tmux prefix automatically"))
	b.WriteString("\n\n")

	// Close hint
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	b.WriteString(hintStyle.Render("esc to close"))

	return b.String()
}
