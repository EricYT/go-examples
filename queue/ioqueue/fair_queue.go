package ioqueue

import (
	"container/heap"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/EricYT/go-examples/queue/ioqueue/pkg/queue"
)

/*
 This is a fair queue, allowing multiple request producers to queue requests
 that will then be served proportionally to their classes' shares.

 To each request, a weight can also be associated. A request of weight 1 will consume
 1 share. Higher weights for a request will consume a proportionally higher amount of
 shares.

 The user of this interface is expected to register multiple \ref priority_class
 objects, which will each have a shares attribute.

 Internally, each priority class may keep a separate queue of requests.
 Requests pertaining to a class can go through even if they are over its
 share limit, provided that the other classes have empty queues.

 When the classes that lag behind start seeing requests, the fair queue will serve
 them first, until balance is restored. This balancing is expected to happen within
 a certain time window that obeys an exponential decay.
*/

var (
	ErrFairQueueClosed                error = errors.New("fair queue closed")
	ErrFairQueueEmpty                 error = errors.New("fair queue empty")
	ErrFairQueuePriorityClassNotFound error = errors.New("fair queue priority class not found")
)

type FairQueueConfig struct {
	maxReqCount   uint64
	maxBytesCount uint64
	tau           uint64
}

type FairQueue struct {
	mu     sync.Mutex
	config FairQueueConfig

	base       time.Time
	handles    queue.PriorityQueue
	allClasses map[string]*PriorityClass
}

func NewFairQueue(cfg FairQueueConfig, cap int) *FairQueue {
	return &FairQueue{
		config:     cfg,
		base:       time.Now(),
		handles:    queue.NewPqueue(cap),
		allClasses: make(map[string]*PriorityClass),
	}
}

func (fq *FairQueue) Close() {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	for _, pc := range fq.allClasses {
		for !pc.Empty() {
			req := pc.Dequeue()
			req.desc.Done(ErrFairQueueClosed)
		}
	}
	fq.allClasses = nil
}

func (fq *FairQueue) RegisterPriorityClass(name string, shares uint32) *PriorityClass {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	if pc, ok := fq.allClasses[name]; ok {
		return pc
	}
	pc := NewPriorityClass(name, shares)
	fq.allClasses[name] = pc

	return pc
}

func (fq *FairQueue) UnregisterPriorityClass(name string) {
	fq.mu.Lock()
	defer fq.mu.Unlock()
	if _, ok := fq.allClasses[name]; !ok {
		return
	}
	delete(fq.allClasses, name)
}

func (fq *FairQueue) Size() int {
	fq.mu.Lock()
	defer fq.mu.Unlock()
	return fq.size()
}

func (fq *FairQueue) size() int {
	var total int
	for _, pc := range fq.allClasses {
		if pc.Queued() {
			total += pc.Size()
		}
	}
	return total
}

func (fq *FairQueue) Enqueue(name string, desc *FairQueueRequestDescriptor) (int, error) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	pc, ok := fq.allClasses[name]
	if !ok {
		return -1, ErrFairQueuePriorityClassNotFound
	}
	pc.Enqueue(&request{desc: desc})
	fq.pushPriorityClass(pc)
	return fq.size(), nil
}

func (fq *FairQueue) Dequeue() (*FairQueueRequestDescriptor, bool) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	pc := fq.popPriorityClass()
	if pc == nil {
		return nil, true
	}

	req := pc.Dequeue()
	nextAccumulated := fq.nextAccumulated(pc, req)
	pc.SetAccumulated(nextAccumulated)

	if !pc.Empty() {
		fq.pushPriorityClass(pc)
	}

	return req.desc, fq.handles.Len() == 0 // no pc in priority queue
}

var _nowFn = time.Now

