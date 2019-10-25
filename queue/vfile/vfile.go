package vfile

import (
	"os"
	"sync"

	"github.com/EricYT/go-examples/queue/ioqueue"
)

// virtual file interface based on a specifical
// userspace io queue.

type VFiler interface {
	RegisterPriorityClass(name string, shares int)
	UnregisterPriorityClass(name string)

	WriteAt(pc string, fh *os.File, off int64, buf []byte) (n int, err error)
	ReadAt(pc, fh *os.File, off int64, buf []byte) (n int, err error)
}

type vFile struct {
	sync.Mutex

	mp    ioqueue.Mountpoint
	queue *ioqueue.IOQueue
}

func NewVirtualFile(mp ioqueue.Mountpoint) *vFile {
	vf := &vFile{
		queue: ioqueue.NewIOQueue(mp),
	}
	return vf
}

func (f *vFile) Close() {
	f.Lock()
	defer f.Unlock()
	f.queue.Close()
}

func (f *vFile) RegisterPriorityClass(pc string, shares uint32) {
	f.queue.RegisterPriorityClass(pc, shares)
}

func (f *vFile) UnregisterPriorityClass(pc string) {
	f.queue.UnregisterPriorityClass(pc)
}

func (f *vFile) WriteAt(pc string, fh *os.File, off int64, buf []byte) (n int, err error) {
	//TODO: validate fh name and mount point

	writeFn := func() {
		n, err = fh.WriteAt(buf, off)
	}
	fut, errq := f.queue.QueueRequest(pc, len(buf), ioqueue.RequestTypeWrite, writeFn)
	if errq != nil {
		return n, errq
	}
	return n, fut.Done()
}

func (f *vFile) ReadAt(pc string, fh *os.File, off int64, buf []byte) (n int, err error) {
	//TODO: validate fh name and mount point

	readFn := func() {
		n, err = fh.ReadAt(buf, off)
	}
	fut, errq := f.queue.QueueRequest(pc, len(buf), ioqueue.RequestTypeRead, readFn)
	if errq != nil {
		return n, errq
	}
	return n, fut.Done()
}
