package sessiond

import (
	"github.com/regenrek/peakypanes/internal/sessionrestore"
	"github.com/regenrek/peakypanes/internal/termframe"
)

func offlinePaneView(req PaneViewRequest, snap sessionrestore.PaneSnapshot) PaneViewResponse {
	cols := req.Cols
	rows := req.Rows
	if cols <= 0 {
		cols = snap.Terminal.Cols
	}
	if rows <= 0 {
		rows = snap.Terminal.Rows
	}
	lines := sessionrestore.RenderPlainLines(cols, rows, snap.Terminal.ScrollbackLines, snap.Terminal.ScreenLines)
	frame := termframe.FrameFromLines(cols, rows, lines)
	return PaneViewResponse{
		PaneID:      snap.PaneID,
		Cols:        cols,
		Rows:        rows,
		UpdateSeq:   0,
		NotModified: false,
		Frame:       frame,
		HasMouse:    false,
		AllowMotion: false,
	}
}
