package dashboard

import (
	"bytes"
	"time"
)

type sgrMouseFragmentKind uint8

const (
	sgrMouseFragUnknown sgrMouseFragmentKind = iota
	sgrMouseFragNumM
	sgrMouseFragNumNumM
	sgrMouseFragNumNumNumM
	sgrMouseFragSemiNumM
	sgrMouseFragSemiNumNumM
)

func looksLikeMousePrefix(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	switch b[0] {
	case '<', '[', ';':
		return true
	case escByte:
		return len(b) > 1 && b[1] == '['
	default:
		return b[0] >= '0' && b[0] <= '9' && bytes.IndexByte(b, ';') >= 0
	}
}

func isIncompleteSGRMousePrefix(b []byte) bool {
	if len(b) < 3 {
		return false
	}
	if b[0] != escByte || b[1] != '[' {
		return false
	}
	k := 2
	for k < len(b) && b[k] == '[' {
		k++
	}
	if k >= len(b) {
		return true
	}
	return b[k] == '<'
}

func isIncompleteEscapeSequence(b []byte) bool {
	// Allow lone ESC to be flushed as the Escape key after singlePrefixWait.
	if len(b) <= 1 || b[0] != escByte {
		return false
	}
	_, ok := scanEscapeSequence(b, 0)
	return !ok
}

func safeReadLen(out []byte, max int) int {
	if max <= 0 || len(out) == 0 {
		return 0
	}
	cut := max
	if cut > len(out) {
		cut = len(out)
	}

	escIdx := bytes.LastIndexByte(out[:cut], escByte)
	if escIdx < 0 {
		return cut
	}
	end, ok := scanEscapeSequence(out, escIdx)
	if ok && end <= cut {
		return cut
	}
	if escIdx == 0 {
		// Avoid returning (0, nil) from Read(). If we can't safely return a full
		// escape sequence, always make progress by returning a single byte.
		return 1
	}
	return escIdx
}

func scanEscapeSequence(b []byte, start int) (end int, ok bool) {
	if start < 0 || start >= len(b) {
		return start, false
	}
	if b[start] != escByte {
		return start + 1, true
	}
	if start+1 >= len(b) {
		return start, false
	}

	switch b[start+1] {
	case '[':
		// Some terminals emit ESC[[... sequences (function keys). Treat that as a
		// special CSI-like variant so we don't split and leak stray '['.
		if start+2 < len(b) && b[start+2] == '[' {
			for i := start + 3; i < len(b); i++ {
				c := b[i]
				if c >= 0x40 && c <= 0x7e {
					return i + 1, true
				}
			}
			return start, false
		}
		for i := start + 2; i < len(b); i++ {
			c := b[i]
			if c >= 0x40 && c <= 0x7e {
				return i + 1, true
			}
		}
		return start, false
	case 'O':
		if start+2 >= len(b) {
			return start, false
		}
		return start + 3, true
	case ']':
		for i := start + 2; i < len(b); i++ {
			if b[i] == 0x07 {
				return i + 1, true
			}
			if b[i] == escByte && i+1 < len(b) && b[i+1] == '\\' {
				return i + 2, true
			}
		}
		return start, false
	case 'P', '_', '^', 'X':
		for i := start + 2; i < len(b); i++ {
			if b[i] == escByte && i+1 < len(b) && b[i+1] == '\\' {
				return i + 2, true
			}
		}
		return start, false
	default:
		return start + 2, true
	}
}

func scanSGRMouse(b []byte, start int) (end int, needMore bool, ok bool) {
	if start >= len(b) {
		return start, true, false
	}
	if b[start] != '<' {
		return start, false, false
	}
	i := start + 1

	i, needMore, ok = scanUint(b, i)
	if !ok || needMore {
		return start, needMore, false
	}
	if i >= len(b) {
		return start, true, false
	}
	if b[i] != ';' {
		return start, false, false
	}
	i++

	i, needMore, ok = scanUint(b, i)
	if !ok || needMore {
		return start, needMore, false
	}
	if i >= len(b) {
		return start, true, false
	}
	if b[i] != ';' {
		return start, false, false
	}
	i++

	i, needMore, ok = scanUint(b, i)
	if !ok || needMore {
		return start, needMore, false
	}
	if i >= len(b) {
		return start, true, false
	}
	switch b[i] {
	case 'M', 'm':
		return i + 1, false, true
	default:
		return start, false, false
	}
}

