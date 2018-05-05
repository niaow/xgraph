package xgraph

import (
	"io"
	"runtime"
	"sync"
)

// WorkRunner is a type
type WorkRunner interface {
	// Do calls a func asynchronously.
	DoTask(task Task, tracker WorkTracker)

	// WorkRunner is closable - Close should clean up all existing state.
	io.Closer
}

// Task is a task function.
type Task func() error

// Run runs a task on the local goroutine, using the tracker to notify of completion.
func (t Task) Run(tracker WorkTracker) {
	tracker.OnComplete(t())
}

// poolWorkRunner is a WorkRunner implementation which uses a pool of goroutines.
type poolWorkRunner struct {
	//jobch is a channel where jobs can be sent
	//all goroutines should stop after this channel is closed
	workch chan work

	//stop is a WaitGroup which waits on all the goroutines
	//goroutines will release the WaitGroup when they shut down
	stop sync.WaitGroup
}

// NewWorkPool returns a WorkRunner that uses a fixed pool of goroutines.
// parallel is the number of goroutines to use in the pool.
// If parallel is 0, then one goroutine will be used per CPU.
func NewWorkPool(parallel uint16) WorkRunner {
	if parallel == 0 {
		parallel = uint16(runtime.NumCPU())
	}
	pwr := &poolWorkRunner{
		workch: make(chan work),
	}
	pwr.stop.Add(int(parallel))
	for parallel != 0 {
		go pwr.worker()
		parallel--
	}
	return pwr
}

// work is a container holding a task-tracker pair.
type work struct {
	task    Task
	tracker WorkTracker
}

// worker is run in a goroutine to do work from the queue.
// decrements the WaitGroup when it finishes.
func (pwr *poolWorkRunner) worker() {
	defer pwr.stop.Done()
	for work := range pwr.workch {
		work.task.Run(work.tracker)
	}
}

func (pwr *poolWorkRunner) DoTask(task Task, tracker WorkTracker) {
	pwr.workch <- work{
		task:    task,
		tracker: tracker,
	}
}

func (pwr *poolWorkRunner) Close() error {
	close(pwr.workch)
	pwr.stop.Wait()
	return nil
}

// WorkTracker is an interface used to track completion of jobs.
type WorkTracker interface {
	// OnComplete is called when a task completes.
	// The error value from the job is passed as an argument.
	OnComplete(err error)
}

// CallbackTracker is a WorkTracker which calls a function on completion.
type CallbackTracker func(err error)

// OnComplete calls the callback and implements WorkTracker.
func (ct CallbackTracker) OnComplete(err error) {
	ct(err)
}
