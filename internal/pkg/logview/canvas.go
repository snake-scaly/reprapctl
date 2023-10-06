package logview

import (
	"cmp"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"reprapctl/pkg/alg"
	"reprapctl/pkg/doc"
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
		c.logView.scrollPointToVisible(e.Position)
	} else {
		c.logView.requestFocus()
		c.logView.document.SetBookmark(bookmarkSelectionStart, a)
		c.selecting.Store(true)
	}
	c.logView.document.SetBookmark(bookmarkSelectionEnd, a)
	c.logView.Refresh()
}

func (c *logCanvas) DragEnd() {
	c.selecting.Store(false)
}

func (c *logCanvas) Tapped(_ *fyne.PointEvent) {
	c.logView.requestFocus()
	c.logView.document.RemoveBookmark(bookmarkSelectionStart)
	c.logView.document.RemoveBookmark(bookmarkSelectionEnd)
	c.Refresh()
}

func (c *logCanvas) TappedSecondary(e *fyne.PointEvent) {
	c.logView.requestFocus()
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

		return doc.Anchor{
			LineIndex:  item.Anchor.LineIndex,
			LineOffset: item.Anchor.LineOffset + item.XToChar(p.X-ip.X),
		}
	}

	// cursor is below all lines; return end of the last line
	lastItem := sorted[len(sorted)-1]
	return doc.Anchor{
		LineIndex:  lastItem.Anchor.LineIndex,
		LineOffset: lastItem.Anchor.LineOffset + len(lastItem.Text()),
	}
}

var _ fyne.WidgetRenderer = (*logCanvasRenderer)(nil)

type logCanvasRenderer struct {
	logView *LogView

	wrappedLines []doc.Fragment
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
	lineSpacing := theme.LineSpacing()
	lineHeight := r.itemHeight() + lineSpacing

	r.itemsLock.Lock()
	defer r.itemsLock.Unlock()

	r.renderItems(lines, lineHeight)
	r.renderSelection(lineHeight)
	r.cacheObjects()
}

// renderItems assumes a write lock on r.itemsLock.
func (r *logCanvasRenderer) renderItems(lines []doc.Fragment, lineHeight float32) {
	visible := make(map[int]*logCanvasItem)

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
			item.SetText(lines[i].Text, textSize, textStyle)
			item.Anchor = lines[i].Anchor
			item.Refresh()
			visible[i] = item
		}
	}

	// recycle unused items
	for i, item := range r.visibleItems {
		if _, ok := visible[i]; !ok {
			r.recycleItem(item)
		}
	}

	r.visibleItems = visible
}

// renderSelection assumes a write lock on r.itemsLock.
func (r *logCanvasRenderer) renderSelection(lineHeight float32) {
	oldSelections := r.visibleSelections
	var selections []*canvas.Rectangle
	selStart, haveSelStart := r.logView.document.GetBookmark(bookmarkSelectionStart)
	selEnd, haveSelEnd := r.logView.document.GetBookmark(bookmarkSelectionEnd)

	if haveSelStart && haveSelEnd && selStart.Compare(selEnd) != 0 {
		selections = make([]*canvas.Rectangle, 0, len(r.visibleItems))
		if selStart.Compare(selEnd) > 0 {
			selStart, selEnd = selEnd, selStart
		}

		for _, item := range r.visibleItems {
			itemStart, itemEnd := item.Anchor, item.Anchor
			itemEnd.LineOffset += len(item.Text())

			if itemEnd.Compare(selStart) < 0 || itemStart.Compare(selEnd) > 0 {
				continue
			}

			itemPos, itemWidth := item.Position(), item.MinSize().Width
			var x1, x2 float32

			switch {
			case itemStart.Compare(selStart) <= 0 && itemEnd.Compare(selEnd) >= 0:
				x1 = itemPos.X + item.CharToX(selStart.LineOffset-itemStart.LineOffset)
				x2 = itemPos.X + item.CharToX(selEnd.LineOffset-itemStart.LineOffset)
			case itemStart.Compare(selStart) <= 0 && itemEnd.Compare(selStart) >= 0:
				x1 = itemPos.X + item.CharToX(selStart.LineOffset-itemStart.LineOffset)
				x2 = itemPos.X + itemWidth
			case itemStart.Compare(selEnd) <= 0 && itemEnd.Compare(selEnd) >= 0:
				x1 = itemPos.X
				x2 = itemPos.X + item.CharToX(selEnd.LineOffset-itemStart.LineOffset)
			default:
				x1, x2 = itemPos.X, itemPos.X+itemWidth
			}

			var rect *canvas.Rectangle
			if len(oldSelections) != 0 {
				last := len(oldSelections) - 1
				rect, oldSelections = oldSelections[last], oldSelections[:last]
			} else {
				rect = r.rectCache.Get().(*canvas.Rectangle)
			}

			rect.Move(fyne.Position{X: x1, Y: itemPos.Y})
			rect.Resize(fyne.Size{Width: x2 - x1, Height: lineHeight})
			selections = append(selections, rect)
		}
	}

	// recycle unused rects
	for _, o := range oldSelections {
		r.rectCache.Put(o)
	}

	r.visibleSelections = selections
}

func (r *logCanvasRenderer) cacheObjects() {
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

func (r *logCanvasRenderer) rewrap() []doc.Fragment {
	width := r.logView.scroller.Size().Width - theme.InnerPadding()*2

	r.wrapLock.RLock()
	lastWrapped := r.wrappedLines
	lastContext := r.wrapContext
	r.wrapLock.RUnlock()

	if width <= 0 {
		return lastWrapped
	}

	context := wrapContext{
		documentVersion: r.logView.document.Version(),
		width:           width,
		wrap:            r.logView.Wrapping(),
		textSize:        r.logView.TextSize(),
		textStyle:       r.logView.TextStyle(),
	}

	// no need to rewrap if context didn't change
	if context == lastContext {
		return lastWrapped
	}

	var lines []string
	r.logView.document.Read(func(ll []string) {
		lines = make([]string, len(ll))
		copy(lines, ll)
	})

	wrapped := doc.WrapDocument(lines, context.width, context.wrap, func(s string) float32 {
		return fyne.MeasureText(s, context.textSize, context.textStyle).Width
	})

	func() {
		r.wrapLock.Lock()
		defer r.wrapLock.Unlock()
		r.wrappedLines = wrapped
		r.wrapContext = context
	}()

	return wrapped
}
