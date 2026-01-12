package app

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type commandID string

type commandArgs struct {
	Raw    string
	Tokens []string
}

type commandSpec struct {
	ID       commandID
	Label    string
	Desc     string
	Aliases  []string
	Shortcut string
	Run      func(*Model, commandArgs) tea.Cmd
}

type commandGroup struct {
	Name     string
	Commands []commandSpec
}

type commandAlias struct {
	key     string
	tokens  []string
	command *commandSpec
}

type commandRegistry struct {
	Groups  []commandGroup
	Aliases []commandAlias
}

func (m *Model) commandRegistry() (commandRegistry, error) {
	var shortcutOpenProject, shortcutCloseProject, shortcutNewSession, shortcutKillSession string
	var shortcutFilter, shortcutHelp, shortcutQuit, shortcutToggleSidebar, shortcutTogglePanes string
	if m.keys != nil {
		shortcutOpenProject = keyLabel(m.keys.openProject)
		shortcutCloseProject = keyLabel(m.keys.closeProject)
		shortcutToggleSidebar = keyLabel(m.keys.toggleSidebar)
		shortcutNewSession = keyLabel(m.keys.newSession)
		shortcutKillSession = keyLabel(m.keys.kill)
		shortcutTogglePanes = keyLabel(m.keys.togglePanes)
		shortcutFilter = keyLabel(m.keys.filter)
		shortcutHelp = keyLabel(m.keys.help)
		shortcutQuit = keyLabel(m.keys.quit)
	}

	groups := []commandGroup{
		{
			Name: "pane",
			Commands: []commandSpec{
				{
					ID:      "pane_add",
					Label:   "Pane: Add pane",
					Desc:    "Add a pane to the current session",
					Aliases: []string{"add", "add-pane", "pane add"},
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						return m.addPaneAuto()
					},
				},
				{
					ID:      "pane_swap",
					Label:   "Pane: Swap pane",
					Desc:    "Swap the selected pane with another",
					Aliases: []string{"pane swap"},
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						m.openPaneSwapPicker()
						return nil
					},
				},
				{
					ID:      "pane_close",
					Label:   "Pane: Close pane",
					Desc:    "Close the selected pane",
					Aliases: []string{"kill", "close-pane", "pane close", "pane kill"},
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						return m.openClosePaneConfirm()
					},
				},
				{
					ID:      "pane_cleanup",
					Label:   "Pane: Cleanup dead panes",
					Desc:    "Restart dead/offline panes in the selected session (or all offline sessions if none are live)",
					Aliases: []string{"cleanup", "pane cleanup", "panes cleanup", "cleanup panes"},
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						return m.cleanupDeadPanes()
					},
				},
				{
					ID:      "pane_rename",
					Label:   "Pane: Rename pane",
					Desc:    "Rename the selected pane title",
					Aliases: []string{"rename", "rename-pane", "pane rename"},
					Run: func(m *Model, args commandArgs) tea.Cmd {
						if strings.TrimSpace(args.Raw) == "" {
							m.openRenamePane()
							return nil
						}
						return m.renamePaneDirect(args.Raw)
					},
				},
			},
		},
		{
			Name: "session",
			Commands: []commandSpec{
				{
					ID:       "session_new",
					Label:    "Session: New session",
					Desc:     "Pick a layout and create a new session",
					Aliases:  []string{"new", "new-session", "session new"},
					Shortcut: shortcutNewSession,
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						m.openLayoutPicker()
						return nil
					},
				},
				{
					ID:       "session_kill",
					Label:    "Session: Close session",
					Desc:     "Close the selected session",
					Aliases:  []string{"close-session", "session close"},
					Shortcut: shortcutKillSession,
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						m.openKillConfirm()
						return nil
					},
				},
				{
					ID:      "session_rename",
					Label:   "Session: Rename session",
					Desc:    "Rename the selected session",
					Aliases: []string{"rename-session", "session rename"},
					Run: func(m *Model, args commandArgs) tea.Cmd {
						if strings.TrimSpace(args.Raw) == "" {
							m.openRenameSession()
							return nil
						}
						return m.renameSessionDirect(args.Raw)
					},
				},
				{
					ID:       "session_filter",
					Label:    "Session: Filter",
					Desc:     "Filter session list",
					Aliases:  []string{"filter", "session filter"},
					Shortcut: shortcutFilter,
					Run: func(m *Model, args commandArgs) tea.Cmd {
						return m.applySessionFilter(args.Raw)
					},
				},
				{
					ID:       "session_toggle_panes",
					Label:    "Session: Toggle panes",
					Desc:     "Expand or collapse panes for the selected session",
					Shortcut: shortcutTogglePanes,
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						m.togglePanes()
						return nil
					},
				},
			},
		},
		{
			Name: "project",
			Commands: []commandSpec{
				{
					ID:       "project_open",
					Label:    "Project: Open project picker",
					Desc:     "Scan project roots and create session",
					Aliases:  []string{"open", "open-project", "project open"},
					Shortcut: shortcutOpenProject,
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						m.openProjectPicker()
						return nil
					},
				},
				{
					ID:       "project_toggle_sidebar",
					Label:    "Project: Toggle sidebar",
					Desc:     "Show or hide the project sidebar",
					Shortcut: shortcutToggleSidebar,
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						m.toggleSidebar()
						return nil
					},
				},
				{
					ID:       "project_close",
					Label:    "Project: Close project",
					Desc:     "Hide project from tabs (sessions stay running)",
					Aliases:  []string{"close-project", "project close"},
					Shortcut: shortcutCloseProject,
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						m.openCloseProjectConfirm()
						return nil
					},
				},
				{
					ID:      "project_close_all",
					Label:   "Project: Close all projects",
					Desc:    "Hide all projects from tabs (sessions stay running)",
					Aliases: []string{"close-all-projects", "project close-all"},
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						m.openCloseAllProjectsConfirm()
						return nil
					},
				},
			},
		},
		{
			Name: "agent",
			Commands: []commandSpec{
				{
					ID:      "agent_auth",
					Label:   "Agent: Auth",
					Desc:    "Connect an AI provider for peky",
					Aliases: []string{"auth", "login", "agent auth"},
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						return m.prefillQuickReplyInput("/auth ")
					},
				},
				{
					ID:      "agent_model",
					Label:   "Agent: Model",
					Desc:    "Select the model used by peky",
					Aliases: []string{"model", "agent model"},
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						return m.prefillQuickReplyInput("/model ")
					},
				},
			},
		},
		{
			Name: "broadcast",
			Commands: []commandSpec{
				{
					ID:    "broadcast_all",
					Label: "Broadcast: /all",
					Desc:  "Send to session/project/all panes",
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						return m.prefillQuickReplyInput("/all ")
					},
				},
			},
		},
		{
			Name: "menu",
			Commands: []commandSpec{
				{
					ID:      "menu_settings",
					Label:   "Settings",
					Desc:    "Project roots and config",
					Aliases: []string{"settings"},
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						return m.openSettingsMenu()
					},
				},
				{
					ID:      "menu_debug",
					Label:   "Debug",
					Desc:    "Refresh and restart server",
					Aliases: []string{"debug"},
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						return m.openDebugMenu()
					},
				},
			},
		},
		{
			Name: "other",
			Commands: []commandSpec{
				{
					ID:       "other_help",
					Label:    "Help",
					Desc:     "Show help",
					Aliases:  []string{"help"},
					Shortcut: shortcutHelp,
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						m.setState(StateHelp)
						return nil
					},
				},
				{
					ID:       "other_quit",
					Label:    "Quit",
					Desc:     "Exit peky",
					Aliases:  []string{"quit"},
					Shortcut: shortcutQuit,
					Run: func(m *Model, _ commandArgs) tea.Cmd {
						return m.requestQuit()
					},
				},
			},
		},
	}

	registry := commandRegistry{Groups: groups}
	if err := registry.indexAliases(); err != nil {
		return commandRegistry{}, err
	}
	return registry, nil
}

func (r *commandRegistry) indexAliases() error {
	seen := make(map[string]struct{})
	var aliases []commandAlias
	for groupIdx := range r.Groups {
		group := &r.Groups[groupIdx]
		for cmdIdx := range group.Commands {
			cmd := &group.Commands[cmdIdx]
			for _, alias := range cmd.Aliases {
				normalized := normalizeCommandAlias(alias)
				if normalized == "" {
					continue
				}
				if _, ok := seen[normalized]; ok {
					return fmt.Errorf("duplicate command alias %q", normalized)
				}
				seen[normalized] = struct{}{}
				aliases = append(aliases, commandAlias{
					key:     normalized,
					tokens:  strings.Fields(normalized),
					command: cmd,
				})
			}
		}
	}
	sort.SliceStable(aliases, func(i, j int) bool {
		if len(aliases[i].tokens) == len(aliases[j].tokens) {
			return aliases[i].key < aliases[j].key
		}
		return len(aliases[i].tokens) > len(aliases[j].tokens)
	})
	r.Aliases = aliases
	return nil
}

func normalizeCommandAlias(alias string) string {
	trimmed := strings.TrimSpace(alias)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(strings.Join(strings.Fields(trimmed), " "))
}
