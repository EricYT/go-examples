package limit

import (
	"errors"
	"math/rand"
	"time"
)

// Limiter represents a limited resource.
type Limiter interface {
	// Acquire returns a boolean indicate whether there
	// is a availability. Until another one release.
	Acquire() bool
	// AcquireWait obtains one resource until there is
	// one exist.
	AcquireWait()
	// Release try to release one resouce already hold.
	Release() error
}

var _ Limiter = (*limit)(nil)

type empty struct{}

type limit struct {
	wait     chan empty
	minPause time.Duration
	maxPause time.Duration
}

func NewLimiter(allowed int) *limit {
	return NewLimiterWithPause(allowed, 0, 0)
}

func NewLimiterWithPause(allowed int, min, max time.Duration) *limit {
	return &limit{
		wait:     make(chan empty, allowed),
		minPause: min,
		maxPause: max,
	}
}

func (l *limit) Acquire() bool {
	// Pause before attempting to grab it.
	// which is depended how to construct the
	// limiter, and throttle incoming connections.
	l.pause()
	select {
	case l.wait <- empty{}:
		return true
	default:
	}
	return false
}

func (l *limit) AcquireWait() {
	l.wait <- empty{}
}

func (l *limit) Release() error {
	select {
	case <-l.wait:
	default:
		return errors.New("Release one token without holding it")
	}
	return nil
}

func (l *limit) pause() {
	if l.minPause <= 0 || l.maxPause <= 0 {
		// no pausing
		return
	}
	pauseRange := (l.maxPause - l.minPause) / time.Millisecond
	pause := time.Duration(rand.Intn(int(pauseRange))) * time.Millisecond
	pause += l.minPause
	<-time.After(pause)
}
