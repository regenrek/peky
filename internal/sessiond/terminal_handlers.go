package sessiond

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/regenrek/peakypanes/internal/termkeys"
)

func (d *Daemon) terminalAction(req TerminalActionRequest) (TerminalActionResponse, error) {
	win, paneID, err := d.windowFromRequest(req.PaneID)
	if err != nil {
		return TerminalActionResponse{}, err
	}
	handler, ok := terminalActionHandlers[req.Action]
	if !ok {
		return TerminalActionResponse{}, errors.New("sessiond: unknown terminal action")
	}
	return handler(win, req, paneID)
}

func (d *Daemon) handleTerminalKey(req TerminalKeyRequest) (TerminalKeyResponse, error) {
	win, _, err := d.windowFromRequest(req.PaneID)
	if err != nil {
		return TerminalKeyResponse{}, err
	}
	if resp, handled := handleAltScreenKey(win); handled {
		return resp, nil
	}
	if resp, handled := handleCopyModeKey(win, req.Key); handled {
		return resp, nil
	}
	if resp, handled := handleScrollbackKey(win, req); handled {
		return resp, nil
	}
	return handleNormalKey(win, req), nil
}

type terminalActionHandler func(win paneWindow, req TerminalActionRequest, paneID string) (TerminalActionResponse, error)

