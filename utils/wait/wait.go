package wait

import "sync"

// Wait is a interface that provides ability to wait and trigger events that
// are associated with IDs.
type Wait interface {
	Register(id uint64) <-chan interface{}
	IsRegister(id uint64) bool
	Trigger(id uint64, x interface{}) bool
}

type wait struct {
	l sync.RWMutex
	m map[uint64]chan interface{}
}

func NewWait() *wait {
	return &wait{
		m: make(map[uint64]chan interface{}),
	}
}

func (w *wait) Register(id uint64) <-chan interface{} {
	w.l.Lock()
	c := w.m[id]
	w.l.Unlock()
	if c == nil {
		c = make(chan interface{}, 1)
		w.m[id] = c
	} else {
		panic("dup id")
	}
	return c
}

func (w *wait) Trigger(id uint64, x interface{}) bool {
	w.l.Lock()
	c := w.m[id]
	delete(w.m, id)
	w.l.Unlock()
	if c != nil {
		c <- x
		close(c)
		return true
	}
	return false
}

func (w *wait) IsRegister(id uint64) bool {
	w.l.RLock()
	_, ok := w.m[id]
	w.l.RUnlock()
	return ok
}
