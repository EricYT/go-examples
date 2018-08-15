package main

import (
	"log"
	"sync"
	"time"
)

type Foo struct {
	mu       sync.Mutex
	finished int

	finishCond *sync.Cond
}

func NewFoo() *Foo {
	f := &Foo{}
	f.finishCond = sync.NewCond(&f.mu)
	go f.run()
	return f
}

func (f *Foo) Wait(n int) {
	for f.finished < n {
		f.mu.Lock()
		f.finishCond.Wait()
		log.Println("foo wait finished ", f.finished, " now: ", time.Now())
		f.mu.Unlock()
	}
}

func (f *Foo) run() {
	log.Println("foo run ...")

	for {
		f.mu.Lock()
		f.finished++
		log.Println("foo run loop finished: ", f.finished, " now: ", time.Now())
		f.finishCond.Broadcast()
		f.mu.Unlock()
		time.Sleep(time.Second)
	}
}

func main() {
	log.Println("cond go ...")
	foo := NewFoo()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		foo.Wait(8)
	}()

	wg.Wait()
}
