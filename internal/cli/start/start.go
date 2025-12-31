package start

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/dashboard"
	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/app"
)

// Register registers start/open handlers.
func Register(reg *root.Registry) {
	reg.Register("start", runStart)
}

func runStart(ctx root.CommandContext) error {
	layout := strings.TrimSpace(ctx.Cmd.String("layout"))
	session := strings.TrimSpace(ctx.Cmd.String("session"))
	path := strings.TrimSpace(ctx.Cmd.String("path"))
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		path = cwd
	}
	if ctx.JSON {
		start := time.Now()
		meta := output.NewMeta("start", ctx.Deps.Version)
		connect := ctx.Deps.Connect
		if connect == nil {
			return fmt.Errorf("daemon connection not configured")
		}
		ctxTimeout, cancel := context.WithTimeout(ctx.Context, 10*time.Second)
		client, err := connect(ctxTimeout, ctx.Deps.Version)
		if err != nil {
			cancel()
			return err
		}
		defer func() {
			cancel()
			_ = client.Close()
		}()
		resp, err := client.StartSession(ctxTimeout, sessiond.StartSessionRequest{
			Name:       session,
			Path:       path,
			LayoutName: layout,
		})
		if err != nil {
			return err
		}
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "start",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "session", ID: resp.Name}},
			Details: map[string]any{
				"name":   resp.Name,
				"path":   resp.Path,
				"layout": resp.LayoutName,
			},
		})
	}
	autoStart := &app.AutoStartSpec{
		Session: session,
		Path:    path,
		Layout:  layout,
		Focus:   true,
	}
	return dashboard.Run(ctx, autoStart)
}
