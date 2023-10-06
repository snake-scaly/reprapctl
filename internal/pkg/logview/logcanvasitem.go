package logview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"reprapctl/pkg/alg"
	"reprapctl/pkg/doc"
)

var _ fyne.Widget = (*logCanvasItem)(nil)

type logCanvasItem struct {
	widget.BaseWidget
	Anchor doc.Anchor
	text   *canvas.Text
	size   fyne.Size
}

func newLogCanvasItem(color color.Color) *logCanvasItem {
	i := &logCanvasItem{text: canvas.NewText("", color)}
	i.ExtendBaseWidget(i)
	return i
}

func (i *logCanvasItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(i.text)
}

func (i *logCanvasItem) MinSize() fyne.Size {
	return i.size
}

func (i *logCanvasItem) SetText(text string, size float32, style fyne.TextStyle) {
	i.text.Text = text
	i.text.TextSize = size
	i.text.TextStyle = style
	i.size = i.text.MinSize()
}

func (i *logCanvasItem) XToChar(x float32) int {
	char, _ := alg.BinarySearch(len(i.text.Text), x, i.CharToX)
	return char
}

func (i *logCanvasItem) CharToX(offset int) float32 {
	return fyne.MeasureText(i.text.Text[:offset], i.text.TextSize, i.text.TextStyle).Width
}

func (i *logCanvasItem) Text() string {
	return i.text.Text
}
