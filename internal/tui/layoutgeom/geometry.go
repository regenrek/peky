package layoutgeom

import (
	"math"
	"sort"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

const (
	handleThickness = 1
	handlePadding   = 1
)

type SegmentAxis int

const (
	SegmentVertical SegmentAxis = iota
	SegmentHorizontal
)

type Geometry struct {
	Preview  mouse.Rect
	Panes    []Pane
	Edges    []Edge
	Corners  []Corner
	Dividers []DividerSegment
}

type Pane struct {
	ID     string
	Layout layout.Rect
	Screen mouse.Rect
}

type EdgeRef struct {
	PaneID string
	Edge   sessiond.ResizeEdge
}

type CornerRef struct {
	Vertical   EdgeRef
	Horizontal EdgeRef
}

type Edge struct {
	Ref        EdgeRef
	Axis       SegmentAxis
	LayoutPos  int
	RangeStart int
	RangeEnd   int
	HitRect    mouse.Rect
	LineRect   mouse.Rect
}

type Corner struct {
	Ref     CornerRef
	HitRect mouse.Rect
}

type DividerSegment struct {
	Axis SegmentAxis
	Rect mouse.Rect
}

type DividerCell struct {
	X    int
	Y    int
	Rune rune
}

func ContentRect(geom Geometry, outer mouse.Rect) mouse.Rect {
	if outer.Empty() || len(geom.Dividers) == 0 {
		return outer
	}
	left := hasVerticalDividerAt(geom.Dividers, outer.X, outer.Y, outer.Y+outer.H-1)
	right := hasVerticalDividerAt(geom.Dividers, outer.X+outer.W-1, outer.Y, outer.Y+outer.H-1)
	top := hasHorizontalDividerAt(geom.Dividers, outer.Y, outer.X, outer.X+outer.W-1)
	bottom := hasHorizontalDividerAt(geom.Dividers, outer.Y+outer.H-1, outer.X, outer.X+outer.W-1)

	rect := outer
	if left {
		rect.X++
		rect.W--
	}
	if right {
		rect.W--
	}
	if top {
		rect.Y++
		rect.H--
	}
	if bottom {
		rect.H--
	}
	if rect.W <= 0 || rect.H <= 0 {
		return outer
	}
	return rect
}

func Build(preview mouse.Rect, rects map[string]layout.Rect) (Geometry, bool) {
	if preview.Empty() || len(rects) == 0 {
		return Geometry{}, false
	}
	panes := make([]Pane, 0, len(rects))
	for id, rect := range rects {
		if rect.Empty() {
			continue
		}
		screen := ScaleLayoutRect(preview, rect)
		if screen.Empty() {
			continue
		}
		panes = append(panes, Pane{
			ID:     id,
			Layout: rect,
			Screen: screen,
		})
	}
	if len(panes) == 0 {
		return Geometry{}, false
	}
	sort.Slice(panes, func(i, j int) bool { return panes[i].ID < panes[j].ID })

	edges := buildEdges(preview, panes)
	corners := buildCorners(preview, edges)
	dividers := buildDividers(edges)
	frame := buildFrameSegments(preview)
	if len(frame) > 0 {
		dividers = mergeDividerSegments(append(dividers, frame...))
	}

	return Geometry{
		Preview:  preview,
		Panes:    panes,
		Edges:    edges,
		Corners:  corners,
		Dividers: dividers,
	}, true
}

func buildEdges(preview mouse.Rect, panes []Pane) []Edge {
	edges := make([]Edge, 0, len(panes)*2)
	for i := 0; i < len(panes); i++ {
		for j := i + 1; j < len(panes); j++ {
			left, right, ok := sharedVerticalEdge(panes[i], panes[j])
			if ok {
				edge := Edge{
					Ref:        EdgeRef{PaneID: left.ID, Edge: sessiond.ResizeEdgeRight},
					Axis:       SegmentVertical,
					LayoutPos:  left.Layout.X + left.Layout.W,
					RangeStart: max(left.Layout.Y, right.Layout.Y),
					RangeEnd:   min(left.Layout.Y+left.Layout.H, right.Layout.Y+right.Layout.H),
				}
				edge.LineRect = edgeLineRect(preview, edge)
				edge.HitRect = edgeHitRect(preview, edge)
				if !edge.LineRect.Empty() {
					edges = append(edges, edge)
				}
			}
			top, bottom, ok := sharedHorizontalEdge(panes[i], panes[j])
			if ok {
				edge := Edge{
					Ref:        EdgeRef{PaneID: top.ID, Edge: sessiond.ResizeEdgeDown},
					Axis:       SegmentHorizontal,
					LayoutPos:  top.Layout.Y + top.Layout.H,
					RangeStart: max(top.Layout.X, bottom.Layout.X),
					RangeEnd:   min(top.Layout.X+top.Layout.W, bottom.Layout.X+bottom.Layout.W),
				}
				edge.LineRect = edgeLineRect(preview, edge)
				edge.HitRect = edgeHitRect(preview, edge)
				if !edge.LineRect.Empty() {
					edges = append(edges, edge)
				}
			}
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].LayoutPos != edges[j].LayoutPos {
			return edges[i].LayoutPos < edges[j].LayoutPos
		}
		if edges[i].RangeStart != edges[j].RangeStart {
			return edges[i].RangeStart < edges[j].RangeStart
		}
		if edges[i].Ref.PaneID != edges[j].Ref.PaneID {
			return edges[i].Ref.PaneID < edges[j].Ref.PaneID
		}
		return edges[i].Ref.Edge < edges[j].Ref.Edge
	})
	return edges
}

