package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/mux"
	"github.com/regenrek/peakypanes/internal/zellijctl"
)

func loadGlobalConfig() (*layout.Config, string, error) {
	path, err := layout.DefaultConfigPath()
	if err != nil {
		return &layout.Config{}, "", err
	}
	cfg, err := layout.LoadConfig(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &layout.Config{}, path, nil
		}
		return &layout.Config{}, path, err
	}
	return cfg, path, nil
}

func resolveMuxType(cliValue string, cfg *layout.Config, project *layout.ProjectConfig, local *layout.ProjectLocalConfig) mux.Type {
	return mux.ResolveType(cliValue, cfg, project, local)
}

func newMuxClient(muxType mux.Type, cfg *layout.Config) (mux.Client, error) {
	switch muxType {
	case mux.Tmux:
		return mux.NewTmuxClient("")
	case mux.Zellij:
		bridgePath := ""
		if cfg != nil && strings.TrimSpace(cfg.Zellij.BridgePlugin) != "" {
			bridgePath = expandUserPath(cfg.Zellij.BridgePlugin)
		}
		return mux.NewZellijClient("", bridgePath)
	default:
		return nil, fmt.Errorf("unsupported multiplexer %q", muxType)
	}
}

func ensureZellijConfig(cfg *layout.Config, bridgePath string) (string, error) {
	base := ""
	if cfg != nil {
		base = strings.TrimSpace(cfg.Zellij.Config)
		if base != "" {
			base = expandUserPath(base)
		}
	}
	if strings.TrimSpace(bridgePath) == "" {
		return "", fmt.Errorf("zellij bridge plugin path is required")
	}
	return zellijctl.EnsureConfigWithBridge(base, bridgePath)
}

func resolveZellijLayoutDir(cfg *layout.Config) (string, error) {
	if cfg != nil && strings.TrimSpace(cfg.Zellij.LayoutDir) != "" {
		return expandUserPath(cfg.Zellij.LayoutDir), nil
	}
	return zellijctl.DefaultLayoutDir()
}

func findProjectConfig(cfg *layout.Config, projectPath string, sessionName string) *layout.ProjectConfig {
	if cfg == nil {
		return nil
	}
	normalizedProject := ""
	if strings.TrimSpace(projectPath) != "" {
		normalizedProject = filepath.Clean(expandUserPath(projectPath))
	}
	for i := range cfg.Projects {
		pc := &cfg.Projects[i]
		path := strings.TrimSpace(pc.Path)
		if path != "" {
			path = filepath.Clean(expandUserPath(path))
			if normalizedProject != "" && path == normalizedProject {
				return pc
			}
		}
		if sessionName != "" && strings.TrimSpace(pc.Session) == sessionName {
			return pc
		}
	}
	return nil
}
