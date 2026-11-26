//go:build !solution

package dupcall

import (
	"context"
	"sync"
)

type Call struct {
	mu      sync.Mutex
	waiters int
	entry   *entry
}

type entry struct {
	result interface{}
	err    error
	done   chan struct{}
	cancel context.CancelFunc
}

func (o *Call) Do(
	ctx context.Context,
	cb func(context.Context) (interface{}, error),
) (result interface{}, err error) {

	o.mu.Lock()
	e := o.entry

	if e == nil {
		o.waiters++
		cbCtx, cancel := context.WithCancel(ctx)
		e = &entry{
			done:   make(chan struct{}),
			cancel: cancel,
		}
		o.entry = e
		o.mu.Unlock()

		go func(ctx context.Context) {

			res, err := cb(ctx)

			o.mu.Lock()
			e.result = res
			e.err = err
			close(e.done)
			o.mu.Unlock()

		}(cbCtx)
	} else {
		o.waiters++
		o.mu.Unlock()
	}

	select {
	case <-ctx.Done(): // cancel
		o.mu.Lock()
		o.waiters--
		if o.waiters == 0 {
			e.cancel()
			o.entry = nil
		}
		o.mu.Unlock()

		return nil, ctx.Err()

	case <-e.done: // done
		result, err = e.result, e.err

		o.mu.Lock()
		o.waiters--
		if o.waiters == 0 {
			o.entry = nil
		}
		o.mu.Unlock()

		return
	}
}
