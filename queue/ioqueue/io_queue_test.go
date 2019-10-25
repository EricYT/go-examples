package ioqueue

import (
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIOQueue_Run(t *testing.T) {
	q := NewIOQueue(Mountpoint{
		MP:             "/disk1",
		ReadBytesRate:  1,
		WriteBytesRate: 1,
		WriteReqRate:   1,
		ReadReqRate:    2,
		NumIOQueues:    1,
	})

	q.RegisterPriorityClass("a", 6000)
	q.RegisterPriorityClass("b", 2000)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		fn := func(a string, i int) func() {
			wg.Add(1)
			return func() {
				defer wg.Done()
			}
		}
		go q.QueueRequest("a", i, RequestTypeWrite, fn("a-write", i))
		go q.QueueRequest("b", i, RequestTypeRead, fn("b-read", i))
	}
	wg.Wait()

	q.Close()
	q.Close()
}

func TestIOQueue_DispathchClosed(t *testing.T) {
	q := NewIOQueue(Mountpoint{
		MP:             "/disk1",
		ReadBytesRate:  math.MaxUint64,
		WriteBytesRate: 1,
		WriteReqRate:   math.MaxUint64,
		ReadReqRate:    2,
		NumIOQueues:    0,
	})

	q.RegisterPriorityClass("a", 6000)

	fut, err := q.QueueRequest("a", 1, RequestTypeWrite, func() { t.Fatalf("not reach here") })
	if !assert.Nil(t, err) {
		return
	}

	q.Close()

	if !assert.Equal(t, ErrIOQueueClosed, fut.Done()) {
		return
	}

	_, err = q.QueueRequest("a", 1, RequestTypeWrite, func() { t.Fatalf("not reach here") })
	if !assert.Equal(t, ErrFairQueuePriorityClassNotFound, err) {
		return
	}
}
