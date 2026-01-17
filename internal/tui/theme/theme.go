// Package theme provides centralized styling for peky TUI components.
// Following best practices: all styles are defined in one place for consistency.
package theme

import "github.com/charmbracelet/lipgloss"

// Design tokens for consistent TUI colors.
var (
	// Accent colors
	Accent      = lipgloss.Color("#3B82F6") // highlight blue
	AccentSoft  = lipgloss.Color("#60A5FA")
	AccentAlt   = lipgloss.Color("#22C55E")
	AccentFocus = lipgloss.Color("#F9F871")

	// Status colors
	Success = lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#22C55E"}
	Warning = lipgloss.AdaptiveColor{Light: "#F59E0B", Dark: "#FBBF24"}
	Error   = lipgloss.AdaptiveColor{Light: "#EF4444", Dark: "#F87171"}
	Info    = lipgloss.AdaptiveColor{Light: "#38BDF8", Dark: "#60A5FA"}

	// Text colors
	TextPrimary   = lipgloss.Color("#F8FAFC")
	TextSecondary = lipgloss.Color("#CBD5E1")
	TextMuted     = lipgloss.Color("#94A3B8")
	TextDim       = lipgloss.Color("#64748B")

	// Surface colors
	Surface      = lipgloss.Color("#1A1A1A")
	SurfaceAlt   = lipgloss.Color("#242424")
	SurfaceMuted = lipgloss.Color("#2E2E2E")
	SurfaceInset = lipgloss.Color("#3A3A3A")

	// UI element colors
	Border        = lipgloss.Color("#3A3A3A")
	BorderFocused = Accent
	BorderTarget  = AccentAlt
	BorderFocus   = AccentFocus
	Background    = Surface
	Highlight     = SurfaceAlt
	QuickReplyBg  = SurfaceMuted
	QuickReplyTag = SurfaceInset
	QuickReplyAcc = SurfaceInset

	// Dialog colors
	DialogBorderColor = Accent
	DialogLabelColor  = TextMuted
	DialogValueColor  = TextSecondary
	DialogChoiceColor = AccentSoft

	// Logo color
	Logo = lipgloss.Color("#FDE68A")
)

// ===== Base Styles =====

// App wraps the entire application view
var App = lipgloss.NewStyle().Padding(1, 2)

// ===== Title Styles =====

// Title is the main title style (e.g., "peky")
var Title = lipgloss.NewStyle().
	Foreground(TextPrimary).
	Background(Accent).
	Padding(0, 1)

// TitleAlt is an alternative title style (e.g., project picker)
var TitleAlt = lipgloss.NewStyle().
	Foreground(TextPrimary).
	Background(AccentAlt).
	Padding(0, 1)

// HelpTitle for help/shortcut views
var HelpTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(TextPrimary).
	Background(Accent).
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

// DialogCompact is a tighter dialog container for dense pickers (e.g. command palette).
var DialogCompact = Dialog.Padding(0, 1)

// DialogTitle for dialog headings
var DialogTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(DialogBorderColor)

// DialogLabel for labels in dialogs
var DialogLabel = lipgloss.NewStyle().
	Foreground(DialogLabelColor)

// DialogValue for values in dialogs
var DialogValue = lipgloss.NewStyle().
	Foreground(DialogValueColor)

// DialogNote for italic notes
var DialogNote = lipgloss.NewStyle().
	Foreground(DialogLabelColor).
	Italic(true)

// DialogChoiceKey for highlighted keys (y/n)
var DialogChoiceKey = lipgloss.NewStyle().
	Foreground(DialogChoiceColor)

// DialogChoiceSep for separators in choices
var DialogChoiceSep = lipgloss.NewStyle().
	Foreground(DialogLabelColor)

// ===== List Delegate Styles =====

// ListSelectedTitle for selected items in lists
var ListSelectedTitle = lipgloss.NewStyle().
	Foreground(TextPrimary).
	BorderLeftForeground(Accent)

// ListSelectedDesc for selected item descriptions
var ListSelectedDesc = lipgloss.NewStyle().
	Foreground(TextSecondary).
	BorderLeftForeground(Accent)

