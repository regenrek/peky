package vt

// registerDefaultCsiHandlers registers the default CSI escape sequence handlers.
func (e *Emulator) registerDefaultCsiHandlers() {
	e.registerCsiCursorHandlers()
	e.registerCsiEditHandlers()
	e.registerCsiScrollTabHandlers()
	e.registerCsiModeHandlers()
	e.registerCsiDeviceHandlers()
	e.registerCsiMarginHandlers()
}
