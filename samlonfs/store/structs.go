package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

var CastagnoliTable = crc32.MakeTable(crc32.Castagnoli)

const (
	headerSize int = 12
)

type valuePointer struct {
	Fid    uint32
	Len    uint32
	Offset uint32
}

func (vp valuePointer) String() string {
	return fmt.Sprintf("value pointer fid: %d len: %d offset: %d", vp.Fid, vp.Len, vp.Offset)
}

type header struct {
	bid  uint64
	dlen uint32
}

func (h *header) Encode(out []byte) {
	binary.BigEndian.PutUint64(out[:8], h.bid)
	binary.BigEndian.PutUint32(out[8:12], h.dlen)
}

func (h *header) Decode(in []byte) {
	h.bid = binary.BigEndian.Uint64(in[:8])
	h.dlen = binary.BigEndian.Uint32(in[8:12])
}

type Entry struct {
	BId  uint64
	Data []byte
}

func encodeEntry(e *Entry, buf *bytes.Buffer) (n uint32, err error) {
	h := header{
		bid:  e.BId,
		dlen: uint32(len(e.Data)),
	}

	hash := crc32.New(CastagnoliTable)

	var head [headerSize]byte
	h.Encode(head[:])
	buf.Write(head[:])
	hash.Write(head[:])

	buf.Write(e.Data[:])
	hash.Write(e.Data[:])

	// crc32
	var crc [crc32.Size]byte
	binary.BigEndian.PutUint32(crc[:], hash.Sum32())
	buf.Write(crc[:])

	return uint32(len(head) + len(e.Data) + len(crc)), nil
}
