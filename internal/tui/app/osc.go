package app

import "io"

// SetOSCWriter configures where OSC sequences (like cursor shape) are emitted.
func (m *Model) SetOSCWriter(w io.Writer) {
	if m == nil {
		return
	}
	if w == nil {
		m.oscEmit = nil
		return
	}
	m.oscEmit = func(seq string) {
		if seq == "" {
			return
		}
		_, _ = io.WriteString(w, seq)
	}
}
