package reprapctl

import (
	"fmt"
	"fyne.io/fyne/v2"
	"reprapctl/internal/pkg/logview"
	"strings"
)

func NewTerminal() fyne.CanvasObject {
	b := strings.Builder{}
	for i := 0; i < 35; i++ {
		b.WriteString(fmt.Sprintf("Line %v\n", i))
	}

	//return container.NewScroll(widget.NewLabel(b.String()))

	//sb := binding.NewString()
	//sb.Set(b.String())
	//
	//e := widget.NewEntry()
	//e.MultiLine = true
	//e.Bind(sb)
	//return e

	//richText := widget.NewRichTextWithText(b.String())
	//textGrid := widget.NewTextGridFromString(b.String())
	//return container.NewScroll(textGrid)

	return logview.New()
}
