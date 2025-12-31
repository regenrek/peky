//go:build windows

package dashboard

import "os"

func ensureBlocking(_ *os.File) error {
	return nil
}