func buildCorners(preview mouse.Rect, edges []Edge) []Corner {
	if preview.Empty() || len(edges) == 0 {
		return nil
	}
	corners := make([]Corner, 0, len(edges))
	seen := make(map[[2]int]struct{})
	for _, v := range edges {
		if v.Axis != SegmentVertical {
			continue
		}
		for _, h := range edges {
			if h.Axis != SegmentHorizontal {
				continue
			}
			if !rangesIntersect(v.RangeStart, v.RangeEnd, h.LayoutPos, h.LayoutPos) {
				continue
			}
			if !rangesIntersect(h.RangeStart, h.RangeEnd, v.LayoutPos, v.LayoutPos) {
				continue
			}
			key := [2]int{v.LayoutPos, h.LayoutPos}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			x := clampInt(ScaleLayoutPos(preview.X, preview.W, v.LayoutPos), preview.X, preview.X+preview.W-1)
			y := clampInt(ScaleLayoutPos(preview.Y, preview.H, h.LayoutPos), preview.Y, preview.Y+preview.H-1)
			corners = append(corners, Corner{
				Ref: CornerRef{
					Vertical:   v.Ref,
					Horizontal: h.Ref,
				},
				HitRect: mouse.Rect{
					X: x - handlePadding,
					Y: y - handlePadding,
					W: handleThickness + handlePadding*2,
					H: handleThickness + handlePadding*2,
				},
			})
		}
	}
	sort.Slice(corners, func(i, j int) bool {
		if corners[i].HitRect.Y != corners[j].HitRect.Y {
			return corners[i].HitRect.Y < corners[j].HitRect.Y
		}
		if corners[i].HitRect.X != corners[j].HitRect.X {
			return corners[i].HitRect.X < corners[j].HitRect.X
		}
		return corners[i].Ref.Vertical.PaneID < corners[j].Ref.Vertical.PaneID
	})
	return corners
}

func buildDividers(edges []Edge) []DividerSegment {
	if len(edges) == 0 {
		return nil
	}
	segments := make([]DividerSegment, 0, len(edges))
	for _, edge := range edges {
		rect := edge.LineRect
		if rect.Empty() {
			continue
		}
		segments = append(segments, DividerSegment{
			Axis: edge.Axis,
			Rect: rect,
		})
	}
	return mergeDividerSegments(segments)
}

