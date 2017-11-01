package scheduler

import (
	"context"
	"testing"
	"time"

	tomb "gopkg.in/tomb.v1"
)

func TestSchedulerDynamicRun(t *testing.T) {
	t.Log("TestSchedulerDynamicRun go")
	reactor := NewDynamicReactor(5, 3, 2)
	defer reactor.Kill()
	var jobs []*fakeJob
	for i := 0; i < 10; i++ {
		job := &fakeJob{id: i, tomb: new(tomb.Tomb)}
		jobFunc := func(ctx context.Context) {
			select {
			case <-ctx.Done():
				return
			default:
			}
			defer job.Done()
			job.Kill(job.Run())
		}
		jobWrapper := NewJobWrapper(jobFunc, func(err error) { defer job.Done(); job.Kill(err) })
		go func(j JobWrapper) {
			err := reactor.Schedule(j)
			if err != nil {
				j.Interrupt(err)
			}
		}(jobWrapper)
		jobs = append(jobs, job)
	}
	waitAllJobsDone(t, jobs, time.Second*30)
	t.Log("TestSchedulerDynamicRun over")
}

func TestSchedulerDynamicRunDone(t *testing.T) {
	t.Log("TestSchedulerDynamicRunDone go")
	reactor := NewDynamicReactor(5, 3, 2)
	defer reactor.Kill()
	var jobs []*fakeJob
	for i := 0; i < 10; i++ {
		job := &fakeJob{id: i, tomb: new(tomb.Tomb)}
		jobFunc := func(ctx context.Context) {
			select {
			case <-ctx.Done():
				return
			default:
			}
			defer job.Done()
			job.Kill(job.Run())
		}
		jobWrapper := NewJobWrapper(jobFunc, func(err error) { defer job.Done(); job.Kill(err) })
		go func(j JobWrapper) {
			err := reactor.Schedule(j)
			if err != nil {
				j.Interrupt(err)
			}
		}(jobWrapper)
		jobs = append(jobs, job)
	}
	waitAllJobsDone(t, jobs, time.Second*30)
	if !reactor.Idle() {
		t.Error("all jobs already done, reactor should be idle")
	}
	t.Log("TestSchedulerDynamicRunDone over")
}
