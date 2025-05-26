package crdt

import "sync"

type GCounter struct {
	mu     sync.RWMutex
	id     string
	counts map[string]int
}

func NewGCounter(id string) *GCounter {
	return &GCounter{
		id:     id,
		counts: map[string]int{id: 0},
	}
}

func (g *GCounter) Increment() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counts[g.id]++
}

func (g *GCounter) Merge(other map[string]int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for id, val := range other {
		if current, ok := g.counts[id]; !ok || val > current {
			g.counts[id] = val
		}
	}
}

func (g *GCounter) Value() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	total := 0
	for _, v := range g.counts {
		total += v
	}
	return total
}

func (g *GCounter) Snapshot() map[string]int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	cp := make(map[string]int)
	for k, v := range g.counts {
		cp[k] = v
	}
	return cp
}

func (g *GCounter) Delta() map[string]int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return map[string]int{g.id: g.counts[g.id]}
}
