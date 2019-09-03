package multipipe

import (
	"errors"
	"io"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestEmpty(t *testing.T) {
	p := &MultiPipe{}
	n, err := p.readAt(0, make([]byte, 0))
	if !assert.Equal(t, 0, n) || !assert.Nil(t, err) {
		return
	}

	n, err = p.readAt(1, make([]byte, 1))
	if !assert.Equal(t, 0, n) || !assert.Equal(t, ErrOffset, err) {
		return
	}

	n, err = p.Write(nil)
	if !assert.Equal(t, 0, n) || !assert.Nil(t, err) {
		return
	}
}

func TestPipeClosed(t *testing.T) {
	p := &MultiPipe{}
	p.Close()
	n, err := p.readAt(0, make([]byte, 0))
	if !assert.Equal(t, 0, n) || !assert.Equal(t, err, io.EOF) {
		return
	}

	// other error discard
	erragain := errors.New("again")
	p.CloseWithError(erragain)
	n, err = p.readAt(0, make([]byte, 0))
	if !assert.Equal(t, 0, n) || !assert.Equal(t, err, io.EOF) {
		return
	}

	n, err = p.Write(make([]byte, 3))
	if !assert.Equal(t, 0, n) || !assert.Equal(t, err, io.EOF) {
		return
	}
}

func TestNoMoreData(t *testing.T) {
	p := &MultiPipe{}

	buf := make([]byte, 5)
	n, err := p.readAt(0, buf)
	if !assert.Equal(t, 0, n) || !assert.Nil(t, err) {
		return
	}

	data := make([]byte, 3)
	rand.Read(data)
	nr, err := p.Write(data)
	if !assert.Equal(t, 3, nr) || !assert.Nil(t, err) {
		return
	}

	n, err = p.readAt(0, buf)
	if !assert.Equal(t, 3, n) || !assert.Nil(t, err) || !assert.Equal(t, data, buf[:n]) {
		return
	}

	n, err = p.readAt(1, buf)
	if !assert.Equal(t, 2, n) || !assert.Nil(t, err) || !assert.Equal(t, data[1:], buf[:1+n-1]) {
		return
	}

	small := make([]byte, 1)
	n, err = p.readAt(1, small)
	if !assert.Equal(t, 1, n) || !assert.Nil(t, err) || !assert.Equal(t, data[1:2], buf[:1+n-1]) {
		return
	}

	// close
	p.Close()
	n, err = p.readAt(1, buf)
	if !assert.Equal(t, 2, n) || !assert.Equal(t, io.EOF, err) || !assert.Equal(t, data[1:], buf[:1+n-1]) {
		return
	}
}

func TestClosedReader(t *testing.T) {
	r := &reader{closedCh: make(chan struct{})}
	r.Close()

	n, err := r.Read(make([]byte, 1))
	if !assert.Equal(t, 0, n) || !assert.Equal(t, err, ErrReaderClosed) {
		return
	}
}

func TestEmptyRead(t *testing.T) {
	r := &reader{closedCh: make(chan struct{})}
	n, err := r.Read(nil)
	if !assert.Equal(t, 0, n) || !assert.Nil(t, err) {
		return
	}
}

var (
	stubReadAt = func(off int, buf []byte) (n int, err error) {
		return 0, nil
	}
)

type fakePipe struct {
}

func (f *fakePipe) readAt(off int, buf []byte) (n int, err error) {
	return stubReadAt(off, buf)
}

func TestReadFailed(t *testing.T) {
	defer func(old func(int, []byte) (int, error)) { stubReadAt = old }(stubReadAt)

	fperr := errors.New("fake error")
	stubReadAt = func(off int, buf []byte) (n int, err error) {
		return n, fperr
	}
	r := &reader{p: &fakePipe{}}
	_, err := r.Read(make([]byte, 1))
	if !assert.Equal(t, fperr, err) {
		return
	}

	stubReadAt = func(off int, buf []byte) (n int, err error) {
		return 3, io.EOF
	}

	n, err := r.Read(make([]byte, 5))
	if !assert.Equal(t, 3, n) || !assert.Equal(t, io.EOF, err) || !assert.Equal(t, 3, r.off) {
		return
	}
}

func TestReadBlock(t *testing.T) {
	defer func(old func(int, []byte) (int, error)) { stubReadAt = old }(stubReadAt)

	stubReadAt = func(off int, buf []byte) (n int, err error) {
		return 0, nil
	}
	r := &reader{p: &fakePipe{}, signalCh: make(chan struct{}), closedCh: make(chan struct{})}

	var wg sync.WaitGroup
	wg.Add(1)

	data := make([]byte, 5)
	rand.Read(data)

	go func() {
		defer wg.Done()
		buf := make([]byte, 5)
		n, err := r.Read(buf)
		if !assert.Equal(t, 5, n) || !assert.Equal(t, io.EOF, err) || !assert.Equal(t, data, buf) {
			return
		}
	}()

	time.Sleep(time.Millisecond * 1)

	stubReadAt = func(off int, buf []byte) (n int, err error) {
		nr := copy(buf, data)
		return nr, io.EOF
	}
	r.wakeup()

	wg.Wait()
}

func TestReadBlockPart(t *testing.T) {
	defer func(old func(int, []byte) (int, error)) { stubReadAt = old }(stubReadAt)

	stubReadAt = func(off int, buf []byte) (n int, err error) {
		return 0, nil
	}
	r := &reader{p: &fakePipe{}, signalCh: make(chan struct{}, 1), closedCh: make(chan struct{})}

	var wg sync.WaitGroup
	wg.Add(1)

	data1 := make([]byte, 5)
	rand.Read(data1)
	data2 := make([]byte, 10)
	rand.Read(data2)

	next := make(chan struct{})

	go func() {
		defer wg.Done()

		cases := []struct {
			n    int
			data []byte
			err  error
		}{
			{n: 5, data: data1},
			{n: 10, data: data2},
			{data: []byte{}, err: io.EOF},
		}

		buf := make([]byte, 10)
		for _, c := range cases {
			n, err := r.Read(buf)
			if !assert.Equal(t, c.n, n) ||
				!assert.Equal(t, c.data, buf[:n]) ||
				!assert.Equal(t, c.err, err) {
				return
			}
			next <- struct{}{}
		}
	}()
	time.Sleep(time.Millisecond * 1)

	stubReadAt = func(off int, buf []byte) (n int, err error) {
		if off == 0 {
			n = copy(buf, data1)
		} else if off == 5 {
			n = copy(buf, data2)
		} else {
			return 0, io.EOF
		}
		return n, nil
	}
	r.wakeup()
	<-next
	r.wakeup()
	<-next
	r.wakeup()
	<-next

	wg.Wait()
}

func TestMultiPipeSingle(t *testing.T) {
	p := NewMultiPipe()

	var wg sync.WaitGroup
	data := make([]byte, 3)
	rand.Read(data)

	r, err := p.NewReader()
	if !assert.Nil(t, err) {
		return
	}

	var count int
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 3)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				if !assert.Equal(t, data, buf) || !assert.Equal(t, 3, n) {
					return
				}
				count++
			}
			if err != nil {
				if !assert.Equal(t, err, io.EOF) {
					return
				}
				return
			}
		}
	}()

	p.Write(data)
	p.Write(data)
	p.Write(data)
	p.Write(data)
	p.Close()

	wg.Wait()

	if !assert.Equal(t, 4, count) {
		return
	}
}

