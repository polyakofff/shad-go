//go:build !solution

package tparallel

import (
	"sync"
)

type T struct {
	parentWg       *sync.WaitGroup
	parentParallel *[]chan struct{}
	wg             *sync.WaitGroup
	parallel       *[]chan struct{}
}

func (t *T) Parallel() {
	ch := make(chan struct{})
	*t.parentParallel = append(*t.parentParallel, ch)
	t.parentWg.Done()
	<-ch // block
}

func (t *T) Run(subtest func(t *T)) {
	runTest(t.wg, t.parallel, subtest)
}

func Run(topTests []func(t *T)) {

	wg := sync.WaitGroup{}
	parallel := make([]chan struct{}, 0)

	for _, test := range topTests {
		runTest(&wg, &parallel, test)
	}

	wg.Add(len(parallel))

	for _, ch := range parallel {
		close(ch)
	}

	wg.Wait()
}

func runTest(parentWg *sync.WaitGroup, parentParallel *[]chan struct{}, test func(t *T)) {
	wg := sync.WaitGroup{}
	parallel := make([]chan struct{}, 0)
	t := &T{
		parentWg:       parentWg,
		parentParallel: parentParallel,
		wg:             &wg,
		parallel:       &parallel,
	}

	parentWg.Add(1)

	go func() {
		test(t)

		wg.Add(len(parallel))

		for _, ch := range parallel {
			close(ch)
		}

		wg.Wait()

		parentWg.Done()
	}()

	parentWg.Wait()
}
