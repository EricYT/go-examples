package merkletree

type Entry struct {
	Value []byte
	Next  *Entry
}

type EntryList struct {
	ents []*Entry
}

func NewEntryList() *EntryList {
	return &EntryList{}
}

func (l *EntryList) Push(val []byte) *EntryList {
	entry := &Entry{Value: val}
	l.ents = append(l.ents, Entry{Value: val})
	l.ents[height-2].Next = &l.en
}
