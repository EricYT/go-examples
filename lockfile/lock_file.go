package lockfile

import (
	"errors"
	"os"
)

// FIXME: not for NFS

var ErrLocked error = errors.New("lock file: file locked")

type FileLock struct {
	*os.File
}

// try to lock file no blocking
func TryLockFile(name string, flag int, perm os.FileMode) (*FileLock, error) {
	fd, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	if err := tryLockFile(fd.Fd()); err != nil {
		fd.Close()
		return nil, err
	}
	return &FileLock{fd}, nil
}

// lock file. maybe blocking until got it
func LockFile(name string, flag int, perm os.FileMode) (*FileLock, error) {
	fd, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	if err := lockFile(fd.Fd()); err != nil {
		fd.Close()
		return nil, err
	}
	return &FileLock{fd}, nil
}
