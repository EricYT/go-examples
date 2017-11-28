package skiplist

import (
	"log"
	"testing"
)

func TestSkipList(t *testing.T) {
	sl := NewSkipList()

	for i := 1; i < 19; i++ {
		sl.Insert(i, i)
	}
	sl.Display()
	if i, ok := sl.Search(4); ok {
		log.Printf("search 4 index: %d\n", i)
	}
	if ok := sl.Delete(4); ok {
		log.Println("delete 4 success")
	}
	sl.Display()
}
