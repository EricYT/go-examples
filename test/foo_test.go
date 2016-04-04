package foo

import (
	"testing"
)

func Test_Division_1(t *testing.T) {
	if i, e := Division(6, 2); i != 3 || e != nil {
		t.Error("not pass")
	} else {
		t.Log("pass Division")
	}
}

func TestaDiv(t *testing.T) {
	t.Log("Pass anyway")
	//t.FailNow()
	//t.Error("not pass anyway")
}

func TestDiv(t *testing.T) {
	t.Error("Test function name")
}

func Benchmark_Division(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Division(4, 5)
	}
}