func buildFrameSegments(preview mouse.Rect) []DividerSegment {
	if preview.Empty() {
		return nil
	}
	segments := make([]DividerSegment, 0, 4)
	segments = append(segments, DividerSegment{
		Axis: SegmentHorizontal,
		Rect: mouse.Rect{X: preview.X, Y: preview.Y, W: preview.W, H: 1},
	})
	if preview.H > 1 {
		segments = append(segments, DividerSegment{
			Axis: SegmentHorizontal,
			Rect: mouse.Rect{X: preview.X, Y: preview.Y + preview.H - 1, W: preview.W, H: 1},
		})
	}
	segments = append(segments, DividerSegment{
		Axis: SegmentVertical,
		Rect: mouse.Rect{X: preview.X, Y: preview.Y, W: 1, H: preview.H},
	})
	if preview.W > 1 {
		segments = append(segments, DividerSegment{
			Axis: SegmentVertical,
			Rect: mouse.Rect{X: preview.X + preview.W - 1, Y: preview.Y, W: 1, H: preview.H},
		})
	}
	return segments
}

func mergeDividerSegments(segments []DividerSegment) []DividerSegment {
	if len(segments) <= 1 {
		return segments
	}
	sort.Slice(segments, func(i, j int) bool {
		if segments[i].Axis != segments[j].Axis {
			return segments[i].Axis < segments[j].Axis
		}
		if segments[i].Axis == SegmentVertical {
			if segments[i].Rect.X != segments[j].Rect.X {
				return segments[i].Rect.X < segments[j].Rect.X
			}
			if segments[i].Rect.Y != segments[j].Rect.Y {
				return segments[i].Rect.Y < segments[j].Rect.Y
			}
			return segments[i].Rect.H < segments[j].Rect.H
		}
		if segments[i].Rect.Y != segments[j].Rect.Y {
			return segments[i].Rect.Y < segments[j].Rect.Y
		}
		if segments[i].Rect.X != segments[j].Rect.X {
			return segments[i].Rect.X < segments[j].Rect.X
		}
		return segments[i].Rect.W < segments[j].Rect.W
	})

	merged := make([]DividerSegment, 0, len(segments))
	current := segments[0]
	for i := 1; i < len(segments); i++ {
		next := segments[i]
		if current.Axis != next.Axis {
			merged = append(merged, current)
			current = next
			continue
		}
		if current.Axis == SegmentVertical {
			if current.Rect.X != next.Rect.X {
				merged = append(merged, current)
				current = next
				continue
			}
			end := current.Rect.Y + current.Rect.H - 1
			nextStart := next.Rect.Y
			nextEnd := next.Rect.Y + next.Rect.H - 1
			if nextStart <= end+1 {
				if nextEnd > end {
					current.Rect.H = nextEnd - current.Rect.Y + 1
				}
				continue
			}
			merged = append(merged, current)
			current = next
			continue
		}
		if current.Rect.Y != next.Rect.Y {
			merged = append(merged, current)
			current = next
			continue
		}
		end := current.Rect.X + current.Rect.W - 1
		nextStart := next.Rect.X
		nextEnd := next.Rect.X + next.Rect.W - 1
		if nextStart <= end+1 {
			if nextEnd > end {
				current.Rect.W = nextEnd - current.Rect.X + 1
			}
			continue
		}
		merged = append(merged, current)
		current = next
	}
	merged = append(merged, current)
	return merged
}

