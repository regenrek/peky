package sessiond

import (
	"fmt"
	"os"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/tool"
)

func loadToolRegistry() (*tool.Registry, error) {
	configPath, err := layout.DefaultConfigPath()
	if err != nil {
		return tool.DefaultRegistry()
	}
	if configPath == "" {
		return tool.DefaultRegistry()
	}
	info, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return tool.DefaultRegistry()
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("sessiond: config path %q is a directory", configPath)
	}
	cfg, err := layout.LoadConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return tool.DefaultRegistry()
		}
		return nil, err
	}
	registry, err := tool.RegistryFromConfig(cfg.ToolDetection)
	if err != nil {
		return nil, err
	}
	return registry, nil
}

func (d *Daemon) toolRegistryRef() *tool.Registry {
	if d == nil || d.toolRegistry == nil {
		return nil
	}
	return d.toolRegistry
}
