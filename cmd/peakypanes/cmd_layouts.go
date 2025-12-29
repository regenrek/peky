package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/regenrek/peakypanes/internal/layout"
)

func runLayouts(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "export":
			if len(args) < 2 {
				fatal("usage: peakypanes layouts export <name>")
			}
			exportLayout(args[1])
			return
		case "-h", "--help":
			fmt.Print(layoutsHelpText)
			return
		}
	}

	listLayouts()
}

func listLayouts() {
	loader, err := layout.NewLoader()
	if err != nil {
		fatal("failed to create loader: %v", err)
	}

	cwd, _ := os.Getwd()
	loader.SetProjectDir(cwd)

	if err := loader.LoadAll(); err != nil {
		fatal("failed to load layouts: %v", err)
	}

	layouts := loader.ListLayouts()
	if len(layouts) == 0 {
		fmt.Println("No layouts found.")
		return
	}

	fmt.Println("🎩 Available Layouts")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "NAME\tSOURCE\tDESCRIPTION"); err != nil {
		fatal("failed to write layout table header: %v", err)
	}
	if _, err := fmt.Fprintln(w, "----\t------\t-----------"); err != nil {
		fatal("failed to write layout table divider: %v", err)
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
			fatal("failed to write layout row: %v", err)
		}
	}
	if err := w.Flush(); err != nil {
		fatal("failed to flush layout table: %v", err)
	}

	fmt.Println()
	fmt.Println("Use 'peakypanes layouts export <name>' to view layout YAML")
}

func exportLayout(name string) {
	loader, err := layout.NewLoader()
	if err != nil {
		fatal("failed to create loader: %v", err)
	}

	if err := loader.LoadAll(); err != nil {
		fatal("failed to load layouts: %v", err)
	}

	yaml, err := loader.ExportLayout(name)
	if err != nil {
		fatal("layout %q not found", name)
	}

	fmt.Printf("# Peaky Panes Layout: %s\n", name)
	fmt.Printf("# Save as .peakypanes.yml in your project root\n")
	fmt.Printf("# session: your-session-name  # uncomment to set session name\n\n")
	fmt.Print(yaml)
}
