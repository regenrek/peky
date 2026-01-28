package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	bundleEnvVar       = "PEKY_SKILLS_DIR"
	manifestFilename   = "manifest.json"
	bundleDirName      = "skills"
	bundleSearchMaxAsc = 4
)

// Bundle holds the loaded skill bundle.
type Bundle struct {
	Root    string
	Version int
	Skills  []Skill
}

// LoadBundle locates and loads the bundled skills manifest.
func LoadBundle() (Bundle, error) {
	root, err := bundledDir()
	if err != nil {
		return Bundle{}, err
	}
	manifest, err := loadManifest(root)
	if err != nil {
		return Bundle{}, err
	}
	skills, err := manifest.resolveSkills(root)
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{Root: root, Version: manifest.Version, Skills: skills}, nil
}

func bundledDir() (string, error) {
	if value := strings.TrimSpace(os.Getenv(bundleEnvVar)); value != "" {
		return ensureBundleDir(value)
	}
	cwd, _ := os.Getwd()
	if dir := findBundleFromBase(cwd); dir != "" {
		return dir, nil
	}
	exe, err := os.Executable()
	if err == nil {
		if dir := findBundleFromBase(filepath.Dir(exe)); dir != "" {
			return dir, nil
		}
	}
	return "", errors.New("skills bundle not found")
}

func findBundleFromBase(base string) string {
	if base == "" {
		return ""
	}
	start := filepath.Clean(base)
	current := start
	for i := 0; i <= bundleSearchMaxAsc; i++ {
		candidate := filepath.Join(current, bundleDirName)
		if dir, err := ensureBundleDir(candidate); err == nil {
			return dir
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

func ensureBundleDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", errors.New("bundle dir is empty")
	}
	manifest := filepath.Join(dir, manifestFilename)
	info, err := os.Stat(manifest)
	if err != nil {
		return "", fmt.Errorf("manifest missing at %s", manifest)
	}
	if info.IsDir() {
		return "", fmt.Errorf("manifest path is a directory: %s", manifest)
	}
	return dir, nil
}
