package store

import (
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValueStoreWrite(t *testing.T) {
	opts := Opts{
		LoadingMode:         FileIO,
		FileBlockMaxSize:    7,
		FileBlockMaxEntries: 20,
		SyncedFileIO:        true,
	}
	tmpdir := path.Join(os.TempDir(), "value_store")
	err := os.MkdirAll(tmpdir, 0755)
	assert.Nil(t, err)
	defer os.RemoveAll(tmpdir)
	vs := NewValueStore(tmpdir, opts)
	err = vs.Load()
	assert.Nil(t, err)

	var ents []*Entry
	for i := 0; i < 103; i++ {
		e := &Entry{
			BId:  uint64(i),
			Data: []byte(strconv.Itoa(i)),
		}
		ents = append(ents, e)
	}

	for i := 0; i < len(ents)/8; i += 8 {
		req := &request{
			Ents: ents[i : i+8],
		}
		err = vs.Write(req)
		assert.Nil(t, err)
	}
	if len(ents)%8 != 0 {
		req := &request{
			Ents: ents[len(ents)-len(ents)%8:],
		}
		err = vs.Write(req)
		assert.Nil(t, err)
	}
}

func TestValueStoreRead(t *testing.T) {
	opts := Opts{
		LoadingMode:         MemoryMap,
		FileBlockMaxSize:    50,
		FileBlockMaxEntries: 20,
		SyncedFileIO:        true,
	}
	tmpdir := path.Join(os.TempDir(), "value_store")
	err := os.MkdirAll(tmpdir, 0755)
	assert.Nil(t, err)
	defer os.RemoveAll(tmpdir)
	vs := NewValueStore(tmpdir, opts)
	err = vs.Load()
	assert.Nil(t, err)

	var ents []*Entry
	for i := 0; i < 103; i++ {
		e := &Entry{
			BId:  uint64(i),
			Data: []byte(strconv.Itoa(i)),
		}
		ents = append(ents, e)
	}

	var reqs []*request
	for i := 0; i < len(ents)/8; i += 8 {
		req := &request{
			Ents: ents[i : i+8],
		}
		err = vs.Write(req)
		assert.Nil(t, err)
		reqs = append(reqs, req)
	}
	if len(ents)%8 != 0 {
		req := &request{
			Ents: ents[len(ents)-len(ents)%8:],
		}
		err = vs.Write(req)
		assert.Nil(t, err)
		reqs = append(reqs, req)
	}

	// validate data
	for _, req := range reqs {
		for i := range req.Ents {
			ent := req.Ents[i]
			vp := req.Ptrs[i]

			s := &Slice{}
			buf, unlock, err := vs.Read(vp, s)
			if assert.Nil(t, err) {
				assert.Equal(t, ent.Data, buf)
				if unlock != nil {
					unlock()
				}
			}
		}
	}
}
