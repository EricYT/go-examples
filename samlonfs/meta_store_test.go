package samlonfs

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetaStore(t *testing.T) {
	tmpdir := path.Join(os.TempDir(), "meta_store")
	os.MkdirAll(tmpdir, 0755)
	defer os.RemoveAll(tmpdir)

	ms := NewMetaStore(tmpdir)
	err := ms.Open()
	if !assert.Nil(t, err) {
		return
	}
	defer func() {
		err := ms.Close()
		assert.Nil(t, err)
	}()

	for i := 0; i < 10; i++ {
		k := []byte(fmt.Sprintf("key#%d", i))
		v := []byte(fmt.Sprintf("value#%d", i))
		err := ms.Put(k, v)
		if !assert.Nil(t, err) {
			return
		}
	}

	// validate
	for i := 0; i < 10; i++ {
		k := []byte(fmt.Sprintf("key#%d", i))
		v := []byte(fmt.Sprintf("value#%d", i))

		v1, err := ms.Get(k)
		if !assert.Nil(t, err) {
			return
		}
		if !assert.Equal(t, v, v1) {
			return
		}
	}
}
