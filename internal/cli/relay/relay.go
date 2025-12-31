package relay

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

// Register registers relay handlers.
func Register(reg *root.Registry) {
	reg.Register("relay.create", runCreate)
	reg.Register("relay.list", runList)
	reg.Register("relay.stop", runStop)
	reg.Register("relay.stop-all", runStopAll)
}

func runCreate(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("relay.create", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	cfg := sessiond.RelayConfig{
		FromPaneID: strings.TrimSpace(ctx.Cmd.String("from")),
		ToPaneIDs:  ctx.Cmd.StringSlice("to"),
		Scope:      strings.TrimSpace(ctx.Cmd.String("scope")),
		Mode:       sessiond.RelayMode(strings.TrimSpace(ctx.Cmd.String("mode"))),
		Delay:      ctx.Cmd.Duration("delay"),
		Prefix:     strings.TrimSpace(ctx.Cmd.String("prefix")),
		TTL:        ctx.Cmd.Duration("ttl"),
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resp, err := client.RelayCreate(ctxTimeout, cfg)
	if err != nil {
		return err
	}
	relay := relayFromInfo(resp)
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, struct {
			Relay output.Relay `json:"relay"`
		}{Relay: relay})
	}
	if _, err := fmt.Fprintf(ctx.Out, "Relay %s created\n", relay.ID); err != nil {
		return err
	}
	return nil
}

func runList(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("relay.list", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resp, err := client.RelayList(ctxTimeout)
	if err != nil {
		return err
	}
	relays := make([]output.Relay, 0, len(resp))
	for _, info := range resp {
		relays = append(relays, relayFromInfo(info))
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, struct {
			Relays []output.Relay `json:"relays"`
			Total  int            `json:"total"`
		}{Relays: relays, Total: len(relays)})
	}
	for _, relay := range relays {
		if _, err := fmt.Fprintln(ctx.Out, relay.ID); err != nil {
			return err
		}
	}
	return nil
}

func runStop(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("relay.stop", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	id := strings.TrimSpace(ctx.Cmd.String("id"))
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if err := client.RelayStop(ctxTimeout, id); err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "relay.stop",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "relay", ID: id}},
		})
	}
	return nil
}

func runStopAll(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("relay.stop-all", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if err := client.RelayStopAll(ctxTimeout); err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action: "relay.stop-all",
			Status: "ok",
		})
	}
	return nil
}

func relayFromInfo(info sessiond.RelayInfo) output.Relay {
	return output.Relay{
		ID:         info.ID,
		FromPaneID: info.FromPane,
		ToPaneIDs:  append([]string(nil), info.ToPanes...),
		Scope:      info.Scope,
		Mode:       string(info.Mode),
		Status:     string(info.Status),
		Delay:      durationString(info.Delay),
		Prefix:     info.Prefix,
		TTL:        durationString(info.TTL),
		CreatedAt:  info.CreatedAt,
		Stats: output.RelayStats{
			Lines:        info.Stats.Lines,
			Bytes:        info.Stats.Bytes,
			LastActivity: info.Stats.LastActivity,
		},
	}
}

func durationString(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return d.String()
}

func connect(ctx root.CommandContext) (*sessiond.Client, func(), error) {
	connect := ctx.Deps.Connect
	if connect == nil {
		return nil, func() {}, fmt.Errorf("daemon connection not configured")
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	client, err := connect(ctxTimeout, ctx.Deps.Version)
	if err != nil {
		cancel()
		return nil, func() {}, err
	}
	cleanup := func() {
		cancel()
		_ = client.Close()
	}
	return client, cleanup, nil
}

func commandTimeout(ctx root.CommandContext) time.Duration {
	if ctx.Cmd.IsSet("timeout") {
		return ctx.Cmd.Duration("timeout")
	}
	return 10 * time.Second
}
