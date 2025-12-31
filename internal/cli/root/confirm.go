package root

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// PromptConfirm asks the user to confirm a side-effect action.
func PromptConfirm(in io.Reader, out io.Writer, message string) (bool, error) {
	if out != nil {
		if _, err := fmt.Fprintf(out, "%s [y/N]: ", message); err != nil {
			return false, err
		}
	}
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	switch answer {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// RequireAck prompts for an explicit acknowledgement token.
func RequireAck(in io.Reader, out io.Writer, token string) (bool, error) {
	if token == "" {
		token = "ack"
	}
	if out != nil {
		if _, err := fmt.Fprintf(out, "Type %q to confirm: ", token); err != nil {
			return false, err
		}
	}
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == strings.ToLower(token) {
		return true, nil
	}
	return false, nil
}
