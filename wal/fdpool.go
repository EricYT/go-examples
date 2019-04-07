package wal

import (
	"os"

	"github.com/pkg/errors"
)

type FDPool interface {
	Do(p string, fn func(*os.File) error) error
}

type createFDPool struct {
}

func (c *createFDPool) Do(f string, fn func(*os.File) error) error {
	fd, err := os.OpenFile(f, os.O_RDONLY, 0)
	if err != nil {
		return errors.Wrap(err, "unable to open file")
	}
	defer fd.Close()
	if err := fn(fd); err != nil {
		return err
	}
	return nil
}
