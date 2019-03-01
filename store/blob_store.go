package store

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"git.jd.com/cloud-storage/newds-datanode/pkg/logutil"
	"github.com/pkg/errors"
	"go.etcd.io/etcd/pkg/fileutil"
)

var (
	ErrBlockFileNotFound error = errors.New("block file not found")
)

var (
	blockFilePrefix string = "block_"
	blockFileSuffix string = ".data"
)

const maxBlockFileSize int64 = 128
const blockFilePreallocateSize int64 = 1 * 1024

type BlobStore interface {
	Load() error
	Close()

	Write(reqs []*Request) error
	Read(bp BlobPointer) (val []byte, err error)

	// FIXME: snapshot
}

type blobStore struct {
	logger logutil.Logger

	root string

	blockFileLock  sync.RWMutex
	blocks         map[uint32]*blockFile
	maxBlockFileId uint32
	Sync           bool
}

func NewBlobStore(logger logutil.Logger, root string) *blobStore {
	bs := &blobStore{
		logger: logger,
		root:   root,
	}
	return bs
}

func (b *blobStore) Load() error {
	b.logger.Info("blob store loading start.")

	if err := b.populateBlockFiles(); err != nil {
		b.logger.Errorf("blob store %s populated block files failed. %v", b.root, err)
		return err
	}

	if len(b.blocks) == 0 {
		fid := b.maxBlockFileId + 1
		blockFile, err := b.createBlockFile(fid)
		if err != nil {
			b.logger.Errorf("blob store %s create block file %d failed. %v", b.root, fid)
			return err
		}
		atomic.StoreUint32(&b.maxBlockFileId, fid)
		b.blocks[fid] = blockFile
	}

	b.logger.Infof("blob store loading stopped. Max block file id: %d", atomic.LoadUint32(&b.maxBlockFileId))
	return nil
}

func (b *blobStore) populateBlockFiles() error {
	b.logger.Infof("blob store %s populate block files start.", b.root)

	blockFiles, err := ioutil.ReadDir(b.root)
	if err != nil {
		b.logger.Errorf("blob store read %s directory failed. %v", b.root, err)
		return errors.Wrapf(err, "Unable to read blob store %s directory.", b.root)
	}

	var (
		maxBlockFileId uint32 = 0
		fileBlocks            = make(map[uint32]*blockFile)
	)
	for _, bf := range blockFiles {
		if bf.IsDir() {
			continue
		}

		name := bf.Name()
		if !strings.HasPrefix(name, blockFilePrefix) || !strings.HasSuffix(name, blockFileSuffix) {
			b.logger.Errorf("blob store %s has wrong name file %s", b.root, name)
			continue
		}
		blockFileIdTmp := []byte(name)[len(blockFilePrefix) : len(blockFilePrefix)+12]
		blockFileId, err := strconv.ParseUint(string(blockFileIdTmp), 10, 32)
		if err != nil {
			b.logger.Errorf("blob store %s hash wrong block file id %s", b.root, name)
			return err
		}

		id := uint32(blockFileId)
		path := b.blockFilePathById(id)
		blockFile, err := newBlockFile(id, path, b.Sync)
		if err != nil {
			b.logger.Errorf("blob store %s load block file %d failed. %v", b.root, id, err)
			return err
		}
		fileBlocks[id] = blockFile

		if id > maxBlockFileId {
			maxBlockFileId = id
		}
	}
	b.blocks = fileBlocks
	b.maxBlockFileId = maxBlockFileId

	b.logger.Infof("blob store populate file blocks done. %s", b.root)

	return nil
}

func (b *blobStore) Close() {
	b.logger.Infof("blob store %s close.", b.root)
	b.blockFileLock.Lock()
	defer b.blockFileLock.Unlock()
	for _, block := range b.blocks {
		block.Close()
	}
}

func (b *blobStore) blockFilePathById(fid uint32) string {
	return fmt.Sprintf("%s%c%s%012d%s", b.root, filepath.Separator, blockFilePrefix, fid, blockFileSuffix)
}

