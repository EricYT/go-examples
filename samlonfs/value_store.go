package samlonfs

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
	"time"

	"github.com/pkg/errors"
)

var (
	ErrRetry      error = errors.New("log file retry") // log file maybe has garbage collection
	ErrCrcInvalid error = errors.New("crc invalid")
)

var (
	ErrValuePointerNotFound error = errors.New("value pointer not found")
)

type IndexEngine interface {
	Get(key []byte) (valuePointer, error)
}

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

	runGC chan struct{}

	// FIXME: index engine for checking whether any one holding this data now.
	indexEngine IndexEngine
}

func NewValueStore(dir string, opts Opts, ie IndexEngine) *ValueStore {
	e := &ValueStore{
		opts:        opts,
		dirPath:     dir,
		runGC:       make(chan struct{}, 1),
		indexEngine: ie,
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
	filesToBeDeleted := make(map[uint32]struct{})
	for i := range vs.filesToBeDeleted {
		filesToBeDeleted[vs.filesToBeDeleted[i]] = struct{}{}
	}
	filesId := make([]uint32, 0, len(vs.filesMap))
	for fileId, _ := range vs.filesMap {
		if _, ok := filesToBeDeleted[fileId]; ok {
			continue
		}
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

// FIXME: Now, it's not thread-safe function.
// Maybe it's better way to allocate the offset for current
// writing.
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
			return errors.Wrapf(err, "Unable to encode entry %q", string(e.Key))
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

// FIXME: thread-safe
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
	return buf[uint32(headerSize)+head.klen : uint32(headerSize)+head.klen+head.vlen], unlock, nil
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
		return nil, ErrRetry
	}
	lf.lock.RLock()
	return lf, nil
}

var (
	ErrStop = errors.New("iterate stop")
)

type valueEntry func(e *Entry, vp valuePointer) error

type safeRead struct {
	k []byte
	v []byte

	recordOffset uint32
}

func (r *safeRead) Entry(reader *bufio.Reader) (*Entry, error) {
	hash := crc32.New(CastagnoliTable)
	tee := io.TeeReader(reader, hash)

	var hbuf [headerSize]byte
	if _, err := io.ReadFull(tee, hbuf[:]); err != nil {
		return nil, err
	}
	var head header
	head.Decode(hbuf[:])
	if cap(r.k) < int(head.klen) {
		r.k = make([]byte, int(2*head.klen))
	}
	if cap(r.v) < int(head.vlen) {
		r.v = make([]byte, int(2*head.vlen))
	}

	e := &Entry{}
	e.Key = r.k[:head.klen]
	e.Value = r.v[:head.vlen]

	if _, err := io.ReadFull(tee, e.Key); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(tee, e.Value); err != nil {
		return nil, err
	}

	var crcbuf [crc32.Size]byte
	if _, err := io.ReadFull(reader, crcbuf[:]); err != nil {
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
		if err == io.EOF || err == io.ErrUnexpectedEOF { // io.ReadFull
			break
		}
		if err != nil {
			return validEof, err
		}

		var vp valuePointer
		vp.Fid = lf.fid
		vp.Offset = read.recordOffset
		vp.Len = uint32(headerSize + len(entry.Key) + len(entry.Value) + crc32.Size)

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

var (
	ErrDataMissing error = errors.New("data missing")
)

func (vs *ValueStore) pickLogFiles(head valuePointer, ratio float64) (lfs []*logFile, err error) {
	maxFid := atomic.LoadUint32(&vs.maxFid)
	writeOffset := atomic.LoadUint32(&vs.writableBlockOffset)
	if head.Fid > maxFid || (head.Fid == maxFid && head.Offset > writeOffset) {
		return nil, ErrDataMissing
	}

	// From the oldest file
	sortedFilesId := vs.sortedFilesId()
	for i := range sortedFilesId {
		fileId := sortedFilesId[i]
		if fileId > head.Fid {
			continue
		}

		vs.filesLock.RLock()
		lf, ok := vs.filesMap[fileId]
		vs.filesLock.RUnlock()
		if !ok {
			log.Printf("value store pick log file %d missing", fileId)
			// Continue or die
			continue
		}

		var discard uint32
		eofOffset, err := vs.iterate(lf, 0, func(entry *Entry, vp valuePointer) error {
			if vp.Fid == head.Fid && vp.Offset >= head.Offset {
				return ErrStop
			}

			key := entry.Key
			currvp, err := vs.indexEngine.Get(key)
			if err != nil {
				if ErrValuePointerNotFound == err {
					discard += vp.Len
					return nil
				}
				return errors.Wrapf(err, "Unable to get %q value pointer from index", key)
			}

			if currvp.Fid > vp.Fid {
				// the entry already move to the head
				discard += vp.Len
			} else if currvp.Fid == vp.Fid {
				if currvp.Offset > vp.Offset {
					// the entry already move to the head
					discard += vp.Len
				}
			}
			// FIXME: small than the index store ? Damn it.

			return nil
		})
		if err != nil {
			log.Printf("value store iterate file %q fail. %v", lf.path, err)
			return nil, err
		}
		if eofOffset == 0 {
			// Empty file
			log.Printf("value store got a empty log file %q", lf.path)
			lfs = append(lfs, lf)
			continue
		}

		sparseRatio := float64(discard) / float64(eofOffset)
		log.Printf("value store file %q discard: %d log size: %d sparse ratio: %f target ratio: %f", lf.path, discard, eofOffset, sparseRatio, ratio)
		if sparseRatio >= ratio {
			lfs = append(lfs, lf)
		}
	}

	return lfs, nil
}

func (vs *ValueStore) runGCLogFile(lf *logFile, ratio float64) error {
	log.Printf("value store ready to gc log file %q", lf.path)

	_, err := vs.iterate(lf, 0, func(e *Entry, vp valuePointer) error {

		key := e.Key
		currvp, err := vs.indexEngine.Get(key)
		if err != nil {
			if err == ErrValuePointerNotFound {
				// discard the entry
				log.Printf("discard entry key: %v", key)
				return nil
			}
			return errors.Wrapf(err, "Unable to index %v from index engine", key)
		}

		if currvp.Fid > vp.Fid {
			// there is a new value after this one, discard it.
			return nil
		} else if currvp.Fid == vp.Fid {
			if currvp.Offset > vp.Offset {
				// discard it
				return nil
			}
			// FIXME: rewrite this one.
			log.Printf("rewrite entry key: %v", key)
		}
		// FIXME: currence writing ?
		// discard it
		return nil
	})
	if err != nil {
		log.Printf("value store run gc in log file %q fail. %v", lf.path, err)
		return errors.Wrapf(err, "Unable to gc file %q", lf.path)
	}

	return nil
}

var (
	ErrGCRunning error = errors.New("gc running")
)

// GC
func (vs *ValueStore) RunGC(head valuePointer, ratio float64) error {
	log.Printf("value store ready to run gc head: %#v ratio: %f", head, ratio)
	select {
	case vs.runGC <- struct{}{}:
		log.Printf("value store running gc head: %#v ratio: %f", head, ratio)

		start := time.Now()
		defer func() {
			log.Printf("value store gc done. time used: %s files: %#v", time.Now().Sub(start), vs.filesToBeDeleted)
			// release the access
			<-vs.runGC
		}()

		lfs, err := vs.pickLogFiles(head, ratio)
		if err != nil {
			return errors.Wrapf(err, "Unable to pick gc log files in %#v", head)
		}
		for i := range lfs {
			lf := lfs[i]
			log.Printf("value store gc file log %q", lf.path)
			err := vs.runGCLogFile(lf, ratio)
			if err != nil {
				return errors.Wrapf(err, "Unable to gc log file %q", lf.path)
			}
			vs.filesToBeDeleted = append(vs.filesToBeDeleted, lf.fid)
		}

	default:
		log.Printf("value store already one gc running")
		return ErrGCRunning
	}
	return nil
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
	// FIXME: maybe it's no necessary to reopen the file readonly, just
	// keep it read-write.
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
