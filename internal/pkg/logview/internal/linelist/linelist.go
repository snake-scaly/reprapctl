package linelist

import "sync"

type LineList struct {
	lines   []string
	version uint64
	lock    sync.RWMutex
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
