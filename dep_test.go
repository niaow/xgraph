package xgraph

import (
	"errors"
	"testing"
)

func TestDepErr(t *testing.T) {
	tests := []testCase{
		{
			Name:   "multierror",
			Func:   MultiError{errors.New("wow"), errors.New("ok")}.Error,
			Expect: []interface{}{"wow\nok"},
		},
		{
			Name:   "dependency-tree-error-basic",
			Func:   DependencyTreeError{JobName: "testjob", Err: errors.New("test")}.Error,
			Expect: []interface{}{"error \"test\" in testjob"},
		},
		{
			Name: "dependency-tree-error-core-error",
			Func: DependencyTreeError{
				JobName: "testjob",
				Err: DependencyTreeError{
					JobName: "test2",
					Err:     errors.New("test"),
				},
			}.coreError,
			Expect: []interface{}{errors.New("test")},
		},
		{
			Name: "dependency-tree-error-backtrace",
			Func: DependencyTreeError{
				JobName: "testjob",
				Err: DependencyTreeError{
					JobName: "test2",
					Err:     errors.New("test"),
				},
			}.Backtrace,
			Expect: []interface{}{[]string{"test2", "testjob"}},
		},
		{
			Name: "dependency-tree-error-recursive",
			Func: DependencyTreeError{
				JobName: "testjob",
				Err: DependencyTreeError{
					JobName: "test2",
					Err:     errors.New("test"),
				},
			}.Error,
			Expect: []interface{}{"error \"test\" in test2 in testjob"},
		},
		{
			Name: "error-flatten",
			Func: func() []error {
				return flatten(DependencyTreeError{
					JobName: "testjob",
					Err: MultiError{
						DependencyTreeError{
							JobName: "test1",
							Err:     errors.New("error 1"),
						},
						DependencyTreeError{
							JobName: "test2",
							Err:     errors.New("error 2"),
						},
						DependencyTreeError{
							JobName: "test3",
							Err: MultiError{
								DependencyTreeError{
									JobName: "test4",
									Err:     errors.New("error 4"),
								},
								DependencyTreeError{
									JobName: "test5",
									Err:     errors.New("error 5"),
								},
							},
						},
					},
				})
			},
			Expect: []interface{}{[]error{
				DependencyTreeError{
					JobName: "testjob",
					Err: DependencyTreeError{
						JobName: "test1",
						Err:     errors.New("error 1"),
					},
				},
				DependencyTreeError{
					JobName: "testjob",
					Err: DependencyTreeError{
						JobName: "test2",
						Err:     errors.New("error 2"),
					},
				},
				DependencyTreeError{
					JobName: "testjob",
					Err: DependencyTreeError{
						JobName: "test3",
						Err: DependencyTreeError{
							JobName: "test4",
							Err:     errors.New("error 4"),
						},
					},
				},
				DependencyTreeError{
					JobName: "testjob",
					Err: DependencyTreeError{
						JobName: "test3",
						Err: DependencyTreeError{
							JobName: "test5",
							Err:     errors.New("error 5"),
						},
					},
				},
			}},
		},
		//TODO: test flatten
	}
	for _, tv := range tests {
		tv.genTest("DepErr", t)
	}
}
