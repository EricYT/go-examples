package queue

import (
	"strconv"
	"testing"
)

func TestQueueEnqueue(t *testing.T) {
	queue := NewQueue()
	queueSize := 100

	// Populate test queue and assertFn Enqueue
	// function does not fail
	for i := 0; i < queueSize; i++ {
		var value string = strconv.Itoa(i)
		queue.Enqueue(value)
	}

	assertFn(
		t,
		queue.Size() == queueSize,
		"queue.Size() = %d; want %d", queue.Size(), 3,
	)

	assertFn(
		t,
		queue.First() == "99",
		"queue.Size() = %s; want %s", queue.First(), "99")

	assertFn(
		t,
		queue.Last() == "0",
		"queue.Last() = %s; want %s", queue.Last(), "0",
	)
}

func TestQueueDequeue_fulfilled(t *testing.T) {
	queue := NewQueue()
	queueSize := 100

	// Populate test queue and assertFn Enqueue
	// function does not fail
	for i := 0; i < queueSize; i++ {
		var value string = strconv.Itoa(i)
		queue.Enqueue(value)
	}

	// Check that while deuqueing, elements come out in
	// their insertion order
	for i := 0; i < queueSize; i++ {
		item := queue.Dequeue()
		expectedValue := strconv.Itoa(i)

		assertFn(
			t,
			item == expectedValue,
			"queue.Dequeue() = %s; want %s", item, expectedValue,
		)

		assertFn(
			t,
			queue.Size() == queueSize-(i+1),
			"queue.Size() = %d; want %d", queue.Size(), queueSize-(i+1),
		)
	}
}

func TestQueueDequeue_empty(t *testing.T) {
	queue := NewQueue()
	item := queue.Dequeue()

	assertFn(
		t,
		item == nil,
		"queue.Dequeue() = %v; want %v", item, nil,
	)

	assertFn(
		t,
		queue.Size() == 0,
		"queue.Size() = %d; want %d", queue.Size(), 0,
	)
}

func TestQueueHead_fulfilled(t *testing.T) {
	queue := NewQueue()
	queue.Enqueue("1")
	item := queue.Head()

	assertFn(
		t,
		item == "1",
		"queue.Enqueue() = %s; want %s", item, "1",
	)

	assertFn(
		t,
		queue.Size() == 1,
		"queue.Size() = %d; want %d", queue.Size(), 1,
	)
}

func TestQueueHead_empty(t *testing.T) {
	queue := NewQueue()
	item := queue.Head()

	assertFn(
		t,
		item == nil,
		"queue.Head() = %v; want %v", item, nil,
	)

	assertFn(
		t,
		queue.Size() == 0,
		"queue.Size() = %d; want %d", queue.Size(), 0,
	)
}
