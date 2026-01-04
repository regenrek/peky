package logging

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Sink string

const (
	SinkStderr Sink = "stderr"
	SinkFile   Sink = "file"
	SinkNone   Sink = "none"
)

const (
	EnvLogLevel           = "PEAKYPANES_LOG_LEVEL"
	EnvLogFormat          = "PEAKYPANES_LOG_FORMAT"
	EnvLogSink            = "PEAKYPANES_LOG_SINK"
	EnvLogFile            = "PEAKYPANES_LOG_FILE"
	EnvLogAddSource       = "PEAKYPANES_LOG_ADD_SOURCE"
	EnvLogIncludePayloads = "PEAKYPANES_LOG_INCLUDE_PAYLOADS"
	EnvLogMaxSizeMB       = "PEAKYPANES_LOG_MAX_SIZE_MB"
	EnvLogMaxBackups      = "PEAKYPANES_LOG_MAX_BACKUPS"
	EnvLogMaxAgeDays      = "PEAKYPANES_LOG_MAX_AGE_DAYS"
	EnvLogCompress        = "PEAKYPANES_LOG_COMPRESS"
)

type Config struct {
	Level           *string `yaml:"level,omitempty"`
	Format          *string `yaml:"format,omitempty"`
	Sink            *string `yaml:"sink,omitempty"`
	File            *string `yaml:"file,omitempty"`
	AddSource       *bool   `yaml:"add_source,omitempty"`
	IncludePayloads *bool   `yaml:"include_payloads,omitempty"`

	MaxSizeMB  *int  `yaml:"max_size_mb,omitempty"`
	MaxBackups *int  `yaml:"max_backups,omitempty"`
	MaxAgeDays *int  `yaml:"max_age_days,omitempty"`
	Compress   *bool `yaml:"compress,omitempty"`
}

func DefaultConfig(mode Mode) Config {
	// Defaults are selected to be quiet on CLI and informative for the daemon.
	level := "error"
	sink := string(SinkStderr)
	format := string(FormatText)
	addSource := false

	if mode == ModeDaemon {
		level = "info"
		sink = string(SinkFile)
		format = string(FormatJSON)
	}

	maxSizeMB := 20
	maxBackups := 5
	maxAgeDays := 7
	compress := true
	includePayloads := false

	return Config{
		Level:           &level,
		Format:          &format,
		Sink:            &sink,
		AddSource:       &addSource,
		IncludePayloads: &includePayloads,
		MaxSizeMB:       &maxSizeMB,
		MaxBackups:      &maxBackups,
		MaxAgeDays:      &maxAgeDays,
		Compress:        &compress,
	}
}

func (c Config) WithEnv() Config {
	applyString := func(dst **string, env string) {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			*dst = &v
		}
	}
	applyBool := func(dst **bool, env string) {
		raw := strings.TrimSpace(os.Getenv(env))
		if raw == "" {
			return
		}
		v := !isDisabledString(raw)
		*dst = &v
	}
	applyInt := func(dst **int, env string) {
		raw := strings.TrimSpace(os.Getenv(env))
		if raw == "" {
			return
		}
		n, err := strconv.Atoi(raw)
		if err != nil {
			return
		}
		*dst = &n
	}

	applyString(&c.Level, EnvLogLevel)
	applyString(&c.Format, EnvLogFormat)
	applyString(&c.Sink, EnvLogSink)
	applyString(&c.File, EnvLogFile)
	applyBool(&c.AddSource, EnvLogAddSource)
	applyBool(&c.IncludePayloads, EnvLogIncludePayloads)
	applyInt(&c.MaxSizeMB, EnvLogMaxSizeMB)
	applyInt(&c.MaxBackups, EnvLogMaxBackups)
	applyInt(&c.MaxAgeDays, EnvLogMaxAgeDays)
	applyBool(&c.Compress, EnvLogCompress)
	return c
}

func (c Config) Normalize() (Config, error) {
	normalizeString := func(s *string) *string {
		if s == nil {
			return nil
		}
		v := strings.ToLower(strings.TrimSpace(*s))
		if v == "" {
			return nil
		}
		return &v
	}
	c.Level = normalizeString(c.Level)
	c.Format = normalizeString(c.Format)
	c.Sink = normalizeString(c.Sink)
	if c.File != nil {
		v := strings.TrimSpace(*c.File)
		if v == "" {
			c.File = nil
		} else {
			c.File = &v
		}
	}
	if c.MaxSizeMB != nil && *c.MaxSizeMB < 0 {
		zero := 0
		c.MaxSizeMB = &zero
	}
	if c.MaxBackups != nil && *c.MaxBackups < 0 {
		zero := 0
		c.MaxBackups = &zero
	}
	if c.MaxAgeDays != nil && *c.MaxAgeDays < 0 {
		zero := 0
		c.MaxAgeDays = &zero
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	if c.Level != nil {
		switch *c.Level {
		case "debug", "info", "warn", "warning", "error":
		default:
			return fmt.Errorf("logging.level: invalid %q", *c.Level)
		}
	}
	if c.Format != nil {
		switch Format(*c.Format) {
		case FormatText, FormatJSON:
		default:
			return fmt.Errorf("logging.format: invalid %q", *c.Format)
		}
	}
	if c.Sink != nil {
		switch Sink(*c.Sink) {
		case SinkStderr, SinkFile, SinkNone:
		default:
			return fmt.Errorf("logging.sink: invalid %q", *c.Sink)
		}
	}
	return nil
}

func isDisabledString(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "no", "off":
		return true
	default:
		return false
	}
}
