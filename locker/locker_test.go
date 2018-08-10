package locker

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeLocker struct {
	ch chan struct{}
}

func (f *fakeLocker) Wait() <-chan struct{} {
	return f.ch
}

func (f *fakeLocker) Lock() {}

func (f *fakeLocker) Unlock() {
	close(f.ch)
}

func TestRLock(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		f := &fakeLocker{ch: make(chan struct{})}
		close(f.ch)
		rlock := NewRLock(f, nil)
		ch := make(chan struct{})
		go func() {
			rlock.Add()
			rlock.Lock()
			ch <- struct{}{}
		}()
		waitChanImmediately(t, ch)
		rlock.Unlock()
		w := MustWaiter(rlock)
		waitChanImmediately(t, w)
	})

	t.Run("not add before locking", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil || (r.(string) != "not add lock") {
				assert.Fail(t, "not add before lock")
			}
		}()
		f := &fakeLocker{ch: make(chan struct{})}
		close(f.ch)
		rlock := NewRLock(f, nil)
		rlock.Lock()
	})

	t.Run("not lock before unlocking", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil || (r.(string) != "unlock item before holding it") {
				assert.Fail(t, "not lock before unlocking")
			}
		}()
		f := &fakeLocker{ch: make(chan struct{})}
		close(f.ch)
		rlock := NewRLock(f, nil)
		rlock.Add()
		rlock.Unlock()
	})

	t.Run("lock before unlocking", func(t *testing.T) {
		rlock := NewRLock(nil, nil)
		rlock.Add()
		rlock.Lock()
		rlock.Unlock()

		ch := make(chan struct{})
		go func() {
			rlock.Add()
			rlock.Lock()
			ch <- struct{}{}

			rlock.Add()
			rlock.Lock()
			ch <- struct{}{}
		}()
		waitChanImmediately(t, ch)
		rlock.Unlock()
		waitChanImmediately(t, ch)
		rlock.Unlock()
		w := MustWaiter(rlock)
		waitChanImmediately(t, w)

		rlock.Add()
		rlock.Lock()
		rlock.Unlock()
		waitChanImmediately(t, rlock.Wait())
	})
}

func TestLock(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		f := &fakeLocker{ch: make(chan struct{})}
		lock := NewLock(f, nil)
		done := make(chan struct{})
		go func() {
			lock.Lock()
			done <- struct{}{}
		}()
		go func() {
			select {
			case <-done:
				assert.Fail(t, "holding lock before parent release it")
			case <-time.After(time.Millisecond * 5):
			}
		}()
		time.Sleep(time.Millisecond * 6)
		close(f.ch)
		<-done
		lock.Unlock()
		waitChanImmediately(t, lock.Wait())
	})

	t.Run("unlock before locking", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil || (r.(string) != "unlock item before holding it") {
				assert.Fail(t, "not lock before unlocking")
			}
		}()
		lock := NewLock(nil, nil)
		lock.Unlock()
	})
}

func TestLockGroup(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		var lockers = make([]Locker, 2)
		lockers[0] = NewLock(nil, 1)
		lockers[1] = NewLock(nil, 2)
		lg := NewLockGroup(nil, lockers)
		done := make(chan struct{})
		go func() {
			lg.Lock()
			done <- struct{}{}
		}()
		waitChanImmediately(t, done)
		lg.Unlock()
		waitChanImmediately(t, MustWaiter(lockers[0]))
		waitChanImmediately(t, MustWaiter(lockers[1]))
	})

	t.Run("unlock before locking", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil || (r.(string) != "unlock item before holding it") {
				assert.Fail(t, "not lock before unlocking")
			}
		}()
		lock := NewLockGroup(nil, []Locker{})
		lock.Unlock()
	})
}

func TestLockPipe(t *testing.T) {
	t.Run("normal pipe", func(t *testing.T) {
		var count int
		var wg sync.WaitGroup
		wg.Add(3)
		r1 := NewRLock(nil, 1)
		r1.Add()
		go func() {
			defer wg.Done()
			r1.Lock()
			count++
			r1.Unlock()
		}()
		r2 := NewRLock(r1, 2)
		r2.Add()
		go func() {
			defer wg.Done()
			r2.Lock()
			count++
			r2.Unlock()
		}()

		w1 := NewLock(r2, 3)
		go func() {
			defer wg.Done()
			w1.Lock()
			count++
			w1.Unlock()
		}()
		wg.Wait()
		assert.Equal(t, 3, count)
		waitChanImmediately(t, w1.Wait())
	})
}

func waitChanImmediately(t *testing.T, ch <-chan struct{}) {
	select {
	case <-ch:
	case <-time.After(time.Millisecond * 10000):
		assert.Fail(t, "should receive immediately")
	}
}
