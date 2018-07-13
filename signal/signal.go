package signal

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
)

var ErrStop error = errors.New("signal: signal term receive")

// signal function handle
type SignalHandleFunc func(os.Signal) error

// signal function registers
// TODO: maybe for every one signal, there is more than one handle need to
// capture it.
var handles = make(map[os.Signal]SignalHandleFunc)

func init() {
	// default capture TERM signal
	handles[syscall.SIGTERM] = sigtermDefaultHandle
}

func sigtermDefaultHandle(os.Signal) error {
	return ErrStop
}

// reset
func ResetHandles() {
	handles = make(map[os.Signal]SignalHandleFunc)
}

// set handle for signals which one we want to
// capture it.
func SetHandleForSignals(handle SignalHandleFunc, signals ...os.Signal) {
	for _, signal := range signals {
		handles[signal] = handle
	}
}

// Wait for signals
func ServeHandleSignals() (err error) {
	var signals = make([]os.Signal, 0, len(handles))
	for sig, _ := range handles {
		signals = append(signals, sig)
	}

	signalChan := make(chan os.Signal, 8)
	signal.Notify(signalChan, signals...)

	for sig := range signalChan {
		err = handles[sig](sig)
		if err != nil {
			break
		}
	}

	signal.Stop(signalChan)

	if err == ErrStop {
		err = nil
	}

	return
}
