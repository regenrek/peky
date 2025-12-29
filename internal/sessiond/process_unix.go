//go:build !windows

package sessiond

import (
	"os"
	"os/exec"
	"syscall"
)

func configureDaemonCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}

func signalDaemon(pid int) error {
	if pid <= 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}
