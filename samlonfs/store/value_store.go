package store

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
)

var (
	ErrLogFileNotFound error = errors.New("log file not found")
	ErrCrcInvalid      error = errors.New("crc invalid")
)

type ValueStore struct {
	opts    Opts
	dirPath string

	// guards our view of which files exists
	filesLock sync.RWMutex
	filesMap  map[uint32]*logFile

	maxFid              uint32 // accessed via atomic
	writableBlockOffset uint32 // read by read, write by write. Must access via atomics.
	numEntriesWritten   uint32

	// gc
	filesToBeDeleted []uint32

	// FIXME: index engine for checking whether any one holding this data now.
}

func NewValueStore(dir string, opts Opts) *ValueStore {
	e := &ValueStore{
		opts:    opts,
		dirPath: dir,
	}
	return e
}

func (vs *ValueStore) Load() error {
	if err := vs.populateFilesMap(); err != nil {
		return errors.Wrap(err, "Unable to populate files map")
	}
	// zero files exist
	if len(vs.filesMap) == 0 {
		_, err := vs.createLogFile(0)
		if err != nil {
			return errors.Wrap(err, "Unable to initialize the first log file")
		}
		return nil
	}

	filesId := vs.sortedFilesId()
	for _, fileId := range filesId {
		lf := vs.filesMap[fileId]
		if err := lf.openReadOnly(); err != nil {
			return errors.Wrapf(err, "Unable to open log file %q", lf.path)
		}
	}

	// appending a new log file for current writing
	// FIXME: we don't use the last one file any more, in case
	// something wrong cause we modified the file.
	maxFid := atomic.AddUint32(&vs.maxFid, 1)
	_, err := vs.createLogFile(maxFid)
	if err != nil {
		return errors.Wrapf(err, "Unable to create the current writing file %q", vs.fpath(maxFid))
	}

	return nil
}

func (vs *ValueStore) Close() error {
	for _, lf := range vs.filesMap {
		if err := lf.sync(); err != nil {
			return errors.Wrapf(err, "Unable to sync file %q", lf.path)
		}
		if err := lf.munmap(); err != nil {
			return errors.Wrapf(err, "Unable to munmap file %q", lf.path)
		}
		if err := lf.fd.Close(); err != nil {
			return errors.Wrapf(err, "Unable to close file %q", lf.path)
		}
	}
	return nil
}

func (vs *ValueStore) sortedFilesId() []uint32 {
	// FIXME: maybe there is a list contains ready to delete files.
	filesId := make([]uint32, 0, len(vs.filesMap))
	for fileId, _ := range vs.filesMap {
		filesId = append(filesId, fileId)
	}
	sort.Slice(filesId, func(i, j int) bool {
		return filesId[i] < filesId[j]
	})
	return filesId
}

func logFilePath(dirPath string, fid uint32) string {
	return fmt.Sprintf("%s%s%06d.data", dirPath, string(os.PathSeparator), fid)
}

func (vs *ValueStore) fpath(fid uint32) string {
	return logFilePath(vs.dirPath, fid)
}

func (vs *ValueStore) populateFilesMap() error {
	vs.filesMap = make(map[uint32]*logFile)
	files, err := ioutil.ReadDir(vs.dirPath)
	if err != nil {
		return errors.Wrapf(err, "Unable to read directory %q", vs.dirPath)
	}

	found := make(map[uint64]struct{})
	for i := range files {
		file := files[i]
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".data") {
			continue
		}

		fsz := len(file.Name())
		fid, err := strconv.ParseUint(file.Name()[:fsz-5], 10, 32)
		if err != nil {
			return fmt.Errorf("Unable to parse file %q id. %v", file.Name(), err)
		}
		// FIXME: The file system ensure just one file with same name can exists.
		if _, ok := found[fid]; ok {
			return fmt.Errorf("Duplicated file %q found", file.Name())
		}
		found[fid] = struct{}{}

		lf := &logFile{
			fid:         uint32(fid),
			path:        vs.fpath(uint32(fid)),
			loadingMode: vs.opts.LoadingMode,
		}
		vs.filesMap[uint32(fid)] = lf
		if lf.fid > vs.maxFid {
			vs.maxFid = lf.fid
		}
	}

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

func (vs *ValueStore) woffset() uint32 {
	return atomic.LoadUint32(&vs.writableBlockOffset)
}

