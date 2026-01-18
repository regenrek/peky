package initcfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/layout"
)

// Register registers init handler.
func Register(reg *root.Registry) {
	reg.Register("init", runInit)
}

func runInit(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("init", ctx.Deps.Version)
	local := ctx.Cmd.Bool("local")
	layoutName := strings.TrimSpace(ctx.Cmd.String("layout"))
	if layoutName == "" {
		layoutName = layout.DefaultLayoutName
	}
	force := ctx.Cmd.Bool("force")
	appName := identity.CLIName
	var err error
	if local {
		cwd, resolveErr := root.ResolveWorkDir(ctx)
		if resolveErr != nil {
			return resolveErr
		}
		err = initLocal(appName, layoutName, force, cwd)
	} else {
		err = initGlobal(appName, layoutName, force)
	}
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action: "init",
			Status: "ok",
		})
	}
	return nil
}

func initLocal(appName, layoutName string, force bool, cwd string) error {
	configPath := filepath.Join(cwd, identity.ProjectConfigFileYML)
	if !force {
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", identity.ProjectConfigFileYML)
		}
	}

	baseLayout, err := layout.GetBuiltinLayout(layoutName)
	if err != nil {
		return fmt.Errorf("layout %q not found", layoutName)
	}

	projectName := filepath.Base(cwd)
	content := fmt.Sprintf(`# peky - Project Layout Configuration
# This file defines the peky layout for this project.
# Teammates with %s installed will get this layout automatically.
#
# Variables: ${PROJECT_NAME}, ${PROJECT_PATH}, ${EDITOR}, or any env var
# Use ${VAR:-default} for defaults

session: %s

layout:
`, appName, projectName)

	yamlContent, err := baseLayout.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to serialize layout: %w", err)
	}

	lines := strings.Split(yamlContent, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "name:") || strings.HasPrefix(line, "description:") {
			continue
		}
		if line != "" {
			content += "  " + line + "\n"
		}
	}

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", configPath, err)
	}

	fmt.Printf("✨ Created %s\n", configPath)
	fmt.Printf("   Based on layout: %s\n", layoutName)
	fmt.Printf("\n   Edit it to customize, then:\n")
	fmt.Printf("   • Run '%s start' to start the session\n", appName)
	fmt.Printf("   • Run '%s' to open the dashboard\n", appName)
	fmt.Printf("   • Commit to git so teammates get the same layout\n")
	return nil
}

func initGlobal(appName, layoutName string, force bool) error {
	configPath, err := layout.DefaultConfigPath()
	if err != nil {
		return fmt.Errorf("cannot determine config path: %w", err)
	}

	layoutsDir, err := layout.DefaultLayoutsDir()
	if err != nil {
		return fmt.Errorf("cannot determine layouts dir: %w", err)
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	if err := os.MkdirAll(layoutsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create layouts dir: %w", err)
	}

	if !force {
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config already exists: %s\n", configPath)
			fmt.Printf("Use --force to overwrite\n")
			return nil
		}
	}

	if err := os.WriteFile(configPath, []byte(layout.DefaultGlobalConfigContent()), 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✨ Initialized peky!\n\n")
	fmt.Printf("   Config: %s\n", configPath)
	fmt.Printf("   Layouts: %s\n\n", layoutsDir)
	fmt.Printf("   Next steps:\n")
	fmt.Printf("   • Run '%s layouts' to see available layouts\n", appName)
	fmt.Printf("   • Run '%s init --local' in a project to create %s\n", appName, identity.ProjectConfigFileYML)
	fmt.Printf("   • Run '%s start' in any directory to start a session\n", appName)
	fmt.Printf("   • Run '%s' to open the dashboard\n", appName)
	return nil
}
