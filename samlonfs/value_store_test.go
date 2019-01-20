package samlonfs

import (
	"crypto/md5"
	"encoding/binary"
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
		k := make([]byte, 4)
		binary.BigEndian.PutUint32(k, uint32(i))
		e := &Entry{Key: k, Value: dataFunc(i)}
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
				if !assert.Equal(t, ent.Value, buf) {
					return
				}
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
	vs := NewValueStore(tmpdir, opts, nil)
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
	vs := NewValueStore(tmpdir, opts, nil)
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
	vs := NewValueStore(tmpdir, opts, nil)
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
	vs = NewValueStore(tmpdir, opts, nil)
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

	vs := NewValueStore(tmpdir, opts, nil)
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
		index := binary.BigEndian.Uint32(e.Key[:4])
		remd5 := md5.Sum(ents[int(index)].Value)
		emd5 := md5.Sum(e.Value)
		if !assert.Equal(t, remd5, emd5) {
			return fmt.Errorf("Got missmatch entry (%q).", string(e.Key))
		}

		rvp := vps[int(index)]
		if !assert.Equal(t, rvp, vp) {
			return fmt.Errorf("Got missmatch value point (%q). Original (%#v) Current (%#v)",
				string(e.Key), rvp, vp)
		}
		return nil
	}

	sortedFilesId := vs.sortedFilesId()
	for _, fid := range sortedFilesId {
		lf := vs.filesMap[fid]
		var eof uint32
		eof, err = vs.iterate(lf, 0, displayEntry)
		if !assert.Nil(t, err) {
			t.Logf("iterate log file %q current offset: %d. %v", lf.path, eof, err)
			return
		}
		//t.Logf("the file %s eof: %d", lf.path, eof)
		stat, _ := lf.fd.Stat()
		if eof > uint32(stat.Size()) {
			assert.Failf(t, "iterate eof mismatch", "iterate returns eof offset %d bigger than file size %d", eof, stat.Size())
		}
	}
}

var (
	stubIndexEngineGet = func(_ []byte) (vp valuePointer, err error) { return }
)

type fakeIndexEngine struct{}

func (f fakeIndexEngine) Get(key []byte) (valuePointer, error) {
	return stubIndexEngineGet(key)
}

func TestValueStorePickLogs(t *testing.T) {
	tmpdir := path.Join(os.TempDir(), "value_store")
	t.Logf("tmpdir: %q", tmpdir)
	err := os.MkdirAll(tmpdir, 0755)
	assert.Nil(t, err)
	defer os.RemoveAll(tmpdir)

	opts := Opts{
		LoadingMode:         MemoryMap,
		FileBlockMaxSize:    50,
		FileBlockMaxEntries: 20,
		SyncedFileIO:        true,
	}
	engine := &fakeIndexEngine{}
	vs := NewValueStore(tmpdir, opts, engine)
	err = vs.Load()
	assert.Nil(t, err)

	ents := prepareEntries(103, func(i int) []byte {
		return []byte(strconv.Itoa(i))
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

	origStubIndexEngineGet := stubIndexEngineGet
	stubIndexEngineGet = func(key []byte) (valuePointer, error) {
		index := int(binary.BigEndian.Uint32(key[:4]))
		if index < 50 {
			return valuePointer{}, ErrValuePointerNotFound
		}
		return vps[index], nil
	}
	defer func() { stubIndexEngineGet = origStubIndexEngineGet }()

	head := vps[len(ents)-1]
	lfs, err := vs.pickLogFiles(head, 1)
	if assert.Nil(t, err) {
		for i := range lfs {
			t.Logf("Got log files: %q fid: %d", lfs[i].path, lfs[i].fid)
		}
	}
}

func TestValueStoreGC(t *testing.T) {
	tmpdir := path.Join(os.TempDir(), "value_store")
	t.Logf("tmpdir: %q", tmpdir)
	err := os.MkdirAll(tmpdir, 0755)
	assert.Nil(t, err)
	defer os.RemoveAll(tmpdir)

	opts := Opts{
		LoadingMode:         MemoryMap,
		FileBlockMaxSize:    50,
		FileBlockMaxEntries: 20,
		SyncedFileIO:        true,
	}
	engine := &fakeIndexEngine{}
	vs := NewValueStore(tmpdir, opts, engine)
	err = vs.Load()
	assert.Nil(t, err)

	ents := prepareEntries(103, func(i int) []byte {
		return []byte(strconv.Itoa(i))
	})
	reqs := prepareRequests(8, ents)

	vps := make(map[uint32]valuePointer)
	for i := range reqs {
		err := vs.Write(reqs[i])
		if !assert.Nil(t, err) {
			return
		}
		for ii := range reqs[i].Ptrs {
			e := reqs[i].Ents[ii]
			vp := reqs[i].Ptrs[ii]
			key := binary.BigEndian.Uint32(e.Key[:4])
			vps[key] = vp
		}
	}

	// rewrite the first request data
	reqs[0].Ptrs = nil
	err = vs.Write(reqs[0])
	if !assert.Nil(t, err) {
		return
	}
	for i := range reqs[0].Ptrs {
		e := reqs[0].Ents[i]
		vp := reqs[0].Ptrs[i]
		key := binary.BigEndian.Uint32(e.Key[:4])
		vps[key] = vp
	}

	anchor := binary.BigEndian.Uint32(reqs[1].Ents[0].Key[:4])

	origStubIndexEngineGet := stubIndexEngineGet
	stubIndexEngineGet = func(key []byte) (valuePointer, error) {
		k := binary.BigEndian.Uint32(key[:4])
		if k >= anchor {
			if k%2 == 0 {
				return valuePointer{}, ErrValuePointerNotFound
			}
		}
		return vps[k], nil
	}
	defer func() { stubIndexEngineGet = origStubIndexEngineGet }()

	headEntry := ents[len(ents)-3]
	head := vps[binary.BigEndian.Uint32(headEntry.Key[:4])]
	err = vs.RunGC(head, 0.5)
	assert.Nil(t, err)
}