func (b *blobStore) createBlockFile(fid uint32) (*blockFile, error) {
	path := b.blockFilePathById(fid)
	blockFile, err := newBlockFile(fid, path, b.Sync)
	if err != nil {
		b.logger.Errorf("blob store %s create block file %s failed. %v", b.root, path, err)
		return nil, err
	}
	b.blockFileLock.Lock()
	b.blocks[fid] = blockFile
	b.blockFileLock.Unlock()
	return blockFile, nil
}

func (b *blobStore) Write(reqs []*Request) error {
	b.logger.Debugf("blob store write entries count %d", len(reqs))

	b.blockFileLock.RLock()
	fid := atomic.LoadUint32(&b.maxBlockFileId)
	bf := b.blocks[fid]
	b.blockFileLock.RUnlock()

	var err error
	var buf bytes.Buffer
	toDisk := func() error {
		if buf.Len() == 0 {
			return nil
		}

		if err := bf.Write(buf.Bytes()); err != nil {
			b.logger.Errorf("blob store %s write block file %d failed. %v", b.root, fid, err)
			return err
		}

		if int64(buf.Len())+bf.Size() >= maxBlockFileSize {
			fid++
			bf, err = b.createBlockFile(fid)
			if err != nil {
				b.logger.Errorf("blob store %s create new block file %d failed. %v", b.root, fid, err)
				return err
			}
			atomic.StoreUint32(&b.maxBlockFileId, fid)
		}
		buf.Reset()
		return nil
	}

	off := bf.Size()
	for _, req := range reqs {
		n := encodeRequest(req, &buf)

		var bp BlobPointer
		bp.FileId = fid
		bp.Offset = off
		bp.Length = uint32(n)

		off += int64(n)
		req.Ptr = bp

		if int64(buf.Len())+bf.Size() >= maxBlockFileSize {
			if err := toDisk(); err != nil {
				b.logger.Errorf("blob store %s persist to disk %d failed. %v", b.root, fid, err)
				return err
			}
			// reset offset
			off = bf.Size()
		}
	}

	return toDisk()
}

func (b *blobStore) Read(bp BlobPointer) (blob []byte, err error) {
	b.logger.Debugf("blob store %s read ptr: %s", b.root, bp)

	b.blockFileLock.RLock()
	bf, found := b.blocks[bp.FileId]
	b.blockFileLock.RUnlock()
	if !found {
		return nil, ErrBlockFileNotFound
	}

	blob, err = bf.ReadAt(int64(bp.Offset), bp.Length)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to read blob %v from block file", bp)
	}

	var h header
	h.Decode(blob)
	return blob[headerSize+int(h.userMetaLen) : headerSize+int(h.userMetaLen)+int(h.blobLen)], nil
}

// block file
type blockFile struct {
	id   uint32
	path string
	size int64
	fd   *os.File
}

func newBlockFile(id uint32, path string, syn bool) (*blockFile, error) {
	bf := &blockFile{
		id:   id,
		path: path,
	}

	flag := os.O_APPEND | os.O_CREATE | os.O_RDWR
	if syn {
		flag |= os.O_SYNC
	}
	fd, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to open block file %s", path)
	}
	// preallocate file
	if err := fileutil.Preallocate(fd, blockFilePreallocateSize, false); err != nil {
		return nil, errors.Wrapf(err, "Unable to preallocate block file %s", path)
	}
	stat, err := fd.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to state block file %s", path)
	}
	bf.fd = fd
	bf.size = stat.Size()

	return bf, nil
}

func (b *blockFile) Close() {
	b.fd.Close()
}

func (b *blockFile) Size() int64 {
	return b.size
}

func (b *blockFile) Write(buf []byte) (err error) {
	off := 0
	for len(buf) > 0 {
		n, err := b.fd.Write(buf)
		if err != nil {
			if err != io.ErrShortWrite {
				return err
			}
		}
		buf = buf[off+n:]
		off += n
	}
	b.size += int64(off)
	return nil
}

func (b *blockFile) ReadAt(off int64, length uint32) ([]byte, error) {
	//FIXME: boundary check

	buf := make([]byte, length)
	if _, err := b.fd.ReadAt(buf, off); err != nil {
		return nil, err
	}
	return buf, nil
}
