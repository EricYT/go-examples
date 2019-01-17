package store

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func prepareEntries(num int, dataFunc func(i int) []byte) []*Entry {
	ents := make([]*Entry, 0, num)
	for i := 0; i < num; i++ {
		e := &Entry{BId: uint64(i), Data: dataFunc(i)}
		ents = append(ents, e)
	}
	return ents
}

func prepareRequests(round int, ents []*Entry) []*request {
	reqs := make([]*request, 0, len(ents)/round)
	for i := 0; i < len(ents)/round; i++ {
		req := &request{
			Ents: ents[i*round : (i+1)*round],
		}
		reqs = append(reqs, req)
	}
	if len(ents)/round != 0 {
		req := &request{
			Ents: ents[len(ents)-len(ents)%round:],
		}
		reqs = append(reqs, req)
	}
	return reqs
}

func validateEntries(t *testing.T, vs *ValueStore, reqs []*request) {
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

	ents := prepareEntries(103, func(i int) []byte {
		return []byte(strconv.Itoa(i))
	})

	reqs := prepareRequests(8, ents)
	for i := range reqs {
		err := vs.Write(reqs[i])
		if !assert.Nil(t, err) {
			return
		}
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

	ents := prepareEntries(103, func(i int) []byte {
		return []byte(strconv.Itoa(i))
	})

	reqs := prepareRequests(8, ents)

	for i := range reqs {
		err := vs.Write(reqs[i])
		if !assert.Nil(t, err) {
			return
		}
	}

	validateEntries(t, vs, reqs)
}

func TestValueStoreLoad(t *testing.T) {
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

	ents := prepareEntries(103, func(i int) []byte {
		return []byte(strconv.Itoa(i))
	})

	reqs := prepareRequests(8, ents)

	for i := range reqs {
		err := vs.Write(reqs[i])
		if !assert.Nil(t, err) {
			return
		}
	}

	// validate data
	validateEntries(t, vs, reqs)

	// close value store
	err = vs.Close()
	if !assert.Nil(t, err) {
		return
	}

	// reopen
	vs = NewValueStore(tmpdir, opts)
	err = vs.Load()
	if !assert.Nil(t, err) {
		return
	}

	validateEntries(t, vs, reqs)

	ents = prepareEntries(10, func(i int) []byte {
		return []byte(strconv.Itoa(i))
	})
	req := &request{
		Ents: ents,
	}
	err = vs.Write(req)
	if !assert.Nil(t, err) {
		return
	}
	validateEntries(t, vs, []*request{req})
}

func TestValueStoreIterate(t *testing.T) {
	opts := Opts{
		LoadingMode:         MemoryMap,
		FileBlockMaxSize:    12 * 1024,
		FileBlockMaxEntries: 20,
		SyncedFileIO:        true,
	}
	tmpdir := path.Join(os.TempDir(), "value_store")
	err := os.MkdirAll(tmpdir, 0755)
	assert.Nil(t, err)
	defer os.RemoveAll(tmpdir)
	vs := NewValueStore(tmpdir, opts)
	err = vs.Load()
	if !assert.Nil(t, err) {
		return
	}
	defer vs.Close()

	ents := prepareEntries(105, func(i int) []byte {
		len := rand.Intn(1 * 1024)
		randData := make([]byte, len)
		rand.Read(randData)
		return randData
	})

	reqs := prepareRequests(8, ents)

	vps := make([]valuePointer, 0, len(ents))
	for i := range reqs {
		err := vs.Write(reqs[i])
		if !assert.Nil(t, err) {
			return
		}
		vps = append(vps, reqs[i].Ptrs...)
	}

	displayEntry := func(e *Entry, vp valuePointer) error {
		remd5 := md5.Sum(ents[int(e.BId)].Data)
		emd5 := md5.Sum(e.Data)
		if !assert.Equal(t, remd5, emd5) {
			return fmt.Errorf("Got missmatch entry (%d).", e.BId)
		}

		rvp := vps[int(e.BId)]
		if !assert.Equal(t, rvp, vp) {
			return fmt.Errorf("Got missmatch value point (%d). Original (%#v) Current (%#v)",
				e.BId, rvp, vp)
		}
		return nil
	}

	sortedFilesId := vs.sortedFilesId()
	for _, fid := range sortedFilesId {
		lf := vs.filesMap[fid]
		eof, err := vs.iterate(lf, 0, displayEntry)
		if !assert.Nil(t, err) {
			return
		}
		t.Logf("the file %s eof: %d", lf.path, eof)
		stat, _ := lf.fd.Stat()
		if eof > uint32(stat.Size()) {
			assert.Failf(t, "iterate eof mismatch", "iterate returns eof offset %d bigger than file size %d", eof, stat.Size())
		}
	}
}
