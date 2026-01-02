package app

import (
	"fmt"
	"strings"

	"github.com/regenrek/peakypanes/internal/tui/picker"
	"github.com/regenrek/peakypanes/internal/tui/theme"
	"github.com/regenrek/peakypanes/internal/tui/views"
)

type dialogHelpSeverity int

const (
	dialogHelpNormal dialogHelpSeverity = iota
	dialogHelpHeavy
)

const (
	helpKeyPerfPreset        = "perf.preset"
	helpKeyPerfRenderPolicy  = "perf.render_policy"
	helpKeyPerfPreviewRender = "perf.preview_render"
	helpKeyPerfEditConfig    = "perf.edit_config"
)

type dialogHelpContent struct {
	Title    string
	Summary  string
	Body     []string
	Severity dialogHelpSeverity
}

type dialogHelpTemplate struct {
	Title    string
	Summary  func(value string) string
	Body     func(value string) []string
	Severity func(value string) dialogHelpSeverity
}

var dialogHelpRegistry = map[string]dialogHelpTemplate{
	helpKeyPerfPreset: {
		Title: "Performance preset",
		Summary: func(value string) string {
			switch value {
			case "low":
				return "Preset = low. Lowest refresh rate for battery or many panes."
			case "medium":
				return "Preset = medium. Balanced refresh rate (default)."
			case "high":
				return "Preset = high. Smoother updates with more CPU."
			case "max":
				return "Preset = max. Disables throttling for fastest updates."
			default:
				return "Choose a preset to control preview and refresh cadence."
			}
		},
		Body: func(value string) []string {
			current := value
			if current == "" {
				current = "medium"
			}
			return []string{
				fmt.Sprintf("Current: %s.", current),
				"Low: save battery or run many panes.",
				"Medium: balanced default.",
				"High: smoother updates, more CPU.",
				"Max: no throttling; highest load.",
			}
		},
		Severity: func(value string) dialogHelpSeverity {
			if value == "max" {
				return dialogHelpHeavy
			}
			return dialogHelpNormal
		},
	},
	helpKeyPerfRenderPolicy: {
		Title: "Render policy",
		Summary: func(value string) string {
			switch value {
			case "all":
				return "Render policy = all. All panes update live (higher CPU)."
			case "visible":
				return "Render policy = visible. Only on-screen panes update live."
			default:
				return "Choose whether all panes render or only visible ones."
			}
		},
		Body: func(value string) []string {
			return []string{
				"Visible: only on-screen panes update live.",
				"All: renders every pane live (heavy).",
			}
		},
		Severity: func(value string) dialogHelpSeverity {
			if value == "all" {
				return dialogHelpHeavy
			}
			return dialogHelpNormal
		},
	},
	helpKeyPerfPreviewRender: {
		Title: "Preview render mode",
		Summary: func(value string) string {
			switch value {
			case "direct":
				return "Preview render = direct. Renders on demand (heavy)."
			case "off":
				return "Preview render = off. Disables live previews."
			case "cached":
				return "Preview render = cached. Uses background ANSI cache."
			default:
				return "Choose how preview frames are rendered."
			}
		},
		Body: func(value string) []string {
			return []string{
				"Cached: background ANSI cache (default).",
				"Direct: renders on demand; higher CPU.",
				"Off: disable previews to save resources.",
			}
		},
		Severity: func(value string) dialogHelpSeverity {
			if value == "direct" {
				return dialogHelpHeavy
			}
			return dialogHelpNormal
		},
	},
	helpKeyPerfEditConfig: {
		Title: "Edit config",
		Summary: func(value string) string {
			_ = value
			return "Open config to override presets and fine tune performance."
		},
		Body: func(value string) []string {
			_ = value
			return []string{
				"Use custom overrides for advanced tuning.",
				"See docs/performance for details.",
			}
		},
		Severity: func(value string) dialogHelpSeverity {
			return dialogHelpNormal
		},
	},
}

func dialogHelpForItem(item picker.CommandItem) dialogHelpContent {
	return dialogHelpFor(item.HelpKey, item.HelpValue)
}

func dialogHelpFor(key, value string) dialogHelpContent {
	key = strings.TrimSpace(key)
	if key == "" {
		return dialogHelpContent{}
	}
	template, ok := dialogHelpRegistry[key]
	if !ok {
		return dialogHelpContent{}
	}
	value = strings.ToLower(strings.TrimSpace(value))
	summary := ""
	if template.Summary != nil {
		summary = strings.TrimSpace(template.Summary(value))
	}
	body := []string{}
	if template.Body != nil {
		body = template.Body(value)
	}
	severity := dialogHelpNormal
	if template.Severity != nil {
		severity = template.Severity(value)
	}
	return dialogHelpContent{
		Title:    strings.TrimSpace(template.Title),
		Summary:  summary,
		Body:     body,
		Severity: severity,
	}
}

func dialogHelpSeverityFor(key, value string) dialogHelpSeverity {
	content := dialogHelpFor(key, value)
	return content.Severity
}

func dialogHelpValueStyle(severity dialogHelpSeverity, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if severity == dialogHelpHeavy {
		return theme.StatusWarning.Render(value)
	}
	return value
}

func dialogHelpLine(content dialogHelpContent) string {
	summary := strings.TrimSpace(content.Summary)
	if summary == "" {
		return theme.DialogNote.Render("Info: Select an option. Press ? for details.")
	}
	prefix := theme.DialogNote.Render("Info: ")
	suffix := theme.DialogNote.Render(" Press ? for details.")
	if content.Severity == dialogHelpHeavy {
		return prefix + theme.StatusWarning.Render(summary) + suffix
	}
	return prefix + theme.DialogNote.Render(summary) + suffix
}

func (m *Model) dialogHelpView() views.DialogHelp {
	if m == nil || m.state != StatePerformanceMenu {
		return views.DialogHelp{}
	}
	item, ok := m.perfMenu.SelectedItem().(picker.CommandItem)
	content := dialogHelpContent{}
	if ok {
		content = dialogHelpForItem(item)
	}
	line := dialogHelpLine(content)
	body := strings.Join(content.Body, "\n")
	open := m.dialogHelpOpen && strings.TrimSpace(content.Title+body) != ""
	return views.DialogHelp{
		Line:  line,
		Title: content.Title,
		Body:  body,
		Open:  open,
	}
}
