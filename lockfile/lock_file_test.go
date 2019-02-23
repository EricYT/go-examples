package lockfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

var (
	filename string      = os.TempDir() + "/test.lock"
	perm     os.FileMode = 0644
)

func TestTryLockFile(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "test-lock")
	if err != nil {
		t.Fatalf("try lock file create temp file failed. %v", err)
	}
	name := file.Name()
	file.Close()
	defer os.Remove(name)

	// try lock it
	lock, err := TryLockFile(name, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("try lock failed on file %s. %v", name, err)
	}
	if _, err := TryLockFile(name, os.O_RDWR, 0644); err != ErrLocked {
		t.Fatalf("shouldn't lock file %s", name)
	}

	// unlock file
	if err := lock.Close(); err != nil {
		t.Fatalf("unlock file %s failed. %v", name, err)
	}

	// lock it again
	lock, err = TryLockFile(name, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("try lock again failed on file %s. %v", name, err)
	}
	if err := lock.Close(); err != nil {
		t.Fatalf("unlock file %s failed. %v", name, err)
	}
}

func TestLockFile(t *testing.T) {
	fileLockName := path.Join(os.TempDir(), fmt.Sprintf("test-lock-%d", time.Now().UnixNano()))
	defer os.Remove(fileLockName)

	// lock it
	l, err := LockFile(fileLockName, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Unable to lock file %s. %v", fileLockName, err)
	}

	waitLockFileC := make(chan struct{}, 1)
	go func() {
		l1, err := LockFile(fileLockName, os.O_RDWR, 0644)
		if err != nil {
			t.Fatalf("Unable to lock file %s on other process. %v", fileLockName, err)
		}
		l1.Close()
		waitLockFileC <- struct{}{}
	}()

	if err := l.Close(); err != nil {
		t.Fatalf("Close file %s lock failed. %v", fileLockName, err)
	}

	select {
	case <-waitLockFileC:
	case <-time.After(1 * time.Second):
		t.Fatalf("Other process not lock file %s again", fileLockName)
	}
}
