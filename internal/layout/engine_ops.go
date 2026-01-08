package layout

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type OpKind string

const (
	OpResize     OpKind = "resize"
	OpSplit      OpKind = "split"
	OpClose      OpKind = "close"
	OpResetSizes OpKind = "reset_sizes"
	OpSwap       OpKind = "swap"
	OpZoom       OpKind = "zoom"
)

type Op interface {
	Kind() OpKind
}

type ResizeEdge int

const (
	ResizeEdgeLeft ResizeEdge = iota
	ResizeEdgeRight
	ResizeEdgeUp
	ResizeEdgeDown
)

func (e ResizeEdge) Axis() Axis {
	switch e {
	case ResizeEdgeLeft, ResizeEdgeRight:
		return AxisHorizontal
	case ResizeEdgeUp, ResizeEdgeDown:
		return AxisVertical
	default:
		return AxisHorizontal
	}
}

type ResizeOp struct {
	PaneID    string
	Edge      ResizeEdge
	Delta     int
	Snap      bool
	SnapState SnapState
}

func (ResizeOp) Kind() OpKind { return OpResize }

type SplitOp struct {
	PaneID    string
	NewPaneID string
	Axis      Axis
	Percent   int
}

func (SplitOp) Kind() OpKind { return OpSplit }

type CloseOp struct {
	PaneID string
}

func (CloseOp) Kind() OpKind { return OpClose }

type ResetSizesOp struct {
	PaneID string
}

func (ResetSizesOp) Kind() OpKind { return OpResetSizes }

type SwapOp struct {
	PaneA string
	PaneB string
}

func (SwapOp) Kind() OpKind { return OpSwap }

type ZoomOp struct {
	PaneID string
	Toggle bool
}

func (ZoomOp) Kind() OpKind { return OpZoom }

type ApplyResult struct {
	Changed   bool
	Snapped   bool
	SnapState SnapState
	Affected  []string
}

type Engine struct {
	Tree        *Tree
	Constraints Constraints
	Snap        SnapConfig
	History     History
}

func NewEngine(tree *Tree) *Engine {
	return &Engine{
		Tree:        tree,
		Constraints: DefaultConstraints(),
		Snap:        DefaultSnapConfig(),
		History:     History{Limit: 200},
	}
}

func (e *Engine) Apply(op Op) (ApplyResult, error) {
	if e == nil {
		return ApplyResult{}, errors.New("layout: engine is nil")
	}
	if e.Tree == nil {
		return ApplyResult{}, errors.New("layout: tree is nil")
	}
	before := e.Tree.Clone()
	result := ApplyResult{}
	var err error

	switch v := op.(type) {
	case ResizeOp:
		result, err = e.applyResize(v)
	case SplitOp:
		result, err = e.applySplit(v)
	case CloseOp:
		result, err = e.applyClose(v)
	case ResetSizesOp:
		result, err = e.applyResetSizes(v)
	case SwapOp:
		result, err = e.applySwap(v)
	case ZoomOp:
		result, err = e.applyZoom(v)
	default:
		return ApplyResult{}, fmt.Errorf("layout: unknown op %T", op)
	}

	if err != nil {
		return ApplyResult{}, err
	}
	if result.Changed {
		e.History.Record(before)
	}
	if op.Kind() == OpResetSizes {
		e.History.Clear()
	}
	return result, nil
}

