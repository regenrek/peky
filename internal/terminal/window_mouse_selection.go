package terminal

import (
	"time"

	uv "github.com/charmbracelet/ultraviolet"
)

func (w *Window) handleMouseSelection(event uv.MouseEvent, route MouseRoute) bool {
	switch ev := event.(type) {
	case uv.MouseClickEvent:
		return w.handleMouseSelectPress(ev, route)
	case uv.MouseMotionEvent:
		return w.handleMouseSelectMotion(ev, route)
	case uv.MouseReleaseEvent:
		return w.handleMouseSelectRelease(ev)
	default:
		return false
	}
}

func (w *Window) mouseNowTime() time.Time {
	if w == nil || w.mouseNow == nil {
		return time.Now()
	}
	return w.mouseNow()
}

func (w *Window) handleMouseSelectPress(event uv.MouseClickEvent, route MouseRoute) bool {
	if w == nil {
		return false
	}
	if event.Button != uv.MouseLeft {
		return false
	}
	if !w.shouldCaptureMouseForSelection(event.Mod, route) {
		return false
	}

	absX, absY, _, ok := w.mouseAbsFromViewport(event.X, event.Y)
	if !ok {
		return false
	}
	absX = w.adjustToNonContinuation(absY, absX, -1)

	clickCount := w.nextMouseClickCount(event, w.mouseNowTime())
	if clickCount == 0 {
		clickCount = 1
	}
	if clickCount > 3 {
		clickCount = 1
	}

	if event.Mod&uv.ModShift != 0 && w.extendSelectionTo(absX, absY) {
		return true
	}

	switch clickCount {
	case 1:
		if !w.exitMouseSelectionCopyModeOnSingleClick() {
			w.clearSelectionOnSingleClick(absX, absY)
		}
		w.setPendingMouseDrag(event, absX, absY, mouseSelectSingle)
		return true
	case 2:
		return w.beginWordSelection(event, absX, absY, route)
	case 3:
		return w.beginLineSelection(event, absX, absY, route)
	default:
		return true
	}
}

func (w *Window) exitMouseSelectionCopyModeOnSingleClick() bool {
	if w == nil {
		return false
	}
	w.stateMu.Lock()
	cm := w.CopyMode
	shouldExit := cm != nil && cm.Active && cm.Selecting && w.mouseSel.fromMouse && w.mouseSel.startedCopyForSelection
	w.stateMu.Unlock()
	if !shouldExit {
		return false
	}
	w.ExitCopyMode()
	return true
}

func (w *Window) setPendingMouseDrag(event uv.MouseClickEvent, absX, absY int, kind mouseSelectionKind) {
	w.stateMu.Lock()
	w.mouseSel.pending = true
	w.mouseSel.pendingX = event.X
	w.mouseSel.pendingY = event.Y
	w.mouseSel.pendingAbsX = absX
	w.mouseSel.pendingAbsY = absY
	w.mouseSel.pendingKind = kind
	w.mouseSel.dragActive = false
	w.mouseSel.anchorAbsY = absY
	w.mouseSel.anchorStartX = absX
	w.mouseSel.anchorEndX = absX
	w.mouseSel.moved = false
	w.stateMu.Unlock()
}

func (w *Window) clearSelectionOnSingleClick(absX, absY int) {
	w.stateMu.Lock()
	cm := w.CopyMode
	if cm != nil && cm.Active {
		cm.CursorX = absX
		cm.CursorAbsY = absY
		cm.Selecting = false
		cm.SelStartX, cm.SelStartAbsY = absX, absY
		cm.SelEndX, cm.SelEndAbsY = absX, absY
		w.mouseSel.clearSelectionFlags()
	}
	w.stateMu.Unlock()
	w.markDirty()
}

func (w *Window) beginWordSelection(event uv.MouseClickEvent, absX, absY int, route MouseRoute) bool {
	startX, endX, ok := w.wordBoundsAt(absY, absX)
	if !ok {
		w.setPendingMouseDrag(event, absX, absY, mouseSelectWord)
		return true
	}

	wasCopyActive := w.CopyModeActive()
	if !wasCopyActive {
		w.enterCopyMode(route == MouseRouteHostSelection)
	}

	w.stateMu.Lock()
	cm := w.CopyMode
	if cm == nil || !cm.Active {
		w.stateMu.Unlock()
		return false
	}
	cm.CursorX = absX
	cm.CursorAbsY = absY
	cm.Selecting = true
	cm.SelStartX, cm.SelStartAbsY = startX, absY
	cm.SelEndX, cm.SelEndAbsY = endX, absY
	w.mouseSel.fromMouse = true
	w.mouseSel.startedCopyForSelection = !wasCopyActive
	w.mouseSel.pending = true
	w.mouseSel.pendingX = event.X
	w.mouseSel.pendingY = event.Y
	w.mouseSel.pendingAbsX = absX
	w.mouseSel.pendingAbsY = absY
	w.mouseSel.pendingKind = mouseSelectWord
	w.mouseSel.dragActive = true
	w.mouseSel.anchorAbsY = absY
	w.mouseSel.anchorStartX = startX
	w.mouseSel.anchorEndX = endX
	w.mouseSel.moved = false
	w.stateMu.Unlock()
	w.markDirty()
	return true
}

