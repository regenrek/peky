package icons

import (
	"os"
	"strings"
)

type Size string

const (
	SizeSmall  Size = "small"
	SizeMedium Size = "medium"
	SizeLarge  Size = "large"
)

type Variants struct {
	Small  string
	Medium string
	Large  string
}

func (v Variants) BySize(size Size) string {
	switch size {
	case SizeSmall:
		if v.Small != "" {
			return v.Small
		}
	case SizeLarge:
		if v.Large != "" {
			return v.Large
		}
	}
	if v.Medium != "" {
		return v.Medium
	}
	if v.Small != "" {
		return v.Small
	}
	return v.Large
}

type IconSet struct {
	Caret       Variants
	PaneDot     Variants
	WindowLabel string
	PaneLabel   string
	Spinner     []string
}

var Unicode = IconSet{
	Caret: Variants{
		Small:  "▹",
		Medium: "▸",
		Large:  "▶",
	},
	PaneDot: Variants{
		Small:  "·",
		Medium: "•",
		Large:  "●",
	},
	WindowLabel: "win",
	PaneLabel:   "pane",
	Spinner:     []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
}

var ASCII = IconSet{
	Caret: Variants{
		Small:  ">",
		Medium: ">",
		Large:  ">",
	},
	PaneDot: Variants{
		Small:  ".",
		Medium: "*",
		Large:  "o",
	},
	WindowLabel: "win",
	PaneLabel:   "pane",
	Spinner:     []string{"-", "\\", "|", "/"},
}

func Active() IconSet {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PEAKYPANES_ICON_SET"))) {
	case "ascii":
		return ASCII
	default:
		return Unicode
	}
}

func ActiveSize() Size {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PEAKYPANES_ICON_SIZE"))) {
	case "small", "sm", "s":
		return SizeSmall
	case "large", "lg", "l":
		return SizeLarge
	default:
		return SizeMedium
	}
}
