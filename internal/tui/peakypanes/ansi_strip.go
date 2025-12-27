package peakypanes

import "strings"

func stripANSI(s string) string {
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
			if b >= 0x20 || b == '\n' || b == '\r' || b == '\t' {
				out = append(out, b)
			}
			i++
			continue
		}

		if i+1 >= len(in) {
			i++
			continue
		}

		next := in[i+1]
		switch next {
		case '[': // CSI
			i += 2
			for i < len(in) {
				c := in[i]
				i++
				if c >= 0x40 && c <= 0x7e {
					break
				}
			}
		case ']': // OSC
			i += 2
			for i < len(in) {
				if in[i] == bel {
					i++
					break
				}
				if in[i] == esc && i+1 < len(in) && in[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}
		case 'P', '^', '_', 'X': // DCS/PM/APC/SOS
			i += 2
			for i < len(in) {
				if in[i] == esc && i+1 < len(in) && in[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}
		default:
			i += 2
		}
	}

	return string(out)
}

func isBlankANSI(line string) bool {
	return strings.TrimSpace(stripANSI(line)) == ""
}
