package sessiond

import "github.com/regenrek/peakypanes/internal/sessionrestore"

func offlinePaneView(req PaneViewRequest, snap sessionrestore.PaneSnapshot) PaneViewResponse {
	cols := req.Cols
	rows := req.Rows
	if cols <= 0 {
		cols = snap.Terminal.Cols
	}
	if rows <= 0 {
		rows = snap.Terminal.Rows
	}
	view := sessionrestore.RenderPlainView(cols, rows, snap.Terminal.ScrollbackLines, snap.Terminal.ScreenLines)
	return PaneViewResponse{
		PaneID:       snap.PaneID,
		Cols:         cols,
		Rows:         rows,
		Mode:         PaneViewANSI,
		ShowCursor:   false,
		ColorProfile: req.ColorProfile,
		UpdateSeq:    0,
		NotModified:  false,
		View:         view,
		HasMouse:     false,
		AllowMotion:  false,
	}
}
