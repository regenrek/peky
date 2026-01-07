package layout

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func BuildTree(layoutCfg *LayoutConfig, paneIDs []string) (*Tree, error) {
	if layoutCfg == nil {
		return nil, errors.New("layout: config is nil")
	}
	if strings.TrimSpace(layoutCfg.Grid) != "" {
		return buildGridTree(layoutCfg, paneIDs)
	}
	if len(layoutCfg.Panes) == 0 {
		return nil, errors.New("layout: no panes defined")
	}
	return buildSplitTree(layoutCfg, paneIDs)
}

func buildGridTree(layoutCfg *LayoutConfig, paneIDs []string) (*Tree, error) {
	grid, err := Parse(layoutCfg.Grid)
	if err != nil {
		return nil, fmt.Errorf("layout: parse grid %q: %w", layoutCfg.Grid, err)
	}
	count := grid.Panes()
	if count <= 0 {
		return nil, errors.New("layout: grid has no panes")
	}
	if len(paneIDs) < count {
		return nil, fmt.Errorf("layout: need %d pane ids, got %d", count, len(paneIDs))
	}
	panes := make(map[string]*Node, count)
	tree := NewTree(nil, panes)

	rows := grid.Rows
	cols := grid.Columns
	idx := 0

	buildRow := func(height int, ids []string) *Node {
		if cols == 1 {
			leaf := &Node{ID: tree.nextNodeID(), PaneID: ids[0], Size: LayoutBaseSize}
			panes[ids[0]] = leaf
			return leaf
		}
		row := &Node{ID: tree.nextNodeID(), Axis: AxisHorizontal, Size: height}
		widths := splitSizes(LayoutBaseSize, cols)
		for c := 0; c < cols; c++ {
			leaf := &Node{ID: tree.nextNodeID(), PaneID: ids[c], Size: widths[c], Parent: row}
			row.Children = append(row.Children, leaf)
			panes[ids[c]] = leaf
		}
		return row
	}

	if rows == 1 {
		rowIDs := paneIDs[idx : idx+cols]
		root := buildRow(LayoutBaseSize, rowIDs)
		root.Parent = nil
		tree.Root = root
		return tree, nil
	}

	root := &Node{ID: tree.nextNodeID(), Axis: AxisVertical, Size: LayoutBaseSize}
	heights := splitSizes(LayoutBaseSize, rows)
	for r := 0; r < rows; r++ {
		rowIDs := paneIDs[idx : idx+cols]
		idx += cols
		row := buildRow(heights[r], rowIDs)
		row.Parent = root
		row.Size = heights[r]
		root.Children = append(root.Children, row)
	}
	tree.Root = root
	return tree, nil
}

func buildSplitTree(layoutCfg *LayoutConfig, paneIDs []string) (*Tree, error) {
	count := len(layoutCfg.Panes)
	if count <= 0 {
		return nil, errors.New("layout: no panes defined")
	}
	if len(paneIDs) < count {
		return nil, fmt.Errorf("layout: need %d pane ids, got %d", count, len(paneIDs))
	}
	panes := make(map[string]*Node, count)
	tree := NewTree(nil, panes)

	root := &Node{ID: tree.nextNodeID(), PaneID: paneIDs[0], Size: LayoutBaseSize}
	panes[paneIDs[0]] = root
	tree.Root = root
	active := root

	for i := 1; i < count; i++ {
		paneDef := layoutCfg.Panes[i]
		axis := splitAxisFromDef(paneDef)
		percent := parsePercent(paneDef.Size)
		newLeaf := &Node{ID: tree.nextNodeID(), PaneID: paneIDs[i]}
		if err := splitLeaf(tree, active, newLeaf, axis, percent, 1); err != nil {
			return nil, err
		}
		panes[paneIDs[i]] = newLeaf
	}

	return tree, nil
}

func splitAxisFromDef(def PaneDef) Axis {
	if strings.EqualFold(def.Split, "vertical") || strings.EqualFold(def.Split, "v") {
		return AxisVertical
	}
	return AxisHorizontal
}

func parsePercent(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	raw = strings.TrimSuffix(raw, "%")
	pct, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return pct
}

func splitSizes(total, parts int) []int {
	if parts <= 0 {
		return nil
	}
	base := total / parts
	remainder := total % parts
	out := make([]int, parts)
	for i := 0; i < parts; i++ {
		out[i] = base
	}
	if remainder > 0 {
		out[parts-1] += remainder
	}
	return out
}
