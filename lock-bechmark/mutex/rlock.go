package mutex

import "sync"

type Node struct {
	mu     sync.RWMutex
	leaves []*Leaf
}

func NewNode() *Node {
	return &Node{}
}

func (n *Node) InsertLeaf(l *Leaf) bool {
	if l == nil {
		return false
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	for _, leaf := range n.leaves {
		if l.key == leaf.key {
			return false
		}
	}
	n.leaves = append(n.leaves, l)
	return true
}

func (n *Node) FindItem(id int32) (*Item, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, leaf := range n.leaves {
		if item, ok := leaf.FindItem(id); ok {
			return item, true
		}
	}
	return nil, false
}

type Leaf struct {
	mu    sync.RWMutex
	key   string
	items map[int32]*Item
}

func NewLeaf(key string) *Leaf {
	return &Leaf{
		key:   key,
		items: make(map[int32]*Item),
	}
}

func (l *Leaf) InsertItem(i *Item) bool {
	if i == nil {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.items[i.ID]; ok {
		return false
	}
	l.items[i.ID] = i
	return true
}

func (l *Leaf) FindItem(id int32) (*Item, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if item, ok := l.items[id]; ok {
		return item, true
	}
	return nil, false
}

type Item struct {
	ID int32
}
