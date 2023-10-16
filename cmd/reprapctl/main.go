package main

import (
	"flag"
	"fmt"
	"fyne.io/fyne/v2/app"
	"log/slog"
	"math"
	"os"
	"reprapctl/internal/app/reprapctl"
	"reprapctl/pkg/yall"
	"runtime/pprof"
)

var cpuprofile = flag.String("cpuprofile", "", "write CPU profile to `file`")

func main() {
	consoleSink := yall.WriterSink{
		Writer: os.Stdout,
		Level:  slog.Level(math.MinInt),
		Format: yall.DefaultFormat(),
	}
	fanOutSink := yall.NewFanOutSink(&consoleSink)
	handler := yall.NewHandler(fanOutSink)
	logger := slog.New(handler)
	defer os.Stdout.Sync()

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(fmt.Sprint("failed to create CPU profile: ", err))
		}
		defer func() {
			if err := f.Close(); err != nil {
				logger.Warn("failed to close CPU profile", "err", err)
			}
		}()
		if err := pprof.StartCPUProfile(f); err != nil {
			panic(fmt.Sprint("failed to start CPU profile: ", err))
		}
		defer pprof.StopCPUProfile()
	}

	// err := tty.ConfigureTty("/dev/ved")
	// if err != nil {
	//	log.Fatal(err)
	// }

	a := app.NewWithID("reprapctl")
	w := reprapctl.CreateMainWindow(a, logger, fanOutSink)
	w.ShowAndRun()
}
