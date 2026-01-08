package app

import (
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

type resizeState struct {
	mode    bool
	snap    bool
	hover   resizeHoverState
	drag    resizeDragState
	preview resizePreviewState
	cache   resizeGeomCache
	key     resizeKeyState
}

func (r *resizeState) invalidateCache() {
	r.cache = resizeGeomCache{}
}

type resizeEdgeRef = layoutgeom.EdgeRef

type resizeCornerRef = layoutgeom.CornerRef

type resizeHoverState struct {
	edge      resizeEdgeRef
	corner    resizeCornerRef
	hasEdge   bool
	hasCorner bool
}

type resizeDragState struct {
	active           bool
	session          string
	edge             resizeEdgeRef
	corner           resizeCornerRef
	cornerActive     bool
	startLayoutX     int
	startLayoutY     int
	baseEdgePos      int
	baseEdgePosAlt   int
	lastAppliedDelta int
	lastAppliedAlt   int
	pendingDelta     int
	pendingAlt       int
	hasPending       bool
	lastSentAt       time.Time
	sendScheduled    bool
	snapEnabled      bool
	snapState        sessiond.SnapState
	snapStateAlt     sessiond.SnapState
	cursorX          int
	cursorY          int
	cursorSet        bool
}

type resizePreviewState struct {
	active       bool
	session      string
	engine       *layout.Engine
	base         *layout.Engine
	edge         resizeEdgeRef
	corner       resizeCornerRef
	cornerActive bool
}

type resizeGeomCache struct {
	version    uint64
	session    string
	selected   string
	preview    mouse.Rect
	geometry   resizeGeometry
	hasPreview bool
}

type resizeKeyState struct {
	edge      resizeEdgeRef
	hasEdge   bool
	snapState sessiond.SnapState
}