func (vs *ValueStore) createLogFile(fid uint32) (lf *logFile, err error) {
	lf = &logFile{
		path:        vs.fpath(fid),
		fid:         fid,
		loadingMode: vs.opts.LoadingMode,
	}
	//log.Printf("Value store prepare to create log file. fid: %d filepath: %q", lf.fid, lf.path)
	atomic.StoreUint32(&vs.writableBlockOffset, 0)
	vs.numEntriesWritten = 0

	// FIXME: Maybe we can truncate file for better writing in XFS.
	lf.fd, err = OpenSyncedFile(lf.path, vs.opts.SyncedFileIO)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to open file %q", lf.path)
	}
	if err := SyncDir(vs.dirPath); err != nil {
		lf.fd.Close()
		return nil, errors.Wrapf(err, "Unable to sync dir %q for create file %q", vs.dirPath, lf.path)
	}
	if err := lf.mmap(2 * vs.opts.FileBlockMaxSize); err != nil {
		lf.fd.Close()
		return nil, errors.Wrapf(err, "Unable to mmap file %q", lf.path)
	}

	vs.filesLock.Lock()
	vs.filesMap[fid] = lf
	vs.filesLock.Unlock()

	return lf, nil
}

func (vs *ValueStore) Write(req *request) (err error) {
	vs.filesLock.RLock()
	currFid := atomic.LoadUint32(&vs.maxFid)
	currlf := vs.filesMap[currFid]
	vs.filesLock.RUnlock()

	var buf bytes.Buffer
	toDisk := func() error {
		if buf.Len() == 0 {
			return nil
		}

		nr, err := currlf.fd.Write(buf.Bytes())
		if err != nil {
			log.Printf("Unable to persist data into disk, %v", err)
			return errors.Wrap(err, "Unable to persist data")
		}
		buf.Reset()

		atomic.AddUint32(&vs.writableBlockOffset, uint32(nr))

		if vs.numEntriesWritten >= vs.opts.FileBlockMaxEntries ||
			vs.woffset() >= uint32(vs.opts.FileBlockMaxSize) {
			if err := currlf.doneWriting(); err != nil {
				return errors.Wrapf(err, "Done writing file %d fail", currlf.fid)
			}
			newid := atomic.AddUint32(&vs.maxFid, 1)
			newlf, err := vs.createLogFile(newid)
			if err != nil {
				return err
			}
			currlf = newlf
		}
		return nil
	}

	for i := range req.Ents {
		e := req.Ents[i]

		var vp valuePointer
		vp.Fid = currlf.fid
		vp.Offset = vs.woffset() + uint32(buf.Len())
		vp.Len, err = encodeEntry(e, &buf)
		if err != nil {
			return errors.Wrapf(err, "Unable to encode entry %d", e.BId)
		}
		vs.numEntriesWritten += 1
		req.Ptrs = append(req.Ptrs, vp)
		writeNow :=
			vs.woffset()+uint32(buf.Len()) > uint32(vs.opts.FileBlockMaxSize) ||
				vs.numEntriesWritten > vs.opts.FileBlockMaxEntries
		if writeNow {
			if err := toDisk(); err != nil {
				return err
			}
		}
	}

	return toDisk()
}

func (vs *ValueStore) Read(vp valuePointer, s *Slice) ([]byte, func(), error) {
	maxFid := atomic.LoadUint32(&vs.maxFid)
	if vp.Fid == maxFid && vp.Offset >= vs.woffset() {
		return nil, nil, errors.Errorf(
			"Invalid value pointer offset: %d greater than current offset: %d",
			vp.Offset, vs.woffset())
	}

	buf, unlock, err := vs.getValueBytes(vp, s)
	if err != nil {
		return nil, nil, err
	}
	var head header
	head.Decode(buf)
	return buf[headerSize : uint32(headerSize)+head.dlen], unlock, nil
}

func (vs *ValueStore) getValueBytes(vp valuePointer, s *Slice) ([]byte, func(), error) {
	lf, err := vs.getFileRLocked(vp.Fid)
	if err != nil {
		return nil, nil, err
	}
	buf, err := lf.read(vp, s)
	if err != nil {
		lf.lock.RUnlock()
		return nil, nil, err
	}
	if vs.opts.LoadingMode == MemoryMap {
		return buf, lf.lock.RUnlock, nil
	}
	lf.lock.RUnlock()
	return buf, nil, nil
}

func (vs *ValueStore) getFileRLocked(fid uint32) (*logFile, error) {
	vs.filesLock.RLock()
	lf, ok := vs.filesMap[fid]
	vs.filesLock.RUnlock()
	if !ok {
		return nil, ErrLogFileNotFound
	}
	lf.lock.RLock()
	return lf, nil
}

var (
	ErrStop = errors.New("iterate stop")
)

type valueEntry func(e *Entry, vp valuePointer) error

type safeRead struct {
	bid uint64
	v   []byte

	recordOffset uint32
}

