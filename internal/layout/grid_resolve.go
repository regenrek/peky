package layout

import "strings"

// ResolveGridCommands expands grid commands into a per-pane list.
func ResolveGridCommands(layoutCfg *LayoutConfig, count int) []string {
	commands := make([]string, 0, count)
	if layoutCfg == nil || count <= 0 {
		return commands
	}
	fallback := strings.TrimSpace(layoutCfg.Command)
	if len(layoutCfg.Commands) > 0 {
		for i := 0; i < count; i++ {
			if i < len(layoutCfg.Commands) {
				commands = append(commands, layoutCfg.Commands[i])
				continue
			}
			if fallback != "" {
				commands = append(commands, fallback)
			} else {
				commands = append(commands, "")
			}
		}
		return commands
	}
	if fallback == "" {
		for i := 0; i < count; i++ {
			commands = append(commands, "")
		}
		return commands
	}
	for i := 0; i < count; i++ {
		commands = append(commands, fallback)
	}
	return commands
}

// ResolveGridTitles expands grid titles into a per-pane list.
func ResolveGridTitles(layoutCfg *LayoutConfig, count int) []string {
	titles := make([]string, 0, count)
	if layoutCfg == nil || count <= 0 {
		return titles
	}
	for i := 0; i < count; i++ {
		if i < len(layoutCfg.Titles) {
			titles = append(titles, layoutCfg.Titles[i])
		} else {
			titles = append(titles, "")
		}
	}
	return titles
}
