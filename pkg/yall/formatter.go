package yall

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unicode"
)

// QuoteType defines the type of string quotation.
type QuoteType int

const (
	QuoteNever  = QuoteType(iota) // Do not add quotes.
	QuoteAlways                   // Always quote using [strconv.Quote].
	QuoteSmart                    // Only quote empty strings and strings containing spaces and/or equal signs.
)

// Formatter creates a text representation of a [slog.Record].
type Formatter interface {
	// Append performs the formatting, appends the result to the given buffer as a sequence
	// of UTF8 bytes, and returns the modified buffer.
	// Append may be called concurrently from multiple goroutines.
	Append(b []byte, c context.Context, r slog.Record) []byte
}

// Layout is a [Formatter] that combines other formatters in the manner of [fmt.Sprintf].
// Internally this works by passing Args to [fmt.Appendf] as []byte slices. Therefore
// only %s, %q, %x, and %X format specifiers make sense in the Format.
type Layout struct {
	Format string
	Args   []Formatter
}

func (l Layout) Append(b []byte, c context.Context, r slog.Record) []byte {
	args := make([]any, len(l.Args))

	tmp := bufferPool.Get().([]byte)[:0]
	defer func() {
		bufferPool.Put(tmp)
	}()

	for i, f := range l.Args {
		start := len(tmp)
		tmp = f.Append(tmp, c, r)
		args[i] = tmp[start:]
	}

	b = fmt.Appendf(b, l.Format, args...)
	return b
}

// Time is a [Formatter] that formats [slog.Record.Time] according to Layout.
// See [time.Layout] and related constants for a list of existing formats and a discussion
// on how to create your own.
// The zero value formats with [time.DateTime].
type Time struct {
	Layout string
}

func (t Time) Append(b []byte, _ context.Context, r slog.Record) []byte {
	if t.Layout == "" {
		return r.Time.AppendFormat(b, time.DateTime)
	}
	return r.Time.AppendFormat(b, t.Layout)
}

// Level is a [Formatter] that formats [slog.Record.Level] using [slog.Level.String].
type Level struct{}

func (l Level) Append(b []byte, _ context.Context, r slog.Record) []byte {
	return fmt.Append(b, r.Level.String())
}

// Source is a [Formatter] that formats [slog.Record.PC]. Set Short to true to only print
// source file name without path.
type Source struct {
	Short bool
}

func (s Source) Append(b []byte, _ context.Context, r slog.Record) []byte {
	fn := runtime.FuncForPC(r.PC)
	if fn == nil {
		return b
	}
	f, l := fn.FileLine(r.PC)
	if s.Short {
		f = path.Base(f)
	}
	return fmt.Append(b, f, ":", l)
}

// Message is a [Formatter] that formats [slog.Record.Message].
// Message is quoted according to Quote.
type Message struct {
	Quote QuoteType
}

func (m Message) Append(b []byte, _ context.Context, r slog.Record) []byte {
	return quote(b, r.Message, m.Quote)
}

// TextAttrs is a [Formatter] that formats [slog.Record.Attrs] as "key=value" pairs.
// Values are quoted according to Quote.
// When the result is non-empty, it includes a leading space.
type TextAttrs struct {
	Quote QuoteType
}

func (t TextAttrs) Append(b []byte, _ context.Context, r slog.Record) []byte {
	r.Attrs(func(a slog.Attr) bool {
		b = t.formatAttr(b, "", a)
		return true
	})
	return b
}

// Conditional is a [Formatter] that wraps the output of the Inner formatter in the
// [fmt.Sprintf]-like format only if Inner produces a non-empty string.
// Internally this works by passing Inner to [fmt.Appendf] as a []byte slice.
// A reasonable format will contain one format specifier, one of %s, %q, %x, or %X.
type Conditional struct {
	Format string
	Inner  Formatter
}

func (co Conditional) Append(b []byte, c context.Context, r slog.Record) []byte {
	tmp := bufferPool.Get().([]byte)[:0]
	defer func() {
		bufferPool.Put(tmp)
	}()

	tmp = co.Inner.Append(tmp, c, r)
	if len(tmp) != 0 {
		b = fmt.Appendf(b, co.Format, tmp)
	}

	return b
}

func (t TextAttrs) formatAttr(b []byte, pfx string, a slog.Attr) []byte {
	if a.Value.Kind() == slog.KindGroup {
		for _, aa := range a.Value.Group() {
			b = t.formatAttr(b, pfx+a.Key+".", aa)
		}
	} else {
		b = fmt.Append(b, " ", pfx, a.Key, "=")
		b = quote(b, a.Value.String(), t.Quote)
	}
	return b
}

// DefaultFormat returns a Formatter that mimics the default log format of slog.
func DefaultFormat() Formatter {
	return &defaultFormat
}

func quote(b []byte, s string, q QuoteType) []byte {
	if needsQuoting(s, q) {
		return strconv.AppendQuote(b, s)
	} else {
		return fmt.Append(b, s)
	}
}

func needsQuoting(s string, q QuoteType) bool {
	switch q {
	case QuoteNever:
		return false
	case QuoteAlways:
		return true
	}
	if s == "" {
		return true
	}
	for _, r := range s {
		if r == '=' || unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 0)
	},
}

var defaultFormat = Layout{
	Format: "%s %s %s%s",
	Args: []Formatter{
		Time{},
		Level{},
		Message{},
		TextAttrs{Quote: QuoteSmart},
	},
}
