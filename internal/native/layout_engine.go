package native

import (
	"fmt"

	"github.com/regenrek/peakypanes/internal/layout"
)

func buildLayoutEngine(layoutCfg *layout.LayoutConfig, panes []*Pane) (*layout.Engine, error) {
	ids := make([]string, 0, len(panes))
	for _, pane := range panes {
		if pane == nil {
			continue
		}
		ids = append(ids, pane.ID)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("native: layout needs panes")
	}
	tree, err := layout.BuildTree(layoutCfg, ids)
	if err != nil {
		return nil, err
	}
	return layout.NewEngine(tree), nil
}

func applyLayoutToPanes(session *Session) error {
	if session == nil || session.Layout == nil || session.Layout.Tree == nil {
		return nil
	}
	rects := session.Layout.Tree.Rects()
	for _, pane := range session.Panes {
		if pane == nil {
			continue
		}
		rect, ok := rects[pane.ID]
		if !ok {
			return fmt.Errorf("native: layout missing pane %q", pane.ID)
		}
		pane.Left = rect.X
		pane.Top = rect.Y
		pane.Width = rect.W
		pane.Height = rect.H
	}
	return nil
}
