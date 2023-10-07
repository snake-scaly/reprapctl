package main

import (
	"flag"
	"fyne.io/fyne/v2/app"
	"log"
	"os"
	"reprapctl/internal/app/reprapctl"
	"runtime/pprof"
)

var cpuprofile = flag.String("cpuprofile", "", "write CPU profile to `file`")

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("failed to create CPU profile: ", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Print("failed to close CPU profile: ", err)
			}
		}()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("failed to start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	// err := tty.ConfigureTty("/dev/ved")
	// if err != nil {
	//	log.Fatal(err)
	// }

	a := app.NewWithID("reprapctl")
	w := reprapctl.CreateMainWindow(a)
	w.ShowAndRun()
}
