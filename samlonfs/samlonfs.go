package samlonfs

import (
	"path"
	"sync"

	"go.uber.org/zap"
)

// A testing simple file system.

const (
	valueStoreDir string = "data"
)

type Samlon struct {
	logger *zap.Logger

	rootDir string

	valueStore *ValueStore // append log value store
	metaStore  *MetaStore  // LSM tree meta store

	writeCh chan *request
}

func NewSamlon(lg *zap.Logger, opts Opts, rootdir string) *Samlon {
	s := &Samlon{
		logger:  lg,
		rootDir: rootdir,
		writeCh: make(chan *request),
	}

	valueStoreDir := path.Join(s.rootDir, valueStoreDir)
	if EnsureDirectory(valueStoreDir) {
	}

	return s
}

func (s *Samlon) Open() error {
	return nil
}

var requestPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return new(request)
	},
}

type request struct {
	// Input
	Ents []*Entry
	// Output
	Ptrs []valuePointer
	Wg   sync.WaitGroup
	Err  error
}

func (req *request) Wait() error {
	req.Wg.Wait()
	req.Ents = nil
	err := req.Err
	requestPool.Put(req)
	return err
}

func (s *Samlon) Put(key, value []byte) (err error) {
	return
}

func (s *Samlon) Get(key []byte) (value []byte, err error) {
	return
}

func EnsureDirectory(dir string) bool {
	return false
}
