package app

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

func openBrowserURL(raw string) error {
	url := strings.TrimSpace(raw)
	if url == "" {
		return errors.New("url is required")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
