package lockfile

import (
	"errors"
	"os"
)

var ErrWouldBlock error = errors.New("lock file: would block")

// lock file
type LockFile struct {
	*os.File
}

func NewLockFile(file *os.File) *LockFile {
	return &LockFile{file}
}

func OpenLockFile(filename string, perm os.FileMode) (lock *LockFile, err error) {
	var file *os.File
	if file, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR, perm); err != nil {
		return nil, err
	}
	return &LockFile{file}, nil
}

func (file *LockFile) Lock() (err error) {
	return lockFile(file.Fd())
}

func (file *LockFile) Unlock() (err error) {
	return unlockFile(file.Fd())
}

func (file *LockFile) Remove() (err error) {
	defer file.Close()

	if err = file.Unlock(); err != nil {
		return err
	}
	return os.Remove(file.Name())
}
