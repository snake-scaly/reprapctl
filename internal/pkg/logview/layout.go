package logview

import (
	"fyne.io/fyne/v2"
	"reprapctl/pkg/alg"
	"reprapctl/pkg/doc"
)

type Box interface {
	Position() fyne.Position
	Size() fyne.Size
	StartAnchor() doc.Anchor
	EndAnchor() doc.Anchor
}

type TextBox struct {
	BoxPosition fyne.Position
	BoxSize     fyne.Size
	Start, End  doc.Anchor
	Text        string
	TextSize    float32
	TextStyle   fyne.TextStyle
}

func NewTextBox(
	pos fyne.Position,
	size fyne.Size,
	start, end doc.Anchor,
	text string,
	textSize float32,
	textStyle fyne.TextStyle,
) *TextBox {
	return &TextBox{
		BoxPosition: pos,
		BoxSize:     size,
		Start:       start,
		End:         end,
		Text:        text,
		TextSize:    textSize,
		TextStyle:   textStyle,
	}
}

func (t *TextBox) Position() fyne.Position { return t.BoxPosition }
func (t *TextBox) Size() fyne.Size         { return t.BoxSize }
func (t *TextBox) StartAnchor() doc.Anchor { return t.Start }
func (t *TextBox) EndAnchor() doc.Anchor   { return t.End }

func (t *TextBox) AnchorAtX(x float32) doc.Anchor {
	char, _ := alg.BinarySearch(len(t.Text), x, t.CharToX)
	anchor := t.Start
	anchor.LineOffset += char
	return anchor
}

func (t *TextBox) CharToX(offset int) float32 {
	return fyne.MeasureText(t.Text[:offset], t.TextSize, t.TextStyle).Width
}
