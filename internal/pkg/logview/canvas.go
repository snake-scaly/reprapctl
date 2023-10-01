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

	wrappedLines []DocumentFragment
	wrapContext  wrapContext
	wrapLock     sync.RWMutex

	visibleItems map[int]*canvas.Text
	itemsLock    sync.RWMutex

	itemCache sync.Pool
	objects   atomic.Value
}

type wrapContext struct {
	documentVersion uint64
	width           float32
	wrap            fyne.TextWrap
	textSize        float32
	textStyle       fyne.TextStyle
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
		r.itemsLock.RLock()
		defer r.itemsLock.RUnlock()
		for _, item := range r.visibleItems {
			minItemWidth = max(minItemWidth, item.MinSize().Width)
		}
	}()

	itemHeight := r.itemHeight()
	innerPadding := theme.InnerPadding()
	lineSpacing := theme.LineSpacing()

	r.wrapLock.RLock()
	itemsCount := len(r.wrappedLines)
	r.wrapLock.RUnlock()

	return fyne.Size{
		Width:  minItemWidth + innerPadding*2,
		Height: (itemHeight+lineSpacing)*float32(itemsCount) + innerPadding*2,
	}
}

func (r *logCanvasRenderer) Objects() []fyne.CanvasObject {
	return r.objects.Load().([]fyne.CanvasObject)
}

func (r *logCanvasRenderer) Refresh() {
	var lines []DocumentFragment

	if wrapped, context, ok := r.rewrap(); ok {
		func() {
			r.wrapLock.Lock()
			defer r.wrapLock.Unlock()
			r.wrappedLines = wrapped
			r.wrapContext = context
		}()
		lines = wrapped
	} else {
		func() {
			r.wrapLock.RLock()
			defer r.wrapLock.RUnlock()
			lines = r.wrappedLines
		}()
	}

	visible := make(map[int]*canvas.Text)

	r.itemsLock.Lock()
	defer r.itemsLock.Unlock()

	if len(lines) > 0 {
		lineHeight := r.itemHeight() + theme.LineSpacing()
		innerPadding := theme.InnerPadding()
		topOffset := innerPadding

		top := Clamp(int((r.parentScroller.Offset.Y-topOffset)/lineHeight), 0, len(lines)-1)
		bottom := Clamp(int((r.parentScroller.Offset.Y+r.parentScroller.Size().Height-topOffset)/lineHeight), 0, len(lines)-1)

		textSize := r.logView.TextSize()
		textStyle := r.logView.TextStyle()

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
	return fyne.MeasureText("", r.logView.TextSize(), r.logView.TextStyle()).Height
}

func (r *logCanvasRenderer) rewrap() ([]DocumentFragment, wrapContext, bool) {
	width := r.parentScroller.Size().Width - theme.InnerPadding()*2
	if width <= 0 {
		return nil, wrapContext{}, false
	}

	context := wrapContext{
		documentVersion: r.logView.lines.Version(),
		width:           width,
		wrap:            r.logView.Wrapping(),
		textSize:        r.logView.TextSize(),
		textStyle:       r.logView.TextStyle(),
	}

	// no need to rewrap if context didn't change
	r.wrapLock.RLock()
	lastContext := r.wrapContext
	r.wrapLock.RUnlock()
	if context == lastContext {
		return nil, wrapContext{}, false
	}

	measure := func(s string) float32 {
		return fyne.MeasureText(s, context.textSize, context.textStyle).Width
	}

	var wrapped []DocumentFragment
	r.logView.lines.Read(func(lines []string) {
		wrapped = WrapDocument(lines, context.width, context.wrap, measure)
	})

	return wrapped, context, true
}
