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
	}

	//run tests
	for _, tv := range tests {
		tv.genTest(t)
	}
}
