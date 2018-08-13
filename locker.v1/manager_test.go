package locker_test

import (
	"sync/atomic"
	"testing"
	"time"

	"git.jd.com/cloud-storage/newds-master/service/metadata/locker"
	"github.com/stretchr/testify/assert"
)

func TestManagerLockGroup(t *testing.T) {
	t.Run("lock items validate", func(t *testing.T) {
		defer func() {
			r := recover()
			err := r.(error)
			if err == nil || (err.Error() != "lock: unknow lock item type") {
				assert.Fail(t, "lock: unknow lock item type")
			}
		}()
		lm := locker.NewLockManagerService()
		items := []locker.LockItem{
			{
				Type: locker.LockTypeRead,
				Item: 1,
			},
			{
				Type: locker.LockItemType(3),
				Item: 2,
			},
		}
		lm.NewLockGroup(items...)
	})

	t.Run("lock items all rlocks", func(t *testing.T) {
		lm := locker.NewLockManagerService()
		items := []locker.LockItem{
			{
				Type: locker.LockTypeRead,
				Item: 1,
			},
			{
				Type: locker.LockTypeRead,
				Item: 2,
			},
			{
				Type: locker.LockTypeRead,
				Item: 1,
			},
		}
		l := lm.NewLockGroup(items...)
		done := make(chan struct{})
		go func() {
			defer func() { done <- struct{}{} }()
			l.Lock()
		}()
		waitChanImmediately(t, done)
		l.Unlock()

		l = lm.NewLockGroup(locker.LockItem{locker.LockTypeRead, 1})
		go func() {
			defer func() { done <- struct{}{} }()
			l.Lock()
		}()
		waitChanImmediately(t, done)
	})

	t.Run("lock items all locks", func(t *testing.T) {
		lm := locker.NewLockManagerService()
		items := []locker.LockItem{
			{
				Type: locker.LockTypeWrite,
				Item: 1,
			},
			{
				Type: locker.LockTypeWrite,
				Item: 2,
			},
		}
		l := lm.NewLockGroup(items...)
		var step int32
		done := make(chan struct{})
		go func() {
			defer func() { done <- struct{}{} }()
			l.Lock()
			assert.True(t, atomic.CompareAndSwapInt32(&step, 0, 1))
		}()
		waitChanImmediately(t, done)
		l.Unlock()
		assert.True(t, atomic.CompareAndSwapInt32(&step, 1, 2))

		l = lm.NewLockGroup(locker.LockItem{locker.LockTypeWrite, 1})
		go func() {
			defer func() { done <- struct{}{} }()
			l.Lock()
			assert.True(t, atomic.CompareAndSwapInt32(&step, 2, 3))
		}()
		waitChanImmediately(t, done)
		assert.True(t, atomic.CompareAndSwapInt32(&step, 3, 4))
	})

	t.Run("lock items mix", func(t *testing.T) {
		lm := locker.NewLockManagerService()
		items := []locker.LockItem{
			{
				Type: locker.LockTypeWrite,
				Item: 1,
			},
			{
				Type: locker.LockTypeWrite,
				Item: 2,
			},
		}
		l := lm.NewLockGroup(items...)
		var step int32
		done := make(chan struct{})
		go func() {
			defer func() { done <- struct{}{} }()
			l.Lock()
			assert.True(t, atomic.CompareAndSwapInt32(&step, 0, 1))
		}()
		waitChanImmediately(t, done)
		go func() {
			time.Sleep(time.Millisecond * 500)
			l.Unlock()
			assert.True(t, atomic.CompareAndSwapInt32(&step, 1, 2))
		}()

		l1 := lm.NewLockGroup(locker.LockItem{locker.LockTypeRead, 1})
		go func() {
			defer func() { done <- struct{}{} }()
			l1.Lock()
			assert.True(t, atomic.CompareAndSwapInt32(&step, 2, 3))
		}()
		waitChanImmediately(t, done)

		go func() {
			time.Sleep(time.Millisecond * 500)
			l1.Unlock()
			assert.True(t, atomic.CompareAndSwapInt32(&step, 5, 6))
		}()

		l2 := lm.NewLockGroup(locker.LockItem{locker.LockTypeRead, 2})
		go func() {
			defer func() { done <- struct{}{} }()
			l2.Lock()
			assert.True(t, atomic.CompareAndSwapInt32(&step, 3, 4))
		}()
		waitChanImmediately(t, done)

		l3 := lm.NewLockGroup(locker.LockItem{locker.LockTypeRead, 1})
		go func() {
			defer func() { done <- struct{}{} }()
			l3.Lock()
			assert.True(t, atomic.CompareAndSwapInt32(&step, 4, 5))
		}()
		waitChanImmediately(t, done)

	})
}

func waitChanImmediately(t *testing.T, ch <-chan struct{}) {
	select {
	case <-ch:
	case <-time.After(time.Millisecond * 3000):
		assert.Fail(t, "should receive immediately")
	}
}
