//go:build windows

package dashboard

import "time"

func inputReady(_ uintptr, _ time.Duration) (bool, error) {
	return false, nil
}
