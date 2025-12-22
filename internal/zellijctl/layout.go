package zellijctl

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/regenrek/peakypanes/internal/layout"
)

type paneNode struct {
	Name     string
	Command  string
	Args     []string
	Cwd      string
	Size     string
	Split    string
	Children []*paneNode
}

// BuildLayout renders a zellij KDL layout for the provided configuration.
func BuildLayout(layoutCfg *layout.LayoutConfig, projectPath string) (string, error) {
	if layoutCfg == nil {
		return "", fmt.Errorf("layout config is required")
	}

	var tabs []*paneNode
	var tabNames []string

	if strings.TrimSpace(layoutCfg.Grid) != "" {
		grid, err := layout.Parse(layoutCfg.Grid)
		if err != nil {
			return "", fmt.Errorf("parse grid %q: %w", layoutCfg.Grid, err)
		}
		tabName := strings.TrimSpace(layoutCfg.Window)
		if tabName == "" {
			tabName = strings.TrimSpace(layoutCfg.Name)
		}
		if tabName == "" {
			tabName = "grid"
		}
		root := buildGridTree(grid.Rows, grid.Columns, buildGridPanes(layoutCfg, grid.Panes(), projectPath))
		tabs = append(tabs, root)
		tabNames = append(tabNames, tabName)
	} else {
		if len(layoutCfg.Windows) == 0 {
			return "", fmt.Errorf("layout has no windows defined")
		}
		for _, win := range layoutCfg.Windows {
			root := buildWindowTree(win, projectPath)
			tabs = append(tabs, root)
			tabNames = append(tabNames, strings.TrimSpace(win.Name))
		}
	}

	var sb strings.Builder
	sb.WriteString("layout {\n")
	writeDefaultTabTemplate(&sb, 1)
	for i, tab := range tabs {
		writeTab(&sb, tabNames[i], tab, 1)
	}
	sb.WriteString("}\n")
	return sb.String(), nil
}

func buildWindowTree(win layout.WindowDef, projectPath string) *paneNode {
	panes := buildPaneNodes(win.Panes, projectPath)
	if len(panes) == 0 {
		return &paneNode{}
	}
	layoutName := strings.TrimSpace(win.Layout)
	if layoutName != "" {
		return applyLayoutAlgorithm(layoutName, panes)
	}
	return buildSequentialTree(panes)
}

func buildPaneNodes(panes []layout.PaneDef, projectPath string) []*paneNode {
	out := make([]*paneNode, 0, len(panes))
	for _, pane := range panes {
		cmd, args := buildCommand(pane.Cmd)
		out = append(out, &paneNode{
			Name:    strings.TrimSpace(pane.Title),
			Command: cmd,
			Args:    args,
			Cwd:     projectPath,
			Size:    strings.TrimSpace(pane.Size),
			Split:   strings.TrimSpace(pane.Split),
		})
	}
	return out
}

func buildGridPanes(layoutCfg *layout.LayoutConfig, count int, projectPath string) []*paneNode {
	commands := resolveGridCommands(layoutCfg, count)
	titles := resolveGridTitles(layoutCfg, count)
	panes := make([]*paneNode, 0, count)
	for i := 0; i < count; i++ {
		cmd, args := buildCommand(commands[i])
		panes = append(panes, &paneNode{
			Name:    strings.TrimSpace(titles[i]),
			Command: cmd,
			Args:    args,
			Cwd:     projectPath,
		})
	}
	return panes
}

func buildSequentialTree(panes []*paneNode) *paneNode {
	if len(panes) == 0 {
		return &paneNode{}
	}
	root := panes[0]
	rootParent := (*paneNode)(nil)
	active := root
	for i := 1; i < len(panes); i++ {
		next := panes[i]
		split := parseSplitDirection(next.Split)
		parent := &paneNode{
			Split:    split,
			Children: []*paneNode{active, next},
		}
		if strings.TrimSpace(next.Size) != "" {
			next.Size = normalizePercent(next.Size)
		}
		if rootParent == nil && active == root {
			root = parent
		} else {
			replaceChild(root, active, parent)
		}
		rootParent = parent
		active = next
	}
	return root
}

