package ioqueue

import "time"

type ioDescriptor interface {
	Do()
	Done(err error)
	Type() RequestType
	RequestSize() int
}

type ioqueue struct {
	schedulerC chan chan ioDescriptor
	queueC     chan ioDescriptor
	doneC      chan struct{}
}

func newIOQueue(scheduler chan chan ioDescriptor, sr func(func())) *ioqueue {
	q := &ioqueue{
		schedulerC: scheduler,
		queueC:     make(chan ioDescriptor),
		doneC:      make(chan struct{}),
	}
	sr(q.run)
	return q
}

func (q *ioqueue) done() {
	close(q.doneC)
}

func (q *ioqueue) run() {
	var start time.Time

	for {
		// register into scheduler
		q.schedulerC <- q.queueC

		select {
		case desc := <-q.queueC:
			start = time.Now()
			desc.Do()
			desc.Done(nil)
			RequestDurationMetric(desc.Type(), start, desc.RequestSize())
		case <-q.doneC:
			return
		}
	}
}
