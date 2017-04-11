package scheduler

import (
	"errors"
	"log"
	"sync"

	tomb "gopkg.in/tomb.v1"
)

var (
	ErrSchedulerOverMaxPendings error = errors.New("scheduler: pendding jobs over max limit")
	ErrSchedulerShutdown        error = errors.New("scheduler: shutdown")
	ErrSchedulerReset           error = errors.New("scheduler: reset")
	ErrSchedulerCurrence        error = errors.New("schduler: currence number must greater than 0")
	ErrSchedulerWorkerDead      error = errors.New("scheduelr: worker ready to die")
)

type Job interface {
	Run() error
	Kill(err error)
	Wait() error
	Done()
}

type reactor struct {
	tomb  *tomb.Tomb
	mutex sync.Mutex

	currence    int
	maxPendings int

	jobsC    chan chan<- Job
	pendings []Job
	resume   chan struct{}
}

func NewReactor(currence int, maxPendings int) *reactor {
	if currence <= 0 {
		panic(ErrSchedulerCurrence)
	}
	r := &reactor{
		tomb:        new(tomb.Tomb),
		currence:    currence,
		maxPendings: maxPendings,
		jobsC:       make(chan chan<- Job, currence),
		pendings:    make([]Job, 0, 200),
		resume:      make(chan struct{}, 1),
	}
	// start workers
	for i := 0; i < currence; i++ {
		go r.workerLoop(r.jobsC)
	}

	// start main loop
	go func() {
		defer r.tomb.Done()
		r.tomb.Kill(r.runLoop())
	}()
	return r
}

func (r *reactor) AddJob(j Job) error {
	log.Printf("reactor: add job: %#v", j)
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if len(r.pendings) >= r.maxPendings {
		return ErrSchedulerOverMaxPendings
	}

	r.pendings = append(r.pendings, j)
	// maybe reset resume
	if len(r.pendings) == 1 {
		r.resume <- struct{}{}
	}
	return nil
}

func (r *reactor) Kill() {
	r.tomb.Kill(nil)
}

func (r *reactor) runLoop() error {
	log.Println("reactor(loop): main loop run")

	// select
	closed := make(chan struct{})
	close(closed)

	var next chan struct{}

	for {
		select {
		case <-r.resume:
			log.Println("reactor(loop): receive a resume signal")
			next = closed
		case <-next:
			// pop one job
			job, empty := r.popOne()
			if empty {
				next = nil
			} else {
				next = closed
			}

			// job run
			if job != nil {
				// this will block main loop if there is not a idle worker
				workerC := r.PickOneWorker()
				select {
				case workerC <- job:
				default:
					// run loop be killed, worker dead
					log.Printf("reactor(loop): dispatch job: %#v error", job)
					job.Kill(ErrSchedulerWorkerDead)
					job.Done()
				}
			}
		case <-r.tomb.Dying():
			log.Println("reactor(loop): ready to die")
			r.reset()
			return nil
		}
	}
	return nil
}

func (r *reactor) reset() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, job := range r.pendings {
		log.Printf("reactor(rest): kill job: %#v", job)
		job.Kill(ErrSchedulerReset)
		job.Done()
	}
	r.pendings = []Job{}
}

func (r *reactor) popOne() (Job, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if len(r.pendings) == 0 {
		return nil, true
	}
	job := r.pendings[0]
	r.pendings = r.pendings[1:]
	return job, len(r.pendings) == 0
}

func (r *reactor) PickOneWorker() chan<- Job {
	select {
	case <-r.tomb.Dying():
		return nil
	case jobC := <-r.jobsC:
		return jobC
	}
}

func (r *reactor) workerLoop(jobsC chan chan<- Job) {
	log.Println("reactor(worker): loop run")
	jobC := make(chan Job)
	for {
		jobsC <- jobC
		select {
		case <-r.tomb.Dying():
			log.Println("reactor(worker): worker done")
			return
		case job := <-jobC:
			log.Printf("reactor(worker): get job %#v", job)
			job.Kill(job.Run())
			job.Done()
		}
	}
}
