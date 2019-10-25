package vfile

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/EricYT/go-examples/queue/ioqueue"
	"github.com/stretchr/testify/assert"
)

func TestVFile_New(t *testing.T) {
	vf := NewVirtualFile(ioqueue.Mountpoint{
		MP:              "/tmp",
		ReadBytesRate:   1,
		WriteBytesRatee: 1,
		WriteReqRate:    1,
		ReadReqRate:     1,
		NumIOQueues:     1,
	})

	tmpdir, err := ioutil.TempDir("", "vfile-")
	if !assert.Nil(t, err) {
		return
	}
	//defer os.RemoveAll(tmpdir)

	tcs := struct {
	}{}

	var files []*os.File
	for i := 0; i < 10; i++ {
	}

}