func (w *Window) beginLineSelection(event uv.MouseClickEvent, absX, absY int, route MouseRoute) bool {
	wasCopyActive := w.CopyModeActive()
	if !wasCopyActive {
		w.enterCopyMode(route == MouseRouteHostSelection)
	}

	startX := 0
	endX := maxInt(0, w.cols-1)

	w.stateMu.Lock()
	cm := w.CopyMode
	if cm == nil || !cm.Active {
		w.stateMu.Unlock()
		return false
	}
	cm.CursorX = absX
	cm.CursorAbsY = absY
	cm.Selecting = true
	cm.SelStartX, cm.SelStartAbsY = startX, absY
	cm.SelEndX, cm.SelEndAbsY = endX, absY
	w.mouseSel.fromMouse = true
	w.mouseSel.startedCopyForSelection = !wasCopyActive
	w.mouseSel.pending = true
	w.mouseSel.pendingX = event.X
	w.mouseSel.pendingY = event.Y
	w.mouseSel.pendingAbsX = absX
	w.mouseSel.pendingAbsY = absY
	w.mouseSel.pendingKind = mouseSelectLine
	w.mouseSel.dragActive = true
	w.mouseSel.anchorAbsY = absY
	w.mouseSel.anchorStartX = startX
	w.mouseSel.anchorEndX = endX
	w.mouseSel.moved = false
	w.stateMu.Unlock()
	w.markDirty()
	return true
}

func (w *Window) extendSelectionTo(absX, absY int) bool {
	if w == nil {
		return false
	}
	sbLen := w.ScrollbackLen()
	total := sbLen + w.rows
	if total <= 0 {
		return false
	}
	absX = w.adjustToNonContinuation(absY, absX, -1)

	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	cm := w.CopyMode
	if cm == nil || !cm.Active || !cm.Selecting {
		return false
	}
	cm.CursorX = absX
	cm.CursorAbsY = clampInt(absY, 0, total-1)
	cm.SelEndX = absX
	cm.SelEndAbsY = cm.CursorAbsY
	w.mouseSel.fromMouse = true
	w.mouseSel.moved = true
	return true
}

func (w *Window) handleMouseSelectMotion(event uv.MouseMotionEvent, route MouseRoute) bool {
	if w == nil {
		return false
	}

	w.stateMu.Lock()
	pending := w.mouseSel.pending
	dragActive := w.mouseSel.dragActive
	startX := w.mouseSel.pendingX
	startY := w.mouseSel.pendingY
	pendingAbsX := w.mouseSel.pendingAbsX
	pendingAbsY := w.mouseSel.pendingAbsY
	kind := w.mouseSel.pendingKind
	anchorAbsY := w.mouseSel.anchorAbsY
	anchorStartX := w.mouseSel.anchorStartX
	anchorEndX := w.mouseSel.anchorEndX
	w.stateMu.Unlock()

	if !pending && !dragActive {
		return false
	}

	absX, absY, sbLen, ok := w.mouseAbsFromViewport(event.X, event.Y)
	if !ok {
		return true
	}
	absX = w.adjustToNonContinuation(absY, absX, 1)

	if kind == mouseSelectSingle && !dragActive {
		if !mouseExceededDragThreshold(startX, startY, event.X, event.Y) {
			return true
		}
		return w.startDragSelection(sbLen, pendingAbsX, pendingAbsY, absX, absY, route)
	}

	return w.updateDragSelection(sbLen, kind, anchorAbsY, anchorStartX, anchorEndX, absX, absY)
}

func (w *Window) startDragSelection(sbLen, startAbsX, startAbsY, endAbsX, endAbsY int, route MouseRoute) bool {
	wasCopyActive := w.CopyModeActive()
	if !wasCopyActive {
		w.enterCopyMode(route == MouseRouteHostSelection)
	}
	if !w.CopyModeActive() {
		return false
	}

	startAbsX = w.adjustToNonContinuation(startAbsY, startAbsX, -1)
	endAbsX = w.adjustToNonContinuation(endAbsY, endAbsX, 1)

	w.stateMu.Lock()
	cm := w.CopyMode
	if cm == nil || !cm.Active {
		w.stateMu.Unlock()
		return false
	}
	cm.CursorX = endAbsX
	cm.CursorAbsY = endAbsY
	cm.Selecting = true
	cm.SelStartX, cm.SelStartAbsY = startAbsX, startAbsY
	cm.SelEndX, cm.SelEndAbsY = endAbsX, endAbsY
	w.mouseSel.fromMouse = true
	w.mouseSel.startedCopyForSelection = !wasCopyActive
	w.mouseSel.dragActive = true
	w.mouseSel.moved = startAbsX != endAbsX || startAbsY != endAbsY
	w.ensureCopyCursorVisibleLocked(sbLen)
	w.stateMu.Unlock()

	w.markDirty()
	return true
}

