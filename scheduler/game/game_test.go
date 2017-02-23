package game

import (
	"fmt"
	"sync"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestGameDirector(t *testing.T) {
	t.Logf("game main go ...")
	gd := NewGameDirector(50)
	gd.Start()
	defer gd.Stop()

	var wg sync.WaitGroup
	var games []Game
	for index := 1; index <= 200; index++ {
		g := NewSimpleGame(fmt.Sprintf("simpleGame#%d", index), func() chan error {
			var errc chan error = make(chan error)
			go func() {
				select {
				case <-time.After(time.Millisecond * 500):
					wg.Done()
					select {
					case errc <- nil:
					default:
					}
				case <-errc:
					return
				}
			}()
			return errc
		})
		games = append(games, g)
	}

	for _, g := range games {
		wg.Add(1)
		gd.Schedule(g)
	}

	time.Sleep(time.Second * 1)
	for _, g := range games {
		if g.Id() == "simpleGame#2" || g.Id() == "simpleGame#7" {
			err := gd.Cancel(g.Id())
			if err != nil {
				continue
			}
			wg.Done()
		}
	}
	wg.Wait()
	time.Sleep(time.Second)
	pending := gd.Pending()
	if pending != 0 {
		t.Fatalf("game director has not complete games %d", pending)
	}
	running := gd.Running()
	if running != 0 {
		t.Fatalf("game director has not finished games %d", running)
	}
}
