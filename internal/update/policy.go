package update

import "time"

const (
	DefaultPromptCooldown = 72 * time.Hour
	DefaultCheckInterval  = 24 * time.Hour
)

// Policy defines update prompt behavior.
type Policy struct {
	PromptCooldown time.Duration
	CheckInterval  time.Duration
}

// DefaultPolicy returns the default update policy.
func DefaultPolicy() Policy {
	return Policy{PromptCooldown: DefaultPromptCooldown, CheckInterval: DefaultCheckInterval}
}

// ShouldPrompt reports whether the update dialog should be shown.
func (p Policy) ShouldPrompt(state State, now time.Time) bool {
	if !state.UpdateAvailable() {
		return false
	}
	if state.IsSkipped() {
		return false
	}
	if state.LastPromptUnixMs <= 0 {
		return true
	}
	cooldown := p.PromptCooldown
	if cooldown <= 0 {
		cooldown = DefaultPromptCooldown
	}
	lastPrompt := time.UnixMilli(state.LastPromptUnixMs)
	return now.Sub(lastPrompt) >= cooldown
}

// ShouldShowBanner reports whether the update banner should be visible.
func (p Policy) ShouldShowBanner(state State) bool {
	if !state.UpdateAvailable() {
		return false
	}
	return !state.IsSkipped()
}

// ShouldCheck reports whether a periodic update check should run.
func (p Policy) ShouldCheck(lastCheckUnixMs int64, now time.Time) bool {
	interval := p.CheckInterval
	if interval <= 0 {
		interval = DefaultCheckInterval
	}
	if lastCheckUnixMs <= 0 {
		return true
	}
	lastCheck := time.UnixMilli(lastCheckUnixMs)
	return now.Sub(lastCheck) >= interval
}
