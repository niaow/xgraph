package xgraph

import (
	"fmt"
	"sort"
	"strings"
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

// BuildDependencyError is an error indicating that dependencies failed
type BuildDependencyError []string

func (bde BuildDependencyError) Error() string {
	return fmt.Sprintf("dependencies failed: (%s)", strings.Join([]string(bde), ","))
}

func newBuildPromise(deps map[string]*Promise) *Promise {
	return NewPromise(func(s FinishHandler, f FailHandler) {
		fails := make(map[string]struct{})
		n := len(deps)
		meh := func() {
			if n == 0 {
				if len(fails) == 0 {
					s()
				} else {
					flst := make([]string, len(fails))
					i := 0
					for n := range fails {
						flst[i] = n
						i++
					}
					sort.Strings(flst)
					f(BuildDependencyError(flst))
				}
				fails = nil
			}
		}
		for i, v := range deps {
			name := i
			v.Then(func() {
				n--
				meh()
			}, func(error) {
				n--
				fails[name] = struct{}{}
				meh()
			})
		}
	})
}
