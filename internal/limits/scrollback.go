package limits

const (
	// TerminalScrollbackMaxBytesDefault matches the “byte budget” approach used by
	// best-in-class terminals. It bounds worst-case memory and reflow cost per pane.
	TerminalScrollbackMaxBytesDefault int64 = 10 * 1024 * 1024

	// TerminalScrollbackMaxBytesMax is a hard ceiling to prevent pathological configs
	// from exhausting memory on multi-pane workloads.
	TerminalScrollbackMaxBytesMax int64 = 512 * 1024 * 1024
)
