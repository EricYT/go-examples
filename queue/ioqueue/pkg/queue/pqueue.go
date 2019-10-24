// Package pqueue provides a priority queue implementation.
package queue

import (
	"container/heap"
)

// Item in the PriorityQueue.
type Item struct {
	Value    interface{}
	Priority float64
	index    int
}

// PriorityQueue as implemented by a min heap
// ie. the 0th element is the *lowest* value.
type PriorityQueue []*Item

// New creates a PriorityQueue of the given capacity.
func NewPqueue(capacity int) PriorityQueue {
	if capacity <= 0 {
		capacity = 1
	}
	return make(PriorityQueue, 0, capacity)
}

// Len returns the length of the queue.
func (pq PriorityQueue) Len() int {
	return len(pq)
}

// Less returns true if the item at index i has a lower priority than the item
// at index j.
func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Priority < pq[j].Priority
}

// Swap the items at index i and j.
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push a new value to the queue.
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	c := cap(*pq)
	if n+1 > c {
		npq := make(PriorityQueue, n, c*2)
		copy(npq, *pq)
		*pq = npq
	}
	*pq = (*pq)[0 : n+1]
	item := x.(*Item)
	item.index = n
	(*pq)[n] = item
}

// Pop an item from the queue.
func (pq *PriorityQueue) Pop() interface{} {
	n := len(*pq)
	c := cap(*pq)
	if n < (c/4) && c > 25 {
		npq := make(PriorityQueue, n, c/2)
		copy(npq, *pq)
		*pq = npq
	}
	item := (*pq)[n-1]
	item.index = -1
	*pq = (*pq)[0 : n-1]
	return item
}

// PeekAndShift based the max priority.
func (pq *PriorityQueue) PeekAndShift(max float64) (*Item, float64) {
	if pq.Len() == 0 {
		return nil, 0
	}

	item := (*pq)[0]
	if item.Priority > max {
		return nil, item.Priority - max
	}
	heap.Remove(pq, 0)

	return item, 0
}
