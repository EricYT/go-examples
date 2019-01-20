package samlonfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

var CastagnoliTable = crc32.MakeTable(crc32.Castagnoli)

const (
	headerSize int = 8
)

type valuePointer struct {
	Fid    uint32
	Len    uint32
	Offset uint32
}

func (vp valuePointer) String() string {
	return fmt.Sprintf("value pointer fid: %d len: %d offset: %d", vp.Fid, vp.Len, vp.Offset)
}

// FIXME: right now, expired or some other meta not supported
type header struct {
	klen uint32
	vlen uint32
}

func (h *header) Encode(out []byte) {
	binary.BigEndian.PutUint32(out[:4], h.klen)
	binary.BigEndian.PutUint32(out[4:8], h.vlen)
}

func (h *header) Decode(in []byte) {
	h.klen = binary.BigEndian.Uint32(in[:4])
	h.vlen = binary.BigEndian.Uint32(in[4:8])
}

type Entry struct {
	Key   []byte
	Value []byte
}

func encodeEntry(e *Entry, buf *bytes.Buffer) (n uint32, err error) {
	h := header{
		klen: uint32(len(e.Key)),
		vlen: uint32(len(e.Value)),
	}

	hash := crc32.New(CastagnoliTable)

	var head [headerSize]byte
	h.Encode(head[:])
	buf.Write(head[:])
	hash.Write(head[:])

	buf.Write(e.Key)
	hash.Write(e.Key)
	buf.Write(e.Value)
	hash.Write(e.Value)

	// crc32
	var crc [crc32.Size]byte
	binary.BigEndian.PutUint32(crc[:], hash.Sum32())
	buf.Write(crc[:])

	return uint32(len(head) + len(e.Key) + len(e.Value) + len(crc)), nil
}
