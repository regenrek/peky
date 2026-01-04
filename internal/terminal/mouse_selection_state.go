package terminal

import (
	"time"

	uv "github.com/charmbracelet/ultraviolet"
)

type mouseSelectionKind uint8

const (
	mouseSelectSingle mouseSelectionKind = iota
	mouseSelectWord
	mouseSelectLine
)

const (
	mouseMultiClickThreshold = 350 * time.Millisecond
	mouseMultiClickMaxDist   = 1
	mouseDragThreshold       = 1
)

// mouseSelectionState tracks host-side mouse selection state.
//
// Guarded by Window.stateMu unless otherwise noted.
type mouseSelectionState struct {
	// Multi-click tracking.
	lastClickAt    time.Time
	lastClickX     int
	lastClickY     int
	lastClickCount int
	lastClickBtn   uv.MouseButton

	// Pending drag selection (mouse pressed but below threshold).
	pending     bool
	pendingX    int
	pendingY    int
	pendingAbsX int
	pendingAbsY int
	pendingKind mouseSelectionKind

	// Active drag selection (mouse pressed, selection updates on motion).
	dragActive bool

	// Anchor is the initial selection origin for word/line selection.
	anchorAbsY   int
	anchorStartX int
	anchorEndX   int

	// Lifecycle markers for mouse-created selections.
	startedCopyForSelection bool
	moved                   bool
	fromMouse               bool
}

func (s *mouseSelectionState) resetPress() {
	s.pending = false
	s.dragActive = false
	s.pendingKind = mouseSelectSingle
	s.anchorAbsY = 0
	s.anchorStartX = 0
	s.anchorEndX = 0
	s.moved = false
}

func (s *mouseSelectionState) clearSelectionFlags() {
	s.fromMouse = false
	s.moved = false
	s.startedCopyForSelection = false
}
