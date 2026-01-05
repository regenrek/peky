package root

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func ResolveWorkDir(ctx CommandContext) (string, error) {
	if strings.TrimSpace(ctx.WorkDir) != "" {
		return normalizeWorkDir(ctx.WorkDir)
	}
	if strings.TrimSpace(ctx.Deps.WorkDir) != "" {
		return normalizeWorkDir(ctx.Deps.WorkDir)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return normalizeWorkDir(cwd)
}

func normalizeWorkDir(dir string) (string, error) {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		return "", errors.New("workdir is empty")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("workdir is not a directory")
	}
	return abs, nil
}
