package multipipe

import (
	"errors"
	"io"
	"sync"
)

var ErrOffset = errors.New("wrong offset")
var ErrReaderClosed = errors.New("reader closed")

type MultiPipe struct {
	mu sync.RWMutex

	data []byte
	rs   []*reader
	err  error
}

func NewMultiPipe() *MultiPipe {
	return &MultiPipe{}
}

func (p *MultiPipe) Write(buf []byte) (n int, err error) {
	if len(buf) == 0 {
		return 0, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return n, p.err
	}
	p.data = append(p.data, buf...)
	p.wakeupReaders()
	return len(buf), nil
}

func (p *MultiPipe) readAt(off int, buf []byte) (n int, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if off > len(p.data) {
		return 0, ErrOffset
	}

	if len(p.data) == off {
		// no more data for reader
		return 0, p.err
	}

	// not enough
	nr := off + len(buf)
	if nr > len(p.data) {
		nr = len(p.data)
	}

	// enough data
	n = copy(buf, p.data[off:nr])
	if off+n == len(p.data) {
		return n, p.err
	}

	return n, nil
}

func (p *MultiPipe) Close() {
	p.CloseWithError(nil)
}

func (p *MultiPipe) CloseWithError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return
	}
	if err == nil {
		err = io.EOF
	}
	p.err = err
	p.wakeupReaders()
}

func (p *MultiPipe) wakeupReaders() {
	for _, r := range p.rs {
		r.wakeup()
	}
}

func (p *MultiPipe) NewReader() (io.Reader, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r := &reader{
		p:        p,
		closedCh: make(chan struct{}),
		signalCh: make(chan struct{}, 1),
	}
	p.rs = append(p.rs, r)
	return r, nil
}

// reader
type pipeReader interface {
	readAt(off int, buf []byte) (n int, err error)
}

type reader struct {
	mu  sync.Mutex
	p   pipeReader
	off int

	closedCh chan struct{}
	signalCh chan struct{} // with one buffer
}

func (r *reader) Read(buf []byte) (n int, err error) {
	select {
	case <-r.closedCh:
		return n, ErrReaderClosed
	default:
	}

	if len(buf) == 0 {
		return 0, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for {
		n, err = r.p.readAt(r.off, buf)
		if err != nil {
			if err != io.EOF {
				return n, err
			}
			break
		}
		if n > 0 {
			break
		}
		select {
		case <-r.signalCh:
		case <-r.closedCh:
			return n, ErrReaderClosed
		}
	}
	// shift offset of reader
	r.off += n
	return
}

func (r *reader) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	close(r.closedCh)
}

func (r *reader) wakeup() {
	select {
	case r.signalCh <- struct{}{}:
	default:
	}
}
