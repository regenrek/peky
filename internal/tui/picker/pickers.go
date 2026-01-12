package picker

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

// NewProjectPicker builds the list model for selecting projects.
func NewProjectPicker() list.Model {
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
	l.Title = "📁 Open Project"
	l.Styles.Title = theme.TitleAlt
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("project", "projects")
	return l
}

const (
	projectPickerItemHeight  = 2
	projectPickerItemSpacing = 1
)

func ProjectPickerRowMetrics() (itemHeight, rowHeight int) {
	rowHeight = projectPickerItemHeight + projectPickerItemSpacing
	if rowHeight < 1 {
		rowHeight = 1
	}
	return projectPickerItemHeight, rowHeight
}

// NewLayoutPicker builds the list model for selecting layouts.
func NewLayoutPicker() list.Model {
	l := newPaletteList(false)
	l.Title = "🧩 New Session Layout"
	l.Styles.Title = theme.TitleAlt
	l.SetStatusBarItemName("layout", "layouts")
	return l
}

// NewCommandPalette builds the list model for the command palette.
func NewCommandPalette() list.Model {
	return newPaletteList(false)
}

func newPaletteList(showDescription bool) list.Model {
	delegate := newCommandPaletteDelegate()
	delegate.ShowDescription = showDescription
	delegate.SetSpacing(0)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(theme.TextSecondary).
		Padding(0, 0, 0, 1)
	delegate.Styles.DimmedTitle = delegate.Styles.DimmedTitle.
		Foreground(theme.TextDim).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.TextPrimary).
		BorderLeftForeground(theme.AccentAlt).
		Bold(true).
		Padding(0, 0, 0, 0)
	delegate.Styles.FilterMatch = lipgloss.NewStyle().
		Foreground(theme.AccentFocus).
		Bold(true)

	l := list.New(nil, delegate, 0, 0)
	l.Title = "⌘ Command Palette"
	l.Styles.Title = lipgloss.NewStyle().Foreground(theme.TextPrimary)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("command", "commands")
	l.SetShowHelp(true)
	l.KeyMap.AcceptWhileFiltering = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	)

	styles := l.Styles
	styles.TitleBar = lipgloss.NewStyle().
		Padding(0, 0, 1, 1).
		Background(theme.Background)
	styles.FilterPrompt = lipgloss.NewStyle().
		Foreground(theme.TextMuted)
	styles.FilterCursor = lipgloss.NewStyle().
		Foreground(theme.AccentFocus)
	styles.StatusBar = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Padding(0, 0, 0, 1).
		Background(theme.Background)
	styles.StatusBarActiveFilter = lipgloss.NewStyle().
		Foreground(theme.TextSecondary)
	styles.StatusBarFilterCount = lipgloss.NewStyle().
		Foreground(theme.TextDim)
	styles.NoItems = lipgloss.NewStyle().
		Foreground(theme.TextDim)
	styles.StatusEmpty = lipgloss.NewStyle().
		Foreground(theme.TextDim)
	styles.PaginationStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		Background(theme.Background)
	styles.HelpStyle = lipgloss.NewStyle().
		Padding(1, 0, 0, 1).
		Background(theme.Background)
	styles.DividerDot = lipgloss.NewStyle().
		Foreground(theme.TextDim).
		SetString(" • ")
	l.Styles = styles
	l.FilterInput.PromptStyle = styles.FilterPrompt
	l.FilterInput.Cursor.Style = styles.FilterCursor
	l.FilterInput.TextStyle = lipgloss.NewStyle().Foreground(theme.TextPrimary)
	l.Help.Styles.ShortKey = lipgloss.NewStyle().Foreground(theme.TextMuted)
	l.Help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(theme.TextDim)
	l.Help.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(theme.TextDim)
	return l
}

// NewDialogMenu builds a command-style list without filtering for compact menus.
func NewDialogMenu() list.Model {
	l := NewCommandPalette()
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(true)
	return l
}
