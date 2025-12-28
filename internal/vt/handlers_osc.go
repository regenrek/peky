package vt

// registerDefaultOscHandlers registers the default OSC escape sequence handlers.
func (e *Emulator) registerDefaultOscHandlers() {
	for _, cmd := range []int{
		0, // Set window title and icon name
		1, // Set icon name
		2, // Set window title
	} {
		e.RegisterOscHandler(cmd, func(data []byte) bool {
			e.handleTitle(cmd, data)
			return true
		})
	}

	e.RegisterOscHandler(7, func(data []byte) bool {
		// Report the shell current working directory
		// [ansi.NotifyWorkingDirectory].
		e.handleWorkingDirectory(7, data)
		return true
	})

	e.RegisterOscHandler(8, func(data []byte) bool {
		// Set/Query Hyperlink [ansi.SetHyperlink]
		e.handleHyperlink(8, data)
		return true
	})

	for _, cmd := range []int{
		10,  // Set/Query foreground color
		11,  // Set/Query background color
		12,  // Set/Query cursor color
		110, // Reset foreground color
		111, // Reset background color
		112, // Reset cursor color
	} {
		e.RegisterOscHandler(cmd, func(data []byte) bool {
			e.handleDefaultColor(cmd, data)
			return true
		})
	}
}
