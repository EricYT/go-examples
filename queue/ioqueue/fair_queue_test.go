package ioqueue

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFairQueue_RegisterPriorityClass(t *testing.T) {
	fq := NewFairQueue(FairQueueConfig{}, 10)
	pc1 := fq.RegisterPriorityClass("xxx", 1)
	pc2 := fq.RegisterPriorityClass("xxx", 2)
	if !assert.Equal(t, pc1, pc2) {
		return
	}
	fq.UnregisterPriorityClass("xxx")
	if !assert.Equal(t, 0, len(fq.handles)) {
		return
	}
	fq.UnregisterPriorityClass("xxx")
}

func TestFairQueue_Enqueue(t *testing.T) {
	fq := NewFairQueue(FairQueueConfig{maxReqCount: 1, maxBytesCount: 1, tau: 100 * 1000}, 10)

	fq.RegisterPriorityClass("a", 1)
	fq.RegisterPriorityClass("b", 2)
	fq.RegisterPriorityClass("c", 3)

	var wg sync.WaitGroup
	wg.Add(3)
	fn := func() func() {
		return func() {
			wg.Done()
		}
	}

	fq.Enqueue("a", &FairQueueRequestDescriptor{Weight: 1, Size: 1, Fn: fn()})
	fq.Enqueue("b", &FairQueueRequestDescriptor{Weight: 1, Size: 1, Fn: fn()})
	fq.Enqueue("c", &FairQueueRequestDescriptor{Weight: 1, Size: 1, Fn: fn()})

	if !assert.Equal(t, 3, fq.Size()) {
		return
	}

	for {
		desc, empty := fq.Dequeue()
		if desc != nil {
			desc.Fn()
		}
		if empty {
			break
		}
	}

	wg.Wait()

	fq.Close()

	size, err := fq.Enqueue("a", &FairQueueRequestDescriptor{Weight: 1, Size: 1, Fn: fn()})
	if !assert.Equal(t, ErrFairQueuePriorityClassNotFound, err) {
		return
	}
	if !assert.Equal(t, -1, size) {
		return
	}
}

func TestFairQueue_Close(t *testing.T) {
	fq := NewFairQueue(FairQueueConfig{maxReqCount: 1, maxBytesCount: 1, tau: 100 * 1000}, 10)
	fq.RegisterPriorityClass("a", 1)

	var descs []*FairQueueRequestDescriptor
	for i := 1; i < 3+1; i++ {
		desc := &FairQueueRequestDescriptor{Weight: 1, Size: 1, Fn: nil, ErrorC: make(chan error, 1)}
		size, err := fq.Enqueue("a", desc)
		if !assert.Nil(t, err) {
			return
		}
		if !assert.Equal(t, size, i) {
			return
		}
		descs = append(descs, desc)
	}

	fq.Close()

	for _, desc := range descs {
		err := <-desc.ErrorC
		if !assert.Equal(t, ErrFairQueueClosed, err) {
			return
		}
	}

	_, err := fq.Enqueue("a", nil)
	if !assert.Equal(t, ErrFairQueuePriorityClassNotFound, err) {
	}
}

func TestFairQueue_NormalizeStats(t *testing.T) {
	start := time.Now()

	fq := NewFairQueue(FairQueueConfig{maxReqCount: 1, maxBytesCount: 1, tau: uint64(40 * 1000)}, 10)
	fq.RegisterPriorityClass("a", 1000)

	pc := fq.allClasses["a"]
	fq.nextAccumulated(pc, &request{desc: &FairQueueRequestDescriptor{Weight: 1, Size: 1}})

	defer func(old func() time.Time) { _nowFn = old }(_nowFn)

	nextTimePoint := fq.base.Add(time.Hour * 24)
	_nowFn = func() time.Time {
		return nextTimePoint.Add(time.Now().Sub(start))
	}
	fq.nextAccumulated(pc, &request{desc: &FairQueueRequestDescriptor{Weight: 1, Size: 1}})

}

func BenchmarkFairQueue_Enqueue(b *testing.B) {
	fq := NewFairQueue(FairQueueConfig{maxReqCount: 1, maxBytesCount: 1, tau: uint64(100 * 1000)}, 10)
	fq.RegisterPriorityClass("a", 1)
	fq.RegisterPriorityClass("b", 2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			fq.Enqueue("a", &FairQueueRequestDescriptor{Weight: 1, Size: 1, Fn: nil})
		} else {
			fq.Enqueue("b", &FairQueueRequestDescriptor{Weight: 1, Size: 1, Fn: nil})
		}
	}
}

func BenchmarkFairQueue_Dequeue(b *testing.B) {
	fq := NewFairQueue(FairQueueConfig{maxReqCount: 1, maxBytesCount: 1, tau: uint64(100 * 1000)}, 10)
	fq.RegisterPriorityClass("a", 1)
	fq.RegisterPriorityClass("b", 2)

	for i := 0; i < 300000; i++ {
		if i%2 == 0 {
			fq.Enqueue("a", &FairQueueRequestDescriptor{Weight: 1, Size: 1, Fn: nil})
		} else {
			fq.Enqueue("b", &FairQueueRequestDescriptor{Weight: 1, Size: 1, Fn: nil})
		}
	}

	b.ResetTimer()
	empty := false
	for i := 0; i < b.N && !empty; i++ {
		_, empty = fq.Dequeue()
	}
}
