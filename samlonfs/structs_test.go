package samlonfs

import (
	"bytes"
	"hash/crc32"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestHeadEncode(t *testing.T) {
	ht1 := &header{klen: 1234, vlen: 4567}

	var head [headerSize]byte
	ht1.Encode(head[:])

	var ht2 = &header{}
	ht2.Decode(head[:])

	assert.Equal(t, *ht2, *ht1)
}

func BenchmarkHeadEncode(b *testing.B) {
	h := &header{klen: 1234, vlen: 4567}
	var head [headerSize]byte
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Encode(head[:])
	}
}

func BenchmarkHeadDecode(b *testing.B) {
	h := &header{klen: 1234, vlen: 4567}
	var head [headerSize]byte
	h.Encode(head[:])
	var box = &header{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		box.Decode(head[:])
	}
}

func TestEncodeEntry(t *testing.T) {
	// 4M
	data := make([]byte, 4*1024*1024)
	n, err := rand.Read(data)
	if assert.Nil(t, err) {
		assert.Equal(t, len(data), n)
	}
	hash := crc32.New(CastagnoliTable)
	hash.Write(data)
	dataCrc32 := hash.Sum32()

	// payload
	payload := make([]byte, 0, 12+4*1024*1024+4)
	pbuf := bytes.NewBuffer(payload)

	key := []byte("foo")
	e := &Entry{
		Key:   []byte("foo"),
		Value: data,
	}
	l, err := encodeEntry(e, pbuf)
	if assert.Nil(t, err) {
		assert.Equal(t, headerSize+len(key)+4*1024*1024+4, int(l))
	}

	// try to decode it
	var head header
	head.Decode(payload[:headerSize])
	assert.Equal(t, len(key), int(head.klen))
	assert.Equal(t, uint32(4*1024*1024), head.vlen)
	assert.Equal(t, data, payload[uint32(headerSize)+head.klen:uint32(headerSize)+head.klen+4*1024*1024])

	// hash
	h1 := crc32.New(CastagnoliTable)
	h1.Write(payload[uint32(headerSize)+head.klen : uint32(headerSize)+head.klen+4*1024*1024])
	assert.Equal(t, dataCrc32, h1.Sum32())
}

func BenchmarkEntryEncode(b *testing.B) {
	data := make([]byte, 4*1024*1024)
	rand.Read(data)
	// payload
	payload := make([]byte, 0, 12+4*1024*1024+4)
	pbuf := bytes.NewBuffer(payload)
	e := &Entry{
		Key:   []byte("foo"),
		Value: data,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodeEntry(e, pbuf)
		pbuf.Reset()
	}
}

func TestEncodeValuePointer(t *testing.T) {
	vp := valuePointer{Fid: 123, Len: 234, Offset: 7890}

	var vpbuf [valuePointerSize]byte
	vp.Encode(vpbuf[:])

	var vp1 valuePointer
	vp1.Decode(vpbuf[:])

	assert.Equal(t, vp, vp1)
}
