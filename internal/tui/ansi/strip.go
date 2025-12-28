package ansi

import "strings"

// Strip removes ANSI escape sequences from a string.
func Strip(s string) string {
	if s == "" {
		return ""
	}

	const (
		esc = byte(0x1b)
		bel = byte(0x07)
	)

	in := []byte(s)
	out := make([]byte, 0, len(in))

	for i := 0; i < len(in); {
		b := in[i]
		if b != esc {
			if isPrintable(b) {
				out = append(out, b)
			}
			i++
			continue
		}
		i = skipEscape(in, i, esc, bel)
	}

	return string(out)
}

func isPrintable(b byte) bool {
	return b >= 0x20 || b == '\n' || b == '\r' || b == '\t'
}

func skipEscape(in []byte, i int, esc, bel byte) int {
	if i+1 >= len(in) {
		return i + 1
	}
	switch in[i+1] {
	case '[': // CSI
		return skipCSI(in, i+2)
	case ']': // OSC
		return skipOSC(in, i+2, esc, bel)
	case 'P', '^', '_', 'X': // DCS/PM/APC/SOS
		return skipStringTerminated(in, i+2, esc)
	default:
		return i + 2
	}
}

func skipCSI(in []byte, i int) int {
	for i < len(in) {
		c := in[i]
		i++
		if c >= 0x40 && c <= 0x7e {
			break
		}
	}
	return i
}

func skipOSC(in []byte, i int, esc, bel byte) int {
	for i < len(in) {
		if in[i] == bel {
			return i + 1
		}
		if in[i] == esc && i+1 < len(in) && in[i+1] == '\\' {
			return i + 2
		}
		i++
	}
	return i
}

func skipStringTerminated(in []byte, i int, esc byte) int {
	for i < len(in) {
		if in[i] == esc && i+1 < len(in) && in[i+1] == '\\' {
			return i + 2
		}
		i++
	}
	return i
}

// IsBlank reports whether the string is blank after ANSI stripping.
func IsBlank(line string) bool {
	return strings.TrimSpace(Strip(line)) == ""
}

// LastNonEmpty returns the last non-empty line after stripping ANSI codes.
func LastNonEmpty(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(Strip(lines[i]))
		if line != "" {
			return line
		}
	}
	return ""
}
