//go:build !unix

package terminal

func signalWINCHForPTY(pid int, pty any) {
	_ = pid
	_ = pty
}
