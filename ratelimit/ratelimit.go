//go:build !solution

package ratelimit

import (
	"context"
	"errors"
	"time"
)

// Limiter is precise rate limiter with context support.
type Limiter struct {
	interval time.Duration
	ch       chan struct{}
	stop     chan struct{}
}

var ErrStopped = errors.New("limiter stopped")

// NewLimiter returns limiter that throttles rate of successful Acquire() calls
// to maxSize events at any given interval.
func NewLimiter(maxCount int, interval time.Duration) *Limiter {
	ch := make(chan struct{}, maxCount)
	stop := make(chan struct{})
	return &Limiter{interval: interval, ch: ch, stop: stop}
}

func (l *Limiter) isStopped() bool {
	select {
	case <-l.stop:
		return true
	default:
		return false
	}
}

func (l *Limiter) Acquire(ctx context.Context) error {
	select {
	case l.ch <- struct{}{}:

		if l.isStopped() {
			<-l.ch
			return ErrStopped
		}

		if err := ctx.Err(); err != nil {
			<-l.ch
			return err
		}

		tick := time.After(l.interval)
		go func() {
			select {
			case <-tick:
			case <-l.stop:
			}
			<-l.ch
		}()

		return nil

	case <-l.stop:
		return ErrStopped

	case <-ctx.Done():
		if l.isStopped() {
			return ErrStopped
		}

		return ctx.Err()
	}

}

func (l *Limiter) Stop() {
	close(l.stop)
}
