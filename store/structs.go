package store

import (
	"encoding/binary"
	"fmt"
)

const blobPointerSize = 16

type BlobPointer struct {
	FileId uint32
	Length uint32
	Offset int64
}

func (b BlobPointer) Encode(buf []byte) {
	binary.BigEndian.PutUint32(buf[0:4], b.FileId)
	binary.BigEndian.PutUint32(buf[4:8], b.Length)
	binary.BigEndian.PutUint64(buf[8:16], uint64(b.Offset))
}

func (b *BlobPointer) Decode(buf []byte) {
	b.FileId = binary.BigEndian.Uint32(buf[0:4])
	b.Length = binary.BigEndian.Uint32(buf[4:8])
	b.Offset = int64(binary.BigEndian.Uint64(buf[8:16]))
}

func (b BlobPointer) String() string {
	return fmt.Sprintf("BlockFileId: %012d Length: %d Offset: %d", b.FileId, b.Length, b.Offset)
}
