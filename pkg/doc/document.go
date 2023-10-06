package doc

import (
	"strings"
	"sync"
)

const (
	// BookmarkStart is a special bookmark that always points to the start of the document.
	BookmarkStart = bookmark(iota)

	// BookmarkEnd is a special bookmark that always points to the end of the document.
	BookmarkEnd
)

// Document is a thread safe, ordered collection of lines.
//
// Document keeps track of an arbitrary number of bookmarks and updates them when lines are added
// or removed.
//
// A Document can have Capacity expressed as the total number of lines that the document can hold.
// If any operation that adds lines exceeds capacity, lines at the top are removed accordingly.
// Setting capacity to zero or negative disables the capacity checks.
//
// Use New to create document instances.
type Document interface {
	// Capacity is the maximum number of lines that the document can hold.
	Capacity() int

	// SetCapacity updates the document capacity. If c is positive and is less than the current
	// number of lines in the document, lines at the top are removed to satisfy the new
	// capacity.
	SetCapacity(c int)

	// Read allows to perform a read-only action on the list of lines.
	//
	// The lines slice must be treated as read-only and volatile. No portion
	// of this slice must be cached nor modified. The strings themselves can be
	// reused.
	Read(action func(lines []string))

	// Add adds lines at the bottom of the document.
	// If number of lines in the document exceeds Capacity, lines at the top are removed
	// and bookmarks are adjusted accordingly.
	Add(lines ...string)

	// Version is an opaque value that changes every time the Document is mutated.
	Version() uint64

	// GetBookmark retrieves a bookmark with the given key.
	// If there is no bookmark with the given key, the return value will be (Anchor{}, false).
	//
	// The special bookmarks BookmarkStart and BookmarkEnd can be used to retrieve anchors for
	// the start and end of the current document, respectively.
	GetBookmark(key any) (anchor Anchor, ok bool)

	// SetBookmark sets or replaces a bookmark with a given key with the new anchor.
	//
	// Key can be anything comparable, but to avoid collisions and improve performance
	// it is recommended to define keys as constants of a distinct unexported type:
	//
	//  type myBookmark int
	//  const (
	//	    bookmarkThing = myBookmark(iota)
	//	    bookmarkSomethingElse
	//  )
	//
	// If anchor is above the first line or below the last line, SetBookmark adjusts it to point
	// to the start or end of the document, respectively. If anchor's LineIndex exists, but
	// LineOffset points to before the line starts or after the line ends, SetBookmark adjusts
	// it to point to the line start or end, respectively.
	//
	// The special bookmarks BookmarkStart and BookmarkEnd cannot be modified. An attempt to set
	// any of them results in a panic.
	SetBookmark(key any, anchor Anchor)

	// RemoveBookmark removes a bookmark with the given key.
	// Removing a bookmark that is not in the document is a no-op.
	RemoveBookmark(key any)

	// String returns contents of the document between two bookmarks.
	// Individual lines are joined with the separator.
	//
	// Any valid bookmarks can be used. Bookmarks can be given in any order. E.g. the following
	// two calls return the same values, assuming that the document didn't change between
	// invocations:
	//
	//	all1, ok1 := document.String(BookmarkStart(), BookmarkEnd(), "\n")
	//	all2, ok2 := document.String(BookmarkEnd(), BookmarkStart(), "\n")
	//
	// If one or both bookmarks do not exist, String returns ("", false).
	String(bookmark1, bookmark2 any, separator string) (string, bool)
}

// New creates a new instance of Document.
func New() Document {
	d := document{}
	d.bookmarks = make(map[any]Anchor)
	return &d
}

type document struct {
	lines          []string
	capacity       int
	version        uint64
	selectionStart Anchor
	selectionEnd   Anchor
	bookmarks      map[any]Anchor
	lock           sync.RWMutex
}

type bookmark int

func (d *document) Capacity() int {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.capacity
}

func (d *document) SetCapacity(c int) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.capacity = c
	d.enforceCapacity()
}

