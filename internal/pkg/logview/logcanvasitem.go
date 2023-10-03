package logview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"reprapctl/pkg/doc"
)

var _ fyne.Widget = (*logCanvasItem)(nil)

type logCanvasItem struct {
	widget.BaseWidget
	Text   *canvas.Text
	Anchor doc.Anchor
}

func newLogCanvasItem(color color.Color) *logCanvasItem {
	i := &logCanvasItem{Text: canvas.NewText("", color)}
	i.ExtendBaseWidget(i)
	return i
}

func (i *logCanvasItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(i.Text)
}

func (i *logCanvasItem) charPos(offset int) float32 {
	return i.Position().X + fyne.MeasureText(i.Text.Text[:offset], i.Text.TextSize, i.Text.TextStyle).Width
}
