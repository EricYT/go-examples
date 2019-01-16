package store

type FileLoadingMode int

const (
	FileIO FileLoadingMode = iota
	MemoryMap
)

type Opts struct {
	LoadingMode         FileLoadingMode
	FileBlockMaxSize    int64
	FileBlockMaxEntries uint32
	SyncedFileIO        bool
}
