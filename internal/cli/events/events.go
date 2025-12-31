package events

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

// Register registers event handlers.
func Register(reg *root.Registry) {
	reg.Register("events.watch", runWatch)
	reg.Register("events.replay", runReplay)
}

func runWatch(ctx root.CommandContext) error {
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	filters := eventFilter(ctx.Cmd.StringSlice("types"))
	streamSeq := int64(0)
	for {
		select {
		case <-ctx.Context.Done():
			return nil
		case ev, ok := <-client.Events():
			if !ok {
				return nil
			}
			if len(filters) > 0 {
				if _, ok := filters[ev.Type]; !ok {
					continue
				}
			}
			out := output.Event{ID: ev.ID, Type: string(ev.Type), TS: ev.TS, Payload: ev.Payload}
			if ctx.JSON {
				streamSeq++
				metaFrame := output.NewStreamMeta("events.watch", ctx.Deps.Version, streamSeq, false)
				if err := output.WriteSuccess(ctx.Out, metaFrame, struct {
					Event output.Event `json:"event"`
				}{Event: out}); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(ctx.Out, "%s\t%s\n", ev.TS.Format(time.RFC3339), ev.Type); err != nil {
				return err
			}
		}
	}
}

func runReplay(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("events.replay", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	since, err := parseRFC3339(ctx.Cmd.String("since"))
	if err != nil {
		return err
	}
	until, err := parseRFC3339(ctx.Cmd.String("until"))
	if err != nil {
		return err
	}
	limit := ctx.Cmd.Int("limit")
	types := ctx.Cmd.StringSlice("types")
	payloadTypes := make([]sessiond.EventType, 0, len(types))
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		payloadTypes = append(payloadTypes, sessiond.EventType(t))
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resp, err := client.EventsReplay(ctxTimeout, sessiond.EventsReplayRequest{Since: since, Until: until, Limit: limit, Types: payloadTypes})
	if err != nil {
		return err
	}
	out := make([]output.Event, 0, len(resp.Events))
	for _, ev := range resp.Events {
		out = append(out, output.Event{ID: ev.ID, Type: string(ev.Type), TS: ev.TS, Payload: ev.Payload})
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, struct {
			Events []output.Event `json:"events"`
			Total  int            `json:"total"`
		}{Events: out, Total: len(out)})
	}
	for _, ev := range out {
		if _, err := fmt.Fprintf(ctx.Out, "%s\t%s\n", ev.TS.Format(time.RFC3339), ev.Type); err != nil {
			return err
		}
	}
	return nil
}

func parseRFC3339(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func eventFilter(values []string) map[sessiond.EventType]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[sessiond.EventType]struct{})
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out[sessiond.EventType(value)] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
