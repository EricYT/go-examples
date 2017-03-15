package lessor

import (
	"testing"
	"time"
)

const (
	defaultHeartbeat time.Duration = time.Second * 3
	defaultTTL       time.Duration = time.Second * 5
)

func TestLessorExpired(t *testing.T) {
	lessor := NewLeasor(time.Second, time.Second*3)
	i1 := &LeaseItem{Client: "127.0.0.1", Host: "192.168.0.2", MountPoint: "/test"}
	_, err := lessor.Grant(i1)
	if err != nil {
		t.Fatalf("grant t1 should not return error: %s", err)
	}

	select {
	case <-time.After(time.Second * 5):
		_, err = lessor.Lookup(i1)
		if err == nil {
			t.Fatalf("t1 should expired")
		}
	}
}

func TestLessorRefresh(t *testing.T) {
	lessor := NewLeasor(time.Second, time.Second*3)
	i1 := &LeaseItem{Client: "127.0.0.1", Host: "192.168.0.2", MountPoint: "/test"}
	_, err := lessor.Grant(i1)
	if err != nil {
		t.Fatalf("grant t1 should not return error: %s", err)
	}

	time.AfterFunc(time.Second*2, func() {
		err := lessor.Renew(i1)
		if err != nil {
			t.Fatalf("lease i1 should exists")
		}
	})

	select {
	case <-time.After(time.Second * 5):
		_, err = lessor.Lookup(i1)
		if err != nil {
			t.Fatalf("t1 should not expired")
		}
	}
}
