//go:build !solution

package httpgauge

import (
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/go-chi/chi/v5"
)

type Gauge struct {
	mu    sync.RWMutex
	count map[string]int
}

func New() *Gauge {
	return &Gauge{
		count: make(map[string]int),
	}
}

func (g *Gauge) Snapshot() map[string]int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	snapshot := make(map[string]int)
	for k, v := range g.count {
		snapshot[k] = v
	}
	return snapshot
}

// ServeHTTP returns accumulated statistics in text format ordered by pattern.
//
// For example:
//
//	/a 10
//	/b 5
//	/c/{id} 7
func (g *Gauge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	snapshot := g.Snapshot()

	keys := make([]string, 0, len(snapshot))
	for k := range snapshot {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Fprintf(w, "%s %d\n", k, snapshot[k])
	}
}

func (g *Gauge) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			pattern := getRoutePattern(r)
			if pattern != "" {
				g.mu.Lock()
				g.count[pattern]++
				g.mu.Unlock()
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func getRoutePattern(r *http.Request) string {
	ctx := chi.RouteContext(r.Context())

	return ctx.RoutePattern()
}
