package store

import (
	"sync"

	"git.jd.com/cloud-storage/newds-datanode/pkg/logutil"
	"github.com/pkg/errors"
)

var (
	ErrMetaNotFound error = errors.New("meta not found")
)

type MetaStore interface {
	Load() error
	Close() error

	Put(key int64, value []byte) (err error)
	Get(key int64) (val []byte, err error)
	Delete(key int64) (err error)

	// FIXME: snapshot
}

// memory meta store
type MemoryMetaStore struct {
	metaLock sync.RWMutex
	meta     map[int64][]byte
}

func NewMemoryMetaStore(logger logutil.Logger) *MemoryMetaStore {
	ms := &MemoryMetaStore{
		meta: make(map[int64][]byte),
	}
	return ms
}

func (m *MemoryMetaStore) Load() error {
	return nil
}

func (m *MemoryMetaStore) Close() error { return nil }

func (m *MemoryMetaStore) Put(key int64, meta []byte) error {
	m.metaLock.Lock()
	defer m.metaLock.Unlock()
	m.meta[key] = meta
	return nil
}

func (m *MemoryMetaStore) Get(key int64) (meta []byte, err error) {
	m.metaLock.RLock()
	defer m.metaLock.RUnlock()
	meta, found := m.meta[key]
	if !found {
		return meta, ErrMetaNotFound
	}
	return meta, nil
}

func (m *MemoryMetaStore) Delete(key int64) (err error) {
	m.metaLock.Lock()
	defer m.metaLock.Unlock()
	delete(m.meta, key)
	return nil
}
