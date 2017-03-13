package reactor

import (
	"log"
	"sync"
	"time"

	tomb "gopkg.in/tomb.v1"
)

type Productor interface {
}

type nuclearBucket struct {
	tomb  *tomb.Tomb
	mutex sync.Mutex

	reactor   *reactor
	generator NuclearGenerator
	notify    chan struct{}

	fillUpRunning bool
	interval      time.Duration
	frequence     int
}

func NewNuclearBucket(reactor *reactor, generator NuclearGenerator, interval time.Duration, frequence int) *nuclearBucket {
	nb := &nuclearBucket{
		tomb:      new(tomb.Tomb),
		notify:    make(chan struct{}, 1),
		reactor:   reactor,
		generator: generator,
		interval:  interval,
		frequence: frequence,
	}
	go func() {
		defer nb.tomb.Done()
		nb.tomb.Kill(nb.fillUp())
	}()
	return nb
}

func (n *nuclearBucket) Kill() error {
	n.tomb.Kill(nil)
	return n.tomb.Wait()
}

func (n *nuclearBucket) fillUp() error {
	log.Printf("nuclear bucket: fill up run")

	closed := make(chan struct{})
	close(closed)
	var next chan struct{}

	ticker := time.NewTicker(n.interval)
	defer ticker.Stop()

	for {
		select {
		case <-n.tomb.Dying():
			log.Printf("nuclear bucket: shutdown")
			return nil
		case <-ticker.C:
			//TODO: fill up too quickly, ignore this time
			if next == nil {
				next = closed
			}
		case <-n.notify: //FIXME: maybe add a feedback control
			// consumer need more material
			//n.notifyFeedback()
			//next = closed
		case <-next:
			// fill up bucket
			n.fillUpBucket()
			next = nil
		}
	}
}

func (n *nuclearBucket) notifyFeedback() {
	// consumer give a feedback because productor product nuclear slowly
	// 1. increase n.frequence count;
	// 2. decrease n.interval
}

func (n *nuclearBucket) fillUpBucket() {
	n.mutex.Lock()
	if !n.fillUpRunning {
		n.fillUpRunning = true
		n.mutex.Unlock()
		// fillup
		go func() {
			defer func() {
				// reset the fille up status
				n.mutex.Lock()
				n.fillUpRunning = false
				n.mutex.Unlock()
			}()
			var count int = n.frequence
			for count > 0 {
				ns, err := n.generator.Next()
				if err != nil {
					log.Printf("nuclear bucket: generate nuclear error: %s", err)
					return
				}
				if len(ns) == 0 {
					//TODO: no more material can be producted
					return
				}
				for _, nuclear := range ns {
					err := n.reactor.AddNuclear(nuclear)
					switch err {
					case ErrorReactorCapacity:
						return
					default:
						count--
					}
				}
			}
		}()
		return
	}
	n.mutex.Unlock()
	return
}
