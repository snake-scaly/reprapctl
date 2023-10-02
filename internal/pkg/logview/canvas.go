package logview

import (
	"cmp"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"reprapctl/internal/pkg/doc"
	"reprapctl/pkg/alg"
	"slices"
	"sync"
	"sync/atomic"
)

var _ fyne.Widget = (*logCanvas)(nil)
var _ fyne.Draggable = (*logCanvas)(nil)
var _ fyne.Tappable = (*logCanvas)(nil)
var _ fyne.SecondaryTappable = (*logCanvas)(nil)
var _ desktop.Cursorable = (*logCanvas)(nil)
var _ desktop.Mouseable = (*logCanvas)(nil)

type logCanvas struct {
	widget.BaseWidget
	logView   *LogView
	renderer  atomic.Pointer[logCanvasRenderer]
	selecting atomic.Bool
}

func newLogCanvas(logView *LogView) *logCanvas {
	c := &logCanvas{logView: logView}
	c.ExtendBaseWidget(c)
	return c
}

func (c *logCanvas) CreateRenderer() fyne.WidgetRenderer {
	r := newLogCanvasRenderer(c.logView)
	c.renderer.Store(r)
	return r
}

func (c *logCanvas) Dragged(e *fyne.DragEvent) {
	a := c.getAnchorAtPoint(e.Position)
	if c.selecting.Load() {
		c.logView.lines.SetSelectionEnd(a)
		c.logView.scrollPointToVisible(e.Position)
	} else {
		c.logView.requestFocus()
		c.logView.lines.StartSelection(a)
		c.selecting.Store(true)
	}
	c.logView.Refresh()
}

func (c *logCanvas) DragEnd() {
	c.selecting.Store(false)
}

func (c *logCanvas) Tapped(_ *fyne.PointEvent) {
	c.logView.requestFocus()
	c.logView.lines.SelectNone()
	c.Refresh()
}

func (c *logCanvas) TappedSecondary(e *fyne.PointEvent) {
	c.logView.showContextMenu(e.AbsolutePosition)
}

func (c *logCanvas) Cursor() desktop.Cursor {
	return desktop.TextCursor
}

func (c *logCanvas) MouseDown(_ *desktop.MouseEvent) {
	c.logView.requestFocus()
}

func (c *logCanvas) MouseUp(_ *desktop.MouseEvent) {
}

func (c *logCanvas) getAnchorAtPoint(p fyne.Position) doc.Anchor {
	renderer := c.renderer.Load()
	if renderer == nil {
		// no renderer, nothing to select
		return doc.Anchor{}
	}

	objects := renderer.Objects()
	sorted := make([]*logCanvasItem, 0, len(objects))
	for _, o := range objects {
		if item, ok := o.(*logCanvasItem); ok {
			sorted = append(sorted, item)
		}
	}
	if len(sorted) == 0 {
		// empty document, nothing to select
		return doc.Anchor{}
	}
	slices.SortFunc(sorted, func(a, b *logCanvasItem) int {
		return cmp.Compare(a.Position().Y, b.Position().Y)
	})

	lineSpacing := theme.LineSpacing()

	if p.Y < sorted[0].Position().Y {
		// cursor is above all lines; return start of the first line
		return sorted[0].Anchor
	}

	for _, item := range sorted {
		ip := item.Position()
		is := item.MinSize()
		if p.Y >= ip.Y+is.Height+lineSpacing {
			// cursor is below the current line
			continue
		}

		offset, _ := alg.BinarySearch(len(item.Text.Text), p.X-ip.X, func(i int) float32 {
			return fyne.MeasureText(item.Text.Text[:i], item.Text.TextSize, item.Text.TextStyle).Width
		})

		return doc.Anchor{
			LineIndex:  item.Anchor.LineIndex,
			LineOffset: item.Anchor.LineOffset + offset,
		}
	}

	// cursor is below all lines; return end of the last line
	lastItem := sorted[len(sorted)-1]
	return doc.Anchor{
		LineIndex:  lastItem.Anchor.LineIndex,
		LineOffset: lastItem.Anchor.LineOffset + len(lastItem.Text.Text),
	}
}

var _ fyne.WidgetRenderer = (*logCanvasRenderer)(nil)

type logCanvasRenderer struct {
	logView *LogView

	wrappedLines []doc.DocumentFragment
	wrapContext  wrapContext
	wrapLock     sync.RWMutex

	visibleItems      map[int]*logCanvasItem
	visibleSelections []*canvas.Rectangle
	itemsLock         sync.RWMutex

	itemCache sync.Pool
	rectCache sync.Pool
	objects   atomic.Value
}

type wrapContext struct {
	documentVersion uint64
	width           float32
	wrap            fyne.TextWrap
	textSize        float32
	textStyle       fyne.TextStyle
}

