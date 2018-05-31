package xgraph

import (
	"errors"
	"testing"
	"time"
)

func timeout() func() {
	finch := make(chan struct{})
	go func() {
		t := time.NewTimer(time.Second * 10)
		select {
		case <-t.C:
			panic(errors.New("timeout"))
		case <-finch:
		}
	}()
	return func() { finch <- struct{}{} }
}

func TestPromise(t *testing.T) {
	tests := []testCase{
		{
			Name: "basic",
			Func: func() (bool, error) {
				defer timeout()()
				var run bool
				var e error
				NewPromise(func(s FinishHandler, f FailHandler) {
					s()
				}).Then(func() { run = true }, func(err error) { e = err })
				return run, e
			},
			Expect: []interface{}{true, nil},
		},
		{
			Name: "basic-error",
			Func: func() (bool, error) {
				defer timeout()()
				var run bool
				var e error
				NewPromise(func(s FinishHandler, f FailHandler) {
					f(errors.New("this is an error"))
				}).Then(func() { run = true }, func(err error) { e = err })
				return run, e
			},
			Expect: []interface{}{false, errors.New("this is an error")},
		},
		{
			Name: "cache",
			Func: func() (bool, bool, error, error) {
				defer timeout()()
				var run1, run2 bool
				var e1, e2 error
				p := NewPromise(func(s FinishHandler, f FailHandler) {
					f(errors.New("this is an error"))
				})
				p.Then(func() { run1 = true }, func(err error) { e1 = err })
				p.Then(func() { run2 = true }, func(err error) { e2 = err })
				return run1, run2, e1, e2
			},
			Expect: []interface{}{false, false, errors.New("this is an error"), errors.New("this is an error")},
		},
		{
			Name: "cache-error",
			Func: func() (bool, bool, error, error) {
				defer timeout()()
				var run1, run2 bool
				var e1, e2 error
				p := NewPromise(func(s FinishHandler, f FailHandler) {
					s()
				})
				p.Then(func() { run1 = true }, func(err error) { e1 = err })
				p.Then(func() { run2 = true }, func(err error) { e2 = err })
				return run1, run2, e1, e2
			},
			Expect: []interface{}{true, true, nil, nil},
		},
		{
			Name: "cache-single-run",
			Func: func() int {
				defer timeout()()
				runs := 0
				p := NewPromise(func(s FinishHandler, f FailHandler) {
					runs++
					s()
				})
				p.Then(func() {}, func(err error) {})
				p.Then(func() {}, func(err error) {})
				return runs
			},
			Expect: []interface{}{1},
		},
	}
	for _, tv := range tests {
		tv.genTest(t)
	}
}
