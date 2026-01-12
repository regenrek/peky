package runenv

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	RuntimeDirEnv          = "PEKY_RUNTIME_DIR"
	DataDirEnv             = "PEKY_DATA_DIR"
	ConfigDirEnv           = "PEKY_CONFIG_DIR"
	FreshConfigEnv         = "PEKY_FRESH_CONFIG"
	StartSessionTimeoutEnv = "PEKY_START_TIMEOUT"
)

func enabledEnv(name string) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return false
	}
	switch strings.ToLower(value) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func FreshConfigEnabled() bool {
	return enabledEnv(FreshConfigEnv)
}

func ConfigDir() string {
	return strings.TrimSpace(os.Getenv(ConfigDirEnv))
}

func RuntimeDir() string {
	return strings.TrimSpace(os.Getenv(RuntimeDirEnv))
}

func DataDir() string {
	return strings.TrimSpace(os.Getenv(DataDirEnv))
}

func StartSessionTimeout() time.Duration {
	const fallback = 5 * time.Second
	raw := strings.TrimSpace(os.Getenv(StartSessionTimeoutEnv))
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil {
		if d <= 0 {
			return fallback
		}
		return d
	}
	secs, err := strconv.Atoi(raw)
	if err != nil || secs <= 0 {
		return fallback
	}
	return time.Duration(secs) * time.Second
}
