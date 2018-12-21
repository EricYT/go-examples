package file

import (
	"container/list"
	"os"
	"sync"

	"github.com/pkg/errors"
)

// NOTICE: the fd cache is not concurrency safe

type FDCache struct {
	mu sync.Mutex

	capacity int

	fds       map[string]*File
	fileIndex map[*list.Element]string
	lru       *list.List
}

func NewFDCache(c int) *FDCache {
	return &FDCache{
		capacity:  c,
		fds:       make(map[string]*File),
		fileIndex: make(map[*list.Element]string),
		lru:       list.New(),
	}
}

func (fdc *FDCache) ReadAt(path string, b []byte, off int64) (n int, err error) {
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

func (fdc *FDCache) WriteAt(path string, b []byte, off int64) (n int, err error) {
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

func (fdc *FDCache) Reset() {
	fdc.mu.Lock()
	defer fdc.mu.Unlock()
	for _, file := range fdc.fds {
		file.CloseWait()
	}
	fdc.fds = nil
	fdc.fileIndex = nil
	fdc.lru = nil
}

func (fdc *FDCache) ensureFile(path string) (*File, error) {
	if file, ok := fdc.fds[path]; ok {
		fdc.touchFile(file)
		return file, nil
	}
	// open file
	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "FDCache")
	}
	file := newFile(path, f)
	if len(fdc.fds) >= fdc.capacity {
		// eject one fd from cache
		fdc.ejectOneFile()
	}
	fdc.insertFile(file)
	return file, nil
}

func (fdc *FDCache) touchFile(f *File) {
	fdc.lru.Remove(f.ele)
	delete(fdc.fileIndex, f.ele)
	ele := fdc.lru.PushFront(f)
	f.setEle(ele)
	fdc.fileIndex[ele] = f.path
}

func (fdc *FDCache) insertFile(f *File) {
	ele := fdc.lru.PushFront(f)
	f.setEle(ele)
	fdc.fds[f.path] = f
	fdc.fileIndex[ele] = f.path
}

func (fdc *FDCache) ejectOneFile() {
	eject := fdc.lru.Back()
	fdc.lru.Remove(eject)
	path := fdc.fileIndex[eject]
	file := fdc.fds[path]
	file.CloseWait()
	delete(fdc.fds, path)
	delete(fdc.fileIndex, eject)
}

type File struct {
	path string
	wg   sync.WaitGroup
	fh   *os.File
	ele  *list.Element
}

func newFile(p string, fh *os.File) *File {
	return &File{
		path: p,
		fh:   fh,
	}
}

func (f *File) setEle(e *list.Element) {
	f.ele = e
}

func (f *File) add(delta int) {
	f.wg.Add(delta)
}

func (f *File) done() {
	f.wg.Done()
}

func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	return f.fh.ReadAt(b, off)
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	return f.fh.WriteAt(b, off)
}

func (f *File) Sync() error {
	return f.fh.Sync()
}

func (f *File) CloseWait() {
	go func() {
		f.wg.Wait()
		f.fh.Sync()
		f.fh.Close()
	}()
}
