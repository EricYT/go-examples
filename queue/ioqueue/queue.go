package ioqueue

import "time"

type ioqueue struct {
	schedulerC chan chan *FairQueueRequestDescriptor
	queueC     chan *FairQueueRequestDescriptor
	doneC      chan struct{}
}

func newIOQueue(scheduler chan chan *FairQueueRequestDescriptor, sr func(func())) *ioqueue {
	q := &ioqueue{
		schedulerC: scheduler,
		queueC:     make(chan *FairQueueRequestDescriptor),
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
			desc.Fn()
			desc.ErrorC <- nil
			RequestDurationMetric(desc.Type, start, desc.Size)
		case <-q.doneC:
			return
		}
	}
}
