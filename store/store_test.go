package store

import (
	"fmt"
	"testing"

	"git.jd.com/cloud-storage/newds-datanode/pkg/logutil"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	logger := logutil.NewProduction()
	ns := NewStore(logger, "./store_root/")
	err := ns.Load()
	if !assert.Nil(t, err) {
		return
	}

	empty := []byte{}
	reqs := make([]*Request, 0)
	for i := 0; i < 10; i++ {
		blob := fmt.Sprintf("#blob%d#", i)
		req, err := ns.Put(logger, []byte(blob), empty, empty)
		if !assert.Nil(t, err) {
			return
		}
		reqs = append(reqs, req)
	}

	drs := make([]*DeleteRequest, 0)
	for _, req := range reqs {
		e := <-req.ErrCh
		if !assert.Nil(t, e) {
			return
		}

		blob, err := ns.Get(logger, req.BlobId)
		if !assert.Nil(t, err) {
			return
		}
		fmt.Printf("got blob %s\n", string(blob))

		dr, err := ns.Delete(logger, req.BlobId)
		if !assert.Nil(t, err) {
			return
		}
		drs = append(drs, dr)
	}

	for _, dr := range drs {
		e := <-dr.ErrCh
		if !assert.Nil(t, e) {
			return
		}

		_, err = ns.Get(logger, dr.BlobId)
		if !assert.Equal(t, ErrMetaNotFound, err) {
			return
		}
	}

}
