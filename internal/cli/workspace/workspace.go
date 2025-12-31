package workspace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/layout"
	ws "github.com/regenrek/peakypanes/internal/workspace"
)

// Register registers workspace handlers.
func Register(reg *root.Registry) {
	reg.Register("workspace.list", runList)
	reg.Register("workspace.open", runOpen)
	reg.Register("workspace.close", runClose)
	reg.Register("workspace.close-all", runCloseAll)
}

func runList(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("workspace.list", ctx.Deps.Version)
	cfgPath, err := layout.DefaultConfigPath()
	if err != nil {
		return err
	}
	workspace, err := ws.ListWorkspace(cfgPath)
	if err != nil {
		return err
	}
	projects := make([]output.ProjectSummary, 0, len(workspace.Projects))
	counts := projectSessionCounts(ctx, workspace)
	for _, project := range workspace.Projects {
		projects = append(projects, output.ProjectSummary{
			ID:           project.ID,
			Name:         project.Name,
			Path:         project.Path,
			Hidden:       project.Hidden,
			SessionCount: counts[project.ID],
		})
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.WorkspaceList{Projects: projects, Roots: workspace.Roots, Total: len(projects)})
	}
	for _, project := range projects {
		if _, err := fmt.Fprintln(ctx.Out, project.Name); err != nil {
			return err
		}
	}
	return nil
}

func runOpen(ctx root.CommandContext) error {
	return runMutation(ctx, "workspace.open", false)
}

func runClose(ctx root.CommandContext) error {
	return runMutation(ctx, "workspace.close", true)
}

func runMutation(ctx root.CommandContext, action string, hide bool) error {
	start := time.Now()
	meta := output.NewMeta(action, ctx.Deps.Version)
	cfgPath, err := layout.DefaultConfigPath()
	if err != nil {
		return err
	}
	workspace, err := ws.ListWorkspace(cfgPath)
	if err != nil {
		return err
	}
	ref := projectRefFromFlags(ctx)
	project, err := ws.FindProject(workspace.Projects, ref)
	if err != nil {
		return err
	}
	cfg, err := ws.LoadConfig(cfgPath)
	if err != nil {
		return err
	}
	changed := false
	if hide {
		changed, err = ws.HideProject(cfg, ws.ProjectRef{ID: project.ID, Name: project.Name, Path: project.Path})
	} else {
		changed, err = ws.UnhideProject(cfg, ws.ProjectRef{ID: project.ID, Name: project.Name, Path: project.Path})
	}
	if err != nil {
		return err
	}
	if changed {
		if err := ws.SaveConfig(cfgPath, cfg); err != nil {
			return err
		}
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  action,
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "project", ID: project.ID}},
			Details: map[string]any{"changed": changed},
		})
	}
	return nil
}

func runCloseAll(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("workspace.close-all", ctx.Deps.Version)
	cfgPath, err := layout.DefaultConfigPath()
	if err != nil {
		return err
	}
	workspace, err := ws.ListWorkspace(cfgPath)
	if err != nil {
		return err
	}
	cfg, err := ws.LoadConfig(cfgPath)
	if err != nil {
		return err
	}
	count, err := ws.HideAllProjects(cfg, workspace.Projects)
	if err != nil {
		return err
	}
	if count > 0 {
		if err := ws.SaveConfig(cfgPath, cfg); err != nil {
			return err
		}
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "workspace.close-all",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "workspace", ID: "default"}},
			Details: map[string]any{"count": count},
		})
	}
	return nil
}

func projectRefFromFlags(ctx root.CommandContext) ws.ProjectRef {
	return ws.ProjectRef{
		ID:   strings.TrimSpace(ctx.Cmd.String("id")),
		Name: strings.TrimSpace(ctx.Cmd.String("name")),
		Path: strings.TrimSpace(ctx.Cmd.String("path")),
	}
}

func projectSessionCounts(ctx root.CommandContext, workspace ws.Workspace) map[string]int {
	counts := make(map[string]int)
	connect := ctx.Deps.Connect
	if connect == nil {
		return counts
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, 5*time.Second)
	defer cancel()
	client, err := connect(ctxTimeout, ctx.Deps.Version)
	if err != nil {
		return counts
	}
	defer func() { _ = client.Close() }()
	resp, err := client.SnapshotState(ctxTimeout, 0)
	if err != nil {
		return counts
	}
	byPath := make(map[string]string)
	for _, project := range workspace.Projects {
		key := ws.NormalizeProjectPath(project.Path)
		if key != "" {
			byPath[key] = project.ID
		}
	}
	for _, session := range resp.Sessions {
		key := ws.NormalizeProjectPath(session.Path)
		id := byPath[key]
		if id == "" {
			continue
		}
		counts[id]++
	}
	return counts
}
