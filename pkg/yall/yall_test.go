package yall_test

import (
	"bytes"
	"log/slog"
	"math"
	"os"
	"reprapctl/pkg/yall"
)

func Example() {
	// This example configures YALL to log into two different targets,
	// with different formats and configurations.

	// Configure the STDOUT sink to log in "[  LVL] Message: args" format,
	// ignoring events with levels below Info, with smart quoting of arg values.
	stdoutSink := yall.WriterSink{
		Writer: os.Stdout,
		Level:  slog.LevelInfo,
		Format: &yall.Layout{
			Format: "[%5s] %s%s",
			Args: []yall.Formatter{
				yall.Level{},
				yall.Message{},
				yall.Conditional{
					Format: ":%s",
					Inner:  yall.TextAttrs{Quote: yall.QuoteSmart},
				},
			},
		},
	}

	// Configure an in-memory sink to log in a format similar to slog.TextHandler.
	// Accept events of all levels.
	memoryTarget := bytes.Buffer{}
	memorySink := yall.WriterSink{
		Writer: &memoryTarget,
		Level:  slog.Level(math.MinInt),
		Format: &yall.Layout{
			Format: "time=%s level=%s source=%s msg=%s%s",
			Args: []yall.Formatter{
				yall.Time{Layout: "2006-01-02T15:04:05.999Z07:00"},
				yall.Level{},
				yall.Source{},
				yall.Message{Quote: yall.QuoteSmart},
				yall.TextAttrs{Quote: yall.QuoteSmart},
			},
		},
	}

	// Create a FanOutSink to send logging events to both target sinks.
	fanOutSink := yall.NewFanOutSink(&stdoutSink, &memorySink)

	// Handler takes care of args and groups so that you don't have to.
	handler := yall.NewHandler(fanOutSink)

	// Finnally, all this plugs into the standard Logger.
	logger := slog.New(handler)
	logger.Info("Just a message")

	// Logger can be used as normal.
	logger2 := logger.With("outer", "exampli gratia").WithGroup("inner")
	logger2.Error("Message with args", "answer", 42)

	// Output:
	// [ INFO] Just a message
	// [ERROR] Message with args: inner.answer=42 outer="exampli gratia"
}
