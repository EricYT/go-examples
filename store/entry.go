package store

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
)

var (
	CastagnoliCrcTable = crc32.MakeTable(crc32.Castagnoli)
)

//  data format
//  |--           header           --|--    meta   --|--   data    --|--  crc --|
//  |--64bits--|--32bits--|--32bits--|--dynamice-len-|--dynamic-len--|--32bits--|
//  |--blobId--|--blobLen-|--metaLen-|-- user-meta  -|--   data    --|--  crc --|

const (
	headerSize int = 16
)

type header struct {
	blobId      uint64
	blobLen     uint32
	userMetaLen uint32
}

func (h header) Encode(p []byte) {
	binary.BigEndian.PutUint64(p[0:8], h.blobId)
	binary.BigEndian.PutUint32(p[8:12], h.blobLen)
	binary.BigEndian.PutUint32(p[12:16], h.userMetaLen)
}

func (h *header) Decode(p []byte) {
	h.blobId = binary.BigEndian.Uint64(p[0:8])
	h.blobLen = binary.BigEndian.Uint32(p[8:12])
	h.userMetaLen = binary.BigEndian.Uint32(p[12:16])
}

func encodeRequest(r *Request, buf *bytes.Buffer) int {
	h := header{
		blobId:      uint64(r.BlobId),
		blobLen:     uint32(len(r.Blob)),
		userMetaLen: uint32(len(r.Meta)),
	}

	hash := crc32.New(CastagnoliCrcTable)

	// header
	var hbuf [headerSize]byte
	h.Encode(hbuf[:])
	buf.Write(hbuf[:])
	hash.Write(hbuf[:])

	// user metadata
	buf.Write(r.Meta)
	hash.Write(r.Meta)

	// data
	buf.Write(r.Blob)
	hash.Write(r.Blob)

	// crc32
	var crcbuf [crc32.Size]byte
	binary.BigEndian.PutUint32(crcbuf[:], hash.Sum32())
	buf.Write(crcbuf[:])

	return len(hbuf) + len(r.Blob) + len(r.Meta) + len(crcbuf)
}