func (fq *FairQueue) nextAccumulated(pc *PriorityClass, req *request) float64 {
	delta := _nowFn().Sub(fq.base).Milliseconds()
	reqCost := (float64(req.desc.Weight)/float64(fq.config.maxReqCount) + float64(req.desc.Size)/float64(fq.config.maxBytesCount)) / float64(pc.Shares())
	cost := math.Exp(float64(1)/float64(fq.config.tau)*float64(delta)) * reqCost
	nextAccumulated := pc.Accumulated() + cost
	for math.IsInf(nextAccumulated, 0) {
		fq.normalizeStats()
		// If we have renormalized, our time base will have changed. This should happen very infrequently
		delta = _nowFn().Sub(fq.base).Milliseconds()
		cost = math.Exp(float64(1)/float64(fq.config.tau)*float64(delta)) * reqCost
		nextAccumulated = pc.Accumulated() + cost
	}
	return nextAccumulated
}

func (fq *FairQueue) normalizeFactor() float64 {
	return float64(math.SmallestNonzeroFloat64)
}

func (fq *FairQueue) normalizeStats() {
	timeDelta := math.Log(fq.normalizeFactor()) * float64(fq.config.tau)
	//// time_delta is negative; and this may advance .base into the future
	fq.base = fq.base.Add(time.Duration(-timeDelta))
	for _, pc := range fq.allClasses {
		pc.SetAccumulated(pc.Accumulated() * fq.normalizeFactor())
	}
}

func (fq *FairQueue) pushPriorityClass(pc *PriorityClass) {
	if !pc.Queued() {
		heap.Push(&fq.handles, pc.Item())
		pc.SetQueued(true)
	}
}

func (fq *FairQueue) popPriorityClass() *PriorityClass {
	if fq.handles.Len() == 0 {
		return nil
	}
	pc := heap.Pop(&fq.handles).(*queue.Item).Value.(*PriorityClass)
	pc.SetQueued(false)
	return pc
}

// priority class
type FairQueueRequestDescriptor struct {
	Typ     RequestType
	Fn      func()
	ErrorC  chan error
	ReqSize int

	Weight int
	Size   int
}

func (desc *FairQueueRequestDescriptor) Do() {
	desc.Fn()
}

func (desc *FairQueueRequestDescriptor) Done(err error) {
	desc.ErrorC <- err
	close(desc.ErrorC) // in case someone block after calling more than once
}

func (desc *FairQueueRequestDescriptor) RequestSize() int {
	return desc.ReqSize
}

func (desc *FairQueueRequestDescriptor) Type() RequestType {
	return desc.Typ
}

type request struct {
	desc *FairQueueRequestDescriptor
}

type PriorityClass struct {
	mu sync.Mutex

	name   string
	shares uint32
	queue  *queue.Queue
	queued bool

	item *queue.Item
}

func NewPriorityClass(name string, shares uint32) *PriorityClass {
	pc := &PriorityClass{
		name:   name,
		shares: shares,
		queue:  queue.NewQueue(),
	}
	item := &queue.Item{Value: pc, Priority: 0}
	pc.item = item
	return pc
}

func (pc *PriorityClass) UpdateShares(shares uint32) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if shares <= 1 {
		shares = 1
	}
	pc.shares = shares
}

func (pc *PriorityClass) Shares() uint32 {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.shares
}

func (pc *PriorityClass) Accumulated() float64 {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.item.Priority
}

func (pc *PriorityClass) SetAccumulated(acc float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.item.Priority = acc
}

func (pc *PriorityClass) Queued() bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.queued
}

func (pc *PriorityClass) SetQueued(q bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.queued = q
}

func (pc *PriorityClass) Item() *queue.Item {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.item
}

func (pc *PriorityClass) Enqueue(req *request) {
	defer func() {
		FairQueuePriorityClassRequestsMetric(pc.name, pc.shares)
		FairQueuePriorityClassQueuedRequestsMetric(pc.name, pc.shares, pc.queue.Size())
	}()
	pc.queue.Enqueue(req)
}

func (pc *PriorityClass) Dequeue() *request {
	defer FairQueuePriorityClassQueuedRequestsMetric(pc.name, pc.shares, pc.queue.Size())
	return pc.queue.Dequeue().(*request)
}

func (pc *PriorityClass) Empty() bool {
	return pc.queue.Empty()
}

func (pc *PriorityClass) Size() int {
	return pc.queue.Size()
}
