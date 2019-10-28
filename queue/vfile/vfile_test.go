package vfile

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/EricYT/go-examples/queue/ioqueue"
	"github.com/stretchr/testify/assert"
)

func TestVFile_validateMountpoint(t *testing.T) {
	vf := NewVirtualFile(ioqueue.Mountpoint{
		MP:             "/tmp",
		ReadBytesRate:  1,
		WriteBytesRate: 1,
		WriteReqRate:   1,
		ReadReqRate:    1,
		NumIOQueues:    1,
	})

	err := vf.validateMountpoint("/tmp1/c")
	if assert.NotNil(t, err) {
		return
	}
	if _, ok := err.(*InvalidFilePath); !assert.True(t, ok) {
		return
	}

	if !assert.Nil(t, vf.validateMountpoint("/tmp/xxx/xxxx.1")) {
		return
	}
}

func TestVFile_Mountpoint_not_directory(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer func() {
			pathErr := recover()
			_, ok := pathErr.(*os.PathError)
			assert.True(t, ok)
		}()
		NewVirtualFile(ioqueue.Mountpoint{
			MP: "/dsdfdlsk",
		})
	}()

	go func() {
		defer wg.Done()
		defer func() {
			pathErr := recover()
			assert.NotNil(t, pathErr)
			err := pathErr.(string)
			assert.True(t, strings.Contains(err, "not a directory"))
		}()
		tmpfile, err := ioutil.TempFile("", "fakemountpoint-")
		if !assert.Nil(t, err) {
			return
		}
		defer os.Remove(tmpfile.Name())
		defer tmpfile.Close()

		NewVirtualFile(ioqueue.Mountpoint{
			MP: tmpfile.Name(),
		})

	}()

	wg.Wait()
}

func TestVFile_Read_Write_At(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "vfiledir-")
	if !assert.Nil(t, err) {
		return
	}
	defer os.RemoveAll(tmpdir)

	vf := NewVirtualFile(ioqueue.Mountpoint{
		MP:             tmpdir,
		ReadBytesRate:  1,
		WriteBytesRate: 1,
		WriteReqRate:   1,
		ReadReqRate:    1,
		NumIOQueues:    1,
	})
	defer vf.Close()

	tmpfile, err := ioutil.TempFile(tmpdir, "vfile-")
	if !assert.Nil(t, err) {
		return
	}
	defer tmpfile.Close()

	n, err := vf.WriteAt("class1", tmpfile, 0, []byte("hello,world"))
	if !assert.Equal(t, -1, n) || !assert.Equal(t, ioqueue.ErrFairQueuePriorityClassNotFound, err) {
		return
	}

	vf.RegisterPriorityClass("class1", 1)

	data := []byte("hello,world")
	n, err = vf.WriteAt("class1", tmpfile, 0, data)
	if !assert.Equal(t, len(data), n) || !assert.Nil(t, err) {
		return
	}

	rdata := make([]byte, len(data))
	n, err = vf.ReadAt("class1", tmpfile, 0, rdata)
	if !assert.Equal(t, len(data), n) || !assert.Equal(t, data, rdata) {
		return
	}
}
