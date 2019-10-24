package ioqueue

import (
	"fmt"
	"sync"
	"testing"
)

func TestIOQueue_run(t *testing.T) {
	q := NewIOQueue(Mountpoint{
		MP:             "/disk1",
		ReadBytesRate:  100,
		WriteBytesRate: 100,
		WriteReqRate:   10,
		ReadReqRate:    20,
		NumIOQueues:    1,
	})

	q.RegisterPriorityClass("a", 6000)
	q.RegisterPriorityClass("b", 2000)

	var wg sync.WaitGroup
	wg.Add(20)
	for i := 0; i < 10; i++ {
		fn := func(a string, i int) func() {
			return func() {
				defer wg.Done()
				fmt.Printf("%s#%d\n", a, i)
			}
		}
		q.QueueRequest("a", i, RequestTypeWrite, fn("a-write", i))
		q.QueueRequest("b", i, RequestTypeRead, fn("b-read", i))
	}
	wg.Wait()

	q.Close()
}
