package store

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"sync"
	"sync/atomic"

	"git.jd.com/cloud-storage/newds-datanode/pkg/logutil"
	dsproto "git.jd.com/cloud-storage/newds-datanode/proto"
	"go.etcd.io/etcd/pkg/wait"
	"go.uber.org/zap"
)

const maxWriteBlobSize = 100
const blobStoreDir = "blob"

var (
	ErrStoreStopped error = errors.New("store stopped")
	ErrStaleBlob    error = errors.New("stale blob")
)

type Store struct {
	logger logutil.Logger

	root       string
	attachedWG sync.WaitGroup

	storeLock sync.RWMutex
	// meta store
	ms MetaStore
	// blob store
	bs BlobStore
	// snap store
	ss SnapStore
	// replica group
	rg ReplicaGroup

	maxBlobId     int64 // access atomic
	committedLock sync.Mutex
	committed     int64

	writeBlobCh chan *Request

	wait wait.Wait

	stopCh chan struct{}
}

func NewStore(logger logutil.Logger, root string) *Store {
	logger = logger.Named("store")
	logger = logger.With(zap.String("store-root", root))
	s := &Store{
		logger: logger,
		root:   root,

		writeBlobCh: make(chan *Request, maxWriteBlobSize),
		wait:        wait.New(),
		stopCh:      make(chan struct{}),
	}

	// meta store
	s.ms = NewMemoryMetaStore(logger)
	// blob store
	s.bs = NewBlobStore(logger, s.blobStorePath())
	// replica group
	s.rg = NewReplicaGroup(logger, s)
	// snapshot store

	return s
}

func (s *Store) blobStorePath() string {
	return path.Join(s.root, blobStoreDir)
}

func (s *Store) Load() error {
	s.logger.Info("store loading start.")

	// FIXME:
	if err := s.bs.Load(); err != nil {
		s.logger.Errorf("store loading blob store failed. %v", err)
		return err
	}

	if err := s.ms.Load(); err != nil {
		s.logger.Errorf("store loading meta store failed. %v", err)
		return err
	}

	s.goAttach(s.run)

	s.logger.Info("store loading stopped.")
	return nil
}

func (s *Store) Close() {
	s.logger.Info("store ready to stop.")

	s.storeLock.Lock()
	defer s.storeLock.Unlock()
	close(s.stopCh)

	// meta engine
	// blob engine
	// snapshot engine

	// wait all gorotines done.
	s.attachedWG.Wait()

	s.logger.Info("store stopped.")
}

func (s *Store) run() {
	s.logger.Info("store main loop run.")

	blockWriteCh := make(chan struct{}, 1)
	writeEntries := func(reqs []*Request) {
		if err := s.writeEntries(reqs); err != nil {
			s.logger.Errorf("store main loop write entries failed. %v", err)
		}
		<-blockWriteCh
	}

	reqs := make([]*Request, 0, 10)
	for {
		var r *Request
		select {
		// write request
		case r = <-s.writeBlobCh:
			// trying to merge more Requests
			for {
				reqs = append(reqs, r)

				if len(reqs) >= 3*maxWriteBlobSize {
					blockWriteCh <- struct{}{}
					goto writeCase
				}

				select {
				case r = <-s.writeBlobCh:
				case blockWriteCh <- struct{}{}:
					goto writeCase
				case <-s.stopCh:
					goto closeCase
				}
			}

		case <-s.stopCh:
			s.logger.Info("store main loop stopped.")
			return
		}

	closeCase:
		s.logger.Error("store main loop closed.")
		for _, req := range reqs {
			s.wait.Trigger(req.reqId, ErrStoreStopped)
		}
		return
	writeCase:
		go writeEntries(reqs)
		reqs = make([]*Request, 0, 10)
	}
}

func (s *Store) nextBlobId() int64 {
	return atomic.AddInt64(&s.maxBlobId, 1)
}

func (s *Store) writeEntries(reqs []*Request) error {
	if len(reqs) == 0 {
		return nil
	}

	// generate blob id
	for _, req := range reqs {
		req.BlobId = s.nextBlobId()
	}
	// merge writes
	if err := s.bs.Write(reqs); err != nil {
		// all failed
		for _, req := range reqs {
			s.wait.Trigger(req.reqId, err)
		}
		s.logger.Errorf("store blob store write request failed. %v", err)
		return err
	}
	// propose
	for _, req := range reqs {
		msg := convertPutBlobMessage(req)
		if err := s.rg.Propose(context.TODO(), msg); err != nil {
			s.wait.Trigger(req.reqId, nil)
		}
	}
	return nil
}

type BlobMeta struct {
	BlobId int64       `json:"blob_id"`
	Meta   []byte      `json:"meta"`
	Crc    []byte      `json:"crc"`
	Ptr    BlobPointer `json:"ptr"`
}

func (s *Store) commitPut(reqId int64, blobId int64, meta, crc []byte, ptr BlobPointer) {
	s.logger.Debugf("store commit blob %d ptr: %s", blobId, ptr)

	// FIXME: id compared. In case staled blob committed
	s.committedLock.Lock()
	committed := s.committed
	if blobId < committed {
		s.committedLock.Unlock()
		s.logger.Errorf("store commit blob %d staled. current max blob id %d", blobId, committed)
		s.wait.Trigger(uint64(reqId), ErrStaleBlob)
		return
	}
	if blobId > committed {
		s.committed = blobId
	}
	s.committedLock.Unlock()

	bm := BlobMeta{BlobId: blobId, Meta: meta, Crc: crc, Ptr: ptr}
	blobMeta, _ := json.Marshal(&bm)

	if err := s.ms.Put(blobId, blobMeta); err != nil {
		s.logger.Errorf("store blob id %d meta %s failed. %v", blobId, meta, err)
		s.wait.Trigger(uint64(reqId), err)
		return
	}
	s.wait.Trigger(uint64(reqId), nil)
	return
}

