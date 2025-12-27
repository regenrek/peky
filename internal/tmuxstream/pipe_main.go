package tmuxstream

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

type PipeOptions struct {
	SocketPath string
	Token      string
	PaneID     string
}

func RunPipe(ctx context.Context, opts PipeOptions) error {
	socket := strings.TrimSpace(opts.SocketPath)
	token := strings.TrimSpace(opts.Token)
	paneID := strings.TrimSpace(opts.PaneID)
	if socket == "" || token == "" || paneID == "" {
		return fmt.Errorf("pipe: socket, token, and pane-id are required")
	}

	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(ctx, "unix", socket)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	if _, err := fmt.Fprintf(conn, "PP1 %s %s\n", token, paneID); err != nil {
		return err
	}

	_, err = io.Copy(conn, os.Stdin)
	return err
}
