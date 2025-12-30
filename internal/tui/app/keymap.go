package app

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"

	"github.com/regenrek/peakypanes/internal/layout"
)

type keymapAction struct {
	name     string
	desc     string
	defaults []string
	override []string
	assign   func(*dashboardKeyMap, key.Binding)
}

func buildDashboardKeyMap(cfg layout.DashboardKeymapConfig) (*dashboardKeyMap, error) {
	km := &dashboardKeyMap{}
	used := make(map[string]string)
	actions := []keymapAction{
		{
			name:     "project_left",
			desc:     "project",
			defaults: []string{"ctrl+q"},
			override: cfg.ProjectLeft,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.projectLeft = b },
		},
		{
			name:     "project_right",
			desc:     "project",
			defaults: []string{"ctrl+e"},
			override: cfg.ProjectRight,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.projectRight = b },
		},
		{
			name:     "session_up",
			desc:     "session",
			defaults: []string{"ctrl+w"},
			override: cfg.SessionUp,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.sessionUp = b },
		},
		{
			name:     "session_down",
			desc:     "session",
			defaults: []string{"ctrl+s"},
			override: cfg.SessionDown,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.sessionDown = b },
		},
		{
			name:     "session_only_up",
			desc:     "session (skip panes)",
			defaults: []string{"alt+w"},
			override: cfg.SessionOnlyUp,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.sessionOnlyUp = b },
		},
		{
			name:     "session_only_down",
			desc:     "session (skip panes)",
			defaults: []string{"alt+s"},
			override: cfg.SessionOnlyDown,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.sessionOnlyDown = b },
		},
		{
			name:     "pane_next",
			desc:     "pane",
			defaults: []string{"ctrl+d"},
			override: cfg.PaneNext,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.paneNext = b },
		},
		{
			name:     "pane_prev",
			desc:     "pane",
			defaults: []string{"ctrl+a"},
			override: cfg.PanePrev,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.panePrev = b },
		},
		{
			name:     "attach",
			desc:     "attach",
			defaults: []string{"enter"},
			override: cfg.Attach,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.attach = b },
		},
		{
			name:     "new_session",
			desc:     "new session",
			defaults: []string{"ctrl+n"},
			override: cfg.NewSession,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.newSession = b },
		},
		{
			name:     "terminal_focus",
			desc:     "terminal focus",
			defaults: []string{"ctrl+\\"},
			override: cfg.TerminalFocus,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.terminalFocus = b },
		},
		{
			name:     "toggle_panes",
			desc:     "panes",
			defaults: []string{"ctrl+u"},
			override: cfg.TogglePanes,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.togglePanes = b },
		},
		{
			name:     "toggle_sidebar",
			desc:     "sidebar",
			defaults: []string{"ctrl+b"},
			override: cfg.ToggleSidebar,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.toggleSidebar = b },
		},
		{
			name:     "open_project",
			desc:     "open project",
			defaults: []string{"ctrl+o"},
			override: cfg.OpenProject,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.openProject = b },
		},
		{
			name:     "command_palette",
			desc:     "commands",
			defaults: []string{"ctrl+p"},
			override: cfg.CommandPalette,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.commandPalette = b },
		},
		{
			name:     "refresh",
			desc:     "refresh",
			defaults: []string{"ctrl+r"},
			override: cfg.Refresh,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.refresh = b },
		},
		{
			name:     "edit_config",
			desc:     "edit config",
			defaults: []string{"ctrl+,"},
			override: cfg.EditConfig,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.editConfig = b },
		},
		{
			name:     "kill",
			desc:     "kill session",
			defaults: []string{"ctrl+x"},
			override: cfg.Kill,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.kill = b },
		},
		{
			name:     "close_project",
			desc:     "close project",
			defaults: []string{"alt+c"},
			override: cfg.CloseProject,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.closeProject = b },
		},
		{
			name:     "help",
			desc:     "help",
			defaults: []string{"ctrl+g"},
			override: cfg.Help,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.help = b },
		},
		{
			name:     "quit",
			desc:     "quit",
			defaults: []string{"ctrl+c"},
			override: cfg.Quit,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.quit = b },
		},
		{
			name:     "filter",
			desc:     "filter",
			defaults: []string{"ctrl+f"},
			override: cfg.Filter,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.filter = b },
		},
		{
			name:     "scrollback",
			desc:     "scrollback",
			defaults: []string{"f7"},
			override: cfg.Scrollback,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.scrollback = b },
		},
		{
			name:     "copy_mode",
			desc:     "copy mode",
			defaults: []string{"f8"},
			override: cfg.CopyMode,
			assign:   func(m *dashboardKeyMap, b key.Binding) { m.copyMode = b },
		},
	}

	for _, action := range actions {
		keys, err := resolveKeyList(action.name, action.override, action.defaults)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			if prev, ok := used[k]; ok {
				return nil, fmt.Errorf("dashboard.keymap.%s: key %q already bound to dashboard.keymap.%s", action.name, k, prev)
			}
			used[k] = action.name
		}
		binding := key.NewBinding(
			key.WithKeys(keys...),
			key.WithHelp(formatKeyLabel(keys), action.desc),
		)
		action.assign(km, binding)
	}

	return km, nil
}

