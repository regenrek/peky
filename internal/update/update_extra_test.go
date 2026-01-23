package update

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/runenv"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type execCall struct {
	name string
	args []string
}

type execRecorder struct {
	calls []execCall
	fail  bool
}

func (r *execRecorder) cmd(ctx context.Context, name string, args ...string) *exec.Cmd {
	r.calls = append(r.calls, execCall{name: name, args: append([]string(nil), args...)})
	if r.fail {
		return exec.CommandContext(ctx, "sh", "-c", "echo boom && exit 1")
	}
	return exec.CommandContext(ctx, "sh", "-c", "exit 0")
}

func TestDetectInstallChannels(t *testing.T) {
	ctx := context.Background()

	spec, err := DetectInstall(ctx, "/usr/local/Cellar/peakypanes/1.2.3/bin/peky")
	if err != nil {
		t.Fatalf("DetectInstall homebrew error: %v", err)
	}
	if spec.Channel != ChannelHomebrew {
		t.Fatalf("expected homebrew channel, got %s", spec.Channel)
	}

	spec, err = DetectInstall(ctx, "/tmp/app/node_modules/peakypanes/bin/peky")
	if err != nil {
		t.Fatalf("DetectInstall npm local error: %v", err)
	}
	if spec.Channel != ChannelNPMLocal {
		t.Fatalf("expected npm local channel, got %s", spec.Channel)
	}
	if spec.NPMRoot != filepath.FromSlash("/tmp/app") {
		t.Fatalf("expected npm root /tmp/app, got %q", spec.NPMRoot)
	}

	spec, err = DetectInstall(ctx, "/usr/local/lib/node_modules/peakypanes/bin/peky")
	if err != nil {
		t.Fatalf("DetectInstall npm global error: %v", err)
	}
	if spec.Channel != ChannelNPMGlobal {
		t.Fatalf("expected npm global channel, got %s", spec.Channel)
	}

	tmp := t.TempDir()
	gitRoot := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(filepath.Join(gitRoot, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.Mkdir(filepath.Join(gitRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	exe := filepath.Join(gitRoot, "nested", "peky")
	spec, err = DetectInstall(ctx, exe)
	if err != nil {
		t.Fatalf("DetectInstall git error: %v", err)
	}
	if spec.Channel != ChannelGit {
		t.Fatalf("expected git channel, got %s", spec.Channel)
	}
	if spec.GitRoot != gitRoot {
		t.Fatalf("expected git root %q, got %q", gitRoot, spec.GitRoot)
	}
}

func TestDetectInstallEmptyPath(t *testing.T) {
	_, err := DetectInstall(context.Background(), " ")
	if err == nil {
		t.Fatalf("expected error for empty path")
	}
}

func TestFindGitRoot(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")
	child := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: ."), 0o644); err != nil {
		t.Fatalf("write .git: %v", err)
	}
	found, ok := findGitRoot(context.Background(), child)
	if !ok {
		t.Fatalf("expected git root")
	}
	if found != root {
		t.Fatalf("expected git root %q, got %q", root, found)
	}
}

func TestUpdateCommand(t *testing.T) {
	cases := map[Channel]string{
		ChannelHomebrew:  "brew upgrade peakypanes",
		ChannelNPMGlobal: "npm update -g peakypanes",
		ChannelNPMLocal:  "npm update peakypanes",
		ChannelGit:       "git pull --ff-only && go install ./cmd/peakypanes ./cmd/peky",
		ChannelUnknown:   "Update manually",
	}
	for ch, want := range cases {
		if got := UpdateCommand(InstallSpec{Channel: ch}); got != want {
			t.Fatalf("UpdateCommand(%s)=%q want %q", ch, got, want)
		}
	}
}

func TestNPMClientLatestVersion(t *testing.T) {
	var ua string
	client := NPMClient{
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			ua = req.Header.Get("User-Agent")
			body := io.NopCloser(strings.NewReader(`{"version":"1.2.3"}`))
			return &http.Response{StatusCode: http.StatusOK, Body: body, Header: make(http.Header)}, nil
		})},
	}
	version, err := client.LatestVersion(context.Background())
	if err != nil {
		t.Fatalf("LatestVersion error: %v", err)
	}
	if version != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %q", version)
	}
	if ua == "" {
		t.Fatalf("expected user agent to be set")
	}
}

