// Package theme provides centralized styling for Peaky Panes TUI components.
// Following best practices: all styles are defined in one place for consistency.
package theme

import "github.com/charmbracelet/lipgloss"

// Color palette - using adaptive colors for light/dark terminal support
var (
	// Primary brand colors
	Primary       = lipgloss.Color("#7D56F4") // Purple accent
	PrimaryLight  = lipgloss.Color("#9B7EF7")
	Secondary     = lipgloss.Color("#25A065") // Green accent
	SecondaryDark = lipgloss.Color("#1D8051")

	// Status colors
	Success = lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}
	Warning = lipgloss.AdaptiveColor{Light: "#FFA500", Dark: "#FFB627"}
	Error   = lipgloss.AdaptiveColor{Light: "#FF4444", Dark: "#FF6B6B"}
	Info    = lipgloss.AdaptiveColor{Light: "#3498DB", Dark: "#5DADE2"}

	// Text colors
	TextPrimary   = lipgloss.Color("#FFFDF5")
	TextSecondary = lipgloss.Color("#B8B8B8")
	TextMuted     = lipgloss.Color("#808080")
	TextDim       = lipgloss.Color("#555555")

	// UI element colors
	Border        = lipgloss.Color("210")
	BorderFocused = lipgloss.Color("#7D56F4")
	Background    = lipgloss.Color("#1a1a1a")
	Highlight     = lipgloss.Color("#3a3a3a")

	// Dialog colors
	DialogBorderColor = lipgloss.Color("210")
	DialogLabelColor  = lipgloss.Color("244")
	DialogValueColor  = lipgloss.Color("252")
	DialogChoiceColor = lipgloss.Color("114")

	// Logo color
	Logo = lipgloss.Color("#FFFF99")
)

// ===== Base Styles =====

// App wraps the entire application view
var App = lipgloss.NewStyle().Padding(1, 2)

// ===== Title Styles =====

// Title is the main title style (e.g., "🎩 Peaky Panes")
var Title = lipgloss.NewStyle().
	Foreground(TextPrimary).
	Background(Primary).
	Padding(0, 1)

// TitleAlt is an alternative title style (e.g., project picker)
var TitleAlt = lipgloss.NewStyle().
	Foreground(TextPrimary).
	Background(Secondary).
	Padding(0, 1)

// HelpTitle for help/shortcut views
var HelpTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("229")).
	Background(lipgloss.Color("57")).
	Padding(0, 1).
	MarginBottom(1)

// ===== Status Message Styles =====

// StatusMessage for success/info messages
var StatusMessage = lipgloss.NewStyle().
	Foreground(Success)

// StatusError for error messages
var StatusError = lipgloss.NewStyle().
	Foreground(Error)

// StatusWarning for warning messages
var StatusWarning = lipgloss.NewStyle().
	Foreground(Warning)

// ===== Dialog Styles =====

// Dialog is the container for modal dialogs
var Dialog = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(DialogBorderColor).
	Background(Background).
	Foreground(TextPrimary).
	Padding(1, 2)

// DialogTitle for dialog headings
var DialogTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(DialogBorderColor)

// DialogLabel for labels in dialogs
var DialogLabel = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244"))

// DialogValue for values in dialogs
var DialogValue = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252"))

// DialogNote for italic notes
var DialogNote = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244")).
	Italic(true)

// DialogChoiceKey for highlighted keys (y/n)
var DialogChoiceKey = lipgloss.NewStyle().
	Foreground(DialogChoiceColor)

// DialogChoiceSep for separators in choices
var DialogChoiceSep = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244"))

// ===== List Delegate Styles =====

// ListSelectedTitle for selected items in lists
var ListSelectedTitle = lipgloss.NewStyle().
	Foreground(TextPrimary).
	BorderLeftForeground(Primary)

// ListSelectedDesc for selected item descriptions
var ListSelectedDesc = lipgloss.NewStyle().
	Foreground(TextSecondary).
	BorderLeftForeground(Primary)

// ListSelectedTitleAlt for alternative lists (project picker)
var ListSelectedTitleAlt = lipgloss.NewStyle().
	Foreground(TextPrimary).
	BorderLeftForeground(Secondary)

// ListSelectedDescAlt for alternative list descriptions
var ListSelectedDescAlt = lipgloss.NewStyle().
	Foreground(TextSecondary).
	BorderLeftForeground(Secondary)

// ListDimmed for dimmed/background list views
var ListDimmed = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240"))

// ===== Shortcut/Help Styles =====

// ShortcutKey for keyboard shortcut keys
var ShortcutKey = lipgloss.NewStyle().
	Foreground(lipgloss.Color("114")).
	Bold(true).
	Width(22)

// ShortcutDesc for shortcut descriptions
var ShortcutDesc = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252"))

// ShortcutNote for footnotes in help views
var ShortcutNote = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244")).
	Italic(true)

// ShortcutHint for close/action hints
var ShortcutHint = lipgloss.NewStyle().
	Foreground(lipgloss.Color("241"))

// ===== Tabs and Sections =====

// TabActive for active tabs (projects/windows).
var TabActive = lipgloss.NewStyle().
	Bold(true).
	Foreground(TextPrimary).
	Background(Primary).
	Padding(0, 1)

// TabInactive for inactive tabs.
var TabInactive = lipgloss.NewStyle().
	Foreground(TextMuted).
	Background(Highlight).
	Padding(0, 1)

// TabAdd for the add/new tab.
var TabAdd = lipgloss.NewStyle().
	Bold(true).
	Foreground(TextPrimary).
	Background(Secondary).
	Padding(0, 1)

// SectionTitle for sidebar/panel headers.
var SectionTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(TextPrimary).
	Background(Background).
	Padding(0, 1)

// ===== Status Badges =====

// StatusBadgeRunning for running activity.
var StatusBadgeRunning = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("229")).
	Background(lipgloss.Color("60")).
	Padding(0, 1)

// StatusBadgeDone for successful completion.
var StatusBadgeDone = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("16")).
	Background(lipgloss.Color("114")).
	Padding(0, 1)

// StatusBadgeError for failures.
var StatusBadgeError = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("15")).
	Background(lipgloss.Color("160")).
	Padding(0, 1)

// StatusBadgeIdle for idle/unknown.
var StatusBadgeIdle = lipgloss.NewStyle().
	Foreground(TextMuted).
	Background(Highlight).
	Padding(0, 1)

// ===== Logo Style =====

// LogoStyle for ASCII art logo
var LogoStyle = lipgloss.NewStyle().
	Foreground(Logo).
	Bold(true)

// ===== Error Display Styles =====

// ErrorBox wraps error messages in a visible container
var ErrorBox = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Error).
	Padding(0, 1).
	MarginTop(1)

// ErrorTitle for error headings
var ErrorTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Error)

// ErrorMessage for error body text
var ErrorMessage = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252"))

// ===== Helper Functions =====

// FormatSuccess creates a success message
func FormatSuccess(msg string) string {
	return StatusMessage.Render("✓ " + msg)
}

// FormatError creates an error message
func FormatError(msg string) string {
	return StatusError.Render("✗ " + msg)
}

// FormatWarning creates a warning message
func FormatWarning(msg string) string {
	return StatusWarning.Render("⚠ " + msg)
}

// FormatInfo creates an info message
func FormatInfo(msg string) string {
	return StatusMessage.Render("ℹ " + msg)
}
