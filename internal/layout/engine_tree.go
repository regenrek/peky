package layout

import "sort"

func (t *Tree) Leaf(paneID string) *Node {
	if t == nil || t.Panes == nil {
		return nil
	}
	return t.Panes[paneID]
}

func (t *Tree) Rects() map[string]Rect {
	out := make(map[string]Rect)
	if t == nil || t.Root == nil {
		return out
	}
	root := Rect{X: 0, Y: 0, W: LayoutBaseSize, H: LayoutBaseSize}
	t.rectsForNode(t.Root, root, out)
	return out
}

func (t *Tree) RectForNode(target *Node) (Rect, bool) {
	if t == nil || t.Root == nil || target == nil {
		return Rect{}, false
	}
	path := make([]*Node, 0, 8)
	for node := target; node != nil; node = node.Parent {
		path = append(path, node)
	}
	rect := Rect{X: 0, Y: 0, W: LayoutBaseSize, H: LayoutBaseSize}
	for i := len(path) - 1; i > 0; i-- {
		parent := path[i]
		child := path[i-1]
		if parent == nil || child == nil || len(parent.Children) == 0 {
			return Rect{}, false
		}
		idx := childIndex(parent, child)
		if idx < 0 {
			return Rect{}, false
		}
		if parent.Axis == AxisHorizontal {
			sizes := normalizeSizes(parent.Children, rect.W)
			x := rect.X
			for j := 0; j < idx; j++ {
				x += sizes[j]
			}
			rect = Rect{X: x, Y: rect.Y, W: sizes[idx], H: rect.H}
			continue
		}
		sizes := normalizeSizes(parent.Children, rect.H)
		y := rect.Y
		for j := 0; j < idx; j++ {
			y += sizes[j]
		}
		rect = Rect{X: rect.X, Y: y, W: rect.W, H: sizes[idx]}
	}
	return rect, true
}

func (t *Tree) ViewRects() map[string]Rect {
	if t == nil || t.ZoomedPaneID == "" {
		return t.Rects()
	}
	if leaf := t.Leaf(t.ZoomedPaneID); leaf != nil {
		return map[string]Rect{leaf.PaneID: {X: 0, Y: 0, W: LayoutBaseSize, H: LayoutBaseSize}}
	}
	return t.Rects()
}

func (t *Tree) rectsForNode(node *Node, rect Rect, out map[string]Rect) {
	if node == nil || rect.Empty() {
		return
	}
	if node.IsLeaf() {
		if node.PaneID != "" {
			out[node.PaneID] = rect
		}
		return
	}
	count := len(node.Children)
	if count == 0 {
		return
	}
	if node.Axis == AxisHorizontal {
		sizes := normalizeSizes(node.Children, rect.W)
		x := rect.X
		for i, child := range node.Children {
			w := sizes[i]
			childRect := Rect{X: x, Y: rect.Y, W: w, H: rect.H}
			t.rectsForNode(child, childRect, out)
			x += w
		}
		return
	}

	sizes := normalizeSizes(node.Children, rect.H)
	y := rect.Y
	for i, child := range node.Children {
		h := sizes[i]
		childRect := Rect{X: rect.X, Y: y, W: rect.W, H: h}
		t.rectsForNode(child, childRect, out)
		y += h
	}
}

func normalizeSizes(children []*Node, total int) []int {
	count := len(children)
	sizes := make([]int, count)
	if count == 0 {
		return sizes
	}
	sum := 0
	for i, child := range children {
		if child == nil {
			continue
		}
		if child.Size < 0 {
			child.Size = 0
		}
		sum += child.Size
		sizes[i] = child.Size
	}
	if sum <= 0 {
		base := total / count
		remainder := total % count
		for i := range sizes {
			sizes[i] = base
		}
		if remainder > 0 {
			sizes[count-1] += remainder
		}
		return sizes
	}

	acc := 0
	for i := range sizes {
		sizes[i] = sizes[i] * total / sum
		acc += sizes[i]
	}
	if acc != total && count > 0 {
		sizes[count-1] += total - acc
	}
	return sizes
}

func (t *Tree) Clone() *Tree {
	if t == nil {
		return nil
	}
	clone := &Tree{ZoomedPaneID: t.ZoomedPaneID, nextNodeIndex: t.nextNodeIndex}
	if t.Root == nil {
		return clone
	}
	paneMap := make(map[string]*Node, len(t.Panes))
	clone.Root = cloneNode(t.Root, nil, paneMap)
	clone.Panes = paneMap
	return clone
}

func (t *Tree) PaneIDs() []string {
	if t == nil || len(t.Panes) == 0 {
		return nil
	}
	ids := make([]string, 0, len(t.Panes))
	for id := range t.Panes {
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func cloneNode(node *Node, parent *Node, paneMap map[string]*Node) *Node {
	if node == nil {
		return nil
	}
	copy := &Node{
		ID:     node.ID,
		PaneID: node.PaneID,
		Axis:   node.Axis,
		Size:   node.Size,
		Parent: parent,
	}
	if node.PaneID != "" {
		paneMap[node.PaneID] = copy
	}
	if len(node.Children) > 0 {
		copy.Children = make([]*Node, 0, len(node.Children))
		for _, child := range node.Children {
			cloneChild := cloneNode(child, copy, paneMap)
			copy.Children = append(copy.Children, cloneChild)
		}
	}
	return copy
}
