package logview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"reprapctl/pkg/alg"
	"reprapctl/pkg/doc"
	"sync"
)

var shortcutCut = &fyne.ShortcutCut{}
var shortcutCopy = &fyne.ShortcutCopy{}
var shortcutSelectAll = &fyne.ShortcutSelectAll{}
var shortcutWordWrap = &desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierShortcutDefault}

var _ fyne.Widget = (*LogView)(nil)
var _ fyne.Focusable = (*LogView)(nil)
var _ fyne.Shortcutable = (*LogView)(nil)

type LogView struct {
	widget.BaseWidget

	border   *canvas.Rectangle
	scroller *container.Scroll
	canvas   *logCanvas

	textSize      float32
	textStyle     fyne.TextStyle
	wrapping      fyne.TextWrap
	autoScroll    bool
	viewTopOffset float32
	document      doc.Document
	propertyLock  sync.RWMutex

	shortcutHandler fyne.ShortcutHandler
}

func New() *LogView {
	l := LogView{
		border: &canvas.Rectangle{
			StrokeColor:  theme.InputBorderColor(),
			StrokeWidth:  theme.InputBorderSize(),
			CornerRadius: theme.InputRadiusSize(),
		},
		textSize:   theme.TextSize(),
		textStyle:  fyne.TextStyle{Monospace: true},
		wrapping:   fyne.TextWrapWord,
		autoScroll: true,
		document:   doc.New(),
	}

	l.canvas = newLogCanvas(&l)
	l.scroller = container.NewScroll(l.canvas)
	l.scroller.OnScrolled = func(_ fyne.Position) {
		func() {
			l.propertyLock.Lock()
			defer l.propertyLock.Unlock()
			l.autoScroll = l.scroller.Offset.Y+l.scroller.Size().Height >= l.canvas.Size().Height
			if l.autoScroll {
				l.document.RemoveBookmark(bookmarkViewTop)
			} else {
				topBox := l.canvas.getBoxAtPoint(l.scroller.Offset)
				if topBox != nil {
					l.document.SetBookmark(bookmarkViewTop, topBox.StartAnchor())
					l.viewTopOffset = l.scroller.Offset.Y - topBox.Position().Y
				}
			}
		}()

		l.canvas.Refresh()
	}

	l.shortcutHandler.AddShortcut(shortcutCut, func(shortcut fyne.Shortcut) {
		s, _ := l.document.String(bookmarkSelectionStart, bookmarkSelectionEnd, "\n")
		shortcut.(*fyne.ShortcutCut).Clipboard.SetContent(s)
	})
	l.shortcutHandler.AddShortcut(shortcutCopy, func(shortcut fyne.Shortcut) {
		s, _ := l.document.String(bookmarkSelectionStart, bookmarkSelectionEnd, "\n")
		shortcut.(*fyne.ShortcutCopy).Clipboard.SetContent(s)
	})
	l.shortcutHandler.AddShortcut(shortcutSelectAll, func(_ fyne.Shortcut) {
		start, _ := l.document.GetBookmark(doc.BookmarkStart)
		end, _ := l.document.GetBookmark(doc.BookmarkEnd)
		l.document.SetBookmark(bookmarkSelectionStart, start)
		l.document.SetBookmark(bookmarkSelectionEnd, end)
		l.Refresh()
	})
	l.shortcutHandler.AddShortcut(shortcutWordWrap, func(_ fyne.Shortcut) {
		if l.Wrapping() == fyne.TextWrapWord {
			l.SetWrapping(fyne.TextWrapOff)
		} else {
			l.SetWrapping(fyne.TextWrapWord)
		}
		l.Refresh()
	})

	l.ExtendBaseWidget(&l)

	return &l
}

