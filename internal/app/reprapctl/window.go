package reprapctl

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"log/slog"
	"reprapctl/internal/pkg/logview"
	"reprapctl/pkg/yall"
	"time"
)

func CreateMainWindow(app fyne.App, logger *slog.Logger, logFanOut *yall.FanOutSink) fyne.Window {
	w := app.NewWindow("RepRap Control")
	w.Resize(fyne.Size{Width: 800, Height: 600})
	m := fyne.MainMenu{
		Items: []*fyne.Menu{
			{
				Label: "File",
				Items: []*fyne.MenuItem{{Label: "Exit", IsQuit: true}},
			},
		},
	}
	w.SetMainMenu(&m)
	logView := logview.New()
	logView.SetCapacity(2000)
	logView.AddLine("foo")
	logView.AddLine("bar")
	logView.AddLine("baz")
	logView.AddLine(
		"        Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt " +
			"ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco " +
			"laboris nisi ut aliquip ex ea commodo consequat.\n" +
			"        Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat " +
			"nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia " +
			"deserunt mollit anim id est laborum.")
	var i int
	for ; i < 1990; i++ {
		logView.AddLine(fmt.Sprint("Line #", i))
	}

	lvs := &logViewSink{logView: logView}
	logFanOut.AddSink(lvs)
	w.SetOnClosed(func() {
		logFanOut.RemoveSink(lvs)
	})

	go func() {
		for {
			<-time.After(1 * time.Second)
			logger.Info("Line", "n", i)
			i++
		}
	}()
	w.SetContent(logView)
	return w
}

type logViewSink struct {
	logView *logview.LogView
}

func (s *logViewSink) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelInfo
}

func (s *logViewSink) Handle(context context.Context, record slog.Record) error {
	var buf []byte
	buf = viewLogFormatter.Append(buf, context, record)
	s.logView.AddLine(string(buf))
	s.logView.Refresh()
	return nil
}

var viewLogFormatter = yall.Layout{
	Format: "%s: %s%s",
	Args: []yall.Formatter{
		&viewLogLevelFormatter{},
		&yall.Message{},
		&yall.TextAttrs{Quote: yall.QuoteSmart},
	},
}

type viewLogLevelFormatter struct{}

func (w viewLogLevelFormatter) Append(b []byte, _ context.Context, r slog.Record) []byte {
	return fmt.Append(b, r.Level.String()[:1])
}
