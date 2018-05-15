package xgraph

import (
	"io"
	"testing"
)

func TestGraph(t *testing.T) {
	tests := []testCase{
		{
			Name: "basic",
			Func: func() (Job, error) {
				return NewGraph().
					AddJob(BasicJob{JobName: "test"}).
					GetJob("test")
			},
			Expect: []interface{}{BasicJob{JobName: "test"}, nil},
		},
		{
			Name: "twojobs",
			Func: func() (Job, error) {
				return NewGraph().
					AddJob(BasicJob{JobName: "test"}).
					AddJob(BasicJob{JobName: "test2"}).
					GetJob("test")
			},
			Expect: []interface{}{BasicJob{JobName: "test"}, nil},
		},
		{
			Name: "generate",
			Func: func() (Job, error) {
				return NewGraph().
					AddGenerator(func(name string) (Job, error) {
						return BasicJob{JobName: name}, nil
					}).
					GetJob("test")
			},
			Expect: []interface{}{BasicJob{JobName: "test"}, nil},
		},
		{
			Name: "generate-multiple",
			Func: func() (Job, error) {
				return NewGraph().
					AddGenerator(func(name string) (Job, error) {
						return BasicJob{JobName: name}, nil
					}).
					AddGenerator(func(name string) (Job, error) {
						return nil, nil
					}).
					GetJob("test")
			},
			Expect: []interface{}{BasicJob{JobName: "test"}, nil},
		},
		{
			Name: "generate-error",
			Func: func() (Job, error) {
				return NewGraph().
					AddGenerator(func(name string) (Job, error) {
						return nil, io.EOF
					}).
					GetJob("test")
			},
			Expect: []interface{}{nil, io.EOF},
		},
		{
			Name: "not-found",
			Func: func() (Job, error) {
				return NewGraph().
					GetJob("test")
			},
			Expect: []interface{}{nil, JobNotFoundError("test")},
		},
		{
			Name:   "not-found-error",
			Func:   JobNotFoundError("test").Error,
			Expect: []interface{}{"job not found: \"test\""},
		},
		{
			Name:   "not-found-string",
			Func:   JobNotFoundError("test").String,
			Expect: []interface{}{"job not found: \"test\""},
		},
	}
	for _, tv := range tests {
		tv.genTest(t)
	}
}
