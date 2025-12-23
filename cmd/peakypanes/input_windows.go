//go:build windows

package main

import "os"

func ensureBlocking(_ *os.File) error {
	return nil
}
