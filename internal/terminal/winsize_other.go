//go:build !unix

package terminal

func setPTYSlaveWinsizeBestEffort(pty any, cols, rows int) {
	_ = pty
	_ = cols
	_ = rows
}
