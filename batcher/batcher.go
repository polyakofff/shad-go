//go:build !solution

package batcher

import (
	"sync"

	"gitlab.com/slon/shad-go/batcher/slow"
)

// решение 1 (мое)

type Batcher struct {
	mu      sync.Mutex
	v       *slow.Value
	loading []chan interface{}
	waiting []chan interface{}
}

func NewBatcher(v *slow.Value) *Batcher {
	return &Batcher{
		v:       v,
		loading: []chan interface{}{},
		waiting: []chan interface{}{},
	}
}

func (b *Batcher) Load() interface{} {

	b.mu.Lock()
	ch := make(chan interface{})

	if len(b.loading) == 0 {
		b.loading = append(b.loading, ch)
		b.mu.Unlock()

		go func() {
			for {
				v := b.v.Load()

				b.mu.Lock()
				for _, ch := range b.loading {
					ch <- v
					close(ch)
				}
				b.loading = b.waiting
				b.waiting = []chan interface{}{}
				if len(b.loading) == 0 {
					b.mu.Unlock()
					return
				}
				b.mu.Unlock()
			}
		}()

	} else {
		b.waiting = append(b.waiting, ch)
		b.mu.Unlock()
	}

	return <-ch

}

// решение 2 (гпт)

// type Batcher struct {
// 	mu      sync.Mutex
// 	v       *slow.Value
// 	running bool
// 	active  *batch
// 	next    *batch
// }

// type batch struct {
// 	value interface{}
// 	done  chan struct{}
// }

// func NewBatcher(v *slow.Value) *Batcher {
// 	return &Batcher{
// 		v: v,
// 	}
// }

// func (b *Batcher) Load() interface{} {

// 	b.mu.Lock()

// 	var current *batch

// 	if !b.running {
// 		current = &batch{
// 			done: make(chan struct{}),
// 		}
// 		b.active = current
// 		b.running = true

// 		go b.run()
// 	} else {
// 		if b.next == nil {
// 			b.next = &batch{
// 				done: make(chan struct{}),
// 			}
// 		}
// 		current = b.next
// 	}

// 	b.mu.Unlock()

// 	<-current.done
// 	return current.value
// }

// func (b *Batcher) run() {
// 	for {
// 		value := b.v.Load()

// 		b.mu.Lock()
// 		b.active.value = value
// 		close(b.active.done)

// 		if b.next == nil {
// 			b.running = false
// 			b.mu.Unlock()
// 			return
// 		}

// 		b.active = b.next
// 		b.next = nil
// 		b.mu.Unlock()
// 	}
// }
