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

	validateLockItems(items)
	var lockers = make([]Locker, len(items))
	for i, item := range items {
		lockers[i] = lm.lockItem(item)
	}
	return NewLockGroup(items, lockers)
}

func (lm *LockManagerService) lockItem(item LockItem) Locker {
	l, ok := lm.items[item.Item]
	if item.Type == LockTypeRead {
		// read lock
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
	} else if item.Type == LockTypeWrite {
		// write lock
		tailer := NewLock(l, item.Item)
		lm.items[item.Item] = tailer
		return tailer
	}
	return nil
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

func validateLockItems(items []LockItem) {
	for _, item := range items {
		switch item.Type {
		case LockTypeRead:
		case LockTypeWrite:
		default:
			panic("unknow lock type")
		}
	}
}
