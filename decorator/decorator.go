package decorator

import (
	"fmt"
)

type Foo func(key string, value int) string

func (f Foo) Decorator(key string, value int) string {
	vtmp := f(key, value)
	return fmt.Sprintf("decorator: %s", vtmp)
}
