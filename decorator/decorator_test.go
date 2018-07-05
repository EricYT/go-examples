package decorator

import (
	"fmt"
	"testing"
)

func TestDecorator(t *testing.T) {
	var bar Foo = func(key string, value int) string {
		return fmt.Sprintf("Internal: %s:%d", key, value)
	}
	res := bar.Decorator("hello", 123)
	fmt.Println(res)
}
