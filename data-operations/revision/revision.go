package revision

import (
	"errors"
	"log"
	"sort"
)

var (
	ErrRevisionNotIntersect error = errors.New("revision: merge revision not intersect")
)

type Revision struct {
	vclock uint64
	off    uint64
	data   []byte
}

func NewRevision(v, o uint64, data []byte) *Revision {
	return &Revision{v, o, data}
}

func (r *Revision) Len() int      { return len(r.data) }
func (r *Revision) Begin() uint64 { return r.off }
func (r *Revision) End() uint64   { return r.off + uint64(r.Len()) - 1 }

func (r *Revision) Intersect(l *Revision) bool {
	return !(l.End() < r.Begin() || l.Begin() > r.End())
}

func (r *Revision) Merge(l *Revision) error {
	if !r.Intersect(l) {
		return ErrRevisionNotIntersect
	}
	correctBegin := min(r.Begin(), l.Begin())
	correctEnd := max(r.End(), l.End())
	mergeData := make([]byte, correctEnd-correctBegin+1)
	if r.vclock < l.vclock {
		r.vclock = l.vclock
		// merge r
		lOffInData := r.Begin() - correctBegin
		copy(mergeData[lOffInData:], r.data)
		// merge l
		lOffInData = l.Begin() - correctBegin
		copy(mergeData[lOffInData:], l.data)
	} else {
		// merge l
		lOffInData := l.Begin() - correctBegin
		copy(mergeData[lOffInData:], l.data)
		// merge r
		lOffInData = r.Begin() - correctBegin
		copy(mergeData[lOffInData:], r.data)
	}
	r.data = mergeData
	r.off = correctBegin
	return nil
}

type SparseRevisions struct {
	revisions []*Revision
}

func NewEmptySparseRevisions() *SparseRevisions {
	return &SparseRevisions{
		revisions: []*Revision{},
	}
}

func (s *SparseRevisions) Insert(r *Revision) *SparseRevisions {
	for index, revision := range s.revisions {
		if revision.Intersect(r) {
			log.Printf("revision merge index: %d r(%v) revision(%v)\n", index, r, revision)
			revision.Merge(r)
			s.revisions = append(s.revisions[0:index], s.revisions[index+1:]...)
			s.compaction(revision, index, r.Begin(), r.End())
			return s
		}
	}
	s.revisions = append(s.revisions, r)
	sort.Sort(Revisions(s.revisions))
	return s
}

func (s *SparseRevisions) compaction(rc *Revision, index int, begin, end uint64) {
	var revisions []*Revision
	for i, r := range s.revisions {
		if (i < index) || (r.End() < begin || r.Begin() > end) {
			revisions = append(revisions, r)
			continue
		}
		rc.Merge(r)
	}
	revisions = append(revisions, rc)
	sort.Sort(Revisions(revisions))
	s.revisions = revisions
}

// for revision sorted
type Revisions []*Revision

func (r Revisions) Len() int           { return len(r) }
func (r Revisions) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r Revisions) Less(i, j int) bool { return r[i].vclock < r[j].vclock }

func max(x, y uint64) uint64 {
	if x < y {
		return y
	}
	return x
}

func min(x, y uint64) uint64 {
	if x < y {
		return x
	}
	return y
}
