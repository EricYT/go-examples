package limit_test

import (
	"testing"
	"time"

	"github.com/EricYT/go-examples/limit"
	"github.com/stretchr/testify/assert"
)

func TestLimit(t *testing.T) {
	var lm limit.Limiter = limit.NewLimiter(2)

	assert.True(t, lm.Acquire())
	assert.True(t, lm.Acquire())
	assert.False(t, lm.Acquire())
	assert.Nil(t, lm.Release())
	assert.Nil(t, lm.Release())
	err := lm.Release()
	if assert.NotNil(t, err) {
		assert.Equal(t, "Release one token without holding it", err.Error())
	}
	assert.True(t, lm.Acquire())
	assert.True(t, lm.Acquire())
	assert.False(t, lm.Acquire())
}

func TestLimitBlock(t *testing.T) {
	var lm limit.Limiter = limit.NewLimiter(2)
	wait := make(chan struct{})
	done := make(chan struct{})
	go func() {
		<-wait
		assert.True(t, lm.Acquire())
		assert.True(t, lm.Acquire())
		assert.False(t, lm.Acquire())
		lm.AcquireWait()
		assert.False(t, lm.Acquire())
		assert.Nil(t, lm.Release())
	}()
	wait <- struct{}{}
	assert.Nil(t, lm.Release())
	go func() {
		lm.AcquireWait()
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-time.After(time.Millisecond * 10):
		assert.Fail(t, "Cant acquire one token")
	}
}

func TestLimitWithPause(t *testing.T) {
	var lm limit.Limiter = limit.NewLimiterWithPause(2, time.Millisecond*10, time.Millisecond*20)
	wait := make(chan struct{})
	done := make(chan struct{})
	go func() {
		<-wait
		assert.True(t, lm.Acquire())
		assert.True(t, lm.Acquire())
		done <- struct{}{}
	}()
	wait <- struct{}{}
	select {
	case <-done:
	case <-time.After(time.Millisecond * 45):
		assert.Fail(t, "Over maximu wait timeout")
	}
}
