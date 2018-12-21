package file

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFDCache(t *testing.T) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name := tmpfile.Name()
	defer os.Remove(name)

	fdc := NewFDCache(3)
	defer fdc.Reset()

	val := []byte("hello,world")
	n, err := fdc.WriteAt(name, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	err = fdc.Sync(name)
	assert.Nil(t, err)

	buf := make([]byte, len(val))
	n1, err := fdc.ReadAt(name, buf, 0)
	assert.Nil(t, err)
	assert.Equal(t, n, n1)
	assert.Equal(t, val, buf)
}

func TestFDCacheFileNotExist(t *testing.T) {
	fdc := NewFDCache(3)
	defer fdc.Reset()

	_, err := fdc.WriteAt("./jldkfjdlfj", nil, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "no such file or directory"))

	_, err = fdc.ReadAt("./jldkfjdlfj", nil, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "no such file or directory"))
}

func TestFDCacheSync(t *testing.T) {
	fdc := NewFDCache(1)
	defer fdc.Reset()

	err := fdc.Sync("./jdklfdkjfdl")
	assert.Nil(t, err)
}

func TestFDCacheTouch(t *testing.T) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name1 := tmpfile.Name()
	defer os.Remove(name1)

	fdc := NewFDCache(3)
	defer fdc.Reset()

	val := []byte("hello,world")
	n, err := fdc.WriteAt(name1, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	f1 := fdc.lru.Front()
	assert.Equal(t, name1, fdc.fileIndex[f1])

	tmpfile2, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name2 := tmpfile2.Name()
	defer os.Remove(name2)

	n, err = fdc.WriteAt(name2, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	f2 := fdc.lru.Front()
	assert.Equal(t, name2, fdc.fileIndex[f2])

	buf := make([]byte, len(val))
	n1, err := fdc.ReadAt(name1, buf, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n1)
	assert.Equal(t, val, buf)

	f3 := fdc.lru.Front()
	assert.Equal(t, name1, fdc.fileIndex[f3])
}

func TestFDCacheEject(t *testing.T) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name1 := tmpfile.Name()
	defer os.Remove(name1)

	fdc := NewFDCache(2)
	defer fdc.Reset()

	val := []byte("hello,world")
	n, err := fdc.WriteAt(name1, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	tmpfile2, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name2 := tmpfile2.Name()
	defer os.Remove(name2)

	n, err = fdc.WriteAt(name2, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	tmpfile3, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name3 := tmpfile3.Name()
	defer os.Remove(name3)

	n, err = fdc.WriteAt(name3, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	assert.Equal(t, 2, len(fdc.fds))

	f1 := fdc.lru.Front()
	assert.Equal(t, name3, fdc.fileIndex[f1])
	f2 := fdc.lru.Back()
	assert.Equal(t, name2, fdc.fileIndex[f2])
}

func TestFDCacheConcurrencyControl(t *testing.T) {
	fdc := NewFDCache(1)
	defer fdc.Reset()
}
