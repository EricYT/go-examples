package vfile

import (
	"fmt"
	"os"
	"strings"
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
	// validate mountpoint
	info, err := os.Stat(mp.MP)
	if err != nil {
		panic(err)
	}
	if !info.IsDir() {
		panic(fmt.Sprintf("mountpoint %s is not a directory.", mp.MP))
	}

	// add path separator in mountpoint
	mountpoint := []byte(mp.MP)
	if !os.IsPathSeparator(mountpoint[len(mountpoint)-1]) {
		mp.MP += string(os.PathSeparator)
	}
	vf := &vFile{
		mp:    mp,
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
	if err := f.validateMountpoint(fh.Name()); err != nil {
		return -1, err
	}

	writeFn := func() {
		n, err = fh.WriteAt(buf, off)
	}
	fut, errq := f.queue.QueueRequest(pc, len(buf), ioqueue.RequestTypeWrite, writeFn)
	if errq != nil {
		return -1, errq
	}
	if ferr := fut.Done(); ferr != nil {
		return -1, ferr
	}
	return n, err
}

func (f *vFile) ReadAt(pc string, fh *os.File, off int64, buf []byte) (n int, err error) {
	if err := f.validateMountpoint(fh.Name()); err != nil {
		return -1, err
	}

	readFn := func() {
		n, err = fh.ReadAt(buf, off)
	}
	fut, errq := f.queue.QueueRequest(pc, len(buf), ioqueue.RequestTypeRead, readFn)
	if errq != nil {
		return -1, errq
	}
	if ferr := fut.Done(); ferr != nil {
		return -1, ferr
	}
	return n, err
}

func (f *vFile) validateMountpoint(name string) error {
	if strings.HasPrefix(name, f.mp.MP) {
		return nil
	}
	return &InvalidFilePath{Mountpoint: f.mp.MP, Path: name}
}
