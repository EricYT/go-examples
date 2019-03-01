package store

import (
	"context"

	"git.jd.com/cloud-storage/newds-datanode/pkg/logutil"
	dsproto "git.jd.com/cloud-storage/newds-datanode/proto"
)

type ReplicaGroup interface {
	Propose(ctx context.Context, msg *dsproto.ReplicaMessage) error
}

func NewReplicaGroup(logger logutil.Logger, s *Store) ReplicaGroup {
	rg := &fakeReplicaGroup{s: s}
	go rg.run()
	// Leader ok
	s.ReplicaGroupReady()
	return rg
}

// fake one
type fakeReplicaGroup struct {
	s   *Store
	msg chan *dsproto.ReplicaMessage
}

func (f *fakeReplicaGroup) run() {
	f.msg = make(chan *dsproto.ReplicaMessage, 10)
	for {
		select {
		case msg := <-f.msg:
			f.s.Commit(context.TODO(), msg)
		}
	}
}

func (f *fakeReplicaGroup) Propose(ctx context.Context, msg *dsproto.ReplicaMessage) error {
	f.msg <- msg
	return nil
}
