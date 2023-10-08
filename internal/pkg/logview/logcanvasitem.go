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
	box  *TextBox
	text *canvas.Text
	size fyne.Size
}

func newLogCanvasItem(color color.Color) *logCanvasItem {
	i := &logCanvasItem{text: canvas.NewText("", color)}
	i.ExtendBaseWidget(i)
	return i
}

func (i *logCanvasItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(i.text)
}

func (i *logCanvasItem) MinSize() fyne.Size { return i.size }

func (i *logCanvasItem) Set(box *TextBox) {
	i.box = box
	i.text.Text = box.Text
	i.text.TextSize = box.TextSize
	i.text.TextStyle = box.TextStyle
	i.size = i.text.MinSize()
}

func (i *logCanvasItem) AnchorAtX(x float32) doc.Anchor {
	char, _ := alg.BinarySearch(len(i.box.Text), x, i.CharToX)
	anchor := i.box.StartAnchor()
	anchor.LineOffset += char
	return anchor
}

func (i *logCanvasItem) CharToX(offset int) float32 {
	return fyne.MeasureText(i.text.Text[:offset], i.text.TextSize, i.text.TextStyle).Width
}

var _ fyne.Widget = (*logSelectionRect)(nil)

type logSelectionRect struct {
	widget.BaseWidget
	rect *canvas.Rectangle
	tag  int
}

func newLogSelectionRect() *logSelectionRect {
	r := &logSelectionRect{
		rect: canvas.NewRectangle(color.Black),
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *logSelectionRect) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.rect)
}