func TestMultiPipeMany(t *testing.T) {
	p := NewMultiPipe()

	var wg sync.WaitGroup
	data := make([]byte, 3)
	rand.Read(data)

	readerFn := func() {
		r, err := p.NewReader()
		if !assert.Nil(t, err) {
			return
		}

		var count int
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 3)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					if !assert.Equal(t, data, buf) || !assert.Equal(t, 3, n) {
						return
					}
					count++
				}
				if err != nil {
					if !assert.Equal(t, err, io.EOF) {
						return
					}
					break
				}
			}
			if !assert.Equal(t, 4, count) {
				return
			}
		}()
	}

	for i := 0; i < 5; i++ {
		readerFn()
	}

	p.Write(data)
	p.Write(data)
	p.Write(data)
	p.Write(data)
	p.Close()

	wg.Wait()
}

func TestMultiPipeManyFailed(t *testing.T) {
	p := NewMultiPipe()

	var wg sync.WaitGroup
	data := make([]byte, 3)
	rand.Read(data)

	crash := errors.New("crash")

	readerFn := func() {
		r, err := p.NewReader()
		if !assert.Nil(t, err) {
			return
		}

		var count int
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 3)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					if !assert.Equal(t, data, buf) || !assert.Equal(t, 3, n) {
						return
					}
					count++
				}
				if err != nil {
					if !assert.Equal(t, crash, err) {
					}
					break
				}
			}
			if !assert.Equal(t, 2, count) {
				return
			}
		}()
	}

	for i := 0; i < 5; i++ {
		readerFn()
	}

	p.Write(data)
	p.Write(data)
	p.CloseWithError(crash)

	wg.Wait()
}