func scanUint(b []byte, start int) (end int, needMore bool, ok bool) {
	if start >= len(b) {
		return start, true, false
	}
	i := start
	digits := 0
	for i < len(b) {
		c := b[i]
		if c < '0' || c > '9' {
			break
		}
		digits++
		if digits > 6 {
			return start, false, false
		}
		i++
	}
	if digits == 0 {
		if i >= len(b) {
			return start, true, false
		}
		return start, false, false
	}
	return i, false, true
}

func scanSGRMouseFragment(b []byte, start int) (end int, needMore bool, ok bool, kind sgrMouseFragmentKind) {
	if start >= len(b) {
		return start, true, false, sgrMouseFragUnknown
	}
	switch b[start] {
	case '<', escByte:
		return start, false, false, sgrMouseFragUnknown
	}

	if b[start] == ';' {
		// Missing cb: ";cyM" or ";cx;cyM"
		i := start + 1
		var okNum bool
		i, needMore, okNum = scanUint(b, i)
		if needMore {
			return start, true, false, sgrMouseFragUnknown
		}
		if !okNum {
			return start, false, false, sgrMouseFragUnknown
		}
		if i >= len(b) {
			return start, true, false, sgrMouseFragUnknown
		}
		if b[i] == 'M' {
			return i + 1, false, true, sgrMouseFragSemiNumM
		}
		if b[i] != ';' {
			return start, false, false, sgrMouseFragUnknown
		}
		i++
		i, needMore, okNum = scanUint(b, i)
		if needMore {
			return start, true, false, sgrMouseFragUnknown
		}
		if !okNum {
			return start, false, false, sgrMouseFragUnknown
		}
		if i >= len(b) {
			return start, true, false, sgrMouseFragUnknown
		}
		if b[i] == 'M' {
			return i + 1, false, true, sgrMouseFragSemiNumNumM
		}
		return start, false, false, sgrMouseFragUnknown
	}

	// "cyM", "cx;cyM", or "cb;cx;cyM"
	if b[start] < '0' || b[start] > '9' {
		return start, false, false, sgrMouseFragUnknown
	}
	i := start
	var okNum bool
	i, needMore, okNum = scanUint(b, i)
	if needMore {
		return start, true, false, sgrMouseFragUnknown
	}
	if !okNum {
		return start, false, false, sgrMouseFragUnknown
	}
	if i >= len(b) {
		// Digits alone are not a mouse fragment.
		return start, false, false, sgrMouseFragUnknown
	}
	if b[i] == 'M' {
		return i + 1, false, true, sgrMouseFragNumM
	}
	if b[i] != ';' {
		return start, false, false, sgrMouseFragUnknown
	}
	i++
	i, needMore, okNum = scanUint(b, i)
	if needMore {
		return start, true, false, sgrMouseFragUnknown
	}
	if !okNum {
		return start, false, false, sgrMouseFragUnknown
	}
	if i >= len(b) {
		return start, true, false, sgrMouseFragUnknown
	}
	if b[i] == 'M' {
		return i + 1, false, true, sgrMouseFragNumNumM
	}
	if b[i] != ';' {
		return start, false, false, sgrMouseFragUnknown
	}
	i++
	i, needMore, okNum = scanUint(b, i)
	if needMore {
		return start, true, false, sgrMouseFragUnknown
	}
	if !okNum {
		return start, false, false, sgrMouseFragUnknown
	}
	if i >= len(b) {
		return start, true, false, sgrMouseFragUnknown
	}
	if b[i] == 'M' {
		return i + 1, false, true, sgrMouseFragNumNumNumM
	}
	return start, false, false, sgrMouseFragUnknown
}

func shouldDropMouseFragment(lastMouseSeqAt time.Time, b []byte, end int, kind sgrMouseFragmentKind) bool {
	if end < len(b) {
		if b[end] == escByte && end+1 < len(b) && b[end+1] == '[' {
			k := end + 2
			for k < len(b) && b[k] == '[' {
				k++
			}
			if k < len(b) && b[k] == '<' {
				return true
			}
		}
	}

	switch kind {
	case sgrMouseFragNumNumNumM, sgrMouseFragSemiNumNumM:
		if !lastMouseSeqAt.IsZero() && time.Since(lastMouseSeqAt) < maxMouseFragmentAge {
			return true
		}
	default:
		// For ambiguous tail fragments like "1M" or "66;21M", require adjacency to a
		// real SGR mouse report to avoid dropping legitimate user input.
	}
	return false
}
