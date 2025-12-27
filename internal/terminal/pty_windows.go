//go:build windows

package terminal

import "os/exec"

// setupPTYCommand is a no-op on Windows; PTY setup is handled by the platform.
func setupPTYCommand(_ *exec.Cmd) {}
