package picker

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

// MultiSelectItem represents a selectable item with a checkbox label.
type MultiSelectItem struct {
	ID       string
	Label    string
	Desc     string
	Selected bool
}

func (i *MultiSelectItem) Title() string {
	prefix := "[ ]"
	if i.Selected {
		prefix = "[x]"
	}
	if i.Label == "" {
		return prefix
	}
	return fmt.Sprintf("%s %s", prefix, i.Label)
}

func (i *MultiSelectItem) Description() string { return i.Desc }
func (i *MultiSelectItem) FilterValue() string { return i.Label }

// NewMultiSelectPicker builds a list model for multi-select dialogs.
func NewMultiSelectPicker(title string) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.SetHeight(projectPickerItemHeight)
	delegate.SetSpacing(projectPickerItemSpacing)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.TextPrimary).
		BorderLeftForeground(theme.AccentAlt)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(theme.TextSecondary).
		BorderLeftForeground(theme.AccentAlt)

	l := list.New(nil, delegate, 0, 0)
	l.Title = title
	l.Styles.Title = theme.TitleAlt
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	return l
}
