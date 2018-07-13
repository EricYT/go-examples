package lockfile

import (
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	filename string      = os.TempDir() + "/test.lock"
	perm     os.FileMode = 0644
)

func TestNewLockFile(t *testing.T) {
	lock := NewLockFile(os.NewFile(1001, "not exist"))
	err := lock.Lock()
	assert.Equal(t, err, syscall.EBADF, "A invalidate file descriptor")
	err = lock.Unlock()
	assert.Equal(t, err, syscall.EBADF, "A invalidate file descriptor")
}

func TestLockFile(t *testing.T) {
	lock, err := OpenLockFile(filename, perm)
	assert.Nil(t, err, "Open lock file should not error")
	assert.Nil(t, lock.Lock(), "Lock file should not error")
	defer lock.Remove()

	lock1, err := OpenLockFile(filename, perm)
	assert.Nil(t, err, "Open lock file again should not error")
	assert.Equal(t, lock1.Lock(), ErrWouldBlock, "Lock file would block")
}

func TestLockFileUnlock(t *testing.T) {
	lock, err := OpenLockFile(filename, perm)
	assert.Nil(t, err, "Open lock file should not error")
	assert.Nil(t, lock.Lock(), "Lock file should not error")
	assert.Nil(t, lock.Unlock(), "Unlock should not error")
	defer lock.Remove()

	lock1, err := OpenLockFile(filename, perm)
	assert.Nil(t, err, "Open lock file again should not error")
	assert.Nil(t, lock1.Lock(), "Lock file would success")
	defer lock1.Remove()
}
