package peakypanes

import (
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/terminal"
)

type terminalKeyResult struct {
	Handled bool
	Cmd     tea.Cmd
	Toast   string
	Level   toastLevel
}

// handleNativeTerminalKey intercepts scrollback/copy keys when terminal focus is active.
// It must be called before sending keys to the PTY.
func handleNativeTerminalKey(msg tea.KeyMsg, km *dashboardKeyMap, win *terminal.Window) terminalKeyResult {
	if win == nil || km == nil {
		return terminalKeyResult{}
	}

	// Alt screen: no scrollback/copy mode.
	if win.IsAltScreen() {
		// If user was in these modes, exit so we don't trap input.
		if win.CopyModeActive() {
			win.ExitCopyMode()
		}
		if win.ScrollbackModeActive() || win.GetScrollbackOffset() > 0 {
			win.ExitScrollback()
		}
		return terminalKeyResult{} // let PTY handle keys
	}

	// Copy mode has priority.
	if win.CopyModeActive() {
		switch msg.String() {
		case "esc", "q":
			win.ExitCopyMode()
			return terminalKeyResult{Handled: true, Toast: "Copy mode exited", Level: toastInfo}
		case "up", "k":
			win.CopyMove(0, -1)
			return terminalKeyResult{Handled: true}
		case "down", "j":
			win.CopyMove(0, 1)
			return terminalKeyResult{Handled: true}
		case "left", "h":
			win.CopyMove(-1, 0)
			return terminalKeyResult{Handled: true}
		case "right", "l":
			win.CopyMove(1, 0)
			return terminalKeyResult{Handled: true}
		case "pgup":
			win.CopyPageUp()
			return terminalKeyResult{Handled: true}
		case "pgdown":
			win.CopyPageDown()
			return terminalKeyResult{Handled: true}
		case "v":
			win.CopyToggleSelect()
			return terminalKeyResult{Handled: true, Toast: "Selection toggled (v) | Yank (y) | Exit (esc/q)", Level: toastInfo}
		case "y":
			text := win.CopyYankText()
			// Keep scrollback view, but exit copy mode.
			win.ExitCopyMode()
			if text == "" {
				return terminalKeyResult{Handled: true, Toast: "Nothing to yank", Level: toastWarning}
			}
			return terminalKeyResult{
				Handled: true,
				Cmd:     clipboardCmd(text),
				Toast:   "Yanked to clipboard",
				Level:   toastSuccess,
			}
		default:
			// Suppress PTY input while in copy mode.
			return terminalKeyResult{Handled: true}
		}
	}

	// Scrollback mode (offset > 0 or explicitly enabled).
	if win.ScrollbackModeActive() || win.GetScrollbackOffset() > 0 {
		// Allow copy mode key while in scrollback mode.
		if key.Matches(msg, km.copyMode) {
			win.EnterCopyMode()
			return terminalKeyResult{Handled: true, Toast: "Copy mode: hjkl/arrows | v select | y yank | esc/q exit", Level: toastInfo}
		}

		// Repeated scrollback key acts like page up.
		if key.Matches(msg, km.scrollback) {
			win.PageUp()
			return terminalKeyResult{Handled: true}
		}

		switch msg.String() {
		case "esc", "q":
			win.ExitScrollback()
			return terminalKeyResult{Handled: true, Toast: "Scrollback exited", Level: toastInfo}
		case "up", "k":
			win.ScrollUp(1)
			return terminalKeyResult{Handled: true}
		case "down", "j":
			win.ScrollDown(1)
			return terminalKeyResult{Handled: true}
		case "pgup":
			win.PageUp()
			return terminalKeyResult{Handled: true}
		case "pgdown":
			win.PageDown()
			return terminalKeyResult{Handled: true}
		case "home", "g":
			win.ScrollToTop()
			return terminalKeyResult{Handled: true}
		case "end", "G":
			win.ScrollToBottom()
			return terminalKeyResult{Handled: true}
		default:
			// Suppress PTY input while viewing scrollback.
			return terminalKeyResult{Handled: true}
		}
	}

	// Not in any mode: allow entering scrollback/copy.
	if key.Matches(msg, km.scrollback) {
		win.EnterScrollback()
		win.PageUp()
		return terminalKeyResult{
			Handled: true,
			Toast:   "Scrollback: up/down/pgup/pgdown | Copy (f8) | Exit (esc/q)",
			Level:   toastInfo,
		}
	}
	if key.Matches(msg, km.copyMode) {
		win.EnterCopyMode()
		return terminalKeyResult{
			Handled: true,
			Toast:   "Copy mode: hjkl/arrows | v select | y yank | esc/q exit",
			Level:   toastInfo,
		}
	}

	return terminalKeyResult{}
}

func clipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(text); err != nil {
			return ErrorMsg{Err: err, Context: "copy to clipboard"}
		}
		return nil
	}
}
