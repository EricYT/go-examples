package token_bucket

import (
	"errors"
	"sync"
	"time"

	tomb "gopkg.in/tomb.v1"
)

// inspire from paper: Rate-Based Active Queue Management with Token Buckets
// Section: Rate-Based AQM with Token Buckets

var (
	ErrRBTBW1OrW2Empty     error = errors.New("RateBasedTokenBucketQueue: w1 or w2 is empty")
	ErrRBTBAlreadyDead     error = errors.New("RateBasedTokenBucketQueue: already dead")
	ErrRBTBDequeueTimeout  error = errors.New("RateBasedTokenBucketQueue: dequeue timeout")
	ErrRBTBDequeueEmpty    error = errors.New("RateBasedTokenBucketQueue: dequeue queue is empty")
	ErrRBTBEnqueueOverflow error = errors.New("RateBasedTokenBucketQueue: enqueue queue overflow")
	ErrRBTBEnqueueDeny     error = errors.New("RateBasedTokenBucketQueue: enqueue deny")
)

const (
	defaultW1 float64 = 32
	defaultW2 float64 = 64

	defaultMinFillupSize    int           = 50
	defaultMaxFillupSize    int           = 100
	defaultPeakBustSize     int           = 1000
	defaultIntervalTime     time.Duration = 1 * time.Second
	defaultMaxQueueCapacity int           = 2000
	defaultThreshold        int           = 1200
)

type rateBasedTokenBucketQueue struct {
	tomb  *tomb.Tomb
	mutex sync.Mutex

	w1 float64
	w2 float64

	minFillupSize int
	maxFillupSize int
	peakBustSize  int

	current  int
	capacity int

	threshold int
	interval  time.Duration

	fillupCh chan int
	queue    []*itemWapper
}

func NewRateBasedTokenBucketQueue(w1, w2 float64, minfs, maxfs, pbs, cap, tsd int, interval time.Duration) *rateBasedTokenBucketQueue {
	if w1 == 0 || w2 == 0 {
		panic(ErrRBTBW1OrW2Empty)
	}

	r := &rateBasedTokenBucketQueue{
		tomb:          new(tomb.Tomb),
		w1:            w1,
		w2:            w1,
		minFillupSize: minfs,
		maxFillupSize: maxfs,
		peakBustSize:  pbs,
		capacity:      cap,
		current:       minfs,
		threshold:     tsd,
		interval:      interval,
		queue:         []*itemWapper{},
		fillupCh:      make(chan int, 1),
	}

	// go fillup goroutine
	go r.fillup()

	// start main loop
	go func() {
		defer r.tomb.Done()
		r.tomb.Kill(r.runLoop())
	}()

	return r
}

func NewDefaultRateBasedTokenBucketQueue() *rateBasedTokenBucketQueue {
	return NewRateBasedTokenBucketQueue(
		defaultW1,
		defaultW2,
		defaultMinFillupSize,
		defaultMaxFillupSize,
		defaultPeakBustSize,
		defaultMaxQueueCapacity,
		defaultThreshold,
		defaultIntervalTime,
	)
}

func (r *rateBasedTokenBucketQueue) Enqueue(item interface{}, token int) error {
	select {
	case <-r.tomb.Dying():
		return ErrRBTBAlreadyDead
	default:
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if len(r.queue) > r.capacity {
		return ErrRBTBEnqueueOverflow
	}
	if token > r.current {
		return ErrRBTBEnqueueDeny
	}
	r.current -= token
	r.queue = append(r.queue, &itemWapper{token, item})

	// notify dequeue waiter

	return nil
}

func (r *rateBasedTokenBucketQueue) Dequeue(timeout time.Duration) (interface{}, error) {
	select {
	case <-r.tomb.Dying():
		return nil, ErrRBTBAlreadyDead
	default:
	}

	r.mutex.Lock()
	if len(r.queue) > 0 {
		iw := r.queue[0]
		r.queue = r.queue[1:]
		r.mutex.Unlock()
		return iw.item, nil
	}
	r.mutex.Unlock()
	return nil, ErrRBTBDequeueEmpty
}

func (r *rateBasedTokenBucketQueue) runLoop() error {
	for {
		select {
		case token := <-r.fillupCh:
			r.mutex.Lock()
			curr := r.current + token
			if curr > r.capacity {
				r.current = r.capacity
			}
			r.current = curr
			r.mutex.Unlock()
		case <-r.tomb.Dying():
			return nil
		}
	}
}

func (r *rateBasedTokenBucketQueue) fillup() {
	ticker := time.NewTicker(r.interval)
	lastToken := r.minFillupSize
	for {
		select {
		case <-ticker.C:
			r.mutex.Lock()
			var token int
			if len(r.queue) > r.threshold {
				// decrease put in tokens
				token = int(float64(lastToken) * (1 - 1/r.w2))
			} else {
				token = lastToken + r.maxFillupSize/int(r.w1)
				if token > r.maxFillupSize {
					token = r.maxFillupSize
				} else if token < r.minFillupSize {
					token = r.minFillupSize
				}
			}
			lastToken = token
			r.mutex.Unlock()
			r.fillupCh <- token
		case <-r.tomb.Dying():
			return
		}
	}
}

type itemWapper struct {
	token int
	item  interface{}
}
