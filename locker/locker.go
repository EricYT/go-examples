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
	lg := &lockGroup{
		items:   items,
		lockers: lockers,
	}
	return lg
}

func (l *lockGroup) Lock() {
	atomic.CompareAndSwapInt32(&l.locked, 0, 1)

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
	if l.lockers == 0 {
		panic("not add lock")
	}
	l.locked = true
	l.mutex.Unlock()

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
	atomic.CompareAndSwapInt32(&l.locked, 0, 1)
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
	case *lock:
		return false
	default:
		panic("unknow type locker")
	}
}

func MustWaiter(l Locker) <-chan struct{} {
	if l == nil {
		var closedCh = make(chan struct{})
		close(closedCh)
		return closedCh
	}
	if w, ok := l.(Waiter); ok {
		return w.Wait()
	}
	panic("locker not implement Wait method")
}

func LockPrepare(l Locker) <-chan struct{} {
	type LockPreparer interface {
		LockPrepare() <-chan struct{}
	}
	if lock, ok := l.(LockPreparer); ok {
		return lock.LockPrepare()
	}
	panic("locker in lock group should be read or write lock")
}
