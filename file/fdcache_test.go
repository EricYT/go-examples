package file

import (
	"container/list"
	"context"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFDCache(t *testing.T) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name := tmpfile.Name()
	defer os.Remove(name)

	fdc := NewFDCache(3)
	defer fdc.Reset()

	val := []byte("hello,world")
	n, err := fdc.WriteAt(context.TODO(), name, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	err = fdc.Sync(name)
	assert.Nil(t, err)

	buf := make([]byte, len(val))
	n1, err := fdc.ReadAt(context.TODO(), name, buf, 0)
	assert.Nil(t, err)
	assert.Equal(t, n, n1)
	assert.Equal(t, val, buf)
}

func TestFDCacheFileNotExist(t *testing.T) {
	fdc := NewFDCache(3)
	defer fdc.Reset()

	_, err := fdc.WriteAt(context.TODO(), "./jldkfjdlfj", nil, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "no such file or directory"))

	_, err = fdc.ReadAt(context.TODO(), "./jldkfjdlfj", nil, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "no such file or directory"))
}

func TestFDCacheSync(t *testing.T) {
	fdc := NewFDCache(1)
	defer fdc.Reset()

	err := fdc.Sync("./jdklfdkjfdl")
	assert.Nil(t, err)
}

func TestFDCacheClose(t *testing.T) {
	origOpenFile := _openFileFn
	_openFileFn = func(p string, f int, m os.FileMode) (*os.File, error) {
		return nil, nil
	}
	defer func() { _openFileFn = origOpenFile }()

	origNewFile := _newFileFn
	_newFileFn = newFakeFile
	defer func() { _newFileFn = origNewFile }()

	next := make(chan struct{})

	origStubCloseWait := stubCloseWait
	stubCloseWait = func(f func()) {
		go func() {
			next <- struct{}{}
			f()
		}()
	}
	defer func() { stubCloseWait = origStubCloseWait }()

	fdc := NewFDCache(1)
	_, err := fdc.ReadAt(context.TODO(), "test", nil, 0)
	assert.Nil(t, err)

	assert.Equal(t, "test", fdc.fds["test"].path())

	fdc.Close("test")

	assert.Equal(t, 0, len(fdc.fds))
	assert.Equal(t, 1, len(fdc.ghost))

	waitc := fdc.ghost["test"]

	<-next

	select {
	case <-waitc:
		assert.Equal(t, 0, len(fdc.ghost))
	default:
		assert.Fail(t, "wait nothing")
	}
}

func TestFDCacheTouch(t *testing.T) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name1 := tmpfile.Name()
	defer os.Remove(name1)

	fdc := NewFDCache(3)
	defer fdc.Reset()

	val := []byte("hello,world")
	n, err := fdc.WriteAt(context.TODO(), name1, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	f1 := fdc.lru.Front()
	assert.Equal(t, name1, f1.Value.(filer).path())

	tmpfile2, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name2 := tmpfile2.Name()
	defer os.Remove(name2)

	n, err = fdc.WriteAt(context.TODO(), name2, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	f2 := fdc.lru.Front()
	assert.Equal(t, name2, f2.Value.(filer).path())

	buf := make([]byte, len(val))
	n1, err := fdc.ReadAt(context.TODO(), name1, buf, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n1)
	assert.Equal(t, val, buf)

	f3 := fdc.lru.Front()
	assert.Equal(t, name1, f3.Value.(filer).path())
}

func TestFDCacheGhostFile(t *testing.T) {
	origOpenFile := _openFileFn
	_openFileFn = func(p string, f int, m os.FileMode) (*os.File, error) {
		return nil, nil
	}
	defer func() { _openFileFn = origOpenFile }()

	origNewFile := _newFileFn
	_newFileFn = newFakeFile
	defer func() { _newFileFn = origNewFile }()

	next := make(chan struct{})

	origStubCloseWait := stubCloseWait
	stubCloseWait = func(f func()) {
		go func() {
			next <- struct{}{}
			f()
		}()
	}
	defer func() { stubCloseWait = origStubCloseWait }()

	fdc := NewFDCache(1)
	_, err := fdc.ReadAt(context.TODO(), "test", nil, 0)
	assert.Nil(t, err)

	assert.Equal(t, "test", fdc.fds["test"].path())

	fdc.Close("test")

	ctx, cancel := context.WithCancel(context.TODO())
	cancel()
	_, err = fdc.ReadAt(ctx, "test", nil, 0)
	assert.Equal(t, context.Canceled, err)

	// ghost file
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// FIXME: damn worst design case.
		time.AfterFunc(time.Millisecond*1, func() { <-next })
		_, err := fdc.ReadAt(context.TODO(), "test", nil, 0)
		assert.Nil(t, err)
	}()

	wg.Wait()
}

