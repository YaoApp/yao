package webproxy

import (
	"fmt"
	"sync"
)

// PortPool manages a range of ports for allocation.
type PortPool struct {
	mu        sync.Mutex
	start     int
	end       int
	available map[int]bool
}

// NewPortPool creates a port pool for the given range [start, end].
func NewPortPool(start, end int) *PortPool {
	available := make(map[int]bool, end-start+1)
	for i := start; i <= end; i++ {
		available[i] = true
	}
	return &PortPool{
		start:     start,
		end:       end,
		available: available,
	}
}

// Allocate returns the next available port from the pool.
func (p *PortPool) Allocate() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for port := range p.available {
		delete(p.available, port)
		return port, nil
	}
	return 0, fmt.Errorf("port pool exhausted (range %d-%d)", p.start, p.end)
}

// Release returns a port back to the pool.
func (p *PortPool) Release(port int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if port >= p.start && port <= p.end {
		p.available[port] = true
	}
}

// Available returns the number of ports currently available.
func (p *PortPool) Available() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.available)
}
