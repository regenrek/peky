package contextpack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/cli/transform"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

// Register registers context pack handlers.
func Register(reg *root.Registry) {
	reg.Register("context.pack", runPack)
}

func runPack(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("context.pack", ctx.Deps.Version)
	include := normalizeIncludes(ctx.Cmd.StringSlice("include"))
	maxBytes := ctx.Cmd.Int("max-bytes")
	pack, errors, err := buildContextPack(ctx, include, maxBytes)
	if err != nil {
		return err
	}
	if include["errors"] && len(errors) > 0 {
		pack.Errors = errors
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, struct {
			Pack output.ContextPack `json:"pack"`
		}{Pack: pack})
	}
	return writeContextPackText(ctx, pack)
}

func buildContextPack(ctx root.CommandContext, include map[string]bool, maxBytes int) (output.ContextPack, []string, error) {
	pack := output.ContextPack{MaxBytes: maxBytes}
	errors := []string{}
	if includeSnapshot(include) {
		snapshot, err := loadContextSnapshot(ctx)
		if err != nil {
			return output.ContextPack{}, nil, err
		}
		pack.Snapshot = snapshot
	}
	if include["git"] {
		git, err := gitContext(ctx)
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			pack.Git = git
		}
	}
	if maxBytes > 0 {
		pack = shrinkToFit(pack)
	}
	return pack, errors, nil
}

func includeSnapshot(include map[string]bool) bool {
	return include["panes"] || include["snapshot"]
}

func loadContextSnapshot(ctx root.CommandContext) (*output.Snapshot, error) {
	client, cleanup, err := connect(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	resp, err := client.SnapshotState(ctxTimeout, 0)
	cancel()
	if err != nil {
		return nil, err
	}
	ws, err := transform.LoadWorkspace()
	if err != nil {
		return nil, err
	}
	snapshot := transform.BuildSnapshot(resp.Sessions, ws, resp.FocusedSession, resp.FocusedPaneID)
	return &snapshot, nil
}

func writeContextPackText(ctx root.CommandContext, pack output.ContextPack) error {
	if pack.Snapshot != nil {
		if _, err := fmt.Fprintf(ctx.Out, "snapshot projects: %d\n", len(pack.Snapshot.Projects)); err != nil {
			return err
		}
	}
	if pack.Git != nil {
		if _, err := fmt.Fprintf(ctx.Out, "git: %s\n", pack.Git.Root); err != nil {
			return err
		}
	}
	if len(pack.Errors) > 0 {
		if _, err := fmt.Fprintln(ctx.Out, strings.Join(pack.Errors, "\n")); err != nil {
			return err
		}
	}
	return nil
}

func normalizeIncludes(values []string) map[string]bool {
	out := map[string]bool{"panes": true, "snapshot": true, "git": true, "errors": true}
	if len(values) == 0 {
		return out
	}
	out = map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" {
			continue
		}
		out[value] = true
	}
	return out
}

func shrinkToFit(pack output.ContextPack) output.ContextPack {
	encoded, ok := encodePack(pack)
	if !ok || pack.MaxBytes <= 0 {
		return pack
	}
	if len(encoded) <= pack.MaxBytes {
		return pack
	}
	pack.Truncated = true
	// drop errors
	if len(pack.Errors) > 0 {
		pack.Errors = nil
		if fits(pack) {
			return pack
		}
	}
	// drop git
	if pack.Git != nil {
		pack.Git = nil
		if fits(pack) {
			return pack
		}
	}
	// drop snapshot
	if pack.Snapshot != nil {
		pack.Snapshot = nil
		if fits(pack) {
			return pack
		}
	}
	return pack
}

func fits(pack output.ContextPack) bool {
	encoded, ok := encodePack(pack)
	if !ok {
		return false
	}
	return len(encoded) <= pack.MaxBytes
}

func encodePack(pack output.ContextPack) ([]byte, bool) {
	payload, err := json.Marshal(pack)
	if err != nil {
		return nil, false
	}
	return payload, true
}

func gitContext(ctx root.CommandContext) (*output.GitContext, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, 2*time.Second)
	defer cancel()
	root, err := gitCmd(ctxTimeout, cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("git root: %w", err)
	}
	branch, _ := gitCmd(ctxTimeout, cwd, "rev-parse", "--abbrev-ref", "HEAD")
	head, _ := gitCmd(ctxTimeout, cwd, "rev-parse", "HEAD")
	dirtyOut, _ := gitCmd(ctxTimeout, cwd, "status", "--porcelain")
	ahead, behind := gitAheadBehind(ctxTimeout, cwd)
	return &output.GitContext{
		Root:   root,
		Branch: branch,
		Head:   head,
		Dirty:  strings.TrimSpace(dirtyOut) != "",
		Ahead:  ahead,
		Behind: behind,
	}, nil
}

func gitAheadBehind(ctx context.Context, dir string) (int, int) {
	out, err := gitCmd(ctx, dir, "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	if err != nil {
		return 0, 0
	}
	parts := strings.Fields(out)
	if len(parts) != 2 {
		return 0, 0
	}
	return atoi(parts[0]), atoi(parts[1])
}

func atoi(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func gitCmd(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(out.String()), nil
}

func connect(ctx root.CommandContext) (*sessiond.Client, func(), error) {
	connect := ctx.Deps.Connect
	if connect == nil {
		return nil, func() {}, fmt.Errorf("daemon connection not configured")
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	client, err := connect(ctxTimeout, ctx.Deps.Version)
	if err != nil {
		cancel()
		return nil, func() {}, err
	}
	cleanup := func() {
		cancel()
		_ = client.Close()
	}
	return client, cleanup, nil
}

func commandTimeout(ctx root.CommandContext) time.Duration {
	if ctx.Cmd.IsSet("timeout") {
		return ctx.Cmd.Duration("timeout")
	}
	return 10 * time.Second
}
