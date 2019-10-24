package ioqueue

type Mountpoint struct {
	MP             string
	ReadBytesRate  uint64
	WriteBytesRate uint64
	WriteReqRate   uint64
	ReadReqRate    uint64
	NumIOQueues    uint64
}
