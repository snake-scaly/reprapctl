package yall

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

// Sink receives complete logging events.
// Note that any slog.Handler can be used as a Sink, e.g.
//
//	var s Sink = slog.Default().Handler()
type Sink interface {
	Enabled(c context.Context, l slog.Level) bool
	Handle(c context.Context, r slog.Record) error
}

var _ Sink = (*WriterSink)(nil)

// WriterSink is a sink that writes logs to an io.Writer.
// Each log event is terminated with a new line and is written as a single write on the Writer.
type WriterSink struct {
	Writer io.Writer
	Level  slog.Leveler
	Format Formatter
	buffer []byte
	lock   sync.Mutex
}

func (s *WriterSink) Enabled(_ context.Context, l slog.Level) bool {
	return l >= s.Level.Level()
}

func (s *WriterSink) Handle(c context.Context, r slog.Record) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.buffer = s.Format.Append(s.buffer[:0], c, r)
	s.buffer = append(s.buffer, '\n')
	_, err := s.Writer.Write(s.buffer)
	return err
}