func (w *Window) updateDragSelection(sbLen int, kind mouseSelectionKind, anchorAbsY, anchorStartX, anchorEndX, absX, absY int) bool {
	nextStartX, nextStartY, nextEndX, nextEndY := selectionBoundsForDrag(w, kind, anchorAbsY, anchorStartX, anchorEndX, absX, absY)

	w.stateMu.Lock()
	cm := w.CopyMode
	if cm == nil || !cm.Active || !cm.Selecting {
		w.stateMu.Unlock()
		return true
	}

	changed := cm.SelStartX != nextStartX || cm.SelStartAbsY != nextStartY || cm.SelEndX != nextEndX || cm.SelEndAbsY != nextEndY
	cm.CursorX = absX
	cm.CursorAbsY = absY
	cm.SelStartX, cm.SelStartAbsY = nextStartX, nextStartY
	cm.SelEndX, cm.SelEndAbsY = nextEndX, nextEndY
	if changed {
		w.mouseSel.moved = true
	}
	w.ensureCopyCursorVisibleLocked(sbLen)
	w.stateMu.Unlock()

	w.markDirty()
	return true
}

func selectionBoundsForDrag(w *Window, kind mouseSelectionKind, anchorAbsY, anchorStartX, anchorEndX, absX, absY int) (startX, startY, endX, endY int) {
	switch kind {
	case mouseSelectLine:
		return selectionBoundsLine(w, anchorAbsY, absY)
	case mouseSelectWord:
		return selectionBoundsWord(w, anchorAbsY, anchorStartX, anchorEndX, absX, absY)
	default:
		return anchorStartX, anchorAbsY, absX, absY
	}
}

func selectionBoundsLine(w *Window, anchorAbsY, absY int) (startX, startY, endX, endY int) {
	startX = 0
	endX = maxInt(0, w.cols-1)
	startY = anchorAbsY
	endY = absY
	if endY < startY {
		startY, endY = endY, startY
	}
	return startX, startY, endX, endY
}

func selectionBoundsWord(w *Window, anchorAbsY, anchorStartX, anchorEndX, absX, absY int) (startX, startY, endX, endY int) {
	curStart, curEnd, ok := w.wordBoundsAt(absY, absX)
	if !ok {
		return anchorStartX, anchorAbsY, absX, absY
	}

	aStartX, aStartY := anchorStartX, anchorAbsY
	aEndX, aEndY := anchorEndX, anchorAbsY
	cStartX, cStartY := curStart, absY
	cEndX, cEndY := curEnd, absY

	if cStartY < aStartY || (cStartY == aStartY && cStartX < aStartX) {
		return cStartX, cStartY, aEndX, aEndY
	}
	return aStartX, aStartY, cEndX, cEndY
}

func mouseExceededDragThreshold(startX, startY, x, y int) bool {
	dx := x - startX
	if dx < 0 {
		dx = -dx
	}
	dy := y - startY
	if dy < 0 {
		dy = -dy
	}
	return dx >= mouseDragThreshold || dy >= mouseDragThreshold
}

func (w *Window) handleMouseSelectRelease(event uv.MouseReleaseEvent) bool {
	if w == nil {
		return false
	}
	if event.Button != uv.MouseLeft {
		return false
	}

	w.stateMu.Lock()
	active := w.mouseSel.pending || w.mouseSel.dragActive
	moved := w.mouseSel.moved
	fromMouse := w.mouseSel.fromMouse
	w.mouseSel.resetPress()
	w.stateMu.Unlock()

	if !active {
		return false
	}

	if fromMouse && moved && w.copyMouseSelectionToClipboard() {
		w.notifyToast(mouseSelectionCopiedToast)
	}
	w.markDirty()
	return true
}

func (w *Window) nextMouseClickCount(event uv.MouseClickEvent, now time.Time) int {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()

	if w.mouseSel.lastClickBtn != event.Button {
		w.mouseSel.lastClickCount = 0
	}

	if w.mouseSel.lastClickCount > 0 {
		if now.Sub(w.mouseSel.lastClickAt) > mouseMultiClickThreshold {
			w.mouseSel.lastClickCount = 0
		}
		if !withinMouseClickDistance(w.mouseSel.lastClickX, w.mouseSel.lastClickY, event.X, event.Y) {
			w.mouseSel.lastClickCount = 0
		}
	}

	w.mouseSel.lastClickAt = now
	w.mouseSel.lastClickX = event.X
	w.mouseSel.lastClickY = event.Y
	w.mouseSel.lastClickBtn = event.Button
	w.mouseSel.lastClickCount++
	if w.mouseSel.lastClickCount > 3 {
		w.mouseSel.lastClickCount = 1
	}
	return w.mouseSel.lastClickCount
}

func withinMouseClickDistance(x0, y0, x1, y1 int) bool {
	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y0
	if dy < 0 {
		dy = -dy
	}
	return dx <= mouseMultiClickMaxDist && dy <= mouseMultiClickMaxDist
}