func (e *Engine) applyResize(op ResizeOp) (ApplyResult, error) {
	paneID := strings.TrimSpace(op.PaneID)
	if paneID == "" {
		return ApplyResult{}, errors.New("layout: resize requires pane id")
	}
	leaf := e.Tree.Leaf(paneID)
	if leaf == nil {
		return ApplyResult{}, fmt.Errorf("layout: pane %q not found", paneID)
	}
	split, beforeChild, afterChild, err := findSplitForEdge(leaf, op.Edge)
	if err != nil {
		return ApplyResult{}, err
	}
	total := beforeChild.Size + afterChild.Size
	minSize := e.minSizeFor(split.Axis)
	if total < minSize*2 {
		return ApplyResult{}, fmt.Errorf("layout: split too small for min size %d", minSize)
	}
	beforeSize := beforeChild.Size
	desired := beforeSize + op.Delta

	snapState := op.SnapState
	snapped := false
	if op.Snap {
		var pos int
		extraTargets := snapTargetsForSplit(e.Tree, split, split.Axis, minSize, total-minSize)
		pos, snapState = SnapPositionWithTargets(e.Snap, desired, minSize, total-minSize, snapState, extraTargets)
		desired = pos
		snapped = snapState.Active
	}

	if desired < minSize {
		desired = minSize
	}
	if desired > total-minSize {
		desired = total - minSize
	}
	afterSize := total - desired
	beforeChild.Size = desired
	afterChild.Size = afterSize

	return ApplyResult{Changed: true, Snapped: snapped, SnapState: snapState, Affected: panesUnder(split)}, nil
}

func (e *Engine) applySplit(op SplitOp) (ApplyResult, error) {
	paneID := strings.TrimSpace(op.PaneID)
	newID := strings.TrimSpace(op.NewPaneID)
	if paneID == "" || newID == "" {
		return ApplyResult{}, errors.New("layout: split requires pane id and new pane id")
	}
	leaf := e.Tree.Leaf(paneID)
	if leaf == nil {
		return ApplyResult{}, fmt.Errorf("layout: pane %q not found", paneID)
	}
	if e.Tree.Leaf(newID) != nil {
		return ApplyResult{}, fmt.Errorf("layout: pane %q already exists", newID)
	}
	newLeaf := &Node{ID: e.Tree.nextNodeID(), PaneID: newID}
	minSize := e.minSizeFor(op.Axis)
	if err := splitLeaf(e.Tree, leaf, newLeaf, op.Axis, op.Percent, minSize); err != nil {
		return ApplyResult{}, err
	}
	e.Tree.Panes[newID] = newLeaf
	return ApplyResult{Changed: true, Affected: panesUnder(leaf.Parent)}, nil
}

func (e *Engine) applyClose(op CloseOp) (ApplyResult, error) {
	paneID := strings.TrimSpace(op.PaneID)
	if paneID == "" {
		return ApplyResult{}, errors.New("layout: close requires pane id")
	}
	leaf := e.Tree.Leaf(paneID)
	if leaf == nil {
		return ApplyResult{}, fmt.Errorf("layout: pane %q not found", paneID)
	}
	parent := leaf.Parent
	delete(e.Tree.Panes, paneID)
	if e.Tree.ZoomedPaneID == paneID {
		e.Tree.ZoomedPaneID = ""
	}
	if parent == nil {
		e.Tree.Root = nil
		return ApplyResult{Changed: true, Affected: []string{paneID}}, nil
	}
	var sibling *Node
	for _, child := range parent.Children {
		if child != leaf {
			sibling = child
			break
		}
	}
	if sibling == nil {
		return ApplyResult{}, errors.New("layout: close pane missing sibling")
	}
	grand := parent.Parent
	sibling.Parent = grand
	sibling.Size = parent.Size
	if grand == nil {
		e.Tree.Root = sibling
		return ApplyResult{Changed: true, Affected: panesUnder(sibling)}, nil
	}
	for i, child := range grand.Children {
		if child == parent {
			grand.Children[i] = sibling
			break
		}
	}
	return ApplyResult{Changed: true, Affected: panesUnder(sibling)}, nil
}