func (d *document) Read(action func(lines []string)) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	action(d.lines)
}

func (d *document) Add(lines ...string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.lines = append(d.lines, lines...)
	d.enforceCapacity()
	d.version++
}

func (d *document) Version() uint64 {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.version
}

func (d *document) GetBookmark(key any) (anchor Anchor, ok bool) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.bookmark(key)
}

func (d *document) SetBookmark(key any, anchor Anchor) {
	if key == BookmarkStart || key == BookmarkEnd {
		panic("doc/SetBookmark: BookmarkStart and BookmarkEnd are immutable")
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	d.bookmarks[key] = d.clampAnchor(anchor)
}

func (d *document) RemoveBookmark(key any) {
	d.lock.Lock()
	defer d.lock.Unlock()
	delete(d.bookmarks, key)
}

func (d *document) String(bookmark1, bookmark2 any, separator string) (string, bool) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	start, startOk := d.bookmark(bookmark1)
	end, endOk := d.bookmark(bookmark2)

	if !startOk || !endOk {
		return "", false
	}

	if len(d.lines) == 0 {
		return "", true
	}

	if start.Compare(end) > 0 {
		start, end = end, start
	}

	firstLine := d.lines[start.LineIndex]
	var builder strings.Builder

	if start.LineIndex == end.LineIndex {
		builder.WriteString(firstLine[start.LineOffset:end.LineOffset])
	} else {
		builder.WriteString(firstLine[start.LineOffset:])
		builder.WriteString(separator)
		for i := start.LineIndex + 1; i < end.LineIndex; i++ {
			builder.WriteString(d.lines[i])
			builder.WriteString(separator)
		}
		builder.WriteString(d.lines[end.LineIndex][:end.LineOffset])
	}

	return builder.String(), true
}

// enforceCapacity assumes a write lock.
func (d *document) enforceCapacity() {
	if d.capacity > 0 && len(d.lines) > d.capacity {
		d.removeLines(0, len(d.lines)-d.capacity)
	}
}

// removeLines assumes a write lock.
func (d *document) removeLines(start, end int) {
	n := copy(d.lines[start:], d.lines[end:])
	// allow GC to collect the removed lines
	for i := start + n; i < len(d.lines); i++ {
		d.lines[i] = ""
	}
	d.lines = d.lines[:start+n]

	// update selection
	d.selectionStart = removeLinesFromAnchor(d.selectionStart, start, end)
	d.selectionEnd = removeLinesFromAnchor(d.selectionEnd, start, end)
	for k, v := range d.bookmarks {
		d.bookmarks[k] = removeLinesFromAnchor(v, start, end)
	}
}

// endAnchor assumes at least a read lock.
func (d *document) endAnchor() Anchor {
	if len(d.lines) == 0 {
		return Anchor{}
	} else {
		lastLine := len(d.lines) - 1
		return Anchor{
			LineIndex:  lastLine,
			LineOffset: len(d.lines[lastLine]),
		}
	}
}

// bookmark assumes at least a read lock.
func (d *document) bookmark(key any) (a Anchor, ok bool) {
	if key == BookmarkStart {
		a, ok = Anchor{}, true
	} else if key == BookmarkEnd {
		a, ok = d.endAnchor(), true
	} else {
		a, ok = d.bookmarks[key]
	}
	return
}

// clampAnchor ensures that the anchor is within document boundaries.
// clampAnchor assumes at least a read lock.
func (d *document) clampAnchor(anchor Anchor) Anchor {
	if anchor.Compare(Anchor{}) < 0 {
		return Anchor{}
	}
	end := d.endAnchor()
	if anchor.Compare(end) >= 0 {
		return end
	}
	if anchor.LineOffset < 0 {
		anchor.LineOffset = 0
	} else if anchor.LineOffset > len(d.lines[anchor.LineIndex]) {
		anchor.LineOffset = len(d.lines[anchor.LineIndex])
	}
	return anchor
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
