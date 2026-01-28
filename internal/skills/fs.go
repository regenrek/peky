package skills

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func copyDir(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat src: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source is not a directory")
	}
	if err := os.MkdirAll(dest, info.Mode().Perm()); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink not supported: %s", path)
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(destPath, info.Mode().Perm())
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := copyFile(path, destPath, info.Mode().Perm()); err != nil {
			return err
		}
		return nil
	})
}

func copyFile(src, dest string, perm fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("mkdir dest: %w", err)
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("open dest: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	return nil
}

func hashDir(root string) (string, error) {
	files := make([]string, 0)
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink not supported: %s", path)
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, path := range files {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		_, _ = h.Write([]byte(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		_, _ = h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func trashPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("empty path")
	}
	trash, err := exec.LookPath("trash")
	if err != nil {
		return errors.New("trash command not found")
	}
	cmd := exec.Command(trash, "-F", "--", path)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("trash failed: %w", err)
	}
	return nil
}
