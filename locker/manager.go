package locker

import "sync"

// locker
type LockManager interface {
	LockItems(items ...LockItem) (locker Locker)
}

var _ LockManager = (*LockManagerService)(nil)

type LockManagerService struct {
	mutex sync.Mutex
	items map[interface{}]Locker
}

func NewLockManagerService() *LockManagerService {
	lm := &LockManagerService{
		items: make(map[interface{}]Locker),
	}
	return lm
}

func (lm *LockManagerService) LockItems(items ...LockItem) Locker {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	var lockers = make([]Locker, len(items))
	for i, item := range items {
		lockers[i] = lm.lockItem(item)
	}
	groupLocker := NewLockGroup(items, lockers)
	return groupLocker
}

func (lm *LockManagerService) lockItem(item LockItem) Locker {
	// read lock
	if item.Type == LockTypeRead {
		l, ok := lm.items[item.Item]
		if !ok || (ok && !IsRLock(l)) {
			locker := NewRLock(l, item.Item)
			lm.items[item.Item] = locker
			return locker
		}
		type Adder interface {
			Add()
		}
		rlocker := l.(Adder)
		rlocker.Add()
		return l
	}

	// write lock
	if item.Type == LockTypeWrite {
		l, _ := lm.items[item.Item]
		tailer := NewLock(l, item.Item)
		lm.items[item.Item] = tailer
		return tailer
	}

	panic("unknow lock type")
}

type LockItemType int8

const (
	LockTypeRead LockItemType = iota
	LockTypeWrite
)

type LockItem struct {
	Type LockItemType
	Item interface{}
}