func (e *Engine) applyResetSizes(op ResetSizesOp) (ApplyResult, error) {
	target := e.Tree.Root
	if strings.TrimSpace(op.PaneID) != "" {
		leaf := e.Tree.Leaf(op.PaneID)
		if leaf == nil {
			return ApplyResult{}, fmt.Errorf("layout: pane %q not found", op.PaneID)
		}
		if leaf.Parent != nil {
			target = leaf.Parent
		} else {
			target = leaf
		}
	}
	if target == nil {
		return ApplyResult{}, nil
	}
	rect, ok := rectForNode(e.Tree, target)
	if !ok {
		return ApplyResult{}, errors.New("layout: reset sizes rect missing")
	}
	if err := resetNodeSizes(target, rect, e.Constraints); err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{Changed: true, Affected: panesUnder(target)}, nil
}

func (e *Engine) applySwap(op SwapOp) (ApplyResult, error) {
	paneA := strings.TrimSpace(op.PaneA)
	paneB := strings.TrimSpace(op.PaneB)
	if paneA == "" || paneB == "" {
		return ApplyResult{}, errors.New("layout: swap requires pane ids")
	}
	if paneA == paneB {
		return ApplyResult{}, nil
	}
	leafA := e.Tree.Leaf(paneA)
	leafB := e.Tree.Leaf(paneB)
	if leafA == nil || leafB == nil {
		return ApplyResult{}, fmt.Errorf("layout: panes %q/%q not found", paneA, paneB)
	}
	leafA.PaneID, leafB.PaneID = leafB.PaneID, leafA.PaneID
	e.Tree.Panes[paneA] = leafB
	e.Tree.Panes[paneB] = leafA
	return ApplyResult{Changed: true, Affected: []string{paneA, paneB}}, nil
}

func (e *Engine) applyZoom(op ZoomOp) (ApplyResult, error) {
	paneID := strings.TrimSpace(op.PaneID)
	if paneID == "" {
		return ApplyResult{}, errors.New("layout: zoom requires pane id")
	}
	if e.Tree.Leaf(paneID) == nil {
		return ApplyResult{}, fmt.Errorf("layout: pane %q not found", paneID)
	}
	if op.Toggle {
		if e.Tree.ZoomedPaneID == paneID {
			e.Tree.ZoomedPaneID = ""
			return ApplyResult{Changed: true, Affected: e.Tree.PaneIDs()}, nil
		}
		e.Tree.ZoomedPaneID = paneID
		return ApplyResult{Changed: true, Affected: e.Tree.PaneIDs()}, nil
	}
	if e.Tree.ZoomedPaneID == paneID {
		return ApplyResult{}, nil
	}
	e.Tree.ZoomedPaneID = paneID
	return ApplyResult{Changed: true, Affected: e.Tree.PaneIDs()}, nil
}

func (e *Engine) minSizeFor(axis Axis) int {
	if axis == AxisVertical {
		return e.Constraints.MinHeight
	}
	return e.Constraints.MinWidth
}

func splitLeaf(tree *Tree, leaf, newLeaf *Node, axis Axis, percent, minSize int) error {
	if tree == nil || leaf == nil || newLeaf == nil {
		return errors.New("layout: split requires leaf and new leaf")
	}
	if !leaf.IsLeaf() {
		return errors.New("layout: split target is not leaf")
	}
	rects := tree.Rects()
	rect, ok := rects[leaf.PaneID]
	if !ok {
		return fmt.Errorf("layout: missing rect for pane %q", leaf.PaneID)
	}
	total := splitAxisTotal(rect, axis)
	oldSize, newSize, err := splitComputeSizes(total, percent, minSize)
	if err != nil {
		return err
	}

	parent := leaf.Parent
	split := &Node{ID: tree.nextNodeID(), Axis: axis, Parent: parent, Size: leaf.Size}
	leaf.Parent = split
	leaf.Size = oldSize
	newLeaf.Parent = split
	newLeaf.Size = newSize
	split.Children = []*Node{leaf, newLeaf}
	return replaceLeafWithSplit(tree, parent, leaf, split)
}

func splitAxisTotal(rect Rect, axis Axis) int {
	if axis == AxisVertical {
		return rect.H
	}
	return rect.W
}

