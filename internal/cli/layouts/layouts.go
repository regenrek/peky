package layouts

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/layout"
)

// Register registers layout handlers.
func Register(reg *root.Registry) {
	reg.Register("layouts.list", runList)
	reg.Register("layouts.export", runExport)
}

func runList(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("layouts.list", ctx.Deps.Version)
	layouts, err := loadLayoutsForList(ctx)
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return writeLayoutsJSON(ctx, meta, layouts)
	}
	return writeLayoutsText(ctx, layouts)
}

func loadLayoutsForList(ctx root.CommandContext) ([]layout.LayoutInfo, error) {
	loader, err := layout.NewLoader()
	if err != nil {
		return nil, err
	}
	cwd, err := root.ResolveWorkDir(ctx)
	if err != nil {
		return nil, err
	}
	loader.SetProjectDir(cwd)
	if err := loader.LoadAll(); err != nil {
		return nil, err
	}
	return loader.ListLayouts(), nil
}

func writeLayoutsJSON(ctx root.CommandContext, meta output.Meta, layouts []layout.LayoutInfo) error {
	items := make([]output.LayoutSummary, 0, len(layouts))
	for _, l := range layouts {
		items = append(items, output.LayoutSummary{
			Name:   l.Name,
			Source: l.Source,
			Path:   l.Path,
		})
	}
	return output.WriteSuccess(ctx.Out, meta, output.LayoutList{Layouts: items, Total: len(items)})
}

func writeLayoutsText(ctx root.CommandContext, layouts []layout.LayoutInfo) error {
	if len(layouts) == 0 {
		_, err := fmt.Fprintln(ctx.Out, "No layouts found.")
		return err
	}
	if err := writeLayoutsHeader(ctx); err != nil {
		return err
	}
	if err := writeLayoutsTable(ctx, layouts); err != nil {
		return err
	}
	_, err := fmt.Fprintf(ctx.Out, "Use '%s layouts export <name>' to view layout YAML\n", identity.CLIName)
	return err
}

func writeLayoutsHeader(ctx root.CommandContext) error {
	if _, err := fmt.Fprintln(ctx.Out, "🎩 Available Layouts"); err != nil {
		return err
	}
	_, err := fmt.Fprintln(ctx.Out)
	return err
}

func writeLayoutsTable(ctx root.CommandContext, layouts []layout.LayoutInfo) error {
	w := tabwriter.NewWriter(ctx.Out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "NAME\tSOURCE\tDESCRIPTION"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "----\t------\t-----------"); err != nil {
		return err
	}
	for _, l := range layouts {
		source := formatLayoutSource(l.Source)
		desc := truncateLayoutDescription(l.Description)
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", l.Name, source, desc); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(ctx.Out)
	return err
}

func formatLayoutSource(source string) string {
	switch source {
	case "builtin":
		return "📦 builtin"
	case "global":
		return "🏠 global"
	case "project":
		return "📁 project"
	default:
		return source
	}
}

func truncateLayoutDescription(desc string) string {
	if len(desc) > 50 {
		return desc[:47] + "..."
	}
	return desc
}

func runExport(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("layouts.export", ctx.Deps.Version)
	name := strings.TrimSpace(ctx.Cmd.StringArg("name"))
	if name == "" {
		return fmt.Errorf("layout name is required")
	}
	loader, err := layout.NewLoader()
	if err != nil {
		return err
	}
	if err := loader.LoadAll(); err != nil {
		return err
	}
	yaml, err := loader.ExportLayout(name)
	if err != nil {
		return fmt.Errorf("layout %q not found", name)
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.LayoutExport{Name: name, Content: yaml})
	}
	if _, err := fmt.Fprintf(ctx.Out, "# Peaky Panes Layout: %s\n", name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(ctx.Out, "# Save as %s in your project root\n", identity.ProjectConfigFileYML); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(ctx.Out, "# session: your-session-name  # uncomment to set session name"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(ctx.Out); err != nil {
		return err
	}
	if _, err := fmt.Fprint(ctx.Out, yaml); err != nil {
		return err
	}
	return nil
}
