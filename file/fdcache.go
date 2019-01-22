package file

import (
	"container/list"
	"context"
	"os"
	"sync"

	"github.com/pkg/errors"
)

// NOTICE: the fd cache is not concurrency safe

var _openFileFn = os.OpenFile
var _newFileFn = newFile

type filer interface {
	path() string
	add(delta int)
	done()
	getEle() *list.Element
	setEle(e *list.Element)

	ReadAt(b []byte, off int64) (n int, err error)
	WriteAt(b []byte, off int64) (n int, err error)
	Sync() error
	CloseWait(doneFn func())
}

type FDCache struct {
	mu sync.Mutex

	capacity int

	fds map[string]filer
	lru *list.List

	ghost map[string]chan struct{} // The files are ejected but not sync or close completely.
}

func NewFDCache(c int) *FDCache {
	return &FDCache{
		capacity: c,
		fds:      make(map[string]filer),
		lru:      list.New(),

		ghost: make(map[string]chan struct{}),
	}
}

func (fdc *FDCache) ReadAt(ctx context.Context, path string, b []byte, off int64) (n int, err error) {
	if err := fdc.ghostFile(ctx, path); err != nil {
		return 0, err
	}

	fdc.mu.Lock()
	file, err := fdc.ensureFile(path)
	if err != nil {
		fdc.mu.Unlock()
		return 0, err
	}
	file.add(1)
	defer file.done()
	fdc.mu.Unlock()
	return file.ReadAt(b, off)
}

func (fdc *FDCache) WriteAt(ctx context.Context, path string, b []byte, off int64) (n int, err error) {
	if err := fdc.ghostFile(ctx, path); err != nil {
		return 0, err
	}

	fdc.mu.Lock()
	file, err := fdc.ensureFile(path)
	if err != nil {
		fdc.mu.Unlock()
		return 0, err
	}
	file.add(1)
	defer file.done()
	fdc.mu.Unlock()
	return file.WriteAt(b, off)
}

func (fdc *FDCache) Sync(path string) error {
	fdc.mu.Lock()
	file, ok := fdc.fds[path]
	if !ok {
		fdc.mu.Unlock()
		return nil
	}
	file.add(1)
	defer file.done()
	fdc.mu.Unlock()
	return file.Sync()
}

func (fdc *FDCache) Close(path string) {
	fdc.mu.Lock()
	defer fdc.mu.Unlock()
	f, ok := fdc.fds[path]
	if !ok {
		return
	}
	fdc.removeFile(f)
}

func (fdc *FDCache) Reset() {
	fdc.mu.Lock()
	defer fdc.mu.Unlock()
	for _, file := range fdc.fds {
		file.CloseWait(nil)
	}
	for _, waitc := range fdc.ghost {
		close(waitc)
	}
	fdc.fds = make(map[string]filer)
	fdc.lru = list.New()
	fdc.ghost = make(map[string]chan struct{})
}

func (fdc *FDCache) ensureFile(path string) (filer, error) {
	if f, ok := fdc.fds[path]; ok {
		fdc.touchFile(f)
		return f, nil
	}
	// open file
	fh, err := _openFileFn(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "FDCache")
	}
	f := _newFileFn(path, fh)
	if len(fdc.fds) >= fdc.capacity {
		// eject one fd from cache
		fdc.ejectOneFile()
	}
	fdc.insertFile(f)
	return f, nil
}

func (fdc *FDCache) touchFile(f filer) {
	fdc.lru.MoveToFront(f.getEle())
}

func (fdc *FDCache) insertFile(f filer) {
	ele := fdc.lru.PushFront(f)
	f.setEle(ele)
	fdc.fds[f.path()] = f
}

func (fdc *FDCache) removeFile(f filer) {
	fdc.lru.Remove(f.getEle())
	delete(fdc.fds, f.path())

	waitc := make(chan struct{})
	fdc.ghost[f.path()] = waitc
	f.CloseWait(func() { fdc.removeGhostFile(f.path()) })
}

func (fdc *FDCache) ejectOneFile() {
	eject := fdc.lru.Back()
	if eject != nil {
		path := eject.Value.(filer).path()
		f := fdc.fds[path]
		fdc.removeFile(f)
	}
}

func (fdc *FDCache) removeGhostFile(path string) {
	fdc.mu.Lock()
	defer fdc.mu.Unlock()
	waitc, ok := fdc.ghost[path]
	if !ok {
		return
	}
	delete(fdc.ghost, path)
	close(waitc)
}

func (fdc *FDCache) ghostFile(ctx context.Context, path string) error {
	fdc.mu.Lock()
	waitc, ok := fdc.ghost[path]
	if !ok {
		fdc.mu.Unlock()
		return nil
	}
	fdc.mu.Unlock()

	select {
	case <-waitc:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

type file struct {
	p   string
	wg  sync.WaitGroup
	fh  *os.File
	ele *list.Element
}

func newFile(p string, fh *os.File) filer {
	return &file{
		p:  p,
		fh: fh,
	}
}

func (f *file) path() string {
	return f.p
}

func (f *file) getEle() *list.Element {
	return f.ele
}

func (f *file) setEle(e *list.Element) {
	f.ele = e
}

func (f *file) add(delta int) {
	f.wg.Add(delta)
}

func (f *file) done() {
	f.wg.Done()
}

func (f *file) ReadAt(b []byte, off int64) (n int, err error) {
	return f.fh.ReadAt(b, off)
}

func (f *file) WriteAt(b []byte, off int64) (n int, err error) {
	return f.fh.WriteAt(b, off)
}

func (f *file) Sync() error {
	return f.fh.Sync()
}

func (f *file) CloseWait(doneFn func()) {
	go func() {
		f.wg.Wait()
		f.fh.Sync()
		f.fh.Close()
		if doneFn != nil {
			doneFn()
		}
	}()
}