func resolveKeyList(field string, override, defaults []string) ([]string, error) {
	keys := override
	if len(keys) == 0 {
		keys = defaults
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("dashboard.keymap.%s: no keys configured", field)
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(keys))
	for _, raw := range keys {
		normalized, err := normalizeKeyString(raw)
		if err != nil {
			return nil, fmt.Errorf("dashboard.keymap.%s: %w", field, err)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("dashboard.keymap.%s: no valid keys configured", field)
	}
	return out, nil
}

func normalizeKeyString(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("invalid key %q (empty)", raw)
	}
	if strings.EqualFold(value, "space") {
		return " ", nil
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "alt+") {
		base := strings.TrimSpace(value[4:])
		if base == "" {
			return "", fmt.Errorf("invalid key %q (missing base key)", raw)
		}
		if strings.EqualFold(base, "space") {
			base = " "
		}
		if isSingleRune(base) {
			return "alt+" + base, nil
		}
		baseLower := strings.ToLower(base)
		if isSupportedKeyName(baseLower) {
			return "alt+" + baseLower, nil
		}
		return "", invalidKeyError(raw)
	}

	if isSingleRune(value) {
		return value, nil
	}
	if isSupportedKeyName(lower) {
		return lower, nil
	}
	return "", invalidKeyError(raw)
}

func isSingleRune(value string) bool {
	if value == "" {
		return false
	}
	r, size := utf8.DecodeRuneInString(value)
	if r == utf8.RuneError {
		return false
	}
	return size == len(value)
}

func invalidKeyError(raw string) error {
	return fmt.Errorf(
		"invalid key %q (use a single character like \"k\", or named keys like \"tab\", \"shift+tab\", \"enter\", \"esc\", \"up\", \"ctrl+a\", \"alt+k\")",
		raw,
	)
}

func formatKeyLabel(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	labels := make([]string, 0, len(keys))
	for _, k := range keys {
		labels = append(labels, prettyKeyLabel(k))
	}
	return strings.Join(labels, "/")
}

func prettyKeyLabel(key string) string {
	switch key {
	case "shift+tab":
		return "⇧tab"
	case " ":
		return "space"
	default:
		return key
	}
}

func isSupportedKeyName(key string) bool {
	_, ok := supportedSpecialKeys[key]
	return ok
}

var supportedSpecialKeys = func() map[string]struct{} {
	keys := map[string]struct{}{
		"tab":              {},
		"shift+tab":        {},
		"enter":            {},
		"esc":              {},
		"backspace":        {},
		"delete":           {},
		"insert":           {},
		"home":             {},
		"end":              {},
		"pgup":             {},
		"pgdown":           {},
		"ctrl+pgup":        {},
		"ctrl+pgdown":      {},
		"up":               {},
		"down":             {},
		"left":             {},
		"right":            {},
		"ctrl+up":          {},
		"ctrl+down":        {},
		"ctrl+left":        {},
		"ctrl+right":       {},
		"shift+up":         {},
		"shift+down":       {},
		"shift+left":       {},
		"shift+right":      {},
		"ctrl+shift+up":    {},
		"ctrl+shift+down":  {},
		"ctrl+shift+left":  {},
		"ctrl+shift+right": {},
		"ctrl+home":        {},
		"ctrl+end":         {},
		"shift+home":       {},
		"shift+end":        {},
		"ctrl+shift+home":  {},
		"ctrl+shift+end":   {},
		"f1":               {},
		"f2":               {},
		"f3":               {},
		"f4":               {},
		"f5":               {},
		"f6":               {},
		"f7":               {},
		"f8":               {},
		"f9":               {},
		"f10":              {},
		"f11":              {},
		"f12":              {},
		"f13":              {},
		"f14":              {},
		"f15":              {},
		"f16":              {},
		"f17":              {},
		"f18":              {},
		"f19":              {},
		"f20":              {},
		"ctrl+@":           {},
		"ctrl+,":           {},
		"ctrl+\\":          {},
		"ctrl+]":           {},
		"ctrl+^":           {},
		"ctrl+_":           {},
	}
	for r := 'a'; r <= 'z'; r++ {
		keys["ctrl+"+string(r)] = struct{}{}
	}
	return keys
}()

func joinKeyLabels(bindings ...key.Binding) string {
	labels := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		labels = append(labels, keyLabel(binding))
	}
	return strings.Join(labels, "/")
}

func keyLabel(binding key.Binding) string {
	help := binding.Help().Key
	if help != "" {
		return help
	}
	return formatKeyLabel(binding.Keys())
}
