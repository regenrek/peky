package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

type dialogSize struct {
	Width  int
	Height int
}

type dialogSpec struct {
	Content         string
	Size            dialogSize
	RequireViewport bool
	Style           lipgloss.Style
	UseStyle        bool
}

type dialogChoice struct {
	Key   string
	Label string
}

func dialogContent(sections ...string) string {
	trimmed := make([]string, 0, len(sections))
	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}
		trimmed = append(trimmed, strings.TrimRight(section, "\n"))
	}
	return strings.Join(trimmed, "\n\n")
}

func dialogSizeForContent(contentW, contentH int) dialogSize {
	return dialogSizeForContentWithStyle(dialogStyle, contentW, contentH)
}

func dialogSizeForContentWithStyle(style lipgloss.Style, contentW, contentH int) dialogSize {
	frameW, frameH := style.GetFrameSize()
	return dialogSize{
		Width:  contentW + frameW,
		Height: contentH + frameH,
	}
}

func (m Model) renderDialog(spec dialogSpec) string {
	if spec.RequireViewport && (m.Width == 0 || m.Height == 0) {
		return ""
	}
	style := dialogStyle
	if spec.UseStyle {
		style = spec.Style
	}
	if spec.Size.Width > 0 {
		style = style.Width(spec.Size.Width)
	}
	if spec.Size.Height > 0 {
		style = style.Height(spec.Size.Height)
	}
	dialog := style.Render(spec.Content)
	return m.overlayDialog(dialog)
}

func renderDialogChoices(choices []dialogChoice) string {
	if len(choices) == 0 {
		return ""
	}
	var builder strings.Builder
	for i, choice := range choices {
		if key := strings.TrimSpace(choice.Key); key != "" {
			builder.WriteString(theme.DialogChoiceKey.Render(key))
		}
		if label := strings.TrimSpace(choice.Label); label != "" {
			prefix := " "
			if strings.TrimSpace(choice.Key) == "" {
				prefix = ""
			}
			builder.WriteString(theme.DialogChoiceSep.Render(prefix + label))
		}
		if i < len(choices)-1 {
			builder.WriteString(theme.DialogChoiceSep.Render(" • "))
		}
	}
	return builder.String()
}

func (m Model) overlayDialog(dialog string) string {
	if m.Width == 0 || m.Height == 0 {
		return appStyle.Render(dialog)
	}
	base := appStyle.Render(theme.ListDimmed.Render(m.viewDashboardContent()))
	return overlayCentered(base, dialog, m.Width, m.Height)
}
