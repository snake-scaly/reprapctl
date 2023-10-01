package logview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"reprapctl/internal/pkg/logview/internal/linelist"
	"sync"
)

var _ fyne.Widget = (*LogView)(nil)

type LogView struct {
	widget.BaseWidget
	textSize     float32
	textStyle    fyne.TextStyle
	wrapping     fyne.TextWrap
	lines        linelist.LineList
	propertyLock sync.RWMutex
}

func New() *LogView {
	lv := LogView{
		textSize:  theme.TextSize(),
		textStyle: fyne.TextStyle{Monospace: true},
		wrapping:  fyne.TextWrapWord,
	}
	lv.ExtendBaseWidget(&lv)
	return &lv
}

func (l *LogView) CreateRenderer() fyne.WidgetRenderer {
	return newViewRenderer(l)
}

func (l *LogView) TextSize() float32 {
	l.propertyLock.RLock()
	defer l.propertyLock.RUnlock()
	return l.textSize
}

func (l *LogView) SetTextSize(s float32) {
	l.propertyLock.Lock()
	defer l.propertyLock.Unlock()
	l.textSize = s
}

func (l *LogView) TextStyle() fyne.TextStyle {
	l.propertyLock.RLock()
	defer l.propertyLock.RUnlock()
	return l.textStyle
}

func (l *LogView) SetTextStyle(textStyle fyne.TextStyle) {
	l.propertyLock.Lock()
	defer l.propertyLock.Unlock()
	l.textStyle = textStyle
}

func (l *LogView) Wrapping() fyne.TextWrap {
	l.propertyLock.RLock()
	defer l.propertyLock.RUnlock()
	return l.wrapping
}

func (l *LogView) SetWrapping(wrapping fyne.TextWrap) {
	l.propertyLock.Lock()
	defer l.propertyLock.Unlock()
	l.wrapping = wrapping
}

func (l *LogView) AddLine(line string) {
	l.lines.Add(line)
}

var _ fyne.WidgetRenderer = (*viewRenderer)(nil)

type viewRenderer struct {
	logView  *LogView
	border   *canvas.Rectangle
	scroller *container.Scroll
	canvas   *logCanvas
}

func newViewRenderer(logView *LogView) *viewRenderer {
	border := &canvas.Rectangle{
		StrokeColor:  theme.InputBorderColor(),
		CornerRadius: theme.InputRadiusSize(),
		StrokeWidth:  theme.InputBorderSize(),
	}

	scroller := container.NewScroll(nil)

	renderer := &viewRenderer{
		logView:  logView,
		border:   border,
		scroller: scroller,
		canvas:   newLogCanvas(logView, scroller),
	}

	scroller.Content = renderer.canvas

	return renderer
}

func (r *viewRenderer) Destroy() {
}

func (r *viewRenderer) Layout(size fyne.Size) {
	r.border.Resize(size)
	r.scroller.Resize(size)
	r.canvas.Refresh()
}

func (r *viewRenderer) MinSize() fyne.Size {
	padding := theme.InnerPadding()
	return fyne.Size{Width: padding * 2, Height: padding * 2}
}

func (r *viewRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.border, r.scroller}
}

func (r *viewRenderer) Refresh() {
	r.canvas.Refresh()
}