func (l *LogView) CreateRenderer() fyne.WidgetRenderer {
	r := NewStackRenderer(l.scroller, l.border)
	r.OnLayout = func(_ fyne.Size) {
		l.Refresh()
	}
	r.OnPreRefresh = func() {
		l.canvas.Refresh()
	}
	r.OnRefresh = func() {
		l.propertyLock.RLock()
		autoScroll := l.autoScroll
		l.propertyLock.RUnlock()

		if autoScroll {
			l.scroller.ScrollToBottom()
		} else if a, ok := l.document.GetBookmark(bookmarkViewTop); ok {
			if box := l.canvas.getBoxAtAnchor(a); box != nil {
				l.scrollToOffset(fyne.NewPos(l.scroller.Offset.X, box.Position().Y+l.viewTopOffset))
				l.scroller.Refresh()
			}
		}
	}
	return r
}

func (l *LogView) FocusGained() {
}

func (l *LogView) FocusLost() {
}

func (l *LogView) TypedRune(_ rune) {
}

func (l *LogView) TypedKey(_ *fyne.KeyEvent) {
}

func (l *LogView) TypedShortcut(shortcut fyne.Shortcut) {
	l.shortcutHandler.TypedShortcut(shortcut)
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

func (l *LogView) Capacity() int {
	return l.document.Capacity()
}

func (l *LogView) SetCapacity(c int) {
	l.document.SetCapacity(c)
}

func (l *LogView) AddLine(line string) {
	l.document.Add(line)
}

func (l *LogView) requestFocus() {
	if c := fyne.CurrentApp().Driver().CanvasForObject(l); c != nil {
		c.Focus(l)
	}
}

func (l *LogView) showContextMenu(absolutePos fyne.Position) {
	driver := fyne.CurrentApp().Driver()
	cb := driver.AllWindows()[0].Clipboard()

	copyItem := &fyne.MenuItem{
		Label:    "Copy",
		Shortcut: shortcutCopy,
		Action: func() {
			l.shortcutHandler.TypedShortcut(&fyne.ShortcutCopy{Clipboard: cb})
		},
	}
	selectAllItem := &fyne.MenuItem{
		Label:    "Select all",
		Shortcut: shortcutSelectAll,
		Action: func() {
			l.shortcutHandler.TypedShortcut(shortcutSelectAll)
		},
	}
	wordWrapItem := &fyne.MenuItem{
		Label:    "Word wrap",
		Shortcut: shortcutWordWrap,
		Checked:  l.Wrapping() == fyne.TextWrapWord,
		Action: func() {
			l.shortcutHandler.TypedShortcut(shortcutWordWrap)
		},
	}

	selStart, haveSelStart := l.document.GetBookmark(bookmarkSelectionStart)
	selEnd, haveSelEnd := l.document.GetBookmark(bookmarkSelectionEnd)
	copyItem.Disabled = !haveSelStart || !haveSelEnd || selStart.Compare(selEnd) == 0

	menu := fyne.NewMenu("", copyItem, selectAllItem, wordWrapItem)

	cv := driver.CanvasForObject(l)
	popup := widget.NewPopUpMenu(menu, cv)
	popup.ShowAtPosition(absolutePos)
}

func (l *LogView) scrollPointToVisible(p fyne.Position) {
	startOffset, viewSize, canvasSize := l.scroller.Offset, l.scroller.Size(), l.canvas.Size()
	var newOffset fyne.Position
	newOffset.X = alg.Clamp(startOffset.X, p.X-viewSize.Width, p.X)
	newOffset.X = alg.Clamp(newOffset.X, 0, max(0, canvasSize.Width-viewSize.Width))
	newOffset.Y = alg.Clamp(startOffset.Y, p.Y-viewSize.Height, p.Y)
	newOffset.Y = alg.Clamp(newOffset.Y, 0, max(0, canvasSize.Height-viewSize.Height))
	l.scrollToOffset(newOffset)
}

func (l *LogView) scrollToOffset(newOffset fyne.Position) {
	if l.scroller.Offset != newOffset {
		l.scroller.Offset = newOffset
		if onScrolled := l.scroller.OnScrolled; onScrolled != nil {
			onScrolled(newOffset)
		}
	}
}

type bookmark int

const (
	bookmarkSelectionStart = bookmark(iota)
	bookmarkSelectionEnd
	bookmarkViewTop
)
