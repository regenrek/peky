package sessionrestore

import "time"

const (
	DefaultSnapshotInterval = 2 * time.Second
	DefaultMaxDiskMB        = 512
	DefaultTTLInactive      = 7 * 24 * time.Hour
)

// Config configures session restore persistence.
type Config struct {
	Enabled            bool
	BaseDir            string
	MaxScrollbackLines int
	MaxScrollbackBytes int64
	SnapshotInterval   time.Duration
	MaxDiskBytes       int64
	TTLInactive        time.Duration
}

func (c Config) Normalized() Config {
	cfg := c
	if cfg.SnapshotInterval <= 0 {
		cfg.SnapshotInterval = DefaultSnapshotInterval
	}
	if cfg.MaxDiskBytes <= 0 && DefaultMaxDiskMB > 0 {
		cfg.MaxDiskBytes = int64(DefaultMaxDiskMB) * 1024 * 1024
	}
	if cfg.TTLInactive <= 0 {
		cfg.TTLInactive = DefaultTTLInactive
	}
	return cfg
}
