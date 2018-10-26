package ringbuf_test

import (
	"testing"

	"github.com/EricYT/go-examples/ringbuf"
	"github.com/stretchr/testify/assert"
)

func TestRinBuf(t *testing.T) {
	var r ringbuf.RingBuffer
	r = ringbuf.New(10)

	part1 := []ringbuf.Entry{
		ringbuf.Entry{1},
		ringbuf.Entry{2},
		ringbuf.Entry{3},
		ringbuf.Entry{4},
	}

	r.Append(part1)
	all := r.All()
	assert.Equal(t, part1, all)

	part2 := []ringbuf.Entry{
		ringbuf.Entry{5},
		ringbuf.Entry{6},
		ringbuf.Entry{7},
		ringbuf.Entry{8},
	}
	r.Append(part2)
	all = r.All()
	assert.Equal(t, append(part1, part2...), all)

	part3 := []ringbuf.Entry{
		ringbuf.Entry{9},
		ringbuf.Entry{10},
		ringbuf.Entry{11},
		ringbuf.Entry{12},
	}
	r.Append(part3)
	all = r.All()
	should := append(part3[2:], part1[2:]...)
	should = append(should, part2...)
	should = append(should, part3[:2]...)
	assert.Equal(t, should, all)

}
