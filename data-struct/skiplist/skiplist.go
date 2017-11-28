package skiplist

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	MAX_LEVEL int = 10
)

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type Node struct {
	key     int
	value   int
	forward []*Node
}

func NewNode(level, key, value int) *Node {
	n := new(Node)
	n.key = key
	n.value = value
	n.forward = make([]*Node, level)
	return n
}

func (n *Node) Key() int         { return n.key }
func (n *Node) Value() int       { return n.value }
func (n *Node) Forward() []*Node { return n.forward }

type SkipList struct {
	level  int
	header *Node
}

func NewSkipList() *SkipList {
	sl := new(SkipList)
	sl.level = 0
	sl.header = NewNode(MAX_LEVEL, 0, 0)
	return sl
}

func (sl *SkipList) randomLevel() int {
	return random.Intn(MAX_LEVEL) + 1
}

func (sl *SkipList) Insert(key, value int) bool {
	var updates = make([]*Node, MAX_LEVEL)

	var p *Node
	var q *Node

	p = sl.header
	k := sl.level

	// find update nodes
	for i := k - 1; i >= 0; i-- {
		for {
			if q = p.forward[i]; q != nil && q.key < key {
				p = q
				continue
			}
			break
		}
		updates[i] = p
	}

	// can not insert same key
	if q != nil && q.key == key {
		return false
	}

	// random a level to insert
	k = sl.randomLevel()
	if k > sl.level {
		// generate a new level
		for i := sl.level; i < k; i++ {
			updates[i] = sl.header
		}
		sl.level = k
	}

	// insert new node
	node := NewNode(k, key, value)
	for i := 0; i < k; i++ {
		node.forward[i] = updates[i].forward[i]
		updates[i].forward[i] = node
	}

	return true
}

func (sl *SkipList) Delete(key int) bool {
	var updates = make([]*Node, MAX_LEVEL)

	var p *Node
	var q *Node
	p = sl.header
	k := sl.level

	// updates
	for i := k - 1; i >= 0; i-- {
		for {
			q = p.forward[i]
			if q != nil && q.key < key {
				p = q
				continue
			}
			break
		}
		updates[i] = p
	}

	if q != nil && q.key == key {
		for i := 0; i < k; i++ {
			if updates[i].forward[i] == q {
				updates[i].forward[i] = q.forward[i]
			}
			for i := k - 1; i >= 0; i-- {
				if sl.header.forward[i] == nil {
					sl.level--
				}
			}
		}
		return true
	}

	return false
}

func (sl *SkipList) Search(key int) (int, bool) {
	var p *Node
	var q *Node

	p = sl.header
	k := sl.level
	for i := k - 1; i >= 0; i-- {
		for {
			if q = p.forward[i]; q != nil && q.key <= key {
				if q.key == key {
					return q.value, true
				}
				p = q
			}
			break
		}
	}

	return -1, false
}

func (sl *SkipList) Display() {
	var p *Node
	var q *Node

	k := sl.level

	for i := k - 1; i >= 0; i-- {
		p = sl.header
		for {
			if q = p.forward[i]; q != nil {
				fmt.Printf("-> %d", q.value)
				p = q
				continue
			}
			break
		}
		fmt.Printf("\n")
	}
	fmt.Printf("\n")
}