func splitComputeSizes(total, percent, minSize int) (oldSize, newSize int, err error) {
	if total <= 1 {
		return 0, 0, errors.New("layout: split target too small")
	}
	if percent <= 0 || percent >= 100 {
		percent = 50
	}
	newSize = total * percent / 100
	if newSize <= 0 || newSize >= total {
		newSize = total / 2
	}
	oldSize = total - newSize
	if minSize <= 0 {
		minSize = 1
	}
	if oldSize >= minSize && newSize >= minSize {
		return oldSize, newSize, nil
	}
	if total < minSize*2 {
		return 0, 0, fmt.Errorf("layout: split min size %d not satisfied", minSize)
	}
	newSize = total / 2
	oldSize = total - newSize
	if oldSize < minSize || newSize < minSize {
		return 0, 0, fmt.Errorf("layout: split min size %d not satisfied", minSize)
	}
	return oldSize, newSize, nil
}

func replaceLeafWithSplit(tree *Tree, parent, leaf, split *Node) error {
	if tree == nil || split == nil {
		return errors.New("layout: split requires tree and split node")
	}
	if parent == nil {
		tree.Root = split
		return nil
	}
	for i, child := range parent.Children {
		if child == leaf {
			parent.Children[i] = split
			return nil
		}
	}
	return nil
}

func findSplitForEdge(leaf *Node, edge ResizeEdge) (*Node, *Node, *Node, error) {
	if leaf == nil {
		return nil, nil, nil, errors.New("layout: resize target missing")
	}
	axis := edge.Axis()
	for node := leaf; node.Parent != nil; node = node.Parent {
		parent := node.Parent
		if parent.Axis != axis {
			continue
		}
		idx := childIndex(parent, node)
		if idx < 0 || len(parent.Children) < 2 {
			continue
		}
		switch edge {
		case ResizeEdgeLeft:
			if idx > 0 {
				return parent, parent.Children[idx-1], parent.Children[idx], nil
			}
		case ResizeEdgeRight:
			if idx < len(parent.Children)-1 {
				return parent, parent.Children[idx], parent.Children[idx+1], nil
			}
		case ResizeEdgeUp:
			if idx > 0 {
				return parent, parent.Children[idx-1], parent.Children[idx], nil
			}
		case ResizeEdgeDown:
			if idx < len(parent.Children)-1 {
				return parent, parent.Children[idx], parent.Children[idx+1], nil
			}
		}
	}
	return nil, nil, nil, errors.New("layout: no matching split for edge")
}

func childIndex(parent, child *Node) int {
	for i, candidate := range parent.Children {
		if candidate == child {
			return i
		}
	}
	return -1
}

func panesUnder(node *Node) []string {
	if node == nil {
		return nil
	}
	if node.IsLeaf() {
		if node.PaneID != "" {
			return []string{node.PaneID}
		}
		return nil
	}
	var out []string
	for _, child := range node.Children {
		out = append(out, panesUnder(child)...)
	}
	return out
}

func snapTargetsForSplit(tree *Tree, split *Node, axis Axis, min, max int) []int {
	if tree == nil || split == nil {
		return nil
	}
	rect, ok := tree.RectForNode(split)
	if !ok {
		return nil
	}
	start := rect.X
	span := rect.W
	if axis == AxisVertical {
		start = rect.Y
		span = rect.H
	}
	if span <= 0 {
		return nil
	}
	end := start + span
	seen := make(map[int]struct{})
	rects := tree.Rects()
	for _, paneRect := range rects {
		var positions [2]int
		if axis == AxisVertical {
			positions = [2]int{paneRect.Y, paneRect.Y + paneRect.H}
		} else {
			positions = [2]int{paneRect.X, paneRect.X + paneRect.W}
		}
		for _, pos := range positions {
			if pos <= start || pos >= end {
				continue
			}
			rel := pos - start
			if rel < min || rel > max {
				continue
			}
			seen[rel] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]int, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}
