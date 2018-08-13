package locker

import (
	"sync"

	"github.com/pkg/errors"
)

var (
	ErrUnknowLockItemType error = errors.New("lock: unknow lock item type")
)

//FIXME: maybe we need a interface to remove the last one lock,
// nobody need it any more.

// locker
type LockManager interface {
	NewLockGroup(items ...LockItem) (locker Locker)
}

var _ LockManager = (*LockManagerService)(nil)

type LockManagerService struct {
	mutex sync.Mutex

	// items stored all locks in the tail for specific item keys.
	// If we want to require a lock for a item, put a waiting
	// lock chains to the tail and change the tail point to the
	// current one.
	items map[interface{}]Locker
}

func NewLockManagerService() *LockManagerService {
	lm := &LockManagerService{
		items: make(map[interface{}]Locker),
	}
	return lm
}

func (lm *LockManagerService) NewLockGroup(items ...LockItem) Locker {
	if err := ValidateLockItems(items); err != nil {
		// FIXME: A error is useless, because the user isn't interrupted.
		panic(err)
	}

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
		return lm.itemRLock(item)
	}
	// write lock
	if item.Type == LockTypeWrite {
		return lm.itemLock(item)
	}
	return nil
}

func (lm *LockManagerService) itemRLock(item LockItem) Locker {
	locker, ok := lm.items[item.Item]
	if !ok || (ok && !IsRLock(locker)) {
		locker := NewRLock(locker, item.Item)
		locker.Add()
		lm.items[item.Item] = locker
		return locker
	}
	type Adder interface {
		Add()
	}
	rlocker := locker.(Adder)
	// Notice: When the tail is rlock, we not pipe a new one, just
	// add a count for the tail.
	rlocker.Add()
	return locker
}

func (lm *LockManagerService) itemLock(item LockItem) Locker {
	l, _ := lm.items[item.Item]
	tailer := NewLock(l, item.Item)
	lm.items[item.Item] = tailer
	return tailer
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

func ValidateLockItems(items []LockItem) error {
	for _, item := range items {
		switch item.Type {
		case LockTypeRead, LockTypeWrite:
		default:
			return ErrUnknowLockItemType
		}
	}
	return nil
}
