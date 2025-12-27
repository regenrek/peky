package tmuxstream

import "strings"

// shellQuote returns a POSIX-sh safe single-quoted argument.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