var terminalActionHandlers = map[TerminalAction]terminalActionHandler{
	TerminalEnterScrollback: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.EnterScrollback()
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalExitScrollback: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.ExitScrollback()
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalScrollUp: func(win paneWindow, req TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.ScrollUp(req.Lines)
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalScrollDown: func(win paneWindow, req TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.ScrollDown(req.Lines)
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalPageUp: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.PageUp()
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalPageDown: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.PageDown()
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalScrollTop: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.ScrollToTop()
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalScrollBottom: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.ScrollToBottom()
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalEnterCopyMode: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.EnterCopyMode()
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalExitCopyMode: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.ExitCopyMode()
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalCopyMove: func(win paneWindow, req TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.CopyMove(req.DeltaX, req.DeltaY)
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalCopyPageUp: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.CopyPageUp()
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalCopyPageDown: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.CopyPageDown()
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalCopyToggleSelect: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		win.CopyToggleSelect()
		return TerminalActionResponse{PaneID: paneID}, nil
	},
	TerminalCopyYank: func(win paneWindow, _ TerminalActionRequest, paneID string) (TerminalActionResponse, error) {
		text := win.CopyYankText()
		return TerminalActionResponse{PaneID: paneID, Text: text}, nil
	},
}

func (d *Daemon) windowFromRequest(value string) (paneWindow, string, error) {
	paneID := strings.TrimSpace(value)
	if paneID == "" {
		return nil, "", errors.New("sessiond: pane id is required")
	}
	if d.manager == nil {
		return nil, "", errors.New("sessiond: manager unavailable")
	}
	win := d.manager.Window(paneID)
	if win == nil {
		return nil, "", fmt.Errorf("sessiond: pane %q not found", paneID)
	}
	return win, paneID, nil
}

func handleAltScreenKey(win paneWindow) (TerminalKeyResponse, bool) {
	if !win.IsAltScreen() {
		return TerminalKeyResponse{}, false
	}
	if win.CopyModeActive() {
		win.ExitCopyMode()
	}
	if win.ScrollbackModeActive() || win.GetScrollbackOffset() > 0 {
		win.ExitScrollback()
	}
	return TerminalKeyResponse{Handled: false}, true
}

func handleCopyModeKey(win paneWindow, key string) (TerminalKeyResponse, bool) {
	if !win.CopyModeActive() {
		return TerminalKeyResponse{}, false
	}
	if termkeys.IsCopyShortcutKey(key) {
		text := win.CopyYankText()
		win.ExitCopyMode()
		if text == "" {
			return TerminalKeyResponse{Handled: true, Toast: "Nothing to yank", ToastKind: ToastWarning}, true
		}
		return TerminalKeyResponse{Handled: true, Toast: "Yanked to clipboard", ToastKind: ToastSuccess, YankText: text}, true
	}
	if win.CopySelectionFromMouseActive() && isPrintableKey(key) {
		win.ExitCopyMode()
		if win.ScrollbackModeActive() || win.GetScrollbackOffset() > 0 {
			win.ExitScrollback()
		}
		return TerminalKeyResponse{Handled: false}, true
	}
	switch key {
	case "esc", "q":
		win.ExitCopyMode()
		return TerminalKeyResponse{Handled: true, Toast: "Copy mode exited", ToastKind: ToastInfo}, true
	case "up", "k":
		win.CopyMove(0, -1)
		return TerminalKeyResponse{Handled: true}, true
	case "down", "j":
		win.CopyMove(0, 1)
		return TerminalKeyResponse{Handled: true}, true
	case "left", "h":
		win.CopyMove(-1, 0)
		return TerminalKeyResponse{Handled: true}, true
	case "right", "l":
		win.CopyMove(1, 0)
		return TerminalKeyResponse{Handled: true}, true
	case "pgup":
		win.CopyPageUp()
		return TerminalKeyResponse{Handled: true}, true
	case "pgdown":
		win.CopyPageDown()
		return TerminalKeyResponse{Handled: true}, true
	case "v":
		win.CopyToggleSelect()
		return TerminalKeyResponse{Handled: true, Toast: "Selection toggled (v) | Yank (y) | Exit (esc/q)", ToastKind: ToastInfo}, true
	case "y":
		text := win.CopyYankText()
		win.ExitCopyMode()
		if text == "" {
			return TerminalKeyResponse{Handled: true, Toast: "Nothing to yank", ToastKind: ToastWarning}, true
		}
		return TerminalKeyResponse{Handled: true, Toast: "Yanked to clipboard", ToastKind: ToastSuccess, YankText: text}, true
	default:
		return TerminalKeyResponse{Handled: true}, true
	}
}

func isPrintableKey(key string) bool {
	if key == "" {
		return false
	}
	runes := []rune(key)
	if len(runes) != 1 {
		return false
	}
	return unicode.IsPrint(runes[0])
}

func handleScrollbackKey(win paneWindow, req TerminalKeyRequest) (TerminalKeyResponse, bool) {
	if !win.ScrollbackModeActive() && win.GetScrollbackOffset() == 0 {
		return TerminalKeyResponse{}, false
	}
	if req.CopyToggle {
		win.EnterCopyMode()
		return TerminalKeyResponse{Handled: true, Toast: "Copy mode: hjkl/arrows | v select | y yank | esc/q exit", ToastKind: ToastInfo}, true
	}
	if req.ScrollbackToggle {
		win.PageUp()
		return TerminalKeyResponse{Handled: true}, true
	}
	switch req.Key {
	case "esc", "q":
		win.ExitScrollback()
		return TerminalKeyResponse{Handled: true, Toast: "Scrollback exited", ToastKind: ToastInfo}, true
	case "up", "k":
		win.ScrollUp(1)
		return TerminalKeyResponse{Handled: true}, true
	case "down", "j":
		win.ScrollDown(1)
		return TerminalKeyResponse{Handled: true}, true
	case "pgup":
		win.PageUp()
		return TerminalKeyResponse{Handled: true}, true
	case "pgdown":
		win.PageDown()
		return TerminalKeyResponse{Handled: true}, true
	case "home", "g":
		win.ScrollToTop()
		return TerminalKeyResponse{Handled: true}, true
	case "end", "G":
		win.ScrollToBottom()
		return TerminalKeyResponse{Handled: true}, true
	default:
		return TerminalKeyResponse{Handled: true}, true
	}
}

func handleNormalKey(win paneWindow, req TerminalKeyRequest) TerminalKeyResponse {
	if req.ScrollbackToggle {
		win.EnterScrollback()
		win.PageUp()
		return TerminalKeyResponse{Handled: true, Toast: "Scrollback: up/down/pgup/pgdown | Copy (f8) | Exit (esc/q)", ToastKind: ToastInfo}
	}
	if req.CopyToggle {
		win.EnterCopyMode()
		return TerminalKeyResponse{Handled: true, Toast: "Copy mode: hjkl/arrows | v select | y yank | esc/q exit", ToastKind: ToastInfo}
	}
	return TerminalKeyResponse{Handled: false}
}