func (s *Store) commitDelete(reqId int64, blobId int64) {
	s.logger.Debugf("store commit delete blob %d", blobId)

	if err := s.ms.Delete(blobId); err != nil {
		s.logger.Errorf("store blob %d meta delete failed. %v", blobId, err)
		s.wait.Trigger(uint64(reqId), err)
		return
	}
	s.wait.Trigger(uint64(reqId), nil)
	return
}

func convertPutBlobMessage(req *Request) *dsproto.ReplicaMessage {
	return &dsproto.ReplicaMessage{
		Type: dsproto.ReplicaMessage_PUT,
		Msg: &dsproto.ReplicaMessage_Put{
			Put: &dsproto.PutBlob{
				ReqId:  int64(req.reqId),
				BlobId: req.BlobId,
				Meta:   req.Meta,
				Crc:    req.Crc,
				Ptr: &dsproto.BlobPointer{
					BlockId: int32(req.Ptr.FileId),
					Len:     int32(req.Ptr.Length),
					Offset:  req.Ptr.Offset,
				},
			},
		},
	}
}

type Request struct {
	BlobId int64
	Blob   []byte
	Meta   []byte
	Crc    []byte
	Ptr    BlobPointer

	// internal done
	reqId uint64
	ErrCh <-chan interface{}
}

var (
	_reqId uint64
)

func (s *Store) Put(logger logutil.Logger, blob, meta, crc []byte) (req *Request, err error) {
	logger.Debugf("store put blob start.")

	reqId := atomic.AddUint64(&_reqId, 1)
	req = &Request{
		Blob:  blob,
		Meta:  meta,
		Crc:   crc,
		reqId: reqId,
		ErrCh: s.wait.Register(reqId),
	}

	select {
	case s.writeBlobCh <- req:
	case <-s.stopCh:
		return nil, ErrStoreStopped
	}
	return req, nil
}

// replica call back
func (s *Store) Commit(ctx context.Context, msg *dsproto.ReplicaMessage) error {
	s.logger.Debugf("store commit message type %s msg: %s", msg.Type, msg.String())

	switch msg.GetType() {
	case dsproto.ReplicaMessage_PUT:
		pb := msg.GetPut()
		if pb == nil {
			return nil
		}
		ptr := BlobPointer{FileId: uint32(pb.Ptr.BlockId), Length: uint32(pb.Ptr.Len), Offset: pb.Ptr.Offset}
		s.commitPut(pb.ReqId, pb.BlobId, pb.Meta, pb.Crc, ptr)
	case dsproto.ReplicaMessage_DELETE:
		db := msg.GetDel()
		if db == nil {
			return nil
		}
		s.commitDelete(db.ReqId, db.BlobId)
	default:
		s.logger.Errorf("store commit receive unknow message type %s", msg.GetType())
	}

	return nil
}

// NOTICE: If replica group implemented by Raft, so
// when this replica became leader and commited all pending
// requests, there is a callback below.
func (s *Store) ReplicaGroupReady() {
	s.logger.Info("store replica group ready")
	s.committedLock.Lock()
	defer s.committedLock.Unlock()
	atomic.StoreInt64(&s.maxBlobId, s.committed)
}

func (s *Store) Get(logger logutil.Logger, blobId int64) (blob []byte, err error) {
	s.logger.Debugf("store get blob %d", blobId)

	meta, err := s.ms.Get(blobId)
	if err != nil {
		s.logger.Errorf("store get blob %d failed. %v", blobId, err)
		return nil, err
	}

	var ms BlobMeta
	json.Unmarshal(meta, &ms)

	blob, err = s.bs.Read(ms.Ptr)
	if err != nil {
		s.logger.Errorf("Unable to read blob %d ptr %s. %v", blobId, ms.Ptr, err)
		return nil, err
	}

	return blob, nil
}

type DeleteRequest struct {
	BlobId int64

	ErrCh <-chan interface{}
}

func (s *Store) Delete(logger logutil.Logger, blobId int64) (req *DeleteRequest, err error) {
	s.logger.Debugf("store delete blob %d", blobId)

	reqId := atomic.AddUint64(&_reqId, 1)

	msg := &dsproto.ReplicaMessage{
		Type: dsproto.ReplicaMessage_DELETE,
		Msg: &dsproto.ReplicaMessage_Del{
			Del: &dsproto.DeleteBlob{
				ReqId:  int64(reqId),
				BlobId: blobId,
			},
		},
	}

	if s.rg.Propose(context.TODO(), msg); err != nil {
		s.logger.Errorf("store delete blob %d propose failed. %v", blobId, err)
		return nil, err
	}
	req = &DeleteRequest{BlobId: blobId}
	req.ErrCh = s.wait.Register(reqId)

	return req, nil
}

// put blob directly

// get blob directly

func (s *Store) goAttach(f func()) {
	s.attachedWG.Add(1)
	go func() {
		defer s.attachedWG.Done()
		f()
	}()
}
