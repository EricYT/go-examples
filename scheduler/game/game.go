package game

import (
	"errors"
	"sync"
	"time"

	"github.com/EricYT/go-examples/scheduler/runner"
	log "github.com/Sirupsen/logrus"

	tomb "gopkg.in/tomb.v1"
)

var glog = log.WithFields(log.Fields{
	"module": "game",
})

// errors
var (
	ErrorGameDirectorNotFound error = errors.New("game director: game not found")
)

// game scheduler
type GameDirector interface {
	// Schedule schedule the game to director
	Schedule(g Game)
	// Cancel stop the running game
	Cancel(id string) error
	// Pending return the number of pending games in director
	Pending() int
	// Running return the number of running games in director
	Running() int
	// Fetch get the entry of running game
	//	Fetch(id string) (Game, error)
	// Start start the game director
	Start()
	// Stop stop the game director
	Stop()
}

type Game interface {
	Id() string
	Kill()
	Wait() error
	Run() error
}

type gameDirector struct {
	tomb *tomb.Tomb

	mu      sync.Mutex
	pending []Game
	running map[string]Game
	runner  runner.Runner
	resume  chan struct{}
}

func NewGameDirector(currence int) GameDirector {
	g := &gameDirector{
		tomb:    new(tomb.Tomb),
		pending: []Game{},
		running: make(map[string]Game),
		resume:  make(chan struct{}, currence),
	}
	g.runner = runner.NewRunner(isFalt, moreImportant, time.Second*30)
	return g
}

func (g *gameDirector) Start() {
	go func() {
		defer g.tomb.Done()
		g.tomb.Kill(g.runLoop())
	}()
}

func (g *gameDirector) Stop() {
	g.tomb.Kill(nil)
}

func (g *gameDirector) Schedule(game Game) {
	g.mu.Lock()
	g.pending = append(g.pending, game)
	g.mu.Unlock()

	// wakeup the main loop maybe
	select {
	case g.resume <- struct{}{}:
	default:
	}
}

func (g *gameDirector) Pending() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.pending)
}

func (g *gameDirector) Running() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.running)
}

func (g *gameDirector) Cancel(id string) error {
	g.mu.Lock()

	for index, game := range g.pending {
		if game.Id() != id {
			continue
		}
		head := g.pending[0:index]
		tail := g.pending[index+1:]
		g.pending = append(head, tail...)
		g.mu.Unlock()
		return nil
	}

	if _, ok := g.running[id]; ok {
		err := g.runner.StopWorker(id)
		if err != nil {
			glog.Errorf("game director cancel game: %s error: %s", id, err)
			// FIXME: delete it ?
			g.mu.Unlock()
			return err
		}
		g.mu.Unlock()
		return nil
	}
	g.mu.Unlock()
	return ErrorGameDirectorNotFound
}

func (g *gameDirector) runLoop() error {
	glog.Debugln("game director run loop ...")

	for {
		select {
		case <-g.resume:
			var todo Game
			g.mu.Lock()
			if len(g.pending) != 0 {
				todo = g.pending[0]
				g.runWorker(todo)
				g.running[todo.Id()] = todo
				g.pending = g.pending[1:]
			}
			g.mu.Unlock()
		case <-g.tomb.Dying():
			log.Debugln("game director shutdown")
			// FIXME: shutdown the running games and remove remains games
			g.mu.Lock()
			for id, _ := range g.running {
				g.runner.StopWorker(id)
			}
			g.mu.Unlock()
			return nil
		}
	}
}

func (g *gameDirector) runWorker(game Game) {
	workerFunc := func() (runner.Worker, error) {
		go func() {
			defer func() {
				g.mu.Lock()
				delete(g.running, game.Id())
				select {
				case g.resume <- struct{}{}:
				default:
				}
				g.mu.Unlock()
			}()
			game.Run()
		}()
		return game, nil
	}
	g.runner.StartWorker(game.Id(), workerFunc)
}

// runner function
func isFalt(err error) bool {
	return false
}

func moreImportant(err0, err1 error) bool {
	return false
}
