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
		return &canvas.Text{
			TextSize:  theme.TextSize(),
			Color:     theme.ForegroundColor(),
			TextStyle: fyne.TextStyle{Monospace: true},
		}
	}

	parentScroller.OnScrolled = func(_ fyne.Position) {
		r.Refresh()
	}

	return r
}

func (r *logCanvasRenderer) Destroy() {
}

func (r *logCanvasRenderer) Layout(_ fyne.Size) {
}

func (r *logCanvasRenderer) MinSize() fyne.Size {
	var minItemWidth float32
	var itemHeight float32

	func() {
		r.itemLock.RLock()
		defer r.itemLock.RUnlock()
		for _, item := range r.visibleItems {
			minSize := item.MinSize()
			minItemWidth, itemHeight = max(minItemWidth, minSize.Width), minSize.Height
		}
	}()

	if itemHeight == 0 {
		itemHeight = r.itemHeight()
	}

	var itemsCount int
	r.logView.lines.Read(func(lines []string) {
		itemsCount = len(lines)
	})

	return fyne.Size{
		Width:  minItemWidth + theme.InnerPadding()*2,
		Height: (itemHeight+theme.LineSpacing())*float32(itemsCount) + theme.InnerPadding()*2,
	}
}

func (r *logCanvasRenderer) Objects() []fyne.CanvasObject {
	return r.objects.Load().([]fyne.CanvasObject)
}

func (r *logCanvasRenderer) Refresh() {
	lineHeight := r.itemHeight() + theme.LineSpacing()
	topOffset := theme.InnerPadding()

	top := int((r.parentScroller.Offset.Y - topOffset) / lineHeight)
	bottom := int((r.parentScroller.Offset.Y + r.parentScroller.Size().Height - topOffset) / lineHeight)

	visible := make(map[int]*canvas.Text)

	r.itemLock.Lock()
	defer r.itemLock.Unlock()

	r.logView.lines.Read(func(lines []string) {
		top = Clamp(top, 0, len(lines)-1)
		bottom = Clamp(bottom, 0, len(lines)-1)

		// populate visible
		for i := top; i <= bottom; i++ {
			item, ok := r.visibleItems[i]
			if !ok {
				item = r.newItem()
				item.Move(fyne.Position{
					X: theme.InnerPadding(),
					Y: theme.InnerPadding() + lineHeight*float32(i),
				})
			}
			item.Text = lines[i]
			item.Refresh()
			visible[i] = item
		}
	})

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
	item := r.newItem()
	defer r.recycleItem(item)
	return item.MinSize().Height
}
