package create

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kregenrek/tmuxman/internal/layout"
)

var ErrAborted = errors.New("form aborted")

func Prompt(defaultSession, defaultLayout string) (string, string, error) {
	m := newModel(defaultSession, defaultLayout)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", "", err
	}
	res := final.(model)
	if res.aborted {
		return "", "", ErrAborted
	}
	return strings.TrimSpace(res.inputs[0].Value()), strings.TrimSpace(res.inputs[1].Value()), nil
}

type model struct {
	inputs   [2]textinput.Model
	focused  int
	showHelp bool
	errMsg   string
	aborted  bool
}

func newModel(defaultSession, defaultLayout string) model {
	if strings.TrimSpace(defaultSession) == "" {
		defaultSession = layout.Default.String()
	}
	if strings.TrimSpace(defaultLayout) == "" {
		defaultLayout = layout.Default.String()
	}

	sessionInput := textinput.New()
	sessionInput.Placeholder = "session name"
	sessionInput.SetValue(strings.TrimSpace(defaultSession))
	sessionInput.Prompt = "Session: "
	sessionInput.Focus()

	layoutInput := textinput.New()
	layoutInput.Placeholder = "2x2"
	layoutInput.SetValue(strings.TrimSpace(defaultLayout))
	layoutInput.Prompt = "Layout:  "

	return model{
		inputs:  [2]textinput.Model{sessionInput, layoutInput},
		focused: 0,
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.aborted = true
			return m, tea.Quit
		case "tab", "shift+tab":
			if msg.String() == "tab" {
				m.focused = (m.focused + 1) % len(m.inputs)
			} else {
				m.focused = (m.focused - 1 + len(m.inputs)) % len(m.inputs)
			}
			for i := range m.inputs {
				if i == m.focused {
					m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
		case "?":
			m.showHelp = !m.showHelp
		case "enter":
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			session := strings.TrimSpace(m.inputs[0].Value())
			layoutSpec := strings.TrimSpace(m.inputs[1].Value())
			if session == "" {
				m.errMsg = "session name required"
				return m, nil
			}
			if _, err := layout.Parse(layoutSpec); err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
			m.errMsg = ""
			return m, tea.Quit
		}
	}

	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		if cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.showHelp {
		return helpView()
	}
	nav := lipgloss.NewStyle().Bold(true).Padding(1, 2).Render("tmuxman · Workspace Builder")
	body := lipgloss.NewStyle().Padding(1, 4)
	inputs := fmt.Sprintf("%s\n%s", m.inputs[0].View(), m.inputs[1].View())
	err := ""
	if m.errMsg != "" {
		err = lipgloss.NewStyle().Foreground(lipgloss.Color("210")).Render(m.errMsg)
	}
	footer := lipgloss.NewStyle().Padding(1, 2).Foreground(lipgloss.Color("244")).Render("tab/shift+tab move · enter confirm · ? help · q quit")
	return lipgloss.JoinVertical(lipgloss.Left, nav, body.Render(inputs+"\n"+err), footer)
}

func helpView() string {
	border := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	content := `Key bindings:
  tab / shift+tab  move between inputs
  enter            validate and create session
  ?                toggle help
  q / esc          abort without changes

Layout examples: 2x2, 2x3, 3x2 (rows x columns).
Layouts are limited to 12 panes for readability.`
	return border.Render(content)
}
