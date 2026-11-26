//go:build race

package batcher

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/slon/shad-go/batcher/slow"
	"go.uber.org/goleak"
)

func init() {
	race = true
}

func TestRaceConcurrentLoads(t *testing.T) {
	defer goleak.VerifyNone(t)

	const (
		N = 100
		K = 20
	)

	var value slow.Value
	value.Store(42)

	b := NewBatcher(&value)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < N; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			for j := 0; j < K; j++ {
				if got := b.Load(); got != 42 {
					t.Errorf("got %v, want 42", got)
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
}

func TestRaceConcurrentStoreLoad(t *testing.T) {
	defer goleak.VerifyNone(t)

	const (
		N = 32
		K = 50
		M = 200
	)

	var value slow.Value
	value.Store(int64(0))

	b := NewBatcher(&value)

	var published int64
	var wg sync.WaitGroup
	start := make(chan struct{})

	// Одновременно обновляем значение.
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start

		for i := int64(1); i <= M; i++ {
			value.Store(i)
			atomic.StoreInt64(&published, i)

			time.Sleep(100 * time.Microsecond)
		}
	}()

	// Параллельно читаем его.
	for i := 0; i < N; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			for j := 0; j < K; j++ {
				before := atomic.LoadInt64(&published)
				got := b.Load().(int64)

				if got < before {
					t.Errorf(
						"stale read: got %d, want >= %d",
						got, before,
					)
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
}
