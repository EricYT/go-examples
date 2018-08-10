package locker_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/EricYT/go-examples/locker"
	"github.com/stretchr/testify/assert"
)

func TestLockItems(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		lm := locker.NewLockManagerService()

		var wg sync.WaitGroup
		wg.Add(2)
		var count int32

		r1 := locker.LockItem{Type: locker.LockTypeRead, Item: 1}
		r2 := locker.LockItem{Type: locker.LockTypeRead, Item: 2}
		rg1 := lm.LockItems(r1, r2)
		go func() {
			defer wg.Done()
			time.Sleep(time.Millisecond * 5)
			rg1.Lock()
			assert.True(t, atomic.CompareAndSwapInt32(&count, 0, 1))
			rg1.Unlock()
		}()

		w1 := locker.LockItem{Type: locker.LockTypeWrite, Item: 1}
		r21 := locker.LockItem{Type: locker.LockTypeRead, Item: 2}
		wg1 := lm.LockItems(w1, r21)
		go func() {
			defer wg.Done()
			wg1.Lock()
			assert.True(t, atomic.CompareAndSwapInt32(&count, 1, 2))
			wg1.Unlock()
		}()
	})
}
