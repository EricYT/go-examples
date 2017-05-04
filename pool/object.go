package pool

import (
	"reflect"
	"sync"
)

var PoolObjecterType reflect.Type = reflect.TypeOf((*PoolObjecter)(nil)).Elem()

type PoolObjecter interface {
	Return() error
	MarkUnusable()
}

// PoolObject is a wrapper around Object to modify the the behavior of
// Object's Return() method.
type PoolObject struct {
	Object
	mu       sync.RWMutex
	c        *channelPool
	unusable bool
}

// Return() puts the given Object back to the pool instead of closing it.
func (p *PoolObject) Return() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.unusable {
		if p.Object != nil {
			return p.Object.Close()
		}
		return nil
	}
	return p.c.put(p.Object)
}

// MarkUnusable() marks the Object not usable any more, to let the pool close it instead of returning it to pool.
func (p *PoolObject) MarkUnusable() {
	p.mu.Lock()
	p.unusable = true
	p.mu.Unlock()
}

// newConn wraps a standard net.Conn to a poolConn net.Conn.
func (c *channelPool) wrapObj(obj Object) *PoolObject {
	p := &PoolObject{c: c}
	p.Object = obj
	return p
}
