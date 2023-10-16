/*
Package yall is Yet Another Logging Library, a slog backend.

# Introduction

[slog] has a convenient [slog.Logger] API, but the [slog.Handler] backend is cumbersome
and hard to implement. The handler implementations included in the slog library
attempt to do 3 different things at once:

 1. Keep ambient state introduced by WithAttrs and WithGroup.
 2. Format each [slog.Record] into its text representation.
 3. Send the formatted results to the log output.

As a result standard implementations are complicated and impossible to reuse or extend.

YALL embraces the architecture provided by the slog package but implements the three
tasks mentioned above separately.

# Formatter

YALL formatters are simple objects with a single function Append. Append takes all
the same arguments as [slog.Handler.Handle]: a [context.Context] and a [slog.Record],
and a byte slice. Its job is to append the formatting results to the slice. This enables
easy reuse of both existing and custom formatters in the system.

But this is not even the best part. The best part is, a [Formatter] does not have to
necessarily format the whole record. A formatter can choose to specialize on a single
attribute or a group of attributes. This allows to compose formatters in a very
flexible way.

YALL provides a set of built-in formatters:

  - [Time] formats [slog.Record.Time] according to the specified format.
  - [Level] formats [slog.Record.Level] using [slog.Level.String].
  - [Source] formats [slog.Record.PC] in long or short format.
  - [Message] formats [slog.Record.Message] with optional quoting.
  - [TextAttrs] formats [slog.Record.Attrs] in key=value format with optional value quoting.
  - [Layout] composes other formatters in a manner of [fmt.Sprintf].
  - [Conditional] is similar to [Layout] for one argument which only produces output
    if the inner formatter result is non-empty.

Layout is where the real power of this design comes in. For example, here's a formatter
which formats a record exactly how [slog.TextHandler] does it with source logging enabled:

	l := &yall.Layout{
		Format: "time=%s level=%s source=%s msg=%s%s",
		Args: []yall.Formatter{
			yall.Time{Layout: "2006-01-02T15:04:05.999Z07:00"},
			yall.Level{},
			yall.Source{},
			yall.Message{Quote: yall.QuoteSmart},
			yall.TextAttrs{Quote: yall.QuoteSmart},
		},
	}

Here's another, simpler example, but this time the level is formatted to always take
5 characters aligned to the right, e.g. [ERROR] or [ INFO]:

	l := &yall.Layout{
		Format: "[%5s] %s%s",
		Args: []yall.Formatter{
			yall.Level{},
			yall.Message{},
			yall.Conditional{
				Format: ":%s",
				Inner:  yall.TextAttrs{Quote: yall.QuoteSmart},
			},
		},
	}

The [DefaultFormat] function creates a formatter to produce logs that look like

	2020-11-22 12:34:56 INFO Long message foo=bar baz="quote me"

# Sink

[Sink] is responsible for delivering log records to the destination, be it console,
a file, or a remote server. In theory, a sink can interpret the records in any way
it wants. In YALL, sinks are encouraged to use formatters to convert records to text.

The [Sink] interface is compatible with [slog.Handler]. This means that any existing
handler can be used as a sink if desired, e.g. slog.Default().Handler().

YALL provides the following sinks:

  - [FanOutSink] broadcasts log records to any number of other sinks. The list of
    target sinks can be modified at run time.
  - [WriterSink] writes records formatted by any [Formatter] to any [io.Writer].

# Handler

YALL provides an implementation of [slog.Handler] which takes care of additional
attributes and groups imposed by [slog.Logger.With] and [slog.Logger.WithGroup].
It updates each incoming [slog.Record] accordingly, and sends the final, completed
records to a [Sink] which typically will be a [FanOutSink], but can also be
a [WriterSink] or even one of the existing slog handlers like [slog.TextHandler].

Use the [NewHandler] function to create an instance of the handler.
*/
package yall
