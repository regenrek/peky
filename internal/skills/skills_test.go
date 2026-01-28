package skills

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseTargets(t *testing.T) {
	got, err := ParseTargets([]string{"Codex,claude", "cursor"})
	if err != nil {
		t.Fatalf("parse targets: %v", err)
	}
	want := []Target{TargetClaude, TargetCodex, TargetCursor}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("targets=%v want=%v", got, want)
	}

	if _, err := ParseTargets([]string{"nope"}); err == nil {
		t.Fatalf("expected unknown target error")
	}
}

func TestTargetRoot(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)
	got, err := TargetRoot(TargetCodex)
	if err != nil {
		t.Fatalf("target root: %v", err)
	}
	want := filepath.Join(home, ".codex", "skills")
	if got != want {
		t.Fatalf("root=%s want=%s", got, want)
	}
}

func TestLoadBundle(t *testing.T) {
	root := writeTestBundle(t)
	t.Setenv(bundleEnvVar, root)

	bundle, err := LoadBundle()
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if bundle.Version != 1 {
		t.Fatalf("version=%d want=1", bundle.Version)
	}
	if len(bundle.Skills) != 1 {
		t.Fatalf("skills=%d want=1", len(bundle.Skills))
	}
	if bundle.Skills[0].ID != "peky-peky" {
		t.Fatalf("skill id=%s want=peky-peky", bundle.Skills[0].ID)
	}
}

func TestInstallStatusUninstall(t *testing.T) {
	root := writeTestBundle(t)
	t.Setenv(bundleEnvVar, root)
	bundle, err := LoadBundle()
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "dest")
	result, err := Install(bundle, InstallOptions{
		Targets:      []Target{TargetCodex},
		DestOverride: dest,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records=%d want=1", len(result.Records))
	}
	if result.Records[0].Status != "installed" {
		t.Fatalf("status=%s want=installed", result.Records[0].Status)
	}

	destDir := filepath.Join(dest, "peky-peky")
	if _, err := os.Stat(destDir); err != nil {
		t.Fatalf("installed dir: %v", err)
	}

	status, err := Status(bundle, []Target{TargetCodex}, dest)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(status) != 1 {
		t.Fatalf("status records=%d want=1", len(status))
	}
	if !status[0].Present || !status[0].Matches {
		t.Fatalf("status present=%v matches=%v", status[0].Present, status[0].Matches)
	}

	trashBin := writeFakeTrash(t)
	t.Setenv("PATH", filepath.Dir(trashBin)+string(os.PathListSeparator)+os.Getenv("PATH"))

	remove, err := Uninstall(bundle, UninstallOptions{
		Targets:      []Target{TargetCodex},
		DestOverride: dest,
	})
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if len(remove.Records) != 1 {
		t.Fatalf("remove records=%d want=1", len(remove.Records))
	}
	if remove.Records[0].Status != "removed" {
		t.Fatalf("remove status=%s want=removed", remove.Records[0].Status)
	}
	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		t.Fatalf("expected skill removed, err=%v", err)
	}
}

func TestInstallMissingSkill(t *testing.T) {
	root := writeTestBundle(t)
	t.Setenv(bundleEnvVar, root)
	bundle, err := LoadBundle()
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if _, err := Install(bundle, InstallOptions{
		Targets:  []Target{TargetCodex},
		SkillIDs: []string{"missing"},
	}); err == nil {
		t.Fatalf("expected missing skill error")
	}
}

func writeTestBundle(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	skillDir := filepath.Join(root, "peky-peky")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("test"), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
	manifest := `{"version":1,"skills":[{"id":"peky-peky","name":"peky-peky","description":"test","targets":["codex"],"path":"peky-peky"}]}`
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return root
}

func writeFakeTrash(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "trash")
	script := `#!/bin/sh
target=""
for arg in "$@"; do
  case "$arg" in
    -F) ;;
    --) ;;
    *) target="$arg" ;;
  esac
done
if [ -n "$target" ]; then
  rm -rf -- "$target"
fi
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write trash: %v", err)
	}
	return path
}
