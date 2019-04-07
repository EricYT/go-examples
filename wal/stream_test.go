package wal

import (
	"bytes"
	"crypto/rand"
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoder(t *testing.T) {
	buf := new(bytes.Buffer)
	encoder := NewEncoder(buf, 4*1024)

	val := make([]byte, 512)
	rand.Read(val)

	n, err := encoder.Encode(val)
	ensure(t, assert.Nil(t, err))
	ensure(t, assert.Equal(t, len(val)+prefixSize+crc32.Size, n))

	rbuf := bytes.NewBuffer(buf.Bytes())
	decoder := NewDecoder(rbuf)

	v1, err := decoder.Decode()
	ensure(t, assert.Nil(t, err))
	ensure(t, assert.Equal(t, val, v1))
}