func (r *safeRead) Entry(reader *bufio.Reader) (*Entry, error) {
	var hbuf [headerSize]byte

	hash := crc32.New(CastagnoliTable)
	tee := io.TeeReader(reader, hash)

	if _, err := io.ReadFull(tee, hbuf[:]); err != nil {
		return nil, err
	}
	var head header
	head.Decode(hbuf[:])
	if cap(r.v) < int(head.dlen) {
		r.v = make([]byte, int(2*head.dlen))
	}

	e := &Entry{}
	e.BId = head.bid
	e.Data = r.v[:head.dlen]

	if _, err := io.ReadFull(tee, e.Data); err != nil {
		return nil, err
	}

	var crcbuf [crc32.Size]byte
	if _, err := reader.Read(crcbuf[:]); err != nil {
		return nil, err
	}
	crc := binary.BigEndian.Uint32(crcbuf[:])
	if crc != hash.Sum32() {
		return nil, ErrCrcInvalid
	}

	return e, nil
}

func (vs *ValueStore) iterate(lf *logFile, offset uint32, fn valueEntry) (eof uint32, err error) {
	stat, err := lf.fd.Stat()
	if err != nil {
		return 0, err
	}
	if int64(offset) >= stat.Size() {
		return 0, ErrEOF
	}
	if _, err := lf.fd.Seek(int64(offset), io.SeekStart); err != nil {
		return 0, errors.Wrapf(err, "Unable to seek file %q", lf.path)
	}

	reader := bufio.NewReader(lf.fd)
	read := &safeRead{
		v: make([]byte, 10), // maybe 1M
	}

	var validEof uint32
	for {
		var entry *Entry
		entry, err = read.Entry(reader)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return validEof, err
		}

		var vp valuePointer
		vp.Fid = lf.fid
		vp.Offset = read.recordOffset
		vp.Len = uint32(headerSize + len(entry.Data) + crc32.Size)

		if err = fn(entry, vp); err != nil {
			if err == ErrStop {
				break
			}
			return validEof, err
		}
		read.recordOffset += vp.Len
		validEof = read.recordOffset
	}

	return validEof, nil
}

type logFile struct {
	path string

	// when trying to reopen file, it's possible gets a wrong fd in a short
	// time.
	lock        sync.RWMutex
	fd          *os.File
	fid         uint32
	fmap        []byte // for mmap
	size        uint32
	loadingMode FileLoadingMode
}

var (
	ErrEOF error = errors.New("End of mapped region")
)

func (f *logFile) read(vp valuePointer, s *Slice) (buf []byte, err error) {
	//log.Printf("log file read. fid: %d len: %d offset: %d fmap: %d", vp.Fid, vp.Len, vp.Offset, len(f.fmap))

	var nbr int64
	offset := vp.Offset

	if f.loadingMode == FileIO {
		buf = s.Resize(int(vp.Len))
		var n int
		n, err = f.fd.ReadAt(buf, int64(offset))
		nbr = int64(n)
	} else {
		size := int64(len(f.fmap))
		valsz := vp.Len
		if int64(offset) >= size || int64(offset+valsz) > size {
			err = ErrEOF
		}
		if err == nil {
			buf = f.fmap[offset : offset+valsz]
			nbr = int64(valsz)
		}
	}
	_ = nbr
	//log.Printf("read count: %d", nbr)
	return buf, err
}

func (f *logFile) openReadOnly() error {
	var err error
	f.fd, err = os.OpenFile(f.path, os.O_RDONLY, 0666)
	if err != nil {
		return errors.Wrapf(err, "Unable to open %q as RDONLY", f.path)
	}

	stat, err := f.fd.Stat()
	if err != nil {
		return errors.Wrapf(err, "Unable to check stat for: %q", f.path)
	}
	f.size = uint32(stat.Size())

	// FIXME: If mmap faild, we can't read original file any more.
	if err := f.mmap(stat.Size()); err != nil {
		return errors.Wrapf(err, "Unable to mmap file: %q", f.path)
	}
	return nil
}

func (f *logFile) mmap(sz int64) error {
	if f.loadingMode != MemoryMap {
		return nil
	}
	var err error
	if f.fmap, err = Mmap(f.fd, false, sz); err == nil {
		err = Madvise(f.fmap, false) // for random reading
	}
	return err
}

func (f *logFile) munmap() error {
	if f.loadingMode != MemoryMap {
		return nil
	}
	if err := Munmap(f.fmap); err != nil {
		return errors.Wrapf(err, "Unable to munmap log file: %q", f.path)
	}
	return nil
}

func (f *logFile) doneWriting() error {
	if err := f.fd.Sync(); err != nil {
		return errors.Wrapf(err, "Unable to sync log file: %q", f.path)
	}
	f.lock.Lock()
	defer f.lock.Unlock()
	if err := f.munmap(); err != nil {
		return err
	}
	if err := f.fd.Close(); err != nil {
		return errors.Wrapf(err, "Unable to close the file: %q", f.path)
	}
	return f.openReadOnly()
}

func (f *logFile) sync() error {
	return f.fd.Sync()
}