func DividerCells(segments []DividerSegment) []DividerCell {
	if len(segments) == 0 {
		return nil
	}
	masks := make(map[point]lineMask, len(segments)*4)
	for _, seg := range segments {
		rect := seg.Rect
		if rect.Empty() {
			continue
		}
		switch seg.Axis {
		case SegmentVertical:
			y1 := rect.Y
			y2 := rect.Y + rect.H - 1
			if y2 < y1 {
				y1, y2 = y2, y1
			}
			for y := y1; y <= y2; y++ {
				mask := masks[point{X: rect.X, Y: y}]
				if y1 == y2 {
					mask |= lineUp | lineDown
				} else {
					if y > y1 {
						mask |= lineUp
					}
					if y < y2 {
						mask |= lineDown
					}
				}
				masks[point{X: rect.X, Y: y}] = mask
			}
		case SegmentHorizontal:
			x1 := rect.X
			x2 := rect.X + rect.W - 1
			if x2 < x1 {
				x1, x2 = x2, x1
			}
			for x := x1; x <= x2; x++ {
				mask := masks[point{X: x, Y: rect.Y}]
				if x1 == x2 {
					mask |= lineLeft | lineRight
				} else {
					if x > x1 {
						mask |= lineLeft
					}
					if x < x2 {
						mask |= lineRight
					}
				}
				masks[point{X: x, Y: rect.Y}] = mask
			}
		}
	}
	if len(masks) == 0 {
		return nil
	}
	cells := make([]DividerCell, 0, len(masks))
	for pt, mask := range masks {
		cells = append(cells, DividerCell{
			X:    pt.X,
			Y:    pt.Y,
			Rune: mask.rune(),
		})
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].Y != cells[j].Y {
			return cells[i].Y < cells[j].Y
		}
		return cells[i].X < cells[j].X
	})
	return cells
}

func hasVerticalDividerAt(segments []DividerSegment, x, yStart, yEnd int) bool {
	if yStart > yEnd {
		yStart, yEnd = yEnd, yStart
	}
	for _, seg := range segments {
		if seg.Axis != SegmentVertical {
			continue
		}
		if seg.Rect.X != x || seg.Rect.H <= 0 {
			continue
		}
		segStart := seg.Rect.Y
		segEnd := seg.Rect.Y + seg.Rect.H - 1
		if rangesIntersect(segStart, segEnd, yStart, yEnd) {
			return true
		}
	}
	return false
}

func hasHorizontalDividerAt(segments []DividerSegment, y, xStart, xEnd int) bool {
	if xStart > xEnd {
		xStart, xEnd = xEnd, xStart
	}
	for _, seg := range segments {
		if seg.Axis != SegmentHorizontal {
			continue
		}
		if seg.Rect.Y != y || seg.Rect.W <= 0 {
			continue
		}
		segStart := seg.Rect.X
		segEnd := seg.Rect.X + seg.Rect.W - 1
		if rangesIntersect(segStart, segEnd, xStart, xEnd) {
			return true
		}
	}
	return false
}

func EdgeLineRect(geom Geometry, ref EdgeRef) (mouse.Rect, bool) {
	for _, edge := range geom.Edges {
		if edge.Ref == ref {
			return edge.LineRect, true
		}
	}
	for _, pane := range geom.Panes {
		if pane.ID != ref.PaneID {
			continue
		}
		rect := edgeLineRectFromPane(geom.Preview, pane.Layout, ref)
		if rect.Empty() {
			return mouse.Rect{}, false
		}
		return rect, true
	}
	return mouse.Rect{}, false
}

func EdgeHitRect(geom Geometry, ref EdgeRef) (mouse.Rect, bool) {
	for _, edge := range geom.Edges {
		if edge.Ref == ref {
			return edge.HitRect, true
		}
	}
	for _, pane := range geom.Panes {
		if pane.ID != ref.PaneID {
			continue
		}
		rect := edgeHitRectFromPane(geom.Preview, pane.Layout, ref)
		if rect.Empty() {
			return mouse.Rect{}, false
		}
		return rect, true
	}
	return mouse.Rect{}, false
}

