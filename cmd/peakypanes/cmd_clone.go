package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func runClone(args []string) {
	if len(args) == 0 {
		fatal("usage: peakypanes clone <url|user/repo>")
	}

	url := args[0]
	// Expand shorthand (user/repo -> https://github.com/user/repo)
	if !strings.Contains(url, "://") && !strings.HasPrefix(url, "git@") {
		url = "https://github.com/" + url
	}

	// Extract repo name for directory
	repoName := extractRepoName(url)
	if repoName == "" {
		repoName = "repo"
	}

	// Clone to ~/projects/<repo>
	home, _ := os.UserHomeDir()
	targetDir := filepath.Join(home, "projects", repoName)

	// Check if directory already exists
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Printf("📁 Directory exists: %s\n", targetDir)
		fmt.Printf("   Starting session...\n\n")
		runStartWithPath(targetDir)
		return
	}

	fmt.Printf("🔄 Cloning %s...\n", url)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", url, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal("clone failed: %v", err)
	}

	fmt.Printf("\n✅ Cloned to %s\n\n", targetDir)

	runStartWithPath(targetDir)
}

func runStartWithPath(projectPath string) {
	origArgs := os.Args
	os.Args = []string{"peakypanes", "start", "--path", projectPath}
	runStart([]string{"--path", projectPath})
	os.Args = origArgs
}

func extractRepoName(url string) string {
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
