package xgraph

import (
	"errors"
	"testing"
)

func TestRunner(t *testing.T) {
	//run statuses
	runstats := map[string]bool{}

	//create graph to use for tests
	g := NewGraph().AddJob(BasicJob{
		JobName: "test1",
		RunCallback: func() error {
			runstats["test1"] = true
			return nil
		},
	}).AddJob(BasicJob{
		JobName: "test2",
		RunCallback: func() error {
			runstats["test2"] = true
			return nil
		},
	}).AddJob(BasicJob{
		JobName: "test3",
		RunCallback: func() error {
			runstats["test3"] = true
			return nil
		},
	}).AddJob(BasicJob{
		JobName: "test4",
		RunCallback: func() error {
			runstats["test4"] = true
			return nil
		},
	}).AddJob(BasicJob{
		JobName: "test5",
		RunCallback: func() error {
			runstats["test5"] = true
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
				}).Run("test1")
				if !runstats["test1"] {
					return errors.New("test did not run")
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
