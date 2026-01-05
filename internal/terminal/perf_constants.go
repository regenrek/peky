package terminal

import "time"

const (
	perfLogInterval     = 2 * time.Second
	perfSlowLock        = 10 * time.Millisecond
	perfSlowWrite       = 15 * time.Millisecond
	perfSlowFrameRender = 40 * time.Millisecond
)
