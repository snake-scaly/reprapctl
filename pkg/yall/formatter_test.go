package yall_test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"reprapctl/pkg/yall"
	"testing"
)

func TestLayout_Append(t *testing.T) {
	f := yall.Layout{
		"%s %s %s",
		[]yall.Formatter{&testFormatter{"a"}, &testFormatter{"b"}, &testFormatter{"c"}},
	}
	s := formatToString(&f, nil, slog.Record{})
	assert.Equal(t, "a b c", s)
}

func TestSource_Append(t *testing.T) {
	f := yall.Source{}
	s := formatToString(&f, nil, rec())
	assert.Contains(t, s, "reprapctl/pkg/yall/util_test.go:14")
}

func TestTextAttrs_Append(t *testing.T) {
	tests := []struct {
		name string
		rec  slog.Record
		quot yall.QuoteType
		want string
	}{
		{
			name: "Empty",
			rec:  rec(),
			want: "",
		},
		{
			name: "Flat",
			rec:  rec("a", "b", "c", "d"),
			want: " a=b c=d",
		},
		{
			name: "Grouped",
			rec:  rec("a", "b", slog.Group("g", "x", "y", slog.Group("h", "v", "w")), "c", "d"),
			want: " a=b g.x=y g.h.v=w c=d",
		},
		{
			name: "QuoteNever",
			rec:  rec("a", "b = c"),
			quot: yall.QuoteNever,
			want: " a=b = c",
		},
		{
			name: "QuoteAlways",
			rec:  rec("a", "bcd"),
			quot: yall.QuoteAlways,
			want: " a=\"bcd\"",
		},
		{
			name: "QuoteSmartEmpty",
			rec:  rec("a", ""),
			quot: yall.QuoteSmart,
			want: " a=\"\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ta := yall.TextAttrs{tt.quot}
			s := formatToString(&ta, nil, tt.rec)
			assert.Equal(t, tt.want, s)
		})
	}
}

func TestConditional_Append(t *testing.T) {
	tests := []struct {
		name   string
		format string
		inner  yall.Formatter
		want   string
	}{
		{
			name:   "InnerEmpty",
			format: "a%sb",
			inner:  testFormatter{""},
			want:   "",
		},
		{
			name:   "InnerNonEmpty",
			format: "a%sb",
			inner:  testFormatter{"c"},
			want:   "acb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := yall.Conditional{tt.format, tt.inner}
			s := formatToString(c, nil, rec())
			assert.Equal(t, tt.want, s)
		})
	}
}

type testFormatter struct {
	str string
}

func (f testFormatter) Append(b []byte, _ context.Context, _ slog.Record) []byte {
	return fmt.Append(b, f.str)
}

func formatToString(f yall.Formatter, c context.Context, r slog.Record) string {
	b := make([]byte, 0)
	b = f.Append(b, c, r)
	return string(b)
}
