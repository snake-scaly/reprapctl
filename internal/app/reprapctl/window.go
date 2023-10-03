package reprapctl

import (
	"fmt"
	"fyne.io/fyne/v2"
	"reprapctl/internal/pkg/logview"
	"time"
)

func CreateMainWindow(app fyne.App) fyne.Window {
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
	logView.SetCapacity(50)
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
	go func() {
		var i int
		for {
			<-time.After(1 * time.Second)
			logView.AddLine(fmt.Sprintf("Line %v", i))
			logView.Refresh()
			i++
		}
	}()
	w.SetContent(logView)
	return w
}
