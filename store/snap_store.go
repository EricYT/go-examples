package store

type SnapStore interface {
	Load() (ss []Snapshot, err error)
	Close() error

	SaveSnapshot(id int32, s Snapshot) error
	FetchSnapshot(id int32) (s Snapshot, err error)
}

type Snapshot struct {
}
