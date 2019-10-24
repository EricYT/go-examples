package ioqueue

import (
	"fmt"
	"testing"

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
}

func TestFairQueue_Enqueue(t *testing.T) {
	fq := NewFairQueue(FairQueueConfig{maxReqCount: 2, maxBytesCount: 1, tau: 100 * 1000}, 10)

	fq.RegisterPriorityClass("a", 1)
	fq.RegisterPriorityClass("b", 2)
	fq.RegisterPriorityClass("c", 3)

	fq.Enqueue("a", &FairQueueRequestDescriptor{Weight: 3, Size: 20, Fn: func() { fmt.Println("a1") }})
	fq.Enqueue("a", &FairQueueRequestDescriptor{Weight: 2, Size: 20, Fn: func() { fmt.Println("a2") }})
	fq.Enqueue("a", &FairQueueRequestDescriptor{Weight: 1, Size: 20, Fn: func() { fmt.Println("a3") }})

	fq.Enqueue("b", &FairQueueRequestDescriptor{Weight: 3, Size: 10, Fn: func() { fmt.Println("b1") }})
	fq.Enqueue("b", &FairQueueRequestDescriptor{Weight: 3, Size: 10, Fn: func() { fmt.Println("b2") }})
	fq.Enqueue("b", &FairQueueRequestDescriptor{Weight: 3, Size: 10, Fn: func() { fmt.Println("b3") }})

	fq.Enqueue("c", &FairQueueRequestDescriptor{Weight: 4, Size: 1, Fn: func() { fmt.Println("c1") }})
	fq.Enqueue("c", &FairQueueRequestDescriptor{Weight: 2, Size: 1, Fn: func() { fmt.Println("c2") }})
	fq.Enqueue("c", &FairQueueRequestDescriptor{Weight: 4, Size: 1, Fn: func() { fmt.Println("c3") }})

	for {
		desc, empty := fq.Dequeue()
		if desc != nil {
			desc.Fn()
		}
		if empty {
			break
		}
	}
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