func PanesForEdge(geom Geometry, ref EdgeRef) (Pane, Pane, bool) {
	var pane Pane
	found := false
	for _, candidate := range geom.Panes {
		if candidate.ID == ref.PaneID {
			pane = candidate
			found = true
			break
		}
	}
	if !found {
		return Pane{}, Pane{}, false
	}
	for _, other := range geom.Panes {
		if other.ID == pane.ID {
			continue
		}
		switch ref.Edge {
		case sessiond.ResizeEdgeLeft, sessiond.ResizeEdgeRight:
			left, right, ok := sharedVerticalEdge(pane, other)
			if !ok {
				continue
			}
			if ref.Edge == sessiond.ResizeEdgeRight && left.ID == pane.ID {
				return pane, right, true
			}
			if ref.Edge == sessiond.ResizeEdgeLeft && right.ID == pane.ID {
				return pane, left, true
			}
		case sessiond.ResizeEdgeUp, sessiond.ResizeEdgeDown:
			top, bottom, ok := sharedHorizontalEdge(pane, other)
			if !ok {
				continue
			}
			if ref.Edge == sessiond.ResizeEdgeDown && top.ID == pane.ID {
				return pane, bottom, true
			}
			if ref.Edge == sessiond.ResizeEdgeUp && bottom.ID == pane.ID {
				return pane, top, true
			}
		}
	}
	return Pane{}, Pane{}, false
}

func edgeLineRectFromPane(preview mouse.Rect, rect layout.Rect, ref EdgeRef) mouse.Rect {
	switch ref.Edge {
	case sessiond.ResizeEdgeLeft, sessiond.ResizeEdgeRight:
		pos := rect.X
		if ref.Edge == sessiond.ResizeEdgeRight {
			pos = rect.X + rect.W
		}
		edge := Edge{
			Axis:       SegmentVertical,
			LayoutPos:  pos,
			RangeStart: rect.Y,
			RangeEnd:   rect.Y + rect.H,
		}
		return edgeLineRect(preview, edge)
	case sessiond.ResizeEdgeUp, sessiond.ResizeEdgeDown:
		pos := rect.Y
		if ref.Edge == sessiond.ResizeEdgeDown {
			pos = rect.Y + rect.H
		}
		edge := Edge{
			Axis:       SegmentHorizontal,
			LayoutPos:  pos,
			RangeStart: rect.X,
			RangeEnd:   rect.X + rect.W,
		}
		return edgeLineRect(preview, edge)
	default:
		return mouse.Rect{}
	}
}

func edgeHitRectFromPane(preview mouse.Rect, rect layout.Rect, ref EdgeRef) mouse.Rect {
	switch ref.Edge {
	case sessiond.ResizeEdgeLeft, sessiond.ResizeEdgeRight:
		pos := rect.X
		if ref.Edge == sessiond.ResizeEdgeRight {
			pos = rect.X + rect.W
		}
		edge := Edge{
			Axis:       SegmentVertical,
			LayoutPos:  pos,
			RangeStart: rect.Y,
			RangeEnd:   rect.Y + rect.H,
		}
		return edgeHitRect(preview, edge)
	case sessiond.ResizeEdgeUp, sessiond.ResizeEdgeDown:
		pos := rect.Y
		if ref.Edge == sessiond.ResizeEdgeDown {
			pos = rect.Y + rect.H
		}
		edge := Edge{
			Axis:       SegmentHorizontal,
			LayoutPos:  pos,
			RangeStart: rect.X,
			RangeEnd:   rect.X + rect.W,
		}
		return edgeHitRect(preview, edge)
	default:
		return mouse.Rect{}
	}
}