// ListSelectedTitleAlt for alternative lists (project picker)
var ListSelectedTitleAlt = lipgloss.NewStyle().
	Foreground(TextPrimary).
	BorderLeftForeground(AccentAlt)

// ListSelectedDescAlt for alternative list descriptions
var ListSelectedDescAlt = lipgloss.NewStyle().
	Foreground(TextSecondary).
	BorderLeftForeground(AccentAlt)

// ListDimmed for dimmed/background list views
var ListDimmed = lipgloss.NewStyle().
	Foreground(TextDim)

// ===== Sidebar Styles =====

// SidebarCaret highlights the active caret in the sidebar.
var SidebarCaret = lipgloss.NewStyle().
	Foreground(Accent).
	Bold(true)

// SidebarPaneMarker highlights the active pane marker.
var SidebarPaneMarker = lipgloss.NewStyle().
	Foreground(Accent).
	Bold(true)

// SidebarSession for session rows.
var SidebarSession = lipgloss.NewStyle().
	Foreground(TextPrimary)

// SidebarSessionSelected for the active session.
var SidebarSessionSelected = lipgloss.NewStyle().
	Foreground(TextPrimary).
	Bold(true)

// SidebarSessionStopped for stopped sessions.
var SidebarSessionStopped = lipgloss.NewStyle().
	Foreground(TextDim)

// SidebarPane for pane rows.
var SidebarPane = lipgloss.NewStyle().
	Foreground(TextMuted)

// SidebarPaneSelected for the active pane.
var SidebarPaneSelected = lipgloss.NewStyle().
	Foreground(TextSecondary)

// SidebarMeta for counts or metadata.
var SidebarMeta = lipgloss.NewStyle().
	Foreground(TextDim)

// ===== Shortcut/Help Styles =====

// ShortcutKey for keyboard shortcut keys
var ShortcutKey = lipgloss.NewStyle().
	Foreground(AccentSoft).
	Bold(true).
	Width(22)

// ShortcutDesc for shortcut descriptions
var ShortcutDesc = lipgloss.NewStyle().
	Foreground(TextSecondary)

// ShortcutNote for footnotes in help views
var ShortcutNote = lipgloss.NewStyle().
	Foreground(TextMuted).
	Italic(true)

// ShortcutHint for close/action hints
var ShortcutHint = lipgloss.NewStyle().
	Foreground(TextDim)

// ===== Tabs and Sections =====

// TabActive for active tabs (projects/views).
var TabActive = lipgloss.NewStyle().
	Bold(true).
	Foreground(TextPrimary).
	Background(Accent).
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
	Background(AccentAlt).
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
	Foreground(TextPrimary).
	Background(Info).
	Padding(0, 1)

// StatusBadgeDone for successful completion.
var StatusBadgeDone = lipgloss.NewStyle().
	Bold(true).
	Foreground(TextPrimary).
	Background(Success).
	Padding(0, 1)

// StatusBadgeError for failures.
var StatusBadgeError = lipgloss.NewStyle().
	Bold(true).
	Foreground(TextPrimary).
	Background(Error).
	Padding(0, 1)

// StatusBadgeDead for terminated panes.
var StatusBadgeDead = lipgloss.NewStyle().
	Bold(true).
	Foreground(TextPrimary).
	Background(SurfaceInset).
	Padding(0, 1)

// StatusBadgeDisconnected for offline panes.
var StatusBadgeDisconnected = lipgloss.NewStyle().
	Bold(true).
	Foreground(TextPrimary).
	Background(Warning).
	Padding(0, 1)

// StatusBadgeIdle for idle/unknown.
var StatusBadgeIdle = lipgloss.NewStyle().
	Foreground(TextMuted).
	Background(Highlight).
	Padding(0, 1)

// PaneTopbar is the per-pane topbar strip rendered above terminal content.
var PaneTopbar = lipgloss.NewStyle().
	Foreground(TextSecondary).
	Background(lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#181818"})

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
	Foreground(TextSecondary)

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
