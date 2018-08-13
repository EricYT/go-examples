package locker

import (
	"reflect"
	"sync"
	"sync/atomic"
)

type Locker interface {
	Lock()
	Unlock()
}

type Waiter interface {
	Wait() <-chan struct{}
}

var _ Locker = (*lockGroup)(nil)

// a bounch of locks
type lockGroup struct {
	items   []LockItem
	lockers []Locker

	locked int32
}

func NewLockGroup(items []LockItem, lockers []Locker) *lockGroup {
	if len(lockers) == 0 {
		// nothing to require, panic
		panic("no lockers")
	}
	lg := &lockGroup{
		items:   items,
		lockers: lockers,
	}
	return lg
}

func (l *lockGroup) Lock() {
	if !atomic.CompareAndSwapInt32(&l.locked, 0, 1) {
		panic("lock item more than once")
	}

	cases := make([]reflect.SelectCase, len(l.lockers))
	for i, locker := range l.lockers {
		ch := LockPrepare(locker)
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}
	// Waiting all locks are awake.
	for {
		chosen, _, _ := reflect.Select(cases)
		cases = append(cases[:chosen], cases[chosen+1:]...)
		if len(cases) == 0 {
			// all lockers are awake
			return
		}
	}
}

func (l *lockGroup) Unlock() {
	if !atomic.CompareAndSwapInt32(&l.locked, 1, 0) {
		panic("unlock item before holding it")
	}
	for _, locker := range l.lockers {
		locker.Unlock()
	}
}

var _ Locker = (*rlock)(nil)
var _ Waiter = (*rlock)(nil)

type rlock struct {
	parent <-chan struct{}
	item   interface{}

	mutex    sync.Mutex
	locked   bool
	unlocked bool
	lockers  int32
	release  chan struct{}
}

func NewRLock(p Locker, item interface{}) *rlock {
	parent := MustWaiter(p)
	return &rlock{
		parent:  parent,
		item:    item,
		release: make(chan struct{}),
	}
}

func (l *rlock) Wait() <-chan struct{} {
	return l.release
}

func (l *rlock) reset() {
	// reset signal channel
	l.release = make(chan struct{})
	l.unlocked = false
	l.locked = false
}

func (l *rlock) Add() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.unlocked {
		l.reset()
	}
	l.lockers++
}

func (l *rlock) LockPrepare() <-chan struct{} {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.lockers == 0 {
		panic("not add lock")
	}
	l.locked = true
	return l.parent
}

func (l *rlock) Lock() {
	// wait the parent release the lock firstly.
	<-l.LockPrepare()
}

func (l *rlock) Unlock() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.lockers--
	if !l.locked || l.lockers < 0 {
		panic("unlock item before holding it")
	}
	if l.lockers == 0 {
		// no one hold this lock, release it.
		l.unlocked = true
		close(l.release)
	}
}

var _ Locker = (*lock)(nil)
var _ Waiter = (*lock)(nil)

type lock struct {
	parent <-chan struct{}
	item   interface{}

	locked  int32
	release chan struct{}
}

func NewLock(p Locker, item interface{}) *lock {
	parent := MustWaiter(p)
	return &lock{
		parent:  parent,
		item:    item,
		release: make(chan struct{}),
	}
}

func (l *lock) Wait() <-chan struct{} {
	return l.release
}

func (l *lock) LockPrepare() <-chan struct{} {
	if !atomic.CompareAndSwapInt32(&l.locked, 0, 1) {
		panic("lock item more than once")
	}
	return l.parent
}

func (l *lock) Lock() {
	<-l.LockPrepare()
}

func (l *lock) Unlock() {
	if !atomic.CompareAndSwapInt32(&l.locked, 1, 0) {
		panic("unlock item before holding it")
	}
	close(l.release)
}

func IsRLock(l Locker) bool {
	switch l.(type) {
	case *rlock:
		return true
	default:
	}
	return false
}

func MustWaiter(l Locker) <-chan struct{} {
	if l == nil {
		var closedCh = make(chan struct{})
		close(closedCh)
		return closedCh
	}
	w := l.(Waiter)
	return w.Wait()
}

func LockPrepare(locker Locker) <-chan struct{} {
	type LockPreparer interface {
		LockPrepare() <-chan struct{}
	}
	if l, ok := locker.(LockPreparer); ok {
		return l.LockPrepare()
	}
	// The worst one, we spawn a goroutine to wait for holding lock.
	done := make(chan struct{})
	go func() {
		defer close(done)
		locker.Lock()
	}()
	return done
}
