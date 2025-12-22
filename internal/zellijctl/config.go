package zellijctl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sblinch/kdl-go"
	"github.com/sblinch/kdl-go/document"
)

// EnsureConfigWithBridge creates a derived zellij config that loads the bridge plugin.
func EnsureConfigWithBridge(baseConfigPath, pluginPath string) (string, error) {
	baseConfigPath = strings.TrimSpace(baseConfigPath)
	if strings.TrimSpace(pluginPath) == "" {
		return "", fmt.Errorf("zellij bridge plugin path is required")
	}
	if baseConfigPath == "" {
		defaultPath, err := DefaultZellijConfigPath()
		if err != nil {
			return "", err
		}
		baseConfigPath = defaultPath
	}

	doc, err := readConfigDocument(baseConfigPath)
	if err != nil {
		return "", err
	}

	pluginURL := normalizePluginURL(pluginPath)
	loadNode := findNode(doc, "load_plugins")
	if loadNode == nil {
		loadNode = document.NewNode()
		loadNode.SetName("load_plugins")
		doc.AddNode(loadNode)
	}
	if !childNodeExists(loadNode, pluginURL) {
		child := document.NewNode()
		child.SetName(pluginURL)
		loadNode.AddNode(child)
	}

	outDir, err := DefaultBridgeDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("create zellij config dir: %w", err)
	}
	outPath := filepath.Join(outDir, "config.kdl")
	var buf bytes.Buffer
	if err := kdl.Generate(doc, &buf); err != nil {
		return "", fmt.Errorf("render zellij config: %w", err)
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("write zellij config: %w", err)
	}
	return outPath, nil
}

// DefaultZellijConfigPath returns the default zellij config path for this OS.
func DefaultZellijConfigPath() (string, error) {
	if env := strings.TrimSpace(os.Getenv("ZELLIJ_CONFIG_FILE")); env != "" {
		return expandPath(env), nil
	}
	if env := strings.TrimSpace(os.Getenv("ZELLIJ_CONFIG_DIR")); env != "" {
		return filepath.Join(expandPath(env), "config.kdl"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "com.Zellij-Contributors.zellij", "config.kdl"), nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "zellij", "config.kdl"), nil
}

// DefaultZellijLayoutDir returns the default layout directory for zellij.
func DefaultZellijLayoutDir() (string, error) {
	if env := strings.TrimSpace(os.Getenv("ZELLIJ_CONFIG_DIR")); env != "" {
		return filepath.Join(expandPath(env), "layouts"), nil
	}
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "com.Zellij-Contributors.zellij", "layouts"), nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "zellij", "layouts"), nil
}

func readConfigDocument(path string) (*document.Document, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return document.New(), nil
		}
		return nil, fmt.Errorf("read zellij config: %w", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return document.New(), nil
	}
	doc, err := kdl.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse zellij config %q: %w", path, err)
	}
	return doc, nil
}

func normalizePluginURL(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "file:") || strings.HasPrefix(path, "http:") || strings.HasPrefix(path, "https:") {
		return path
	}
	return "file:" + filepath.ToSlash(path)
}

func findNode(doc *document.Document, name string) *document.Node {
	for _, node := range doc.Nodes {
		if node != nil && node.Name != nil && node.Name.ValueString() == name {
			return node
		}
	}
	return nil
}

func childNodeExists(parent *document.Node, name string) bool {
	for _, child := range parent.Children {
		if child != nil && child.Name != nil && child.Name.ValueString() == name {
			return true
		}
	}
	return false
}

func expandPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
