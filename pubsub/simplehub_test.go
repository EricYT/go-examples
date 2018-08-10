package pubsub_test

import (
	"fmt"
	"log"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/EricYT/go-examples/pubsub"
)

const (
	topic pubsub.Topic = "test-topic"
)

func waitForMessagehandlingToBeComplete(done <-chan struct{}, t *testing.T) {
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("publish did not complete")
	}
}

func TestPublish(t *testing.T) {
	hub := pubsub.NewSimplehub()
	done := hub.Publish(topic, nil)
	waitForMessagehandlingToBeComplete(done, t)
}

func TestSubscribe(t *testing.T) {
	var called bool
	hub := pubsub.NewSimplehub()
	unsubscriber := hub.Subscribe(topic, func(topic_ pubsub.Topic, data interface{}) {
		if topic != topic_ {
			t.Fatalf("topic shuold equal %s", topic)
		}
		if data != nil {
			t.Fatalf("handler shuold receive a nil data")
		}
		called = true
	})
	t.Run("publish", func(t *testing.T) {
		done := hub.Publish(topic, nil)
		waitForMessagehandlingToBeComplete(done, t)
		if !called {
			t.Fatalf("publish handle not be called")
		}
	})
	t.Run("unsubscribe", func(t *testing.T) {
		called = false
		unsubscriber.Unsubscribe()
		done := hub.Publish(topic, nil)
		waitForMessagehandlingToBeComplete(done, t)
		if called {
			t.Fatalf("publish handle shouldn't be called again.")
		}
	})
}

func TestPublishCompleterWait(t *testing.T) {
	wait := make(chan struct{})
	hub := pubsub.NewSimplehub()
	hub.Subscribe(topic, func(topic pubsub.Topic, data interface{}) {
		<-wait
	})
	done := hub.Publish(topic, nil)

	select {
	case <-done:
		t.Fatalf("publish didn't wait")
	case <-time.After(time.Millisecond):
	}
	close(wait)
	waitForMessagehandlingToBeComplete(done, t)
}

func TestSubscriberExecInOrder(t *testing.T) {
	mutex := sync.Mutex{}
	hub := pubsub.NewSimplehub()
	var calls []pubsub.Topic
	hub.Subscribe(pubsub.MatchRegexp("test.*"), func(topic pubsub.Topic, data interface{}) {
		mutex.Lock()
		defer mutex.Unlock()
		calls = append(calls, topic)
	})

	var lastdone <-chan struct{}
	var rightCalls []pubsub.Topic
	for index := 0; index < 10; index++ {
		topc := pubsub.Topic(fmt.Sprintf("test.%d", index))
		rightCalls = append(rightCalls, topc)
		lastdone = hub.Publish(topc, nil)
	}
	waitForMessagehandlingToBeComplete(lastdone, t)
	if !reflect.DeepEqual(calls, rightCalls) {
		t.Fatalf("%#v != %#v", calls, rightCalls)
	}
}

func TestPublishNotBlockedByHandleFunc(t *testing.T) {
	wait := make(chan struct{})
	hub := pubsub.NewSimplehub()
	hub.Subscribe(topic, func(t pubsub.Topic, d interface{}) {
		<-wait
		log.Printf("subscriber receive data: %+v", d)
	})

	var lastdone <-chan struct{}
	for i := 0; i < 5; i++ {
		lastdone = hub.Publish(topic, i)
	}
	close(wait)
	waitForMessagehandlingToBeComplete(lastdone, t)
}
