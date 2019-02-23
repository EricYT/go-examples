package samlonfs

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

var (
	metaDataBucket   []byte = []byte("meta-data-bucket")
	metaOthersBucket []byte = []byte("meta-others-bucket")

	boltDBFile string = "meta.db"
)

type MetaStore struct {
	dirPath string

	db *bolt.DB
}

func NewMetaStore(dir string) *MetaStore {
	m := &MetaStore{
		dirPath: dir,
	}

	return m
}

func (m *MetaStore) dbPath() string {
	return fmt.Sprintf("%s%c%s", m.dirPath, os.PathSeparator, boltDBFile)
}

func (m *MetaStore) Open() error {
	// FIXME: options
	opts := bolt.DefaultOptions
	db, err := bolt.Open(m.dbPath(), 0666, opts)
	if err != nil {
		return errors.Wrapf(err, "Unable to open bolt db %q", m.dbPath())
	}

	// initialize buckets
	tx, err := db.Begin(true)
	if err != nil {
		return errors.Wrap(err, "Unable to create db tx")
	}
	defer tx.Rollback()
	if _, err := tx.CreateBucketIfNotExists(metaDataBucket); err != nil {
		return errors.Wrapf(err, "Unable to create bucket %s", string(metaDataBucket))
	}
	if _, err := tx.CreateBucketIfNotExists(metaOthersBucket); err != nil {
		return errors.Wrapf(err, "Unable to create bucket %s", string(metaOthersBucket))
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "Unable to commit bucket tx")
	}
	m.db = db
	return nil
}

func (m *MetaStore) Close() error {
	err := m.db.Close()
	return err
}

func (m *MetaStore) Put(key, value []byte) error {
	err := m.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(metaDataBucket)
		return b.Put(key, value)
	})
	return err
}

func (m *MetaStore) Get(key []byte) ([]byte, error) {
	var value []byte
	err := m.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(metaDataBucket)
		value = b.Get(key)
		return nil
	})
	return value, err
}
