package main

import (
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

type Test struct {
	bar *Bar
}

func (t *Test) Call() {
	start := time.Now()
	t.bar.Call()
	log.Errorln("time call used:", time.Now().Sub(start))
}

type Bar struct {
	bar int64
}

func (b *Bar) Call() {
	b.bar++
}

func main() {
	bar := &Bar{}
	test := Test{bar: bar}

	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			test.Call()
			wg.Done()
		}()
	}

	wg.Wait()
}
