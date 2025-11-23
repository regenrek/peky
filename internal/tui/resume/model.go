package resume

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var ErrAborted = fmt.Errorf("selection aborted")

func SelectSession(sessions []string) (string, error) {
	if len(sessions) == 0 {
		return "", fmt.Errorf("no sessions provided")
	}
	m := model{sessions: sessions}
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	result := final.(model)
	if result.quitWithoutSelection {
		return "", ErrAborted
	}
	if result.selected == "" {
		return "", fmt.Errorf("no session selected")
	}
	return result.selected, nil
}

type model struct {
	sessions             []string
	cursor               int
	selected             string
	showHelp             bool
	quitWithoutSelection bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.quitWithoutSelection = true
			return m, tea.Quit
		case "enter":
			if len(m.sessions) > 0 {
				m.selected = m.sessions[m.cursor]
			}
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
		case "?":
			m.showHelp = !m.showHelp
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.showHelp {
		return helpView()
	}
	var b strings.Builder
	b.WriteString("Select a tmux session\n\n")
	for i, name := range m.sessions {
		cursor := ""
		if i == m.cursor {
			cursor = "> "
		} else {
			cursor = "  "
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, name))
	}
	b.WriteString("\n? help   q quit   enter attach\n")
	return b.String()
}

func helpView() string {
	return "Key bindings:\n" +
		"  ↑/k  move up\n" +
		"  ↓/j  move down\n" +
		"  enter attach to highlighted session\n" +
		"  q/esc  cancel\n" +
		"  ?  toggle this help\n\nPress any movement key to return."
}
