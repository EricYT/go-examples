package foo_test

import (
	"testing"

	"github.com/EricYT/go-examples/import/bar" // imports package foo
	//"github.com/EricYT/go-examples/import/foo"
	. "github.com/EricYT/go-examples/import/foo"

	"github.com/davecgh/go-spew/spew"
)

func TestFoo(t *testing.T) {
	f := Foo{Key: "hello", Value: "world"}

	spew.Dump(f)

	b := bar.Bar{
		Foo: f,
		WTF: "ok",
	}
	spew.Dump(b)
}
