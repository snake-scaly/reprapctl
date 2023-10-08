package logview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"reprapctl/pkg/alg"
	"reprapctl/pkg/doc"
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
	renderer := c.renderer.Load()
	if renderer == nil {
		return
	}
	a := renderer.getAnchorAtPoint(e.Position)
	if c.selecting.Load() {
		c.logView.scrollPointToVisible(e.Position)
	} else {
		c.selecting.Store(true)
		c.logView.requestFocus()
		c.logView.document.SetBookmark(bookmarkSelectionStart, a)
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
	c.logView.Refresh()
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

var _ fyne.WidgetRenderer = (*logCanvasRenderer)(nil)

type logCanvasRenderer struct {
	logView *LogView

	wrapContext       wrapContext
	wrappedLines      []Box
	visibleItems      map[int]*logCanvasItem
	visibleSelections map[int]*logSelectionRect
	refreshId         int
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
		logView:           logView,
		visibleItems:      make(map[int]*logCanvasItem),
		visibleSelections: make(map[int]*logSelectionRect),
	}

	r.itemCache.New = func() any {
		return newLogCanvasItem(theme.ForegroundColor())
	}

	r.rectCache.New = func() any {
		return newLogSelectionRect()
	}

	r.objects.Store(make([]fyne.CanvasObject, 0))

	return r
}

func (r *logCanvasRenderer) Destroy() {
}

func (r *logCanvasRenderer) Layout(_ fyne.Size) {
}

func (r *logCanvasRenderer) MinSize() fyne.Size {
	innerPadding := theme.InnerPadding()

	r.itemsLock.RLock()
	defer r.itemsLock.RUnlock()

	minSize := fyne.Size{Width: innerPadding * 2, Height: innerPadding * 2}

	for _, item := range r.visibleItems {
		minSize.Width = max(minSize.Width, item.Position().X+item.MinSize().Width+innerPadding)
	}

	if nLines := len(r.wrappedLines); nLines > 0 {
		lastBox := r.wrappedLines[nLines-1]
		minSize.Height = lastBox.Position().Y + lastBox.Size().Height + innerPadding
	}

	return minSize
}

func (r *logCanvasRenderer) Objects() []fyne.CanvasObject {
	return r.objects.Load().([]fyne.CanvasObject)
}

func (r *logCanvasRenderer) Refresh() {
	r.itemsLock.Lock()
	defer r.itemsLock.Unlock()

	r.refreshId++

	dirty := r.rewrap()
	r.renderItems(dirty)
	r.renderSelection()
	r.cacheObjects()
}

// renderItems assumes a write lock on r.itemsLock.
func (r *logCanvasRenderer) renderItems(dirty bool) {
	scrollY, scrollH := r.logView.scroller.Offset.Y, r.logView.scroller.Size().Height

	isLineVisible := func(box Box) bool {
		return box.Position().Y+box.Size().Height > scrollY && scrollY+scrollH > box.Position().Y
	}

	for i, item := range r.visibleItems {
		if dirty || !isLineVisible(r.wrappedLines[i]) {
			r.visibleItems[i] = nil
			delete(r.visibleItems, i)
			r.itemCache.Put(item)
		}
	}

	for i, box := range r.wrappedLines {
		textBox := box.(*TextBox)

		if textBox.Position().Y >= scrollY+scrollH {
			break
		}
		if textBox.Position().Y+textBox.Size().Height <= scrollY || r.visibleItems[i] != nil {
			continue
		}

		item := r.itemCache.Get().(*logCanvasItem)
		item.Set(textBox)

		p := textBox.Position()
		p.Y += (textBox.Size().Height - item.MinSize().Height) / 2
		item.Move(p)

		item.Refresh()
		r.visibleItems[i] = item
	}
}

