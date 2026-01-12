//go:build !darwin && !linux

package terminal

func pidCwd(pid int) (string, bool) {
	_ = pid
	return "", false
}
