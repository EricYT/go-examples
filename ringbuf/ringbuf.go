package ringbuf

import (
	"fmt"
	"sync"
)

type RingBuffer interface {
	Append(vals []Entry)
	//Reset()
	All() []Entry
}

type ringbuf struct {
	mu sync.RWMutex

	size int

	written    int
	writeCusor int

	ring []Entry
}

func New(size int) *ringbuf {
	if size <= 0 {
		panic(fmt.Sprintf("wrong size %d", size))
	}
	r := &ringbuf{
		size: size,
		ring: make([]Entry, 0, size),
	}
	return r
}

func (r *ringbuf) Append(vals []Entry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.written += len(vals)

	// we just care about the last entries
	if len(vals) > r.size {
		vals = vals[len(vals)-r.size:]
	}

	remain := r.size - r.writeCusor
	if remain < len(vals) {
		for i, e := range vals[remain:] {
			r.ring[i] = e
		}
		vals = vals[:remain]
	}
	r.ring = append(r.ring, vals...)
	r.writeCusor = (r.writeCusor + len(vals)) % r.size
}

func (r *ringbuf) All() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// FIXME: clone it
	return r.ring
}

type Entry struct {
	Value interface{}
}
