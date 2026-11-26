//go:build !solution

package keylock

import (
	"sort"
	"sync"
)

type KeyLock struct {
	mu    sync.Mutex
	locks map[string]chan struct{}
}

func New() *KeyLock {
	return &KeyLock{
		locks: map[string]chan struct{}{},
	}
}

func (l *KeyLock) LockKeys(keys []string, cancel <-chan struct{}) (canceled bool, unlock func()) {
	sorted := make([]string, len(keys))
	copy(sorted, keys)
	sort.Strings(sorted)

	processed := make([]string, 0, len(keys))
	unlock = func() {
		for _, key := range processed {
			l.mu.Lock()
			lock := l.locks[key]
			l.mu.Unlock()
			<-lock
		}
	}

	for _, key := range sorted {
		// lock key
		l.mu.Lock()
		lock, ok := l.locks[key]
		if !ok {
			lock = make(chan struct{}, 1)
			l.locks[key] = lock
		}
		l.mu.Unlock()

		select {
		case lock <- struct{}{}: // lock acquired
			processed = append(processed, key)
		case <-cancel: // cancel
			canceled = true
			unlock()
			return
		}
	}

	canceled = false

	return
}
