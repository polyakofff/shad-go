//go:build !solution

package pubsub

import (
	"context"
	"errors"
	"sync"
)

var _ Subscription = (*MySubscription)(nil)

type MySubscription struct {
	pubSub  *MyPubSub
	mu      *sync.Mutex
	cond    *sync.Cond
	queue   []interface{}
	stopped bool
}

func (s *MySubscription) Unsubscribe() {
	s.pubSub.mu.Lock()
	for _, subs := range s.pubSub.subs {
		delete(subs, s)
	}
	s.pubSub.mu.Unlock()

	s.mu.Lock()
	s.stopped = true
	s.cond.Signal()
	s.mu.Unlock()
}

var _ PubSub = (*MyPubSub)(nil)

type MyPubSub struct {
	mu      sync.Mutex
	subs    map[string]map[*MySubscription]struct{}
	done    chan struct{}
	workers int
	closed  bool
}

func NewPubSub() PubSub {
	done := make(chan struct{})
	close(done)
	return &MyPubSub{
		subs: make(map[string]map[*MySubscription]struct{}),
		done: done,
	}
}

func (p *MyPubSub) Subscribe(subj string, cb MsgHandler) (Subscription, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, errors.New("pubsub is closed")
	}

	var mu sync.Mutex
	s := &MySubscription{
		pubSub: p,
		mu:     &mu,
		cond:   sync.NewCond(&mu),
		queue:  make([]interface{}, 0),
	}

	if p.subs[subj] == nil {
		p.subs[subj] = make(map[*MySubscription]struct{})
	}
	p.subs[subj][s] = struct{}{}

	p.workers++
	if p.workers == 1 {
		p.done = make(chan struct{})
	}

	// launch subscriber
	go s.run(cb)

	return s, nil
}

func (s *MySubscription) run(cb MsgHandler) {
	defer func() {
		s.pubSub.mu.Lock()
		s.pubSub.workers--
		if s.pubSub.workers == 0 {
			close(s.pubSub.done)
		}
		s.pubSub.mu.Unlock()
	}()

	for {
		s.mu.Lock()

		for len(s.queue) == 0 && !s.stopped {
			s.cond.Wait()
		}

		if len(s.queue) == 0 && s.stopped {
			s.mu.Unlock()
			return
		}

		batch := s.queue
		s.queue = make([]interface{}, 0)

		s.mu.Unlock()

		for _, msg := range batch {
			cb(msg)
		}
	}
}

func (p *MyPubSub) Publish(subj string, msg interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errors.New("pubsub is closed")
	}

	for sub := range p.subs[subj] {
		sub.mu.Lock()
		sub.queue = append(sub.queue, msg)
		sub.cond.Signal()
		sub.mu.Unlock()
	}

	return nil
}

func (p *MyPubSub) Close(ctx context.Context) error {
	p.mu.Lock()

	p.closed = true

	for _, subs := range p.subs {
		for sub := range subs {
			sub.mu.Lock()
			sub.stopped = true
			sub.cond.Signal()
			sub.mu.Unlock()
		}
	}

	p.mu.Unlock()

	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
