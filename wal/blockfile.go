package wal

import (
	"os"
	"sync"

	"github.com/pkg/errors"
)

var (
	ErrReadError  error = errors.New("read block file error")
	ErrReadonly   error = errors.New("block file readonly")
	ErrShortWrite error = errors.New("block file write short")
)

type BlockFile struct {
	sync.RWMutex

	fn   string
	id   int32
	size int64
	w    *os.File
	fp   FDPool
}

func NewBlockFile(fn string, id int32, readonly bool, fp FDPool) (*BlockFile, error) {
	var (
		err error
		w   *os.File
	)

	if !readonly {
		w, err = os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0640)
		if err != nil {
			return nil, errors.Wrap(err, "unable to open file")
		}
	}
	stat, err := os.Stat(fn)
	if err != nil {
		w.Close()
		return nil, errors.Wrap(err, "unable to stat file")
	}
	return &BlockFile{
		fn:   fn,
		id:   id,
		size: stat.Size(),
		w:    w,
		fp:   fp,
	}, nil
}

func (bf *BlockFile) FileId() int32 {
	return bf.id
}

func (bf *BlockFile) Close() error {
	if bf.w == nil {
		return nil
	}
	if err := bf.Sync(); err != nil {
		return err
	}
	return bf.w.Close()
}

func (bf *BlockFile) Sync() error {
	if bf.w == nil {
		return nil
	}
	return bf.w.Sync()
}

func (bf *BlockFile) Size() int64 {
	if bf.w == nil {
		return bf.size
	}
	bf.RLock()
	defer bf.RUnlock()
	return bf.size
}

func (bf *BlockFile) ReadAt(off int64, size int) ([]byte, error) {
	val := make([]byte, size)

	readAtFn := func(fd *os.File) error {
		n, err := fd.ReadAt(val, off)
		if err != nil {
			return err
		}
		if n != size {
			return ErrReadError
		}
		return nil
	}

	if bf.w != nil {
		if err := readAtFn(bf.w); err != nil {
			return nil, err
		}
		return val, nil
	}

	// file descriptor pool
	if err := bf.fp.Do(bf.fn, readAtFn); err != nil {
		return nil, err
	}
	return val, nil
}

func (bf *BlockFile) Write(val []byte) (int64, error) {
	if bf.w == nil {
		return 0, ErrReadonly
	}

	bf.Lock()
	defer bf.Unlock()

	n, err := bf.w.Write(val)
	if err != nil {
		return 0, err
	}
	if n != len(val) {
		return 0, ErrShortWrite
	}
	off := bf.size
	bf.size += int64(len(val))
	return off, nil
}
