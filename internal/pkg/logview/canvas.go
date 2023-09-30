package logview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"sync"
	"sync/atomic"
)

var _ fyne.Widget = (*logCanvas)(nil)

type logCanvas struct {
	widget.BaseWidget
	logView        *LogView
	parentScroller *container.Scroll
}

func newLogCanvas(logView *LogView, parentScroller *container.Scroll) *logCanvas {
	c := &logCanvas{logView: logView, parentScroller: parentScroller}
	c.ExtendBaseWidget(c)
	return c
}

func (l *logCanvas) CreateRenderer() fyne.WidgetRenderer {
	return newLogCanvasRenderer(l.logView, l.parentScroller)
}

var _ fyne.WidgetRenderer = (*logCanvasRenderer)(nil)

type logCanvasRenderer struct {
	logView        *LogView
	parentScroller *container.Scroll
	wrappedLines   atomic.Value
	itemCache      sync.Pool
	visibleItems   map[int]*canvas.Text
	itemLock       sync.RWMutex
	objects        atomic.Value
}

func newLogCanvasRenderer(logView *LogView, parentScroller *container.Scroll) *logCanvasRenderer {
	r := &logCanvasRenderer{
		logView:        logView,
		parentScroller: parentScroller,
		visibleItems:   make(map[int]*canvas.Text),
	}

	r.itemCache.New = func() any {
		return &canvas.Text{Color: theme.ForegroundColor()}
	}

	parentScroller.OnScrolled = func(_ fyne.Position) {
		r.Refresh()
	}

	r.wrappedLines.Store(make([]DocumentFragment, 0))
	r.objects.Store(make([]fyne.CanvasObject, 0))

	return r
}

func (r *logCanvasRenderer) Destroy() {
}

func (r *logCanvasRenderer) Layout(_ fyne.Size) {
}

func (r *logCanvasRenderer) MinSize() fyne.Size {
	var minItemWidth float32
	func() {
		r.itemLock.RLock()
		defer r.itemLock.RUnlock()
		for _, item := range r.visibleItems {
			minItemWidth = max(minItemWidth, item.MinSize().Width)
		}
	}()

	itemHeight := r.itemHeight()
	itemsCount := len(r.wrappedLines.Load().([]DocumentFragment))
	innerPadding := theme.InnerPadding()

	return fyne.Size{
		Width:  minItemWidth + innerPadding*2,
		Height: (itemHeight+theme.LineSpacing())*float32(itemsCount) + innerPadding*2,
	}
}

func (r *logCanvasRenderer) Objects() []fyne.CanvasObject {
	return r.objects.Load().([]fyne.CanvasObject)
}

func (r *logCanvasRenderer) Refresh() {
	r.rewrap()

	lineHeight := r.itemHeight() + theme.LineSpacing()
	innerPadding := theme.InnerPadding()
	topOffset := innerPadding
	lines := r.wrappedLines.Load().([]DocumentFragment)

	if len(lines) == 0 {
		return
	}

	top := Clamp(int((r.parentScroller.Offset.Y-topOffset)/lineHeight), 0, len(lines)-1)
	bottom := Clamp(int((r.parentScroller.Offset.Y+r.parentScroller.Size().Height-topOffset)/lineHeight), 0, len(lines)-1)

	visible := make(map[int]*canvas.Text)
	textSize := r.logView.TextSize
	textStyle := r.logView.TextStyle

	r.itemLock.Lock()
	defer r.itemLock.Unlock()

	// populate visible
	for i := top; i <= bottom; i++ {
		item, ok := r.visibleItems[i]
		if !ok {
			item = r.newItem()
			item.Move(fyne.Position{
				X: innerPadding,
				Y: innerPadding + lineHeight*float32(i),
			})
		}
		item.Text = lines[i].Text
		item.TextSize = textSize
		item.TextStyle = textStyle
		item.Refresh()
		visible[i] = item
	}

	// recycle invisible
	for i, item := range r.visibleItems {
		if _, ok := visible[i]; !ok {
			r.recycleItem(item)
		}
	}

	// replace the visible map
	r.visibleItems = visible

	// prepare object list
	objects := make([]fyne.CanvasObject, 0, len(r.visibleItems))
	for _, item := range r.visibleItems {
		objects = append(objects, item)
	}
	r.objects.Store(objects)
}

func (r *logCanvasRenderer) newItem() *canvas.Text {
	return r.itemCache.Get().(*canvas.Text)
}

func (r *logCanvasRenderer) recycleItem(item *canvas.Text) {
	r.itemCache.Put(item)
}

func (r *logCanvasRenderer) itemHeight() float32 {
	return fyne.MeasureText("", r.logView.TextSize, r.logView.TextStyle).Height
}

func (r *logCanvasRenderer) rewrap() {
	width := r.parentScroller.Size().Width - theme.InnerPadding()*2
	if width <= 0 {
		return
	}

	wrap := r.logView.Wrapping
	textSize := r.logView.TextSize
	textStyle := r.logView.TextStyle
	var wrapped []DocumentFragment

	measure := func(s string) float32 {
		return fyne.MeasureText(s, textSize, textStyle).Width
	}

	r.logView.lines.Read(func(lines []string) {
		wrapped = WrapDocument(lines, width, wrap, measure)
	})

	r.wrappedLines.Store(wrapped)
}
