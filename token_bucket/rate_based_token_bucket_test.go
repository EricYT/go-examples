package token_bucket

import (
	"log"
	"sync"
	"testing"
	"time"
)

var index int

func adjustFunc() bool {
	log.Printf("index: %d\n", index)
	defer func() { index++ }()
	if index%5 != 0 {
		return true
	}
	return false
}

func TestTokenBucket(t *testing.T) {
	tb := NewRateBasedTokenBucket(5, 10, 1, 3, 5, time.Second*1, adjustFunc)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-time.After(time.Second * 1):
				now := time.Now()
				log.Printf("Time now: %s tokens: %d\n", time.Now(), 3)
				tokens := tb.Wait(3)
				log.Printf("Time escape: %s tokens: %d\n", time.Now().Sub(now), tokens)
			}
		}
	}()
	wg.Wait()
}
