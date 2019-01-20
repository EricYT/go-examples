package samlonfs

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func Mmap(fd *os.File, writeable bool, size int64) ([]byte, error) {
	mtype := unix.PROT_READ
	if writeable {
		mtype |= unix.PROT_WRITE
	}
	return unix.Mmap(int(fd.Fd()), 0, int(size), mtype, unix.MAP_SHARED)
}

func Munmap(b []byte) error {
	return unix.Munmap(b)
}

func Madvise(b []byte, readhead bool) error {
	flags := unix.MADV_NORMAL
	if !readhead {
		flags |= unix.MADV_RANDOM
	}
	return madvise(b, flags)
}

func madvise(b []byte, advise int) (err error) {
	_, _, e1 := syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&b[0])),
		uintptr(len(b)), uintptr(advise))
	if e1 != 0 {
		err = e1
	}
	return
}
