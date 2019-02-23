// +build darwin dragonfly freebsd linux netbsd openbsd plan9 solaris

package lockfile

import "syscall"

func tryLockFile(fd uintptr) (err error) {
	err = syscall.Flock(int(fd), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == syscall.EWOULDBLOCK {
		err = ErrLocked
	}
	return err
}

func lockFile(fd uintptr) (err error) {
	err = syscall.Flock(int(fd), syscall.LOCK_EX)
	if err == syscall.EWOULDBLOCK {
		err = ErrLocked
	}
	return err
}
