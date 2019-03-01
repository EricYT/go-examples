package store

import (
	"fmt"
	"os"
	"testing"

	"git.jd.com/cloud-storage/newds-datanode/pkg/logutil"
	"github.com/stretchr/testify/assert"
)

func TestBlockFile(t *testing.T) {
	//d, err := ioutil.TempDir("./", "test_blob_")
	//if !assert.Nil(t, err) {
	//	return
	//}
	//defer os.RemoveAll(d)
	d := "./test_blob"
	os.Mkdir(d, 0755)
	logger := logutil.NewProduction()
	bs := NewBlobStore(logger, d)
	err := bs.Load()
	if !assert.Nil(t, err) {
		return
	}

	reqs := make([]*Request, 0)
	for i := 0; i < 100; i++ {
		blob := fmt.Sprintf("#value%d#", i)
		req := &Request{
			BlobId: int64(i),
			Blob:   []byte(blob),
		}
		reqs = append(reqs, req)
	}

	err = bs.Write(reqs)
	if !assert.Nil(t, err) {
		return
	}
	for _, req := range reqs {
		blob, err := bs.Read(req.Ptr)
		if !assert.Nil(t, err) {
			return
		}
		fmt.Printf("blob: %s\n", string(blob))
	}

	bs.Close()

	bs = NewBlobStore(logger, d)
	err = bs.Load()
	if !assert.Nil(t, err) {
		return
	}
	for _, req := range reqs {
		blob, err := bs.Read(req.Ptr)
		if !assert.Nil(t, err) {
			return
		}
		fmt.Printf("load blob: %s\n", string(blob))
	}

}
