package gate

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrGateClosed error = errors.New("gate closed")

type Gater interface {
	Enter() error
	Leave()
	GetCount() int
	Close()
	IsClosed() bool
}

type Gate struct {
	mu sync.Mutex

	wg    sync.WaitGroup
	count int64 // access atomic

	closed bool
}

func New() *Gate {
	return &Gate{}
}

func (g *Gate) Enter() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.enter()
}

// no lock needed for leave
func (g *Gate) Leave() {
	g.leave()
}

func (g *Gate) enter() error {
	if g.closed {
		return ErrGateClosed
	}
	g.wg.Add(1)
	atomic.AddInt64(&g.count, 1)
	return nil
}

func (g *Gate) leave() {
	g.wg.Done()
	atomic.AddInt64(&g.count, -1)
}

func (g *Gate) With(f func()) error {
	if err := g.Enter(); err != nil {
		return err
	}
	go func() {
		defer g.Leave()
		f()
	}()
	return nil
}

func (g *Gate) GetCount() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.count
}

func (g *Gate) IsClosed() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.closed
}

func (g *Gate) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return
	}
	g.wg.Wait()
	count := atomic.LoadInt64(&g.count)
	if count != 0 {
		panic("count is not zero")
	}
	g.closed = true
}
