package linelist

import "sync"

type LineList struct {
	lines []string
	lock  sync.RWMutex
}

// WithReadLock allows to perform a read-only action on the list of lines.
func (l *LineList) Read(action func(lines []string)) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	action(l.lines)
}

// WithWriteLock allows to perform a mutating action on the list of lines.
func (l *LineList) Modify(action func(lines []string) []string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.lines = action(l.lines)
}

func (l *LineList) Add(lines ...string) {
	l.Modify(func(l []string) []string {
		return append(l, lines...)
	})
}
