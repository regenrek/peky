package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kregenrek/tmuxman/internal/tmuxctl"
	"github.com/kregenrek/tmuxman/internal/tui/projmux"
)

func main() {
	tmuxBin := flag.String("tmux", "", "path to the tmux binary to execute")
	configPath := flag.String("config", "", "path to the projmux YAML config (defaults to ~/.config/projmux/projects.yml)")
	flag.Parse()

	client, err := tmuxctl.NewClient(*tmuxBin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "projmux: %v\n", err)
		os.Exit(1)
	}

	model, err := projmux.NewModel(client, *configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "projmux: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "projmux: %v\n", err)
		os.Exit(1)
	}
}