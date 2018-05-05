package xgraph

import (
	"io"
	"sync"
	"testing"
)

func TestWorkPool(t *testing.T) {
	wp := NewWorkPool(0)
	defer wp.Close()
	tests := []testCase{
		{
			Name: "CallbackTracker",
			Func: func() error {
				var thing error
				lck := new(sync.Mutex)
				lck.Lock()
				CallbackTracker(func(err error) {
					thing = err
					lck.Unlock()
				})(io.EOF)
				return thing
			},
			Expect: []interface{}{io.EOF},
		},
		{
			Name: "basic",
			Func: func() error {
				var thing error
				lck := new(sync.Mutex)
				lck.Lock()
				wp.DoTask(func() error { return nil }, CallbackTracker(func(err error) {
					thing = err
					lck.Unlock()
				}))
				lck.Lock()
				return thing
			},
			Expect: []interface{}{nil},
		},
	}
	for _, tv := range tests {
		tv.genTest("WorkPool", t)
	}
}
