//go:build !windows

package dashboard

import (
	"time"

	"golang.org/x/sys/unix"
)

func inputReady(fd uintptr, timeout time.Duration) (bool, error) {
	ms64 := timeout.Milliseconds()
	ms := int(ms64)
	if timeout > 0 && ms == 0 {
		ms = 1
	}

	pollfd := []unix.PollFd{
		{Fd: int32(fd), Events: unix.POLLIN},
	}
	var n int
	for {
		var err error
		n, err = unix.Poll(pollfd, ms)
		if err == nil {
			break
		}
		if err == unix.EINTR {
			continue
		}
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	re := pollfd[0].Revents
	if re&(unix.POLLIN|unix.POLLHUP|unix.POLLERR) == 0 {
		return false, nil
	}
	return true, nil
}
