package xgraph

import (
	"sync"
)

// Promise is a future
type Promise struct {
	fun               func(FinishHandler, FailHandler)
	started, finished bool
	err               error
	sfs               []FinishHandler
	efs               []FailHandler
}

// Then registers callbacks and starts the Promise if it has not been already started.
// If the Promise has already completed, the success/failure handler is immediately called with the resu;t.
// This is like Promise.then in JavaScript.
func (p *Promise) Then(success FinishHandler, failure FailHandler) {
	if p.finished {
		if p.err == nil {
			if success != nil {
				success()
			}
		} else {
			if failure != nil {
				failure(p.err)
			}
		}
	} else {
		if success != nil {
			p.sfs = append(p.sfs, success)
		}
		if failure != nil {
			p.efs = append(p.efs, failure)
		}
		if !p.started {
			p.fun(p.onFinish, p.onFail)
			p.started = true
		}
	}
}

// onFinish is the FinishHandler passed to the promise function
func (p *Promise) onFinish() {
	p.finished = true
	for _, v := range p.sfs {
		v()
	}
	//save memory
	p.sfs = nil
	p.efs = nil
}

// onFail is the FailHandler passed to the promise function
func (p *Promise) onFail(err error) {
	p.finished = true
	p.err = err
	for _, v := range p.efs {
		v(err)
	}
	//save memory
	p.sfs = nil
	p.efs = nil
}

//FinishHandler is a type of function used as a callback for a Promise on success
type FinishHandler func()

//FailHandler is a type of function used as a callback for a Promise on failure
type FailHandler func(error)

//NewPromise returns a *Promise using the given function
func NewPromise(fun func(FinishHandler, FailHandler)) *Promise {
	return &Promise{
		fun: fun,
		sfs: []FinishHandler{},
		efs: []FailHandler{},
	}
}

// NewMultiPromise returns a Promise which is fullfilled when all of the promises are fullfilled.
// If one promise fails, this error will be propogated.
// If multiple promises fail, a MultiError will be sent to the FailHandler instead.
func NewMultiPromise(p ...*Promise) *Promise {
	return NewPromise(func(s FinishHandler, f FailHandler) {
		var lck sync.Mutex
		n := len(p)
		errs := []error{}
		meh := func() {
			if n == 0 {
				if len(errs) == 1 {
					f(errs[0])
				} else if len(errs) > 1 {
					f(MultiError(errs))
				} else {
					s()
				}
			}
		}
		for _, v := range p {
			v.Then(
				func() {
					lck.Lock()
					defer lck.Unlock()
					n--
					meh()
				},
				func(err error) {
					lck.Lock()
					defer lck.Unlock()
					errs = append(errs, err)
					n--
					meh()
				},
			)
		}
	})
}
