package doc

import (
	"strings"
	"sync"
)

type Document struct {
	lines          []string
	version        uint64
	selectionStart Anchor
	selectionEnd   Anchor
	lock           sync.RWMutex
}

// WithReadLock allows to perform a read-only action on the list of lines.
//
// The lines slice must be treated as read-only and volatile. No portion
// of this slice must be cached nor modified. The strings themselves can be
// reused.
func (d *Document) Read(action func(lines []string)) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	action(d.lines)
}

func (d *Document) Add(lines ...string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.lines = append(d.lines, lines...)
	d.version++
}

// Version is an opaque value that changes every time the Document is mutated.
func (d *Document) Version() uint64 {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.version
}

// Selection returns the current selection boundaries in ascending order.
func (d *Document) Selection() (start, end Anchor) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	start = d.selectionStart
	end = d.selectionEnd
	if start.Compare(end) > 0 {
		start, end = end, start
	}
	return
}

func (d *Document) StartSelection(a Anchor) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.selectionStart = a
	d.selectionEnd = a
}

func (d *Document) SetSelectionEnd(a Anchor) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.selectionEnd = a
}

func (d *Document) SelectNone() {
	d.StartSelection(Anchor{})
}

func (d *Document) SelectAll() {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.selectionStart = Anchor{}
	if len(d.lines) == 0 {
		d.selectionEnd = Anchor{}
	} else {
		d.selectionEnd.LineIndex = len(d.lines) - 1
		d.selectionEnd.LineOffset = len(d.lines[len(d.lines)-1])
	}
}

func (d *Document) SelectionToString() string {
	var builder strings.Builder

	func() {
		d.lock.RLock()
		defer d.lock.RUnlock()

		if len(d.lines) == 0 {
			return
		}

		firstLine := d.lines[d.selectionStart.LineIndex]

		if d.selectionStart.LineIndex == d.selectionEnd.LineIndex {
			builder.WriteString(firstLine[d.selectionStart.LineOffset:d.selectionEnd.LineOffset])
		} else {
			builder.WriteString(firstLine[d.selectionStart.LineOffset:])
			builder.WriteRune('\n')
			for i := d.selectionStart.LineIndex + 1; i < d.selectionEnd.LineIndex; i++ {
				builder.WriteString(d.lines[i])
				builder.WriteRune('\n')
			}
			builder.WriteString(d.lines[d.selectionEnd.LineIndex][:d.selectionEnd.LineOffset])
		}
	}()

	return builder.String()
}
