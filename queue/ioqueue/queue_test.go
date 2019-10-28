package ioqueue

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	stubDo   = func() {}
	stubDone = func(err error) {}
)

type fakeIODesc struct{}

func (f *fakeIODesc) Do() {
	stubDo()
}

func (f *fakeIODesc) Done(err error) {
	stubDone(err)
}

func (f *fakeIODesc) Type() RequestType { return RequestTypeWrite }

func (f *fakeIODesc) RequestSize() int { return 0 }

func TestQueue_New(t *testing.T) {
	var wg sync.WaitGroup
	scheduler := make(chan chan ioDescriptor, 1)
	sr := func(fn func()) {
		wg.Add(1)
		go func() {
			wg.Done()
			fn()
		}()
	}

	defer func(old func()) { stubDo = old }(stubDo)
	defer func(old func(error)) { stubDone = old }(stubDone)

	var count int
	done := make(chan struct{})
	stubDo = func() { count++ }
	stubDone = func(err error) {
		if !assert.Nil(t, err) {
			return
		}
		close(done)
	}

	q := newIOQueue(scheduler, sr)
	q.queueC <- &fakeIODesc{}
	<-done
	if !assert.Equal(t, 1, count) {
		return
	}
	q.done()
	wg.Wait()
}
