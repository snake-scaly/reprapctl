package yall

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
)

var _ Sink = (*FanOutSink)(nil)

// FanOutSink is a Sink that broadcasts logging events to a dynamic list of other sinks.
type FanOutSink struct {
	sinks     atomic.Value
	writeLock sync.Mutex
}

// NewFanOutSink creates a FanOutSink instance with the given initial list of sinks.
func NewFanOutSink(sinks ...Sink) *FanOutSink {
	d := &FanOutSink{}
	ss := make([]Sink, len(sinks))
	copy(ss, sinks)
	d.sinks.Store(ss)
	return d
}

// AddSink adds a Sink to the FanOutSink's list of sinks.
func (f *FanOutSink) AddSink(s Sink) {
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	sinks := f.getSinks()
	sinks = append(sinks, s)
	f.sinks.Store(sinks)
}

// RemoveSink removes the first Sink that matches s from the FanOutSink's list of sinks.
func (f *FanOutSink) RemoveSink(s Sink) bool {
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	sinks := f.getSinks()

	i := slices.Index(sinks, s)
	if i == -1 {
		return false
	}

	n := len(sinks)
	if i != n-1 {
		sinks[i] = sinks[n-1]
	}
	sinks[n-1] = nil
	sinks = sinks[:n-1]

	f.sinks.Store(sinks)
	return true
}

func (f *FanOutSink) Enabled(ctx context.Context, level slog.Level) bool {
	for _, s := range f.getSinks() {
		if s.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (f *FanOutSink) Handle(ctx context.Context, record slog.Record) error {
	sinks := f.getSinks()
	errs := make([]error, len(sinks))
	for i, s := range sinks {
		if s.Enabled(ctx, record.Level) {
			errs[i] = s.Handle(ctx, record)
		}
	}
	return errors.Join(errs...)
}

func (f *FanOutSink) getSinks() []Sink {
	return f.sinks.Load().([]Sink)
}
