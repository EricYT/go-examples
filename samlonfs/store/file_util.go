package store

import "os"

func OpenSyncedFile(path string, synced bool) (fd *os.File, err error) {
	flag := os.O_RDWR | os.O_CREATE | os.O_EXCL
	if synced {
		flag |= os.O_SYNC
	}
	fd, err = os.OpenFile(path, flag, 0666)
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func SyncDir(dirpath string) error {
	fd, err := os.Open(dirpath)
	if err != nil {
		return err
	}
	defer fd.Close()
	if err := fd.Sync(); err != nil {
		return err
	}
	return nil
}
