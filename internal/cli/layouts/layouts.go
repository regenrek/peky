package layouts

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
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
	loader, err := layout.NewLoader()
	if err != nil {
		return err
	}
	cwd, _ := os.Getwd()
	loader.SetProjectDir(cwd)
	if err := loader.LoadAll(); err != nil {
		return err
	}
	layouts := loader.ListLayouts()
	if ctx.JSON {
		items := make([]output.LayoutSummary, 0, len(layouts))
		for _, l := range layouts {
			items = append(items, output.LayoutSummary{
				Name:   l.Name,
				Source: l.Source,
				Path:   l.Path,
			})
		}
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.LayoutList{Layouts: items, Total: len(items)})
	}
	if len(layouts) == 0 {
		if _, err := fmt.Fprintln(ctx.Out, "No layouts found."); err != nil {
			return err
		}
		return nil
	}
	if _, err := fmt.Fprintln(ctx.Out, "🎩 Available Layouts"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(ctx.Out); err != nil {
		return err
	}

	w := tabwriter.NewWriter(ctx.Out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "NAME\tSOURCE\tDESCRIPTION"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "----\t------\t-----------"); err != nil {
		return err
	}
	for _, l := range layouts {
		source := l.Source
		switch source {
		case "builtin":
			source = "📦 builtin"
		case "global":
			source = "🏠 global"
		case "project":
			source = "📁 project"
		}
		desc := l.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", l.Name, source, desc); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(ctx.Out); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(ctx.Out, "Use 'peakypanes layouts export <name>' to view layout YAML"); err != nil {
		return err
	}
	return nil
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
	if _, err := fmt.Fprintln(ctx.Out, "# Save as .peakypanes.yml in your project root"); err != nil {
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
