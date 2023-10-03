package doc

import (
	"strings"
	"sync"
)

// Document is a thread safe, ordered collection of lines.
//
// Document keeps track of the current selection and updates it when lines are added or removed.
//
// A Document can have Capacity expressed as the total number of lines that the document can hold.
// If any operation that adds lines exceeds capacity, lines at the top are removed accordingly.
// Setting capacity to zero or negative disables the capacity checks.
//
// The zero value of Document is ready for use.
type Document struct {
	capacity       int
	lines          []string
	version        uint64
	selectionStart Anchor
	selectionEnd   Anchor
	lock           sync.RWMutex
}

// Capacity is the maximum number of lines that the document can hold.
func (d *Document) Capacity() int {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.capacity
}

// SetCapacity updates the document capacity. If c is positive and is less than the current
// number of lines in the document, lines at the top are removed to satisfy the new
// capacity.
func (d *Document) SetCapacity(c int) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.capacity = c
	d.enforceCapacity()
}

// Read allows to perform a read-only action on the list of lines.
//
// The lines slice must be treated as read-only and volatile. No portion
// of this slice must be cached nor modified. The strings themselves can be
// reused.
func (d *Document) Read(action func(lines []string)) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	action(d.lines)
}

// Add adds lines at the bottom of the document.
// If number of lines in the document exceeds Capacity, lines at the top are removed
// and Selection is adjusted accordingly.
func (d *Document) Add(lines ...string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.lines = append(d.lines, lines...)
	d.enforceCapacity()
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
	start, end = d.selectionStart, d.selectionEnd
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

// enforceCapacity assumes a write lock.
func (d *Document) enforceCapacity() {
	if d.capacity > 0 && len(d.lines) > d.capacity {
		d.removeLines(0, len(d.lines)-d.capacity)
	}
}

// removeLines assumes a write lock.
func (d *Document) removeLines(start, end int) {
	n := copy(d.lines[start:], d.lines[end:])
	// allow GC to collect the removed lines
	for i := start + n; i < len(d.lines); i++ {
		d.lines[i] = ""
	}
	d.lines = d.lines[:start+n]

	// update selection
	d.selectionStart = removeLinesFromAnchor(d.selectionStart, start, end)
	d.selectionEnd = removeLinesFromAnchor(d.selectionEnd, start, end)
}

func removeLinesFromAnchor(a Anchor, start, end int) Anchor {
	if start == end {
		return a
	}
	if a.LineIndex >= end {
		a.LineIndex -= end - start
	} else if a.LineIndex >= start {
		a = Anchor{LineIndex: start}
	}
	return a
}
