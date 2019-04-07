package wal

import (
	"bufio"
	"encoding/binary"
	"hash/crc32"
	"io"

	"github.com/pkg/errors"
)

var (
	ErrCrcMismatch error = errors.New("crc mismatch")
)

const (
	prefixSize = 8
)

var (
	CastagnoliCrcTable = crc32.MakeTable(crc32.Castagnoli)
)

func NewEncoder(w io.Writer, size int) *Encoder {
	return &Encoder{w: bufio.NewWriterSize(w, size)}
}

type Encoder struct {
	w *bufio.Writer
}

func (e *Encoder) Encode(val []byte) (int, error) {
	hash := crc32.New(CastagnoliCrcTable)
	mw := io.MultiWriter(e.w, hash)

	var pbuf [prefixSize]byte
	binary.BigEndian.PutUint64(pbuf[:], uint64(len(val)))

	if _, err := mw.Write(pbuf[:]); err != nil {
		return 0, errors.Wrap(err, "failed to write value prefix size")
	}
	if _, err := mw.Write(val); err != nil {
		return 0, errors.Wrap(err, "failed to write value")
	}

	var crcbuf [crc32.Size]byte
	binary.BigEndian.PutUint32(crcbuf[0:4], hash.Sum32())

	if _, err := e.w.Write(crcbuf[:]); err != nil {
		return 0, errors.Wrap(err, "failed to write crc32")
	}

	if err := e.w.Flush(); err != nil {
		return 0, errors.Wrap(err, "failed to flush data")
	}

	return len(val) + prefixSize + crc32.Size, nil
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

type Decoder struct {
	r io.Reader
}

func (d *Decoder) Decode() ([]byte, error) {
	hash := crc32.New(CastagnoliCrcTable)
	r := io.TeeReader(d.r, hash)

	var pbuf [prefixSize]byte
	if _, err := io.ReadFull(r, pbuf[:]); err != nil {
		return nil, errors.Wrap(err, "failed to read prefix size")
	}
	size := binary.BigEndian.Uint64(pbuf[:])

	val := make([]byte, int(size))
	if _, err := io.ReadFull(r, val); err != nil {
		return nil, errors.Wrap(err, "failed to read value")
	}

	var crcbuf [crc32.Size]byte
	if _, err := io.ReadFull(d.r, crcbuf[:]); err != nil {
		return nil, errors.Wrap(err, "failed to read tail crc32")
	}
	if hash.Sum32() != binary.BigEndian.Uint32(crcbuf[0:4]) {
		return nil, ErrCrcMismatch
	}

	return val, nil
}