func TestFDCacheEject(t *testing.T) {
	fdc := NewFDCache(2)
	defer fdc.Reset()
	val := []byte("hello,world")

	tmpfile, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name1 := tmpfile.Name()
	defer os.Remove(name1)

	n, err := fdc.WriteAt(context.TODO(), name1, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	tmpfile2, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name2 := tmpfile2.Name()
	defer os.Remove(name2)

	n, err = fdc.WriteAt(context.TODO(), name2, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	tmpfile3, err := ioutil.TempFile(os.TempDir(), "")
	assert.Nil(t, err)
	name3 := tmpfile3.Name()
	defer os.Remove(name3)

	n, err = fdc.WriteAt(context.TODO(), name3, val, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(val), n)

	assert.Equal(t, 2, len(fdc.fds))

	f1 := fdc.lru.Front()
	assert.Equal(t, name3, f1.Value.(filer).path())
	f2 := fdc.lru.Back()
	assert.Equal(t, name2, f2.Value.(filer).path())
}

// fake File for test
var (
	stubAdd       = func() {}
	stubDone      = func() {}
	stubReadAt    = func(b []byte, o int64) (n int, e error) { return }
	stubWriteAt   = func(b []byte, o int64) (n int, e error) { return }
	stubSync      = func() error { return nil }
	stubCloseWait = func(f func()) { f() }
)

type fakeFile struct {
	p string
	f *os.File
	e *list.Element
}

func newFakeFile(p string, f *os.File) filer {
	return &fakeFile{p: p, f: f}
}

func (f *fakeFile) path() string {
	return f.p
}

func (f *fakeFile) add(d int) {
	stubAdd()
}

func (f *fakeFile) done() {
	stubDone()
}

func (f *fakeFile) getEle() *list.Element {
	return f.e
}

func (f *fakeFile) setEle(e *list.Element) {
	f.e = e
}

func (f *fakeFile) ReadAt(b []byte, off int64) (n int, err error) {
	return stubReadAt(b, off)
}

func (f *fakeFile) WriteAt(b []byte, off int64) (n int, err error) {
	return stubWriteAt(b, off)
}

func (f *fakeFile) Sync() error {
	return stubSync()
}

func (f *fakeFile) CloseWait(fn func()) {
	stubCloseWait(fn)
}

func TestFDCacheConcurrencyControl(t *testing.T) {
	origOpenFile := _openFileFn
	_openFileFn = func(p string, f int, m os.FileMode) (*os.File, error) {
		return nil, nil
	}
	defer func() { _openFileFn = origOpenFile }()

	origNewFile := _newFileFn
	_newFileFn = newFakeFile
	defer func() { _newFileFn = origNewFile }()

	var wg sync.WaitGroup

	count := make(chan int, 1)
	origStubAdd := stubAdd
	stubAdd = func() {
		wg.Add(1)
	}
	defer func() { stubAdd = origStubAdd }()

	origStubDone := stubDone
	stubDone = func() {
		wg.Done()
	}
	defer func() { stubDone = origStubDone }()

	next := make(chan struct{})
	step := make(chan struct{})

	origStubReadAt := stubReadAt
	stubReadAt = func(b []byte, o int64) (int, error) {
		step <- struct{}{}
		next <- struct{}{}
		count <- int(o)
		return 0, nil
	}
	defer func() { stubReadAt = origStubReadAt }()

	var wgc sync.WaitGroup
	wgc.Add(3)

	origStubCloseWait := stubCloseWait
	stubCloseWait = func(f func()) {
		select {
		case <-count:
			t.Fatal("Coming early")
		default:
		}

		go func() {
			defer wgc.Done()
			backHoll := make(chan struct{}, 1)
			select {
			case backHoll <- func() struct{} {
				wg.Wait()
				select {
				case c := <-count:
					assert.Equal(t, 1, c)
				default:
					t.Fatal("Why you don't come.")
				}
				return struct{}{}
			}():
			case <-time.After(time.Second * 1):
				t.Fatalf("Coming a man, we don't want to see")
			}
		}()
	}
	defer func() { stubCloseWait = origStubCloseWait }()

	fdc := NewFDCache(1)

	// first one
	go func() {
		defer wgc.Done()
		_, err := fdc.ReadAt(context.TODO(), "first", nil, 1)
		assert.Nil(t, err)
	}()

	// second one
	go func() {
		defer wgc.Done()
		<-step
		_, err := fdc.WriteAt(context.TODO(), "second", nil, 2)
		assert.Nil(t, err)
		<-next
	}()

	wgc.Wait()
}
