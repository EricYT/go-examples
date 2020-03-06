package gate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGate_EnterLeave(t *testing.T) {
	gate := New()
	err := gate.Enter()
	assert.Nil(t, err)
	assert.Equal(t, int64(1), gate.GetCount())
	gate.Leave()
	assert.Equal(t, int64(0), gate.GetCount())
	assert.False(t, gate.IsClosed())
	gate.Close()
	assert.True(t, gate.IsClosed())
}

func TestGate_Closed(t *testing.T) {
	gate := New()
	gate.Close()
	err := gate.Enter()
	assert.Equal(t, ErrGateClosed, err)
	gate.Close()
}

func TestGate_With(t *testing.T) {
	gate := New()
	err := gate.With(func() {
		time.Sleep(time.Millisecond * 3)
	})
	assert.Nil(t, err)
	gate.Close()
}

func TestGate_WithClosed(t *testing.T) {
	gate := New()
	gate.Close()
	err := gate.With(nil)
	assert.Equal(t, ErrGateClosed, err)
}
