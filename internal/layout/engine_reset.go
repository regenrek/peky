package layout

import "fmt"

func rectForNode(tree *Tree, target *Node) (Rect, bool) {
	if tree == nil || tree.Root == nil || target == nil {
		return Rect{}, false
	}
	return rectForNodeRecursive(tree.Root, Rect{X: 0, Y: 0, W: LayoutBaseSize, H: LayoutBaseSize}, target)
}

func rectForNodeRecursive(node *Node, rect Rect, target *Node) (Rect, bool) {
	if node == nil {
		return Rect{}, false
	}
	if node == target {
		return rect, true
	}
	if node.IsLeaf() {
		return Rect{}, false
	}
	if node.Axis == AxisHorizontal {
		sizes := normalizeSizes(node.Children, rect.W)
		x := rect.X
		for i, child := range node.Children {
			childRect := Rect{X: x, Y: rect.Y, W: sizes[i], H: rect.H}
			if found, ok := rectForNodeRecursive(child, childRect, target); ok {
				return found, true
			}
			x += sizes[i]
		}
		return Rect{}, false
	}
	sizes := normalizeSizes(node.Children, rect.H)
	y := rect.Y
	for i, child := range node.Children {
		childRect := Rect{X: rect.X, Y: y, W: rect.W, H: sizes[i]}
		if found, ok := rectForNodeRecursive(child, childRect, target); ok {
			return found, true
		}
		y += sizes[i]
	}
	return Rect{}, false
}

func resetNodeSizes(node *Node, rect Rect, constraints Constraints) error {
	if node == nil || node.IsLeaf() {
		return nil
	}
	count := len(node.Children)
	if count == 0 {
		return nil
	}
	axisSize := rect.W
	minSize := constraints.MinWidth
	if node.Axis == AxisVertical {
		axisSize = rect.H
		minSize = constraints.MinHeight
	}
	if axisSize < minSize*count {
		return fmt.Errorf("layout: reset sizes exceeds min size %d", minSize)
	}
	base := axisSize / count
	remainder := axisSize % count
	for i, child := range node.Children {
		size := base
		if i == count-1 {
			size += remainder
		}
		child.Size = size
	}
	if node.Axis == AxisHorizontal {
		x := rect.X
		for _, child := range node.Children {
			childRect := Rect{X: x, Y: rect.Y, W: child.Size, H: rect.H}
			if err := resetNodeSizes(child, childRect, constraints); err != nil {
				return err
			}
			x += child.Size
		}
		return nil
	}
	y := rect.Y
	for _, child := range node.Children {
		childRect := Rect{X: rect.X, Y: y, W: rect.W, H: child.Size}
		if err := resetNodeSizes(child, childRect, constraints); err != nil {
			return err
		}
		y += child.Size
	}
	return nil
}
