package input

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/muesli/cancelreader"
)

type Sender interface {
	Send(tea.Msg)
}

type Stream struct {
	cancel func()
	wg     sync.WaitGroup
	cr     cancelreader.CancelReader
}

func Start(ctx context.Context, sender Sender, r io.Reader, termType string) (*Stream, error) {
	if sender == nil || r == nil {
		return &Stream{cancel: func() {}}, nil
	}
	cr, err := cancelreader.NewReader(r)
	if err != nil {
		return nil, err
	}

	if termType == "" {
		termType = os.Getenv("TERM")
	}

	reader := uv.NewTerminalReader(cr, termType)
	eventc := make(chan uv.Event, 256)

	streamCtx, cancel := context.WithCancel(ctx)
	s := &Stream{cancel: cancel, cr: cr}

	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		err := reader.StreamEvents(streamCtx, eventc)
		if err == nil || errors.Is(err, io.EOF) || errors.Is(err, cancelreader.ErrCanceled) || errors.Is(err, context.Canceled) {
			return
		}
		sender.Send(tea.Quit())
	}()
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-streamCtx.Done():
				return
			case ev := <-eventc:
				if msg, ok := toTeaMsg(ev); ok && msg != nil {
					sender.Send(msg)
				}
			}
		}
	}()

	return s, nil
}

func (s *Stream) Stop() {
	if s == nil {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.cr != nil {
		s.cr.Cancel()
		_ = s.cr.Close()
	}
	s.wg.Wait()
}