func applyLayoutAlgorithm(name string, panes []*paneNode) *paneNode {
	if len(panes) == 0 {
		return &paneNode{}
	}
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case "even-horizontal":
		return buildEvenTree(panes, "Horizontal")
	case "even-vertical":
		return buildEvenTree(panes, "Vertical")
	case "main-horizontal":
		return buildMainTree(panes, "Vertical")
	case "main-vertical":
		return buildMainTree(panes, "Horizontal")
	case "tiled":
		fallthrough
	default:
		return buildTiledTree(panes)
	}
}

func buildEvenTree(panes []*paneNode, split string) *paneNode {
	if len(panes) == 1 {
		return panes[0]
	}
	sizes := evenSizes(len(panes))
	for i, size := range sizes {
		panes[i].Size = size
	}
	return &paneNode{Split: split, Children: panes}
}

func buildMainTree(panes []*paneNode, split string) *paneNode {
	if len(panes) == 1 {
		return panes[0]
	}
	main := panes[0]
	stack := panes[1:]
	main.Size = "60%"
	stackNode := buildEvenTree(stack, oppositeSplit(split))
	return &paneNode{
		Split:    split,
		Children: []*paneNode{main, stackNode},
	}
}

func buildTiledTree(panes []*paneNode) *paneNode {
	if len(panes) == 1 {
		return panes[0]
	}
	cols := int(math.Ceil(math.Sqrt(float64(len(panes)))))
	rows := int(math.Ceil(float64(len(panes)) / float64(cols)))
	return buildGridTree(rows, cols, panes)
}

func buildGridTree(rows, cols int, panes []*paneNode) *paneNode {
	if rows <= 1 && cols <= 1 {
		if len(panes) > 0 {
			return panes[0]
		}
		return &paneNode{}
	}
	rowSizes := evenSizes(rows)
	colSizes := evenSizes(cols)
	rowsNodes := make([]*paneNode, 0, rows)
	paneIndex := 0
	for r := 0; r < rows; r++ {
		rowChildren := make([]*paneNode, 0, cols)
		for c := 0; c < cols; c++ {
			if paneIndex >= len(panes) {
				rowChildren = append(rowChildren, &paneNode{})
				continue
			}
			pane := panes[paneIndex]
			pane.Size = colSizes[c]
			rowChildren = append(rowChildren, pane)
			paneIndex++
		}
		rowNode := &paneNode{
			Split:    "Horizontal",
			Size:     rowSizes[r],
			Children: rowChildren,
		}
		rowsNodes = append(rowsNodes, rowNode)
	}
	return &paneNode{
		Split:    "Vertical",
		Children: rowsNodes,
	}
}

func writeDefaultTabTemplate(sb *strings.Builder, indent int) {
	writeIndent(sb, indent)
	sb.WriteString("default_tab_template {\n")
	writeIndent(sb, indent+1)
	sb.WriteString("pane size=1 borderless=true {\n")
	writeIndent(sb, indent+2)
	sb.WriteString("plugin location=\"zellij:tab-bar\"\n")
	writeIndent(sb, indent+1)
	sb.WriteString("}\n")
	writeIndent(sb, indent+1)
	sb.WriteString("children\n")
	writeIndent(sb, indent+1)
	sb.WriteString("pane size=2 borderless=true {\n")
	writeIndent(sb, indent+2)
	sb.WriteString("plugin location=\"zellij:status-bar\"\n")
	writeIndent(sb, indent+1)
	sb.WriteString("}\n")
	writeIndent(sb, indent)
	sb.WriteString("}\n")
}

func writeTab(sb *strings.Builder, name string, root *paneNode, indent int) {
	writeIndent(sb, indent)
	sb.WriteString("tab")
	if strings.TrimSpace(name) != "" {
		sb.WriteString(" name=")
		sb.WriteString(strconv.Quote(name))
	}
	if root != nil && root.Split != "" {
		sb.WriteString(" split_direction=")
		sb.WriteString(strconv.Quote(root.Split))
	}
	if root == nil {
		sb.WriteString("\n")
		return
	}
	sb.WriteString(" {\n")
	if root.Split != "" {
		for _, child := range root.Children {
			writePane(sb, child, indent+1)
		}
	} else {
		writePane(sb, root, indent+1)
	}
	writeIndent(sb, indent)
	sb.WriteString("}\n")
}

