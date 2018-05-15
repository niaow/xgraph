package xgraph

import (
	"sync"
)

type dispatchTracker struct {
	// job is the Job
	job Job
	// notch is the channel to send notifications to
	notch chan notification
}

// OnComplete is the completion callback used for dispatching Jobs (implements WorkTracker)
func (dt *dispatchTracker) OnComplete(err error) {
	dt.notch <- notification{
		job:   dt.job,
		state: 2,
		err:   err,
	}
}

func (dt *dispatchTracker) task() error {
	dt.notch <- notification{
		job:   dt.job,
		state: 1,
	}
	err := dt.job.Run()
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
	dispatchch chan<- Job
	// bufch is a channel going to a goroutine which buffers jobs and relays them to runch
	bufch chan<- Job
	// notifych is a channel carrying notifications from the running jobs
	notifych chan notification
	// evh is the EventHandler being used to track this build
	evh EventHandler
	// proms is the set of promises for rules
	proms map[string]*Promise
	// cbset is the set of callbacks for Job completion
	cbset map[string]func(error)
}

// startDispatcher populates dispatchch and starts a goroutine which dispatches jobs
// stops on channel close and uses the WaitGroup
func (ex *executor) startDispatcher() {
	dispatch := make(chan Job)
	ex.wg.Add(1)
	go func() {
		defer ex.wg.Done()
		for j := range dispatch {
			dt := &dispatchTracker{
				job:   j,
				notch: ex.notifych,
			}
			ex.runner.DoTask(dt.task, dt)
		}
	}()
	ex.dispatchch = dispatch
}

// startDispatchBuffer starts a goroutine which buffers dispatches between bufch and dispatchch
func (ex *executor) startDispatchBuffer() {
	bufch := make(chan Job)
	ex.wg.Add(1)
	go func() {
		defer ex.wg.Done()
		buf := []Job{} //we dont care about order so just use a stack
		for {
			if len(buf) == 0 {
				j, ok := <-bufch
				if !ok {
					return
				}
				select {
				case ex.dispatchch <- j:
				default:
					buf = append(buf, j)
				}
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
	p := ex.proms[name]
	if p != nil {
		return p
	}
	jt := ex.forest[name]
	return NewPromise(func(s FinishHandler, f FailHandler) {
		//if there is a pre-existing error (e.g. dependency cycle), bail out
		if jt.err != nil {
			f(jt.err)
			return
		}
		//prep dep promise
		var dps *Promise
		if len(jt.deps) > 0 {
			depps := make([]*Promise, len(jt.deps))
			for i, v := range jt.deps {
				depps[i] = ex.promise(v.name)
			}
			dps = NewMultiPromise(depps...)
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

func (ex *executor) execute() {
	// start dispatcher/buffer
	defer ex.wg.Wait()
	ex.startDispatcher()
	defer close(ex.dispatchch)
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
