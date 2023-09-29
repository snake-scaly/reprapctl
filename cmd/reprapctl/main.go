package main

import (
	"fyne.io/fyne/v2/app"
	"reprapctl/internal/app/reprapctl"
)

func main() {
	//err := tty.ConfigureTty("/dev/ved")
	//if err != nil {
	//	log.Fatal(err)
	//}
	a := app.NewWithID("reprapctl")
	w := reprapctl.CreateMainWindow(a)
	w.ShowAndRun()
}
