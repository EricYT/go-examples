package signal

import (
	"errors"
	"os"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func simulateSignalTriggerAndHandle(t *testing.T, handle SignalHandleFunc, sig os.Signal, errExpected error) {
	// clean up hadnles
	ResetHandles()
	// capture SIGHUP
	SetHandleForSignals(handle, sig)

	var pidChan = make(chan int, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pidChan <- os.Getpid()
		// capture signals
		assert.Equal(t, ServeHandleSignals(), errExpected, "simulate signal got unexpected error")
	}()
	// send a SIGHUP signal
	syscall.Kill(<-pidChan, sig.(syscall.Signal))
	wg.Wait()
}

func TestCaptureSignal(t *testing.T) {
	var executorChan = make(chan bool, 1)
	captureSIGHUP := func(sig os.Signal) error {
		assert.Equal(t, sig, syscall.SIGHUP, "We should get a SIGHUP signal")
		// in case the handle just not executed
		executorChan <- true
		return ErrStop
	}
	simulateSignalTriggerAndHandle(t, captureSIGHUP, syscall.SIGHUP, nil)
	assert.Equal(t, <-executorChan, true, "capture signal handle already done")
}

func TestIgnoreSignal(t *testing.T) {
	var executorChan = make(chan bool, 1)
	var errorIgnoreSignal error = errors.New("signal test: ignore signal")
	ignoreSIGINT := func(sig os.Signal) error {
		assert.Equal(t, sig, syscall.SIGINT, "We should get a SIGINT signal")
		executorChan <- true
		// ignore it
		return errorIgnoreSignal
	}
	simulateSignalTriggerAndHandle(t, ignoreSIGINT, syscall.SIGINT, errorIgnoreSignal)
	assert.Equal(t, <-executorChan, true, "capture signal handle already done")
}
