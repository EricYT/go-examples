package mutex

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLeafInsert(t *testing.T) {
	l := NewLeaf("a")
	assert.False(t, l.InsertItem(nil))

	item, ok := l.FindItem(1)
	if assert.False(t, ok) {
		assert.Nil(t, item)
	}

	item = &Item{ID: 1}
	assert.True(t, l.InsertItem(item))

	item1, ok := l.FindItem(1)
	assert.True(t, ok)
	assert.Equal(t, item, item1)
}

func TestNodeInsert(t *testing.T) {
	n := NewNode()
	assert.False(t, n.InsertLeaf(nil))

	l := NewLeaf("a")
	assert.True(t, n.InsertLeaf(l))

	assert.True(t, l.InsertItem(&Item{ID: 1}))

	i, ok := n.FindItem(1)
	if assert.True(t, ok) {
		assert.Equal(t, int32(1), i.ID)
	}
}

func BenchmarkFindItem(b *testing.B) {
	b.ReportAllocs()

	n := NewNode()
	leafCount := 36
	itemCount := 102
	itemIndex := 0
	for i := 0; i < leafCount; i++ {
		// create leaf
		l := NewLeaf(fmt.Sprintf("leaf-%010d", i))
		// create items
		for d := 0; d < itemCount; d++ {
			ok := l.InsertItem(&Item{ID: int32(itemIndex)})
			if !assert.True(b, ok) {
				b.Fatalf("insert item failed")
			}
			itemIndex++
		}
		if !assert.True(b, n.InsertLeaf(l)) {
			b.Fatalf("insert leaf faield")
		}
	}

	b.ResetTimer()

	var index int32
	for i := 0; i < b.N; i++ {
		n.FindItem(index)
		index++
		if index >= int32(3672) {
			index = 0
		}
	}
}

func BenchmarkFindItemParallel(b *testing.B) {
	b.ReportAllocs()

	n := NewNode()
	leafCount := 36
	itemCount := 102
	itemIndex := 0
	for i := 0; i < leafCount; i++ {
		// create leaf
		l := NewLeaf(fmt.Sprintf("leaf-%010d", i))
		// create items
		for d := 0; d < itemCount; d++ {
			ok := l.InsertItem(&Item{ID: int32(itemIndex)})
			if !assert.True(b, ok) {
				b.Fatalf("insert item failed")
			}
			itemIndex++
		}
		if !assert.True(b, n.InsertLeaf(l)) {
			b.Fatalf("insert leaf faield")
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		var index int32
		for pb.Next() {
			n.FindItem(index)
			index++
			if index >= int32(3672) {
				index = 0
			}
		}
	})

}