// renderSelection assumes a write lock on r.itemsLock.
func (r *logCanvasRenderer) renderSelection() {
	selStart, haveSelStart := r.logView.document.GetBookmark(bookmarkSelectionStart)
	selEnd, haveSelEnd := r.logView.document.GetBookmark(bookmarkSelectionEnd)

	selMin, selMax := selStart, selEnd
	if selMin.Compare(selMax) > 0 {
		selMin, selMax = selMax, selMin
	}

	if haveSelStart && haveSelEnd {
		selColor := theme.SelectionColor()
		for i, item := range r.visibleItems {
			itemStart := item.box.StartAnchor()
			itemEnd := item.box.EndAnchor()
			if itemStart.Compare(selMax) >= 0 || itemEnd.Compare(selMin) <= 0 {
				continue
			}

			selPos := item.box.Position()
			selSize := item.box.Size()

			if itemStart.Compare(selMin) < 0 && itemEnd.Compare(selMax) > 0 {
				x1 := item.CharToX(selMin.LineOffset - itemStart.LineOffset)
				selPos.X += x1
				selSize.Width = item.CharToX(selMax.LineOffset-itemStart.LineOffset) - x1
			} else if itemStart.Compare(selMin) < 0 && itemEnd.Compare(selMin) > 0 {
				x1 := item.CharToX(selMin.LineOffset - itemStart.LineOffset)
				selPos.X += x1
				selSize.Width -= x1
			} else if itemStart.Compare(selMax) < 0 && itemEnd.Compare(selMax) > 0 {
				selSize.Width = item.CharToX(selMax.LineOffset - itemStart.LineOffset)
			}

			rect := r.visibleSelections[i]
			if rect == nil {
				rect = r.rectCache.Get().(*logSelectionRect)
				r.visibleSelections[i] = rect
			}

			rect.rect.FillColor = selColor
			rect.tag = r.refreshId
			rect.Move(selPos)
			rect.Resize(selSize)
			rect.Refresh()
		}
	}

	for i, sel := range r.visibleSelections {
		if sel.tag != r.refreshId {
			delete(r.visibleSelections, i)
			r.rectCache.Put(sel)
		}
	}
}

func (r *logCanvasRenderer) cacheObjects() {
	objects := make([]fyne.CanvasObject, 0, len(r.visibleSelections)+len(r.visibleItems))
	for _, rect := range r.visibleSelections {
		objects = append(objects, rect)
	}
	for _, item := range r.visibleItems {
		objects = append(objects, item)
	}
	r.objects.Store(objects)
}

func (r *logCanvasRenderer) itemHeight() float32 {
	h := fyne.MeasureText("", r.logView.TextSize(), r.logView.TextStyle()).Height
	return float32(int(h))
}

func (r *logCanvasRenderer) rewrap() bool {
	padding := theme.InnerPadding()
	width := r.logView.scroller.Size().Width - padding*2
	if width <= 0 {
		return false
	}

	context := wrapContext{
		documentVersion: r.logView.document.Version(),
		width:           width,
		wrap:            r.logView.Wrapping(),
		textSize:        r.logView.TextSize(),
		textStyle:       r.logView.TextStyle(),
	}

	// no need to rewrap if context didn't change
	if context == r.wrapContext {
		return false
	}
	r.wrapContext = context

	var lines []string
	r.logView.document.Read(func(ll []string) {
		lines = make([]string, len(ll))
		copy(lines, ll)
	})

	lineSpacing := theme.LineSpacing()
	pos := fyne.Position{X: padding, Y: padding}
	r.wrappedLines = make([]Box, 0, len(lines))

	for i, line := range lines {
		doc.WrapString(
			line, context.width, context.wrap,
			func(s string) fyne.Size {
				return fyne.MeasureText(s, context.textSize, context.textStyle)
			},
			func(s string, o int, z fyne.Size) {
				// round height down to avoid pixel artifacts
				z.Height = float32(int(z.Height)) + lineSpacing
				start := doc.Anchor{LineIndex: i, LineOffset: o}
				end := doc.Anchor{LineIndex: i, LineOffset: o + len(s)}
				r.wrappedLines = append(
					r.wrappedLines, NewTextBox(pos, z, start, end, s, context.textSize, context.textStyle))
				pos.Y += z.Height
			},
		)
	}

	return true
}

func (r *logCanvasRenderer) getAnchorAtPoint(p fyne.Position) doc.Anchor {
	r.itemsLock.RLock()
	defer r.itemsLock.RUnlock()
	nLines := len(r.wrappedLines)
	if nLines == 0 {
		return doc.Anchor{}
	}
	lastLine := r.wrappedLines[nLines-1]
	i, _ := alg.BinarySearch(nLines, p.Y, func(i int) float32 {
		if i == nLines {
			return lastLine.Position().Y + lastLine.Size().Height
		}
		return r.wrappedLines[i].Position().Y
	})
	if i == nLines {
		return lastLine.EndAnchor()
	}
	if textBox, ok := r.wrappedLines[i].(*TextBox); ok {
		return textBox.AnchorAtX(p.X - textBox.Position().X)
	}
	return r.wrappedLines[i].StartAnchor()
}
