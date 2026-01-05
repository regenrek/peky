package agent

import "strings"

type Config struct {
	Provider        Provider
	Model           string
	AllowedCommands []string
	BlockedCommands []string
}

func (c Config) normalized() Config {
	out := c
	out.Provider = Provider(strings.ToLower(strings.TrimSpace(string(c.Provider))))
	out.Model = strings.TrimSpace(out.Model)
	return out
}