func edgeLineRect(preview mouse.Rect, edge Edge) mouse.Rect {
	if preview.Empty() {
		return mouse.Rect{}
	}
	switch edge.Axis {
	case SegmentVertical:
		x := clampInt(ScaleLayoutPos(preview.X, preview.W, edge.LayoutPos), preview.X, preview.X+preview.W-1)
		y1 := clampInt(ScaleLayoutPos(preview.Y, preview.H, edge.RangeStart), preview.Y, preview.Y+preview.H-1)
		y2 := clampInt(ScaleLayoutPos(preview.Y, preview.H, edge.RangeEnd), preview.Y, preview.Y+preview.H-1)
		if y2 < y1 {
			y1, y2 = y2, y1
		}
		if y1 == y2 {
			return mouse.Rect{X: x, Y: y1, W: 1, H: 1}
		}
		return mouse.Rect{
			X: x,
			Y: y1,
			W: 1,
			H: y2 - y1 + 1,
		}
	case SegmentHorizontal:
		y := clampInt(ScaleLayoutPos(preview.Y, preview.H, edge.LayoutPos), preview.Y, preview.Y+preview.H-1)
		x1 := clampInt(ScaleLayoutPos(preview.X, preview.W, edge.RangeStart), preview.X, preview.X+preview.W-1)
		x2 := clampInt(ScaleLayoutPos(preview.X, preview.W, edge.RangeEnd), preview.X, preview.X+preview.W-1)
		if x2 < x1 {
			x1, x2 = x2, x1
		}
		if x1 == x2 {
			return mouse.Rect{X: x1, Y: y, W: 1, H: 1}
		}
		return mouse.Rect{
			X: x1,
			Y: y,
			W: x2 - x1 + 1,
			H: 1,
		}
	default:
		return mouse.Rect{}
	}
}

func edgeHitRect(preview mouse.Rect, edge Edge) mouse.Rect {
	if preview.Empty() {
		return mouse.Rect{}
	}
	switch edge.Axis {
	case SegmentVertical:
		x := clampInt(ScaleLayoutPos(preview.X, preview.W, edge.LayoutPos), preview.X, preview.X+preview.W-1)
		y1 := clampInt(ScaleLayoutPos(preview.Y, preview.H, edge.RangeStart), preview.Y, preview.Y+preview.H-1)
		y2 := clampInt(ScaleLayoutPos(preview.Y, preview.H, edge.RangeEnd), preview.Y, preview.Y+preview.H)
		if y2 < y1 {
			y1, y2 = y2, y1
		}
		if y2 <= y1 {
			y2 = min(y1+1, preview.Y+preview.H)
		}
		return mouse.Rect{
			X: x - handlePadding,
			Y: y1,
			W: handleThickness + handlePadding*2,
			H: y2 - y1,
		}
	case SegmentHorizontal:
		y := clampInt(ScaleLayoutPos(preview.Y, preview.H, edge.LayoutPos), preview.Y, preview.Y+preview.H-1)
		x1 := clampInt(ScaleLayoutPos(preview.X, preview.W, edge.RangeStart), preview.X, preview.X+preview.W-1)
		x2 := clampInt(ScaleLayoutPos(preview.X, preview.W, edge.RangeEnd), preview.X, preview.X+preview.W)
		if x2 < x1 {
			x1, x2 = x2, x1
		}
		if x2 <= x1 {
			x2 = min(x1+1, preview.X+preview.W)
		}
		return mouse.Rect{
			X: x1,
			Y: y - handlePadding,
			W: x2 - x1,
			H: handleThickness + handlePadding*2,
		}
	default:
		return mouse.Rect{}
	}
}

func sharedVerticalEdge(a, b Pane) (Pane, Pane, bool) {
	if a.Layout.X+a.Layout.W == b.Layout.X {
		if overlap(a.Layout.Y, a.Layout.Y+a.Layout.H, b.Layout.Y, b.Layout.Y+b.Layout.H) > 0 {
			return a, b, true
		}
	}
	if b.Layout.X+b.Layout.W == a.Layout.X {
		if overlap(a.Layout.Y, a.Layout.Y+a.Layout.H, b.Layout.Y, b.Layout.Y+b.Layout.H) > 0 {
			return b, a, true
		}
	}
	return Pane{}, Pane{}, false
}

func sharedHorizontalEdge(a, b Pane) (Pane, Pane, bool) {
	if a.Layout.Y+a.Layout.H == b.Layout.Y {
		if overlap(a.Layout.X, a.Layout.X+a.Layout.W, b.Layout.X, b.Layout.X+b.Layout.W) > 0 {
			return a, b, true
		}
	}
	if b.Layout.Y+b.Layout.H == a.Layout.Y {
		if overlap(a.Layout.X, a.Layout.X+a.Layout.W, b.Layout.X, b.Layout.X+b.Layout.W) > 0 {
			return b, a, true
		}
	}
	return Pane{}, Pane{}, false
}

