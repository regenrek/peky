package dashboard

import (
	"fmt"
	"os"
)

func openTUIInput() (*os.File, func(), error) {
	return openTUIInputWith(os.OpenFile, ensureBlocking, os.Stdin)
}

func openTUIInputWith(
	openFileFn func(string, int, os.FileMode) (*os.File, error),
	ensureBlockingFn func(*os.File) error,
	stdin *os.File,
) (*os.File, func(), error) {
	if openFileFn == nil {
		openFileFn = os.OpenFile
	}
	if ensureBlockingFn == nil {
		ensureBlockingFn = ensureBlocking
	}
	if stdin == nil {
		stdin = os.Stdin
	}

	if f, err := openFileFn("/dev/tty", os.O_RDWR, 0); err == nil {
		if err := ensureBlockingFn(f); err != nil {
			_ = f.Close()
			return nil, func() {}, fmt.Errorf("configure /dev/tty: %w", err)
		}
		return f, func() { _ = f.Close() }, nil
	}
	if err := ensureBlockingFn(stdin); err != nil {
		return nil, func() {}, fmt.Errorf("stdin is not a usable TUI input: %w", err)
	}
	return stdin, func() {}, nil
}
