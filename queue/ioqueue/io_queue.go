package ioqueue

import (
	"errors"
	"math"
	"sync"
)

// A general disk io queue inspired by seastar.
// TODO: how to separate io between normal file io and leveldb log io?

var (
	ErrIOQueueClosed error = errors.New("IO queue closed")
)

type RequestType int

const (
	RequestTypeWrite RequestType = iota + 1
	RequestTypeRead

	/*
	   // We want to represent the fact that write requests are (maybe) more expensive
	   // than read requests. To avoid dealing with floating point math we will scale one
	   // read request to be counted by this amount.
	   //
	   // A write request that is 30% more expensive than a read will be accounted as
	   // (read_request_base_count * 130) / 100.
	   // It is also technically possible for reads to be the expensive ones, in which case
	   // writes will have an integer value lower than read_request_base_count.
	*/
	ReadRequestBaseCount int = 128
)

type ioQueueConfig struct {
	mountpoint                     string
	diskBytesWriteToReadMultiplier uint64
	diskReqWriteToReadMultiplier   uint64
	maxBytesCount                  uint64
	maxReqCount                    uint64
}

func newIOQueueConfig(p Mountpoint) ioQueueConfig {
	cfg := ioQueueConfig{
		mountpoint:                     "undefined",
		maxBytesCount:                  math.MaxUint64,
		maxReqCount:                    math.MaxUint64,
		diskBytesWriteToReadMultiplier: uint64(ReadRequestBaseCount),
		diskReqWriteToReadMultiplier:   uint64(ReadRequestBaseCount),
	}

	maxBandwidth := max(p.ReadBytesRate, p.WriteBytesRate)
	maxIOPS := max(p.ReadReqRate, p.WriteReqRate)

	cfg.diskBytesWriteToReadMultiplier = (uint64(ReadRequestBaseCount) * p.ReadBytesRate) / p.WriteBytesRate
	cfg.diskReqWriteToReadMultiplier = (uint64(ReadRequestBaseCount) * p.ReadReqRate) / p.WriteReqRate
	if maxBandwidth != math.MaxUint64 {
		cfg.maxBytesCount = uint64(ReadRequestBaseCount) * (maxBandwidth / p.NumIOQueues)
	}
	if maxIOPS != math.MaxUint64 {
		cfg.maxReqCount = uint64(ReadRequestBaseCount) * (maxIOPS / p.NumIOQueues)
	}
	cfg.mountpoint = p.MP

	return cfg
}

func newFairQueueConfig(iocfg ioQueueConfig) FairQueueConfig {
	var c FairQueueConfig
	c.maxReqCount = iocfg.maxReqCount
	c.maxBytesCount = iocfg.maxBytesCount
	c.tau = 100 * 1000 // 100ms convert to Nanosecond
	return c
}

type IOQueue struct {
	mu  sync.Mutex
	cfg ioQueueConfig

	fq *FairQueue

	// NOTE: concurrent queue
	schedulerC chan chan *FairQueueRequestDescriptor
	queues     []*ioqueue

	wg      sync.WaitGroup
	closing bool
	closeC  chan struct{}
	signalC chan struct{}
}

func NewIOQueue(mp Mountpoint) *IOQueue {
	q := &IOQueue{
		cfg:     newIOQueueConfig(mp),
		closeC:  make(chan struct{}),
		signalC: make(chan struct{}),
	}
	q.fq = NewFairQueue(newFairQueueConfig(q.cfg), 128)

	// ioqueue
	q.schedulerC = make(chan chan *FairQueueRequestDescriptor, int(mp.NumIOQueues))
	for i := 0; i < int(mp.NumIOQueues); i++ {
		w := newIOQueue(q.schedulerC, q.safeAttach)
		q.queues = append(q.queues, w)
	}

	q.safeAttach(q.run)

	return q
}

func (q *IOQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closing {
		return
	}
	q.closing = true
	close(q.closeC)
	for _, i := range q.queues {
		i.done()
	}
	q.wg.Wait()
	q.fq.Close()
}

func (q *IOQueue) QueueRequest(pc string, size int, reqType RequestType, fn func()) (IOFuture, error) {
	des := &FairQueueRequestDescriptor{Fn: fn, ErrorC: make(chan error, 1)}
	if reqType == RequestTypeWrite {
		des.Weight = int(q.cfg.diskReqWriteToReadMultiplier)
		des.Size = int(q.cfg.diskBytesWriteToReadMultiplier) * size
	} else {
		des.Weight = ReadRequestBaseCount
		des.Size = ReadRequestBaseCount * size
	}

	if err := q.fq.Enqueue(pc, des); err != nil {
		return nil, err
	}

	QueueRequestMetric(reqType, size)

	// wake up main loop to consume io request
	if q.fq.Size() == 1 {
		q.signalC <- struct{}{}
	}

	return &ioFuture{errC: des.ErrorC}, nil
}

func (q *IOQueue) RegisterPriorityClass(name string, shares uint32) {
	q.fq.RegisterPriorityClass(name, shares)
}

func (q *IOQueue) UnregisterPriorityClass(name string) {
	q.fq.UnregisterPriorityClass(name)
}

func (q *IOQueue) dispatchRequest(desc *FairQueueRequestDescriptor) {
	select {
	case jobC := <-q.schedulerC:
		jobC <- desc
	case <-q.closeC:
		// FIXME:
		desc.ErrorC <- ErrIOQueueClosed
		return
	}
}

func (q *IOQueue) popOneRequest() (*FairQueueRequestDescriptor, bool) {
	return q.fq.Dequeue()
}

func (q *IOQueue) run() {

	next := q.signalC
	closedC := make(chan struct{})
	close(closedC)

	for {
		select {
		case <-next:

			req, empty := q.popOneRequest()
			if req != nil {
				q.dispatchRequest(req)
			}
			if empty {
				next = q.signalC
			} else {
				next = closedC
			}

		case <-q.closeC:
			return
		}
	}
}

func (q *IOQueue) safeAttach(f func()) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.wg.Add(1)
	go func() {
		// FIXME: recover ?
		defer q.wg.Done()
		f()
	}()
}

type IOFuture interface {
	Done() error
}

type ioFuture struct {
	errC chan error
}

func (f *ioFuture) Done() error {
	return <-f.errC
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
