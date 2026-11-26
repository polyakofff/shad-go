//go:build !solution

package genericsum

import (
	"sort"
	"sync"

	"golang.org/x/exp/constraints"
)

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func SortSlice[T constraints.Ordered](a []T) {
	sort.Slice(a, func(i, j int) bool {
		return a[i] < a[j]
	})
}

func MapsEqual[K comparable, V comparable](a, b map[K]V) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || v != bv {
			return false
		}
	}
	return true
}

func SliceContains[T comparable](s []T, v T) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func MergeChans[T any](chs ...<-chan T) <-chan T {
	out := make(chan T)

	wg := sync.WaitGroup{}

	for _, ch := range chs {
		wg.Add(1)
		go func(ch <-chan T) {
			for c := range ch {
				out <- c
			}
			wg.Done()
		}(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

type Number interface {
	constraints.Integer | constraints.Float | constraints.Complex
}

func IsHermitianMatrix[T Number](m [][]T) bool {
	rows := len(m)
	if rows == 0 {
		return true
	}
	cols := len(m[0])
	if rows != cols {
		return false
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if !equalsHermitian(m[i][j], m[j][i]) {
				return false
			}
		}
	}
	return true
}

func equalsHermitian[T Number](a, b T) bool {
	switch a := any(a).(type) {
	case complex64:
		b := any(b).(complex64)
		return a == complex(real(b), -imag(b))
	case complex128:
		b := any(b).(complex128)
		return a == complex(real(b), -imag(b))
	default:
		return a == b
	}
}
