package bar_test

import (
	"testing"

	"github.com/EricYT/go-examples/import/bar"
	"github.com/EricYT/go-examples/import/foo"
	"github.com/davecgh/go-spew/spew"
)

func TestBar(t *testing.T) {
	b := bar.Bar{
		Foo: foo.Foo{
			Key:   "foo",
			Value: "fooman",
		},
		WTF: "xxxx",
	}
	spew.Dump(b)
}
