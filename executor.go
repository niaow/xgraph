package xgraph

import (
	"context"
	"sync"
)

type dispatchTracker struct {
	// job is the Job
	job Job
	// ctx is a context for running the Job
	ctx context.Context
	// notch is the channel to send notifications to
	notch chan notification
}

// OnComplete is the completion callback used for dispatching Jobs (implements WorkTracker)
func (dt *dispatchTracker) OnComplete(err error) {
	dt.notch <- notification{
		job:   dt.job,
		state: stateCompleted,
		err:   err,
	}
}

func (dt *dispatchTracker) task() error {
	dt.notch <- notification{
		job:   dt.job,
		state: stateStarted,
	}
	err := dt.job.Run(dt.ctx)
	return err
}

type notification struct {
	// job is the job that this notification is about
	job Job
	// stats is the state which this notification is reporting
	state int
	// err is the error (if applicable) from the run
	err error
}

const (
	stateStarted   = 1
	stateCompleted = 2
)

type executor struct {
	// forest is the jobtree we are using
	forest map[string]*jTree
	// runner is a WorkRunner used to run Jobs
	runner WorkRunner
	// wg is a sync.WaitGroup used to track shutdown of the executor
	wg sync.WaitGroup
	// dispatchch is a channel going to a goroutine which dispatches jobs
	dispatchch chan Job
	// bufch is a channel going to a goroutine which buffers jobs and relays them to runch
	bufch chan Job
	// notifych is a channel carrying notifications from the running jobs
	notifych chan notification
	// evh is the EventHandler being used to track this build
	evh EventHandler
	// proms is the set of promises for rules
	proms map[string]*Promise
	// cbset is the set of callbacks for Job completion
	cbset map[string]func(error)
	// ctx is the context used for execution (with cancel)
	ctx context.Context
}

// startDispatcher populates dispatchch and starts a goroutine which dispatches jobs
// stops on channel close and uses the WaitGroup
func (ex *executor) startDispatcher() {
	dispatch := ex.dispatchch
	ex.wg.Add(1)
	go func() {
		defer ex.wg.Done()
		ctxdone := ex.ctx.Done()
		for {
			select {
			case j, ok := <-dispatch:
				if !ok {
					return
				}
				dt := &dispatchTracker{
					job:   j,
					notch: ex.notifych,
					ctx:   ex.ctx,
				}
				ex.runner.DoTask(dt.task, dt)
			case <-ctxdone:
				for j := range dispatch { //drain dispatch buffer
					ex.notifych <- notification{ //tell controller that they were canceled
						job:   j,
						state: stateCompleted,
						err:   context.Canceled,
					}
				}
				return
			}
		}
	}()
	ex.dispatchch = dispatch
}

// startDispatchBuffer starts a goroutine which buffers dispatches between bufch and dispatchch
func (ex *executor) startDispatchBuffer() {
	bufch := ex.bufch
	ex.wg.Add(1)
	go func() {
		defer ex.wg.Done()
		defer close(ex.dispatchch)
		buf := []Job{} //we dont care about order so just use a stack
		for {
			if len(buf) == 0 {
				j, ok := <-bufch
				if !ok {
					return
				}
				buf = append(buf, j)
			} else {
				select {
				case j, ok := <-bufch:
					if !ok {
						return
					}
					buf = append(buf, j)
				case ex.dispatchch <- buf[len(buf)-1]:
					buf = buf[:len(buf)-1]
				}
			}
		}
	}()
	ex.bufch = bufch
}

// runJob places a job on the queue and returns a promise that resolves when the job completes
func (ex *executor) runJob(jt *jTree) *Promise {
	return NewPromise(func(s FinishHandler, f FailHandler) {
		ex.cbset[jt.name] = func(err error) {
			if err == nil {
				s()
			} else {
				f(err)
			}
		}
		ex.bufch <- jt.job
	})
}

// promise returns a promise that resolves when a given job finished building
func (ex *executor) promise(name string) *Promise {
	var p *Promise
	for p = ex.proms[name]; p == nil; p = ex.proms[name] {
		jt := ex.forest[name]
		ex.proms[name] = NewPromise(func(s FinishHandler, f FailHandler) {
			//if there is a pre-existing error (e.g. dependency cycle), bail out
			if jt.err != nil {
				f(jt.err)
				return
			}

			//prep dep promise
			var dps *Promise
			if len(jt.deps) > 0 {
				depps := make(map[string]*Promise)
				for _, v := range jt.deps {
					depps[v.name] = ex.promise(v.name)
				}
				dps = newBuildPromise(depps)
			} else {
				dps = NewPromise(func(s FinishHandler, f FailHandler) {
					s()
				})
			}

			//run dep promise
			dps.Then(
				func() { //on success, run build
					sr, err := jt.job.ShouldRun() //check if the job should run
					if err != nil {               //error out if we cant tell whether it should be run
						f(err)
					}
					if sr {
						ex.runJob(jt).Then(s, f)
					}
				},
				func(err error) {
					f(err)
				},
			)
		})
	}
	return p
}

func (ex *executor) execute() {
	// start dispatcher/buffer
	defer ex.wg.Wait()
	ex.startDispatcher()
	ex.startDispatchBuffer()
	defer close(ex.bufch)

	// start build promises
	n := len(ex.forest)
	for _, v := range ex.forest {
		name := v.name
		if v.err == nil { //if might be run, mark as queued
			ex.evh.OnQueued(name)
		}
		ex.promise(name).Then( //start promise
			func() {
				ex.evh.OnFinish(name)
				n--
			},
			func(err error) {
				ex.evh.OnError(name, err)
				n--
			},
		)
	}

	// do processing loop
	for n > 0 {
		not := <-ex.notifych
		switch not.state {
		case stateStarted:
			ex.evh.OnStart(not.job.Name())
		case stateCompleted:
			ex.cbset[not.job.Name()](not.err)
		}
	}

	//we are done!
}
