package linelist

import (
	"reprapctl/internal/pkg/doc"
	"strings"
	"sync"
)

type LineList struct {
	lines          []string
	version        uint64
	selectionStart doc.Anchor
	selectionEnd   doc.Anchor
	lock           sync.RWMutex
}

// WithReadLock allows to perform a read-only action on the list of lines.
//
// The lines slice must be treated as read-only and volatile. No portion
// of this slice must be cached nor modified. The strings themselves can be
// reused.
func (l *LineList) Read(action func(lines []string)) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	action(l.lines)
}

func (l *LineList) Add(lines ...string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.lines = append(l.lines, lines...)
	l.version++
}

// Version is an opaque value that changes every time the LineList is mutated.
func (l *LineList) Version() uint64 {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.version
}

// Selection returns the current selection boundaries in ascending order.
func (l *LineList) Selection() (start, end doc.Anchor) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	start = l.selectionStart
	end = l.selectionEnd
	if start.Compare(end) > 0 {
		start, end = end, start
	}
	return
}

func (l *LineList) StartSelection(a doc.Anchor) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.selectionStart = a
	l.selectionEnd = a
}

func (l *LineList) SetSelectionEnd(a doc.Anchor) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.selectionEnd = a
}

func (l *LineList) SelectNone() {
	l.StartSelection(doc.Anchor{})
}

func (l *LineList) SelectAll() {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.selectionStart = doc.Anchor{}
	if len(l.lines) == 0 {
		l.selectionEnd = doc.Anchor{}
	} else {
		l.selectionEnd.LineIndex = len(l.lines) - 1
		l.selectionEnd.LineOffset = len(l.lines[len(l.lines)-1])
	}
}

func (l *LineList) SelectionToString() string {
	var builder strings.Builder

	func() {
		l.lock.RLock()
		defer l.lock.RUnlock()

		if len(l.lines) == 0 {
			return
		}

		firstLine := l.lines[l.selectionStart.LineIndex]

		if l.selectionStart.LineIndex == l.selectionEnd.LineIndex {
			builder.WriteString(firstLine[l.selectionStart.LineOffset:l.selectionEnd.LineOffset])
		} else {
			builder.WriteString(firstLine[l.selectionStart.LineOffset:])
			builder.WriteRune('\n')
			for i := l.selectionStart.LineIndex + 1; i < l.selectionEnd.LineIndex; i++ {
				builder.WriteString(l.lines[i])
				builder.WriteRune('\n')
			}
			builder.WriteString(l.lines[l.selectionEnd.LineIndex][:l.selectionEnd.LineOffset])
		}
	}()

	return builder.String()
}
