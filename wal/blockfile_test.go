package wal

import (
	"crypto/rand"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func ensure(t *testing.T, r bool) {
	if r {
		return
	}
	t.Fatalf("ensure crash")
}

func TestBlockFileReadonly(t *testing.T) {
	tmpdir, err := ioutil.TempDir(os.TempDir(), "test_block_file_")
	ensure(t, assert.Nil(t, err))
	defer os.RemoveAll(tmpdir)

	noexist := filepath.Join(tmpdir, "nop")
	_, err = NewBlockFile(noexist, 0, true, nil)
	ensure(t, assert.True(t, os.IsNotExist(errors.Cause(err))))
}

func TestBlockFileWrite(t *testing.T) {
	tmpdir, err := ioutil.TempDir(os.TempDir(), "test_block_file_")
	ensure(t, assert.Nil(t, err))
	defer os.RemoveAll(tmpdir)

	f := filepath.Join(tmpdir, "00001.data")
	bf, err := NewBlockFile(f, 0, false, &createFDPool{})
	ensure(t, assert.Nil(t, err))
	defer bf.Close()

	data := make([]byte, 512)
	rand.Read(data)

	off, err := bf.Write(data)
	ensure(t, assert.Nil(t, err))
	ensure(t, assert.Equal(t, int64(0), off))
	ensure(t, assert.Equal(t, int64(len(data)), bf.Size()))

	off1, err := bf.Write(data)
	ensure(t, assert.Nil(t, err))
	ensure(t, assert.Equal(t, int64(len(data)), off1))
	ensure(t, assert.Equal(t, int64(len(data)*2), bf.Size()))

	v1, err := bf.ReadAt(0, len(data))
	ensure(t, assert.Nil(t, err))
	ensure(t, assert.Equal(t, data, v1))

	v2, err := bf.ReadAt(int64(len(data)), len(data))
	ensure(t, assert.Nil(t, err))
	ensure(t, assert.Equal(t, data, v2))
}

func TestBlcokFileClosed(t *testing.T) {
	tmpdir, err := ioutil.TempDir(os.TempDir(), "test_block_file_")
	ensure(t, assert.Nil(t, err))
	defer os.RemoveAll(tmpdir)

	f := filepath.Join(tmpdir, "00001.data")
	bf, err := NewBlockFile(f, 0, false, &createFDPool{})
	ensure(t, assert.Nil(t, err))
	err = bf.Close()
	ensure(t, assert.Nil(t, err))

	_, err = bf.Write(nil)
	ensure(t, assert.Contains(t, err.Error(), os.ErrClosed.Error()))
}
