package wait_test

import (
	"sync"
	"testing"

	"github.com/EricYT/go-examples/utils/wait"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		w := wait.NewWait()
		c := w.Register(1)
		if assert.NotNil(t, c) {
			assert.True(t, w.IsRegister(1))
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				r := <-c
				assert.Equal(t, 1, r)
			}()
			w.Trigger(1, 1)
		}
	})

	t.Run("already_registerd", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.Equal(t, r, "dup id")
		}()

		w := wait.NewWait()
		c := w.Register(1)
		assert.NotNil(t, c)
		w.Register(1)
	})

	t.Run("not_registered", func(t *testing.T) {
		w := wait.NewWait()
		if assert.False(t, w.IsRegister(1)) {
			w.Register(1)
			assert.True(t, w.IsRegister(1))
		}
	})
}

func TestTrigger(t *testing.T) {
	t.Run("not_registered", func(t *testing.T) {
		w := wait.NewWait()
		if assert.False(t, w.Trigger(1, 1)) {
			w.Register(1)
			assert.True(t, w.Trigger(1, 1))
			assert.False(t, w.Trigger(1, 1))
		}
	})
}
