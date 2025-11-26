package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kregenrek/tmuxman/internal/tui/ghosttyhelp"
)

func main() {
	m := ghosttyhelp.NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tmuxhelp: %v\n", err)
		os.Exit(1)
	}
}
