package vt

import "github.com/charmbracelet/x/ansi"

// DcsHandler is a function that handles a DCS escape sequence.
type DcsHandler func(params ansi.Params, data []byte) bool

// CsiHandler is a function that handles a CSI escape sequence.
type CsiHandler func(params ansi.Params) bool

// OscHandler is a function that handles an OSC escape sequence.
type OscHandler func(data []byte) bool

// ApcHandler is a function that handles an APC escape sequence.
type ApcHandler func(data []byte) bool

// SosHandler is a function that handles an SOS escape sequence.
type SosHandler func(data []byte) bool

// PmHandler is a function that handles a PM escape sequence.
type PmHandler func(data []byte) bool

// EscHandler is a function that handles an ESC escape sequence.
type EscHandler func() bool

// CcHandler is a function that handles a control character.
type CcHandler func() bool

// handlers contains the terminal's escape sequence handlers.
type handlers struct {
	ccHandlers  map[byte][]CcHandler
	dcsHandlers map[int][]DcsHandler
	csiHandlers map[int][]CsiHandler
	oscHandlers map[int][]OscHandler
	escHandler  map[int][]EscHandler
	apcHandlers []ApcHandler
	sosHandlers []SosHandler
	pmHandlers  []PmHandler
}

// RegisterDcsHandler registers a DCS escape sequence handler.
func (h *handlers) RegisterDcsHandler(cmd int, handler DcsHandler) {
	if h.dcsHandlers == nil {
		h.dcsHandlers = make(map[int][]DcsHandler)
	}
	h.dcsHandlers[cmd] = append(h.dcsHandlers[cmd], handler)
}

// RegisterCsiHandler registers a CSI escape sequence handler.
func (h *handlers) RegisterCsiHandler(cmd int, handler CsiHandler) {
	if h.csiHandlers == nil {
		h.csiHandlers = make(map[int][]CsiHandler)
	}
	h.csiHandlers[cmd] = append(h.csiHandlers[cmd], handler)
}

// RegisterOscHandler registers an OSC escape sequence handler.
func (h *handlers) RegisterOscHandler(cmd int, handler OscHandler) {
	if h.oscHandlers == nil {
		h.oscHandlers = make(map[int][]OscHandler)
	}
	h.oscHandlers[cmd] = append(h.oscHandlers[cmd], handler)
}

// RegisterApcHandler registers an APC escape sequence handler.
func (h *handlers) RegisterApcHandler(handler ApcHandler) {
	h.apcHandlers = append(h.apcHandlers, handler)
}

// RegisterSosHandler registers an SOS escape sequence handler.
func (h *handlers) RegisterSosHandler(handler SosHandler) {
	h.sosHandlers = append(h.sosHandlers, handler)
}

// RegisterPmHandler registers a PM escape sequence handler.
func (h *handlers) RegisterPmHandler(handler PmHandler) {
	h.pmHandlers = append(h.pmHandlers, handler)
}

// RegisterEscHandler registers an ESC escape sequence handler.
func (h *handlers) RegisterEscHandler(cmd int, handler EscHandler) {
	if h.escHandler == nil {
		h.escHandler = make(map[int][]EscHandler)
	}
	h.escHandler[cmd] = append(h.escHandler[cmd], handler)
}

// registerCcHandler registers a control character handler.
func (h *handlers) registerCcHandler(r byte, handler CcHandler) {
	if h.ccHandlers == nil {
		h.ccHandlers = make(map[byte][]CcHandler)
	}
	h.ccHandlers[r] = append(h.ccHandlers[r], handler)
}

// handleCc handles a control character.
// It returns true if the control character was handled.
func (h *handlers) handleCc(r byte) bool {
	// Reverse iterate over the handlers so that the last registered handler
	// is the first to be called.
	for i := len(h.ccHandlers[r]) - 1; i >= 0; i-- {
		if h.ccHandlers[r][i]() {
			return true
		}
	}
	return false
}

// handleDcs handles a DCS escape sequence.
// It returns true if the sequence was handled.
func (h *handlers) handleDcs(cmd ansi.Cmd, params ansi.Params, data []byte) bool {
	// Reverse iterate over the handlers so that the last registered handler
	// is the first to be called.
	if handlers, ok := h.dcsHandlers[int(cmd)]; ok {
		for i := len(handlers) - 1; i >= 0; i-- {
			if handlers[i](params, data) {
				return true
			}
		}
	}
	return false
}

// handleCsi handles a CSI escape sequence.
// It returns true if the sequence was handled.
func (h *handlers) handleCsi(cmd ansi.Cmd, params ansi.Params) bool {
	// Reverse iterate over the handlers so that the last registered handler
	// is the first to be called.
	if handlers, ok := h.csiHandlers[int(cmd)]; ok {
		for i := len(handlers) - 1; i >= 0; i-- {
			if handlers[i](params) {
				return true
			}
		}
	}
	return false
}

// handleOsc handles an OSC escape sequence.
// It returns true if the sequence was handled.
func (h *handlers) handleOsc(cmd int, data []byte) bool {
	// Reverse iterate over the handlers so that the last registered handler
	// is the first to be called.
	if handlers, ok := h.oscHandlers[cmd]; ok {
		for i := len(handlers) - 1; i >= 0; i-- {
			if handlers[i](data) {
				return true
			}
		}
	}
	return false
}

// handleApc handles an APC escape sequence.
// It returns true if the sequence was handled.
func (h *handlers) handleApc(data []byte) bool {
	// Reverse iterate over the handlers so that the last registered handler
	// is the first to be called.
	for i := len(h.apcHandlers) - 1; i >= 0; i-- {
		if h.apcHandlers[i](data) {
			return true
		}
	}
	return false
}

// handleSos handles an SOS escape sequence.
// It returns true if the sequence was handled.
func (h *handlers) handleSos(data []byte) bool {
	// Reverse iterate over the handlers so that the last registered handler
	// is the first to be called.
	for i := len(h.sosHandlers) - 1; i >= 0; i-- {
		if h.sosHandlers[i](data) {
			return true
		}
	}
	return false
}

// handlePm handles a PM escape sequence.
// It returns true if the sequence was handled.
func (h *handlers) handlePm(data []byte) bool {
	// Reverse iterate over the handlers so that the last registered handler
	// is the first to be called.
	for i := len(h.pmHandlers) - 1; i >= 0; i-- {
		if h.pmHandlers[i](data) {
			return true
		}
	}
	return false
}

// handleEsc handles an ESC escape sequence.
// It returns true if the sequence was handled.
func (h *handlers) handleEsc(cmd int) bool {
	// Reverse iterate over the handlers so that the last registered handler
	// is the first to be called.
	if handlers, ok := h.escHandler[cmd]; ok {
		for i := len(handlers) - 1; i >= 0; i-- {
			if handlers[i]() {
				return true
			}
		}
	}
	return false
}

// registerDefaultHandlers registers the default escape sequence handlers.
func (e *Emulator) registerDefaultHandlers() {
	e.registerDefaultCcHandlers()
	e.registerDefaultCsiHandlers()
	e.registerDefaultEscHandlers()
	e.registerDefaultOscHandlers()
}