func newLogCanvasRenderer(logView *LogView) *logCanvasRenderer {
	r := &logCanvasRenderer{
		logView:      logView,
		visibleItems: make(map[int]*logCanvasItem),
	}

	r.itemCache.New = func() any {
		return newLogCanvasItem(theme.ForegroundColor())
	}

	r.rectCache.New = func() any {
		return canvas.NewRectangle(theme.SelectionColor())
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
	lines := r.rewrap()
	visible := make(map[int]*logCanvasItem)
	lineSpacing := theme.LineSpacing()
	lineHeight := r.itemHeight() + lineSpacing

	r.itemsLock.Lock()
	defer r.itemsLock.Unlock()

	if len(lines) > 0 {
		innerPadding := theme.InnerPadding()
		topOffset := innerPadding

		scroller := r.logView.scroller
		top := alg.Clamp(int((scroller.Offset.Y-topOffset)/lineHeight), 0, len(lines)-1)
		bottom := alg.Clamp(int((scroller.Offset.Y+scroller.Size().Height-topOffset)/lineHeight), 0, len(lines)-1)

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
			item.Text.Text = lines[i].Text
			item.Text.TextSize = textSize
			item.Text.TextStyle = textStyle
			item.Anchor = lines[i].Anchor
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

	// selection
	for _, s := range r.visibleSelections {
		r.rectCache.Put(s)
	}
	r.visibleSelections = make([]*canvas.Rectangle, 0, len(r.visibleItems))

	selStart, selEnd := r.logView.lines.Selection()

	for _, item := range r.visibleItems {
		itemStart, itemEnd := item.Anchor, item.Anchor
		itemEnd.LineOffset += len(item.Text.Text)

		if itemEnd.Compare(selStart) < 0 || itemStart.Compare(selEnd) > 0 {
			continue
		}

		itemPos, itemWidth := item.Position(), item.MinSize().Width
		rect := r.rectCache.Get().(*canvas.Rectangle)
		var x1, x2 float32

		switch {
		case itemStart.Compare(selStart) <= 0 && itemEnd.Compare(selEnd) >= 0:
			x1 = item.charPos(selStart.LineOffset - itemStart.LineOffset)
			x2 = item.charPos(selEnd.LineOffset - itemStart.LineOffset)
		case itemStart.Compare(selStart) <= 0 && itemEnd.Compare(selStart) >= 0:
			x1, x2 = item.charPos(selStart.LineOffset-itemStart.LineOffset), itemPos.X+itemWidth
		case itemStart.Compare(selEnd) <= 0 && itemEnd.Compare(selEnd) >= 0:
			x1, x2 = itemPos.X, item.charPos(selEnd.LineOffset-itemStart.LineOffset)
		default:
			x1, x2 = itemPos.X, itemPos.X+itemWidth
		}

		rect.Move(fyne.Position{X: x1, Y: itemPos.Y})
		rect.Resize(fyne.Size{Width: x2 - x1, Height: lineHeight})
		r.visibleSelections = append(r.visibleSelections, rect)
	}

	// prepare object list
	objects := make([]fyne.CanvasObject, 0, len(r.visibleItems))
	for _, rect := range r.visibleSelections {
		objects = append(objects, rect)
	}
	for _, item := range r.visibleItems {
		objects = append(objects, item)
	}
	r.objects.Store(objects)
}

func (r *logCanvasRenderer) newItem() *logCanvasItem {
	return r.itemCache.Get().(*logCanvasItem)
}

func (r *logCanvasRenderer) recycleItem(item *logCanvasItem) {
	r.itemCache.Put(item)
}

func (r *logCanvasRenderer) itemHeight() float32 {
	h := fyne.MeasureText("", r.logView.TextSize(), r.logView.TextStyle()).Height
	return float32(int(h))
}

func (r *logCanvasRenderer) rewrap() []doc.DocumentFragment {
	width := r.logView.scroller.Size().Width - theme.InnerPadding()*2

	r.wrapLock.RLock()
	lastWrapped := r.wrappedLines
	lastContext := r.wrapContext
	r.wrapLock.RUnlock()

	if width <= 0 {
		return lastWrapped
	}

	context := wrapContext{
		documentVersion: r.logView.lines.Version(),
		width:           width,
		wrap:            r.logView.Wrapping(),
		textSize:        r.logView.TextSize(),
		textStyle:       r.logView.TextStyle(),
	}

	// no need to rewrap if context didn't change
	if context == lastContext {
		return lastWrapped
	}

	measure := func(s string) float32 {
		return fyne.MeasureText(s, context.textSize, context.textStyle).Width
	}

	var wrapped []doc.DocumentFragment
	r.logView.lines.Read(func(lines []string) {
		wrapped = doc.WrapDocument(lines, context.width, context.wrap, measure)
	})

	func() {
		r.wrapLock.Lock()
		defer r.wrapLock.Unlock()
		r.wrappedLines = wrapped
		r.wrapContext = context
	}()

	return wrapped
}
