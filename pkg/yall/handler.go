package yall

import (
	"context"
	"log/slog"
)

// NewHandler creates an implementation of slog.Handler that sends logging events to a Sink.
//
// The handler takes care of [slog.Handler.WithAttrs] and [slog.Handler.WithGroup] and always
// sends a complete slog.Record to the Sink. The Sink still needs to resolve and handle the attrs.
func NewHandler(sink Sink) slog.Handler {
	return &sinkHandler{sink: sink}
}

type sinkHandler struct {
	sink Sink
}

func (h *sinkHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.sink.Enabled(ctx, level)
}

func (h *sinkHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.sink.Handle(ctx, record)
}

func (h *sinkHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return withAttrs(h, attrs)
}

func (h *sinkHandler) WithGroup(name string) slog.Handler {
	return withGroup(h, name)
}

type attrsHandler struct {
	next  slog.Handler
	attrs []slog.Attr
}

func withAttrs(next slog.Handler, attrs []slog.Attr) slog.Handler {
	empty := countEmptyGroups(attrs)
	if empty == len(attrs) {
		// attrs is empty or consists exclusively of empty groups
		return next
	}
	if empty != 0 {
		// only leave non-empty attrs
		aa := make([]slog.Attr, 0, len(attrs)-empty)
		for _, a := range attrs {
			if !isEmptyGroup(a) {
				aa = append(aa, a)
			}
		}
		attrs = aa
	}
	return &attrsHandler{next: next, attrs: attrs}
}

func countEmptyGroups(attrs []slog.Attr) (count int) {
	for _, a := range attrs {
		if isEmptyGroup(a) {
			count++
		}
	}
	return
}

func isEmptyGroup(attr slog.Attr) bool {
	if attr.Value.Kind() != slog.KindGroup {
		return false
	}
	g := attr.Value.Group()
	return len(g) == countEmptyGroups(g)
}

func (h *attrsHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *attrsHandler) Handle(ctx context.Context, record slog.Record) error {
	record.AddAttrs(h.attrs...)
	return h.next.Handle(ctx, record)
}

func (h *attrsHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return withAttrs(h, attrs)
}

func (h *attrsHandler) WithGroup(name string) slog.Handler {
	return withGroup(h, name)
}

type groupHandler struct {
	next slog.Handler
	name string
}

func withGroup(next slog.Handler, name string) slog.Handler {
	if name == "" {
		return next
	}
	return &groupHandler{next: next, name: name}
}

func (h *groupHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *groupHandler) Handle(ctx context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	r := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	r.AddAttrs(slog.Attr{Key: h.name, Value: slog.GroupValue(attrs...)})
	return h.next.Handle(ctx, r)
}

func (h *groupHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return withAttrs(h, attrs)
}

func (h *groupHandler) WithGroup(name string) slog.Handler {
	return withGroup(h, name)
}
