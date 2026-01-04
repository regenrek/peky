package limits

const (
	// TerminalScrollbackMaxBytesDefault matches the “byte budget” approach used by
	// best-in-class terminals. It bounds worst-case memory and reflow cost per pane.
	TerminalScrollbackMaxBytesDefault int64 = 10 * 1024 * 1024

	// TerminalScrollbackMaxBytesMax is a hard ceiling to prevent pathological configs
	// from exhausting memory on multi-pane workloads.
	TerminalScrollbackMaxBytesMax int64 = 512 * 1024 * 1024

	// TerminalScrollbackTotalMaxBytesDefault bounds worst-case scrollback memory across
	// all panes in a single daemon instance. The per-pane budget is derived from this
	// when many panes exist to avoid host memory blowups under stress.
	TerminalScrollbackTotalMaxBytesDefault int64 = 512 * 1024 * 1024

	// TerminalScrollbackTotalMaxBytesMax is an upper bound for any future config override.
	TerminalScrollbackTotalMaxBytesMax int64 = 8 * 1024 * 1024 * 1024
)

// ScrollbackMaxBytesPerPane returns the byte budget to assign each pane given the
// number of active panes. It enforces a global cap while preserving the per-pane
// default when pane counts are small.
func ScrollbackMaxBytesPerPane(activePanes int) int64 {
	if activePanes <= 0 {
		return TerminalScrollbackMaxBytesDefault
	}
	per := TerminalScrollbackTotalMaxBytesDefault / int64(activePanes)
	if per <= 0 {
		// Negative disables scrollback; 0 is reserved for “use default”.
		return -1
	}
	if per > TerminalScrollbackMaxBytesMax {
		per = TerminalScrollbackMaxBytesMax
	}
	if per > TerminalScrollbackMaxBytesDefault {
		per = TerminalScrollbackMaxBytesDefault
	}
	return per
}
