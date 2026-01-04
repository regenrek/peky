package tool

var (
	bracketedPasteStart = [...]byte{0x1b, '[', '2', '0', '0', '~'}
	bracketedPasteEnd   = [...]byte{0x1b, '[', '2', '0', '1', '~'}
)

// WrapBracketedPaste wraps payload in bracketed paste sequences.
func WrapBracketedPaste(payload []byte) []byte {
	out := make([]byte, 0, len(bracketedPasteStart)+len(payload)+len(bracketedPasteEnd))
	out = append(out, bracketedPasteStart[:]...)
	out = append(out, payload...)
	out = append(out, bracketedPasteEnd[:]...)
	return out
}

// ApplyProfile wraps payload according to the profile.
func ApplyProfile(payload []byte, profile Profile, raw bool) []byte {
	if raw || !profile.BracketedPaste {
		return append([]byte(nil), payload...)
	}
	return WrapBracketedPaste(payload)
}
