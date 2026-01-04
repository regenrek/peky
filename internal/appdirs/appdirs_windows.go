//go:build windows

package appdirs

import "errors"

func RuntimeDir() (string, error) {
	return "", errors.New("runtime dirs are not supported on windows yet")
}
