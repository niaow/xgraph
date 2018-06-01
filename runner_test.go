package xgraph

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestRunner(t *testing.T) {
	//run statuses
	var lck sync.Mutex
	runstats := map[string]bool{}

	//create graph to use for tests
	g := New().AddJob(BasicJob{
		JobName: "test1",
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test1"] = true
			return nil
		},
	}).AddJob(BasicJob{
		JobName: "test2",
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test2"] = true
			return nil
		},
	}).AddJob(BasicJob{
		JobName: "test3",
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test3"] = true
			return nil
		},
	}).AddJob(BasicJob{
		JobName: "test4",
		Deps:    []string{"test3"},
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test4"] = true
			return nil
		},
	}).AddJob(BasicJob{
		JobName: "test5",
		Deps:    []string{"test4"},
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test5"] = true
			return errors.New("bad")
		},
	}).AddJob(BasicJob{
		JobName: "test6",
		Deps:    []string{"test8"},
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test6"] = true
			return errors.New("bad")
		},
	}).AddJob(BasicJob{
		JobName: "test7",
		Deps:    []string{"test6", "test8"},
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test7"] = true
			return errors.New("bad")
		},
	}).AddJob(BasicJob{
		JobName: "test8",
		Deps:    []string{"test7"},
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test8"] = true
			return errors.New("bad")
		},
	}).AddJob(BasicJob{
		JobName: "test9",
		Deps:    []string{"test10"},
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test9"] = true
			return nil
		},
	}).AddJob(BasicJob{
		JobName: "test10",
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test10"] = true
			return errors.New("bad")
		},
	}).AddJob(BasicJob{
		JobName: "test11",
		Deps:    []string{"t", "test13"},
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test11"] = true
			return errors.New("bad")
		},
	}).AddJob(BasicJob{
		JobName: "test12",
		Deps:    []string{"test13"},
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test12"] = true
			return errors.New("bad")
		},
	}).AddJob(BasicJob{
		JobName: "test13",
		Deps:    []string{"test12", "test11"},
		RunCallback: func() error {
			lck.Lock()
			defer lck.Unlock()
			runstats["test13"] = true
			return errors.New("bad")
		},
	})

	//test table
	tests := []testCase{
		{
			Name: "basic",
			Func: func() error {
				defer timeout()()
				wp := NewWorkPool(1)
				defer wp.Close()
				(&Runner{
					Graph:        g,
					WorkRunner:   wp,
					EventHandler: NoOpEventHandler,
				}).Run(context.Background(), "test1")
				if !runstats["test1"] {
					return errors.New("test did not run")
				}
				return nil
			},
			Expect: []interface{}{nil},
		},
		{
			Name: "multilevel",
			Func: func() error {
				defer timeout()()
				wp := NewWorkPool(1)
				defer wp.Close()
				(&Runner{
					Graph:        g,
					WorkRunner:   wp,
					EventHandler: NoOpEventHandler,
				}).Run(context.Background(), "test5")
				if !runstats["test5"] {
					return errors.New("test did not run")
				}
				return nil
			},
			Expect: []interface{}{nil},
		},
		{
			Name: "cycle",
			Func: func() error {
				defer timeout()()
				wp := NewWorkPool(1)
				defer wp.Close()
				(&Runner{
					Graph:        g,
					WorkRunner:   wp,
					EventHandler: NoOpEventHandler,
				}).Run(context.Background(), "test7")
				if runstats["test7"] {
					return errors.New("test ran")
				}
				return nil
			},
			Expect: []interface{}{nil},
		},
		{
			Name: "depfail-missing",
			Func: func() error {
				defer timeout()()
				wp := NewWorkPool(1)
				defer wp.Close()
				eh := &errCheckEventHandler{m: make(map[string]error)}
				(&Runner{
					Graph:        g,
					WorkRunner:   wp,
					EventHandler: eh,
				}).Run(context.Background(), "test11")
				if runstats["test11"] {
					return errors.New("test ran")
				}
				return eh.m["test11"]
			},
			Expect: []interface{}{JobNotFoundError("t")},
		},
	}

	//run tests
	for _, tv := range tests {
		tv.genTest(t)
	}
}

type errCheckEventHandler struct {
	m map[string]error
	nophandler
}

func (eceh *errCheckEventHandler) OnError(name string, err error) {
	eceh.m[name] = err
}