func TestNPMClientErrors(t *testing.T) {
	client := NPMClient{HTTPClient: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}}
	if _, err := client.LatestVersion(context.Background()); err == nil {
		t.Fatalf("expected status error")
	}

	client = NPMClient{HTTPClient: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{")), Header: make(http.Header)}, nil
	})}}
	if _, err := client.LatestVersion(context.Background()); err == nil {
		t.Fatalf("expected decode error")
	}

	client = NPMClient{HTTPClient: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"version":""}`)), Header: make(http.Header)}, nil
	})}}
	if _, err := client.LatestVersion(context.Background()); err == nil {
		t.Fatalf("expected missing version error")
	}
}

func TestInstallerInstallAndErrors(t *testing.T) {
	recorder := &execRecorder{}
	installer := Installer{execCommand: recorder.cmd}
	if err := installer.Install(context.Background(), InstallSpec{Channel: ChannelHomebrew}); err != nil {
		t.Fatalf("install homebrew error: %v", err)
	}
	if len(recorder.calls) == 0 || recorder.calls[0].name != "brew" {
		t.Fatalf("expected brew command call")
	}

	tmp := t.TempDir()
	recorder.calls = nil
	if err := installer.Install(context.Background(), InstallSpec{Channel: ChannelNPMLocal, NPMRoot: tmp}); err != nil {
		t.Fatalf("install npm local error: %v", err)
	}
	if len(recorder.calls) == 0 || recorder.calls[0].name != "npm" {
		t.Fatalf("expected npm command call")
	}

	recorder.calls = nil
	if err := installer.Install(context.Background(), InstallSpec{Channel: ChannelGit, GitRoot: tmp}); err != nil {
		t.Fatalf("install git error: %v", err)
	}
	if len(recorder.calls) < 2 || recorder.calls[0].name != "git" || recorder.calls[1].name != "go" {
		t.Fatalf("expected git + go commands")
	}

	if err := installer.Install(context.Background(), InstallSpec{Channel: ChannelUnknown}); err == nil {
		t.Fatalf("expected unsupported channel error")
	}

	recorder = &execRecorder{fail: true}
	installer = Installer{execCommand: recorder.cmd}
	if err := installer.Install(context.Background(), InstallSpec{Channel: ChannelHomebrew}); err == nil {
		t.Fatalf("expected command error")
	}
}

func TestCleanRootAndRunCommandErrors(t *testing.T) {
	if _, err := cleanRoot(""); err == nil {
		t.Fatalf("expected empty root error")
	}
	if _, err := cleanRoot("relative/path"); err == nil {
		t.Fatalf("expected relative root error")
	}

	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := cleanRoot(file); err == nil {
		t.Fatalf("expected non-dir error")
	}

	err := runCommand(context.Background(), func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "echo boom && exit 1")
	}, "", "dummy")
	if err == nil {
		t.Fatalf("expected runCommand error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected output in error, got %v", err)
	}

	if err := runCommand(context.Background(), nil, "", "sh", "-c", "exit 0"); err != nil {
		t.Fatalf("expected runCommand success, got %v", err)
	}
}

func TestDetectInstallContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := DetectInstall(ctx, "/usr/local/Cellar/peakypanes/1.2.3/bin/peky")
	if err == nil {
		t.Fatalf("expected context error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestUpdatePolicyAndStateHelpers(t *testing.T) {
	state := State{CurrentVersion: "1.0.0", LatestVersion: "1.1.0"}
	policy := Policy{PromptCooldown: time.Hour, CheckInterval: time.Hour}
	now := time.Now()

	if !policy.ShouldShowBanner(state) {
		t.Fatalf("expected banner for available update")
	}
	if policy.ShouldShowBanner(State{CurrentVersion: "1.0.0", LatestVersion: "1.0.0", SkippedVersion: "1.0.0"}) {
		t.Fatalf("expected banner suppressed for skipped update")
	}
	if !policy.ShouldCheck(0, now) {
		t.Fatalf("expected check when never checked")
	}
	if policy.ShouldCheck(now.Add(-30*time.Minute).UnixMilli(), now) {
		t.Fatalf("expected check suppressed within interval")
	}

	state.MarkPrompted(now)
	if policy.ShouldPrompt(state, now.Add(30*time.Minute)) {
		t.Fatalf("expected prompt cooldown")
	}
	state.MarkChecked(now)
	if state.LastCheckUnixMs == 0 {
		t.Fatalf("expected last check updated")
	}
}

func TestDefaultStatePathAndStore(t *testing.T) {
	t.Setenv(runenv.FreshConfigEnv, "1")
	if _, err := DefaultStatePath(); err != ErrStateDisabled {
		t.Fatalf("expected ErrStateDisabled, got %v", err)
	}
	t.Setenv(runenv.FreshConfigEnv, "")

	configDir := t.TempDir()
	t.Setenv(runenv.ConfigDirEnv, configDir)
	path, err := DefaultStatePath()
	if err != nil {
		t.Fatalf("DefaultStatePath error: %v", err)
	}
	if !strings.HasPrefix(path, configDir) {
		t.Fatalf("expected config dir path, got %q", path)
	}

	store := FileStore{Path: filepath.Join(configDir, "update-state.json")}
	state := State{CurrentVersion: "1.0.0", LatestVersion: "1.2.0"}
	if err := store.Save(context.Background(), state); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.CurrentVersion != state.CurrentVersion || loaded.LatestVersion != state.LatestVersion {
		t.Fatalf("unexpected state: %+v", loaded)
	}

	invalid := FileStore{Path: filepath.Join(configDir, "bad.json")}
	if err := os.WriteFile(invalid.Path, []byte("{"), 0o644); err != nil {
		t.Fatalf("write invalid json: %v", err)
	}
	if _, err := invalid.Load(context.Background()); err == nil {
		t.Fatalf("expected json parse error")
	}

	relStore := FileStore{Path: "relative/path.json"}
	if _, err := relStore.Load(context.Background()); err == nil {
		t.Fatalf("expected relative path error")
	}
}

func TestNewInstallerDefaults(t *testing.T) {
	installer := NewInstaller()
	if installer.execCommand == nil {
		t.Fatalf("expected execCommand")
	}
	recorder := &execRecorder{}
	installer = Installer{execCommand: recorder.cmd}
	if err := installer.Install(nil, InstallSpec{Channel: ChannelHomebrew}); err != nil {
		t.Fatalf("expected install success with nil ctx: %v", err)
	}
}

func TestUpdateStateAvailabilityAndPromptSkips(t *testing.T) {
	if (State{CurrentVersion: "dev", LatestVersion: "1.1.0"}).UpdateAvailable() {
		t.Fatalf("expected dev version to suppress updates")
	}
	if (State{CurrentVersion: "1.0.0", LatestVersion: "bad"}).UpdateAvailable() {
		t.Fatalf("expected invalid version to suppress updates")
	}

	now := time.Now()
	policy := Policy{PromptCooldown: time.Hour}
	state := State{CurrentVersion: "1.0.0", LatestVersion: "1.1.0", SkippedVersion: "1.1.0"}
	if policy.ShouldPrompt(state, now) {
		t.Fatalf("expected skipped update to suppress prompt")
	}
	state = State{CurrentVersion: "1.0.0", LatestVersion: "1.0.0"}
	if policy.ShouldPrompt(state, now) {
		t.Fatalf("expected no update to suppress prompt")
	}
}

func TestStateMarkersNil(t *testing.T) {
	var state *State
	state.MarkPrompted(time.Now())
	state.MarkChecked(time.Now())
}

func TestDefaultStatePathHomeFallbackAndStoreContexts(t *testing.T) {
	t.Setenv(runenv.ConfigDirEnv, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := DefaultStatePath()
	if err != nil {
		t.Fatalf("DefaultStatePath error: %v", err)
	}
	want := filepath.Join(home, ".config", "peakypanes", "update-state.json")
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}

	store := FileStore{Path: path}
	if _, err := store.Load(context.Background()); err != nil {
		t.Fatalf("expected empty load, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Load(ctx); err == nil {
		t.Fatalf("expected load context error")
	}
	if err := store.Save(ctx, State{}); err == nil {
		t.Fatalf("expected save context error")
	}
}