func writePane(sb *strings.Builder, node *paneNode, indent int) {
	if node == nil {
		return
	}
	writeIndent(sb, indent)
	sb.WriteString("pane")
	if strings.TrimSpace(node.Size) != "" {
		sb.WriteString(" size=")
		sb.WriteString(strconv.Quote(normalizePercent(node.Size)))
	}
	if strings.TrimSpace(node.Split) != "" {
		sb.WriteString(" split_direction=")
		sb.WriteString(strconv.Quote(node.Split))
	}
	if strings.TrimSpace(node.Name) != "" {
		sb.WriteString(" name=")
		sb.WriteString(strconv.Quote(node.Name))
	}
	if strings.TrimSpace(node.Command) != "" {
		sb.WriteString(" command=")
		sb.WriteString(strconv.Quote(node.Command))
	}
	if strings.TrimSpace(node.Cwd) != "" {
		sb.WriteString(" cwd=")
		sb.WriteString(strconv.Quote(node.Cwd))
	}
	hasChildren := len(node.Children) > 0
	hasArgs := len(node.Args) > 0
	if !hasChildren && !hasArgs {
		sb.WriteString("\n")
		return
	}
	sb.WriteString(" {\n")
	if hasArgs {
		writeIndent(sb, indent+1)
		sb.WriteString("args")
		for _, arg := range node.Args {
			sb.WriteString(" ")
			sb.WriteString(strconv.Quote(arg))
		}
		sb.WriteString("\n")
	}
	for _, child := range node.Children {
		writePane(sb, child, indent+1)
	}
	writeIndent(sb, indent)
	sb.WriteString("}\n")
}

func writeIndent(sb *strings.Builder, indent int) {
	for i := 0; i < indent; i++ {
		sb.WriteString("    ")
	}
}

func parseSplitDirection(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "v", "vertical":
		return "Vertical"
	case "h", "horizontal", "":
		return "Horizontal"
	default:
		return "Horizontal"
	}
}

func oppositeSplit(direction string) string {
	if direction == "Horizontal" {
		return "Vertical"
	}
	return "Horizontal"
}

func normalizePercent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if strings.HasSuffix(value, "%") {
		return value
	}
	return value + "%"
}

func evenSizes(count int) []string {
	if count <= 0 {
		return nil
	}
	sizes := make([]string, count)
	base := 100 / count
	remainder := 100 - (base * count)
	for i := 0; i < count; i++ {
		size := base
		if remainder > 0 {
			size++
			remainder--
		}
		sizes[i] = fmt.Sprintf("%d%%", size)
	}
	return sizes
}

func replaceChild(root, oldChild, newChild *paneNode) bool {
	if root == nil {
		return false
	}
	for i, child := range root.Children {
		if child == oldChild {
			root.Children[i] = newChild
			return true
		}
		if replaceChild(child, oldChild, newChild) {
			return true
		}
	}
	return false
}

func buildCommand(cmd string) (string, []string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", nil
	}
	return "$SHELL", []string{"-lc", cmd}
}

func resolveGridCommands(layoutCfg *layout.LayoutConfig, count int) []string {
	commands := make([]string, 0, count)
	fallback := strings.TrimSpace(layoutCfg.Command)
	if len(layoutCfg.Commands) > 0 {
		for i := 0; i < count; i++ {
			if i < len(layoutCfg.Commands) {
				commands = append(commands, layoutCfg.Commands[i])
				continue
			}
			if fallback != "" {
				commands = append(commands, fallback)
			} else {
				commands = append(commands, "")
			}
		}
		return commands
	}
	if fallback == "" {
		for i := 0; i < count; i++ {
			commands = append(commands, "")
		}
		return commands
	}
	for i := 0; i < count; i++ {
		commands = append(commands, fallback)
	}
	return commands
}

func resolveGridTitles(layoutCfg *layout.LayoutConfig, count int) []string {
	titles := make([]string, 0, count)
	for i := 0; i < count; i++ {
		if i < len(layoutCfg.Titles) {
			titles = append(titles, layoutCfg.Titles[i])
		} else {
			titles = append(titles, "")
		}
	}
	return titles
}
