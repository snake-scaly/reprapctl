package yall_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"reprapctl/pkg/yall"
	"testing"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name string
		with func(slog.Handler) slog.Handler
		want slog.Record
	}{
		{
			name: "Direct",
			with: func(h slog.Handler) slog.Handler {
				return h
			},
			want: rec("n", "v"),
		},
		{
			name: "WithAttrs",
			with: func(h slog.Handler) slog.Handler {
				return h.WithAttrs([]slog.Attr{slog.String("x", "y")})
			},
			want: rec("n", "v", "x", "y"),
		},
		{
			name: "WithAttrsWithAttrs",
			with: func(h slog.Handler) slog.Handler {
				h = h.WithAttrs([]slog.Attr{slog.String("x", "y")})
				h = h.WithAttrs([]slog.Attr{slog.String("z", "w")})
				return h
			},
			want: rec("n", "v", "z", "w", "x", "y"),
		},
		{
			name: "WithGroup",
			with: func(h slog.Handler) slog.Handler {
				return h.WithGroup("g")
			},
			want: rec(slog.Group("g", "n", "v")),
		},
		{
			name: "WithGroupWithGroup",
			with: func(h slog.Handler) slog.Handler {
				h = h.WithGroup("g")
				h = h.WithGroup("h")
				return h
			},
			want: rec(slog.Group("g", slog.Group("h", "n", "v"))),
		},
		{
			name: "WithAttrsWithGroup",
			with: func(h slog.Handler) slog.Handler {
				h = h.WithAttrs([]slog.Attr{slog.String("x", "y")})
				h = h.WithGroup("g")
				return h
			},
			want: rec(slog.Group("g", "n", "v"), "x", "y"),
		},
		{
			name: "WithGroupWithAttrs",
			with: func(h slog.Handler) slog.Handler {
				h = h.WithGroup("g")
				h = h.WithAttrs([]slog.Attr{slog.String("x", "y")})
				return h
			},
			want: rec(slog.Group("g", "n", "v", "x", "y")),
		},
		{
			name: "WithAttrsEmpty",
			with: func(h slog.Handler) slog.Handler {
				return h.WithAttrs([]slog.Attr{})
			},
			want: rec("n", "v"),
		},
		{
			name: "WithAttrsEmptyGroup",
			with: func(h slog.Handler) slog.Handler {
				return h.WithAttrs([]slog.Attr{
					slog.String("x", "y"),
					slog.Group("g"),
					slog.String("v", "w"),
				})
			},
			want: rec("n", "v", "x", "y", "v", "w"),
		},
		{
			name: "WithGroupEmptyName",
			with: func(h slog.Handler) slog.Handler {
				return h.WithGroup("")
			},
			want: rec("n", "v"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &testSink{enabled: true, err: nil}
			h := yall.NewHandler(s)
			h = tt.with(h)
			e := h.Handle(someCtx, rec("n", "v"))
			assert.Nil(t, e)
			assert.Equal(t, 1, len(s.calls))
			assert.Equal(t, someCtx, s.calls[0].ctx)
			assert.Equal(t, tt.want, s.calls[0].record)
		})
	}
}

func TestHandler_Enabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "Enabled",
			enabled: true,
		},
		{
			name:    "Disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &testSink{enabled: tt.enabled}
			h := yall.NewHandler(s)

			t.Run("Direct", func(t *testing.T) {
				e := h.Enabled(nil, slog.LevelInfo)
				assert.Equal(t, tt.enabled, e)
			})

			t.Run("WithAttrs", func(t *testing.T) {
				h = h.WithAttrs([]slog.Attr{slog.String("a", "b")})
				e := h.Enabled(nil, slog.LevelInfo)
				assert.Equal(t, tt.enabled, e)
			})

			t.Run("WithGroup", func(t *testing.T) {
				h = h.WithGroup("g")
				e := h.Enabled(nil, slog.LevelInfo)
				assert.Equal(t, tt.enabled, e)
			})
		})
	}
}

type testSink struct {
	enabled bool
	err     error
	calls   []call
}

type call struct {
	ctx    context.Context
	record slog.Record
}

func (s *testSink) Enabled(_ context.Context, _ slog.Level) bool {
	return s.enabled
}

func (s *testSink) Handle(ctx context.Context, record slog.Record) error {
	s.calls = append(s.calls, call{ctx, record})
	return s.err
}
