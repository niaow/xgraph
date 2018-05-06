package xgraph

import (
	"errors"
	"io"
	"testing"
)

func TestDep(t *testing.T) {
	tests := []testCase{
		{
			Name: "dedup",
			Args: []interface{}{
				[]string{
					"apple", "apple",
					"bannana", "bannana", "bannana",
					"ok",
					"orange",
					"wow",
				},
			},
			Func: dedup,
			Expect: []interface{}{
				[]string{
					"apple",
					"bannana",
					"ok",
					"orange",
					"wow",
				},
			},
		},
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
		{
			Name: "depcache-get",
			Func: func() ([]string, error) {
				return (&depCache{
					graph: NewGraph().
						AddJob(BasicJob{
							JobName: "wow",
							Deps:    []string{"ok"},
						}),
					cache: map[string]*depCacheEntry{},
				}).getDeps("wow")
			},
			Expect: []interface{}{[]string{"ok"}, nil},
		},
		{
			Name: "depcache-get-cache",
			Func: func() ([]string, error) {
				return (&depCache{
					cache: map[string]*depCacheEntry{
						"wow": &depCacheEntry{
							deps: []string{"ok"},
							err:  nil,
						},
					},
				}).getDeps("wow")
			},
			Expect: []interface{}{[]string{"ok"}, nil},
		},
		{
			Name: "depcache-get-cache-error",
			Func: func() ([]string, error) {
				return (&depCache{
					cache: map[string]*depCacheEntry{
						"wow": &depCacheEntry{
							deps: nil,
							err:  io.EOF,
						},
					},
				}).getDeps("wow")
			},
			Expect: []interface{}{[]string(nil), io.EOF},
		},
		{
			Name: "deptree-checkcycle-cycle",
			Args: []interface{}{"a"},
			Func: (&depTree{
				parent: &depTree{
					name: "b",
				},
				name: "a",
			}).checkCycle,
			Expect: []interface{}{ErrDependencyCycle},
		},
		{
			Name: "deptree-checkcycle-nocycle",
			Args: []interface{}{"c"},
			Func: (&depTree{
				parent: &depTree{
					name: "b",
				},
				name: "a",
			}).checkCycle,
			Expect: []interface{}{nil},
		},
		{
			Name: "getErroredBuilds",
			Args: []interface{}{
				MultiError{
					DependencyTreeError{
						JobName: "test",
						Err: DependencyTreeError{
							JobName: "test2",
							Err:     errors.New("nothing here, move along"),
						},
					},
				},
			},
			Func:   getErroredBuilds,
			Expect: []interface{}{[]string{"test", "test2"}},
		},
	}
	for _, tv := range tests {
		tv.genTest(t)
	}
}