func overlap(a1, a2, b1, b2 int) int {
	start := max(a1, b1)
	end := min(a2, b2)
	if end <= start {
		return 0
	}
	return end - start
}

func rangesIntersect(a1, a2, b1, b2 int) bool {
	if a1 > a2 {
		a1, a2 = a2, a1
	}
	if b1 > b2 {
		b1, b2 = b2, b1
	}
	return a1 <= b2 && b1 <= a2
}

func ScaleLayoutRect(preview mouse.Rect, rect layout.Rect) mouse.Rect {
	if preview.Empty() || rect.Empty() {
		return mouse.Rect{}
	}
	x1 := ScaleLayoutPos(preview.X, preview.W, rect.X)
	y1 := ScaleLayoutPos(preview.Y, preview.H, rect.Y)
	x2 := ScaleLayoutPos(preview.X, preview.W, rect.X+rect.W)
	y2 := ScaleLayoutPos(preview.Y, preview.H, rect.Y+rect.H)
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	w := x2 - x1
	h := y2 - y1
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if x1+w > preview.X+preview.W {
		w = preview.X + preview.W - x1
	}
	if y1+h > preview.Y+preview.H {
		h = preview.Y + preview.H - y1
	}
	if w <= 0 || h <= 0 {
		return mouse.Rect{}
	}
	return mouse.Rect{X: x1, Y: y1, W: w, H: h}
}

func ScaleLayoutPos(base, span, pos int) int {
	if span <= 0 {
		return base
	}
	if pos <= 0 {
		return base
	}
	if pos >= layout.LayoutBaseSize {
		return base + span
	}
	return base + (pos*span)/layout.LayoutBaseSize
}

func LayoutPosFromScreen(preview mouse.Rect, x, y int) (int, int, bool) {
	if preview.Empty() {
		return 0, 0, false
	}
	if x < preview.X {
		x = preview.X
	}
	if y < preview.Y {
		y = preview.Y
	}
	if x >= preview.X+preview.W {
		x = preview.X + preview.W - 1
	}
	if y >= preview.Y+preview.H {
		y = preview.Y + preview.H - 1
	}
	relX := x - preview.X
	relY := y - preview.Y
	if relX < 0 || relY < 0 {
		return 0, 0, false
	}
	lx := int(math.Round(float64(relX) / float64(preview.W) * float64(layout.LayoutBaseSize)))
	ly := int(math.Round(float64(relY) / float64(preview.H) * float64(layout.LayoutBaseSize)))
	if lx < 0 {
		lx = 0
	}
	if ly < 0 {
		ly = 0
	}
	if lx > layout.LayoutBaseSize {
		lx = layout.LayoutBaseSize
	}
	if ly > layout.LayoutBaseSize {
		ly = layout.LayoutBaseSize
	}
	return lx, ly, true
}

type lineMask uint8

const (
	lineUp lineMask = 1 << iota
	lineDown
	lineLeft
	lineRight
)

type point struct {
	X int
	Y int
}

func (m lineMask) rune() rune {
	switch m {
	case lineUp | lineDown | lineLeft | lineRight:
		return '┼'
	case lineUp | lineDown | lineLeft:
		return '┤'
	case lineUp | lineDown | lineRight:
		return '├'
	case lineLeft | lineRight | lineDown:
		return '┬'
	case lineLeft | lineRight | lineUp:
		return '┴'
	case lineDown | lineRight:
		return '┌'
	case lineDown | lineLeft:
		return '┐'
	case lineUp | lineRight:
		return '└'
	case lineUp | lineLeft:
		return '┘'
	case lineUp | lineDown:
		return '│'
	case lineLeft | lineRight:
		return '─'
	case lineUp, lineDown:
		return '│'
	case lineLeft, lineRight:
		return '─'
	default:
		return '─'
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
