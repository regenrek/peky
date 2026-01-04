package terminal

import "time"

const (
	perfLogInterval     = 2 * time.Second
	perfSlowLock        = 10 * time.Millisecond
	perfSlowWrite       = 15 * time.Millisecond
	perfSlowANSIRender  = 25 * time.Millisecond
	perfSlowLipgloss    = 25 * time.Millisecond
	perfSlowLipglossAll = 40 * time.Millisecond
)
