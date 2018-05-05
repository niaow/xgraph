package xgraph

import (
	"io"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

type testCase struct {
	Name   string
	Args   []interface{}
	Func   interface{}
	Expect []interface{}
}

func (tc testCase) runTest(t *testing.T) {
	args := make([]reflect.Value, len(tc.Args))
	for i, v := range tc.Args {
		args[i] = reflect.ValueOf(v)
	}
	vals := reflect.ValueOf(tc.Func).Call(args)
	valint := make([]interface{}, len(vals))
	for i, v := range vals {
		valint[i] = v.Interface()
	}
	if !reflect.DeepEqual(valint, tc.Expect) {
		t.Errorf("Expected %v but got %v for test %q", spew.Sdump(tc.Expect), spew.Sdump(valint), tc.Name)
	}
}

func (tc testCase) genTest(t *testing.T) {
	t.Run(tc.Name, func(t *testing.T) {
		tc.runTest(t)
	})
}

func TestBasicJob(t *testing.T) {
	tests := []testCase{
		{
			Name:   "name",
			Func:   BasicJob{JobName: "test"}.Name,
			Expect: []interface{}{"test"},
		},
		{
			Name:   "run",
			Func:   BasicJob{RunCallback: func() error { return nil }}.Run,
			Expect: []interface{}{nil},
		},
		{
			Name:   "run-error-propogate",
			Func:   BasicJob{RunCallback: func() error { return io.EOF }}.Run,
			Expect: []interface{}{io.EOF},
		},
		{
			Name:   "run-missing-callback",
			Func:   BasicJob{}.Run,
			Expect: []interface{}{ErrMissingCallback},
		},
		{
			Name:   "shouldrun-default",
			Func:   BasicJob{}.ShouldRun,
			Expect: []interface{}{true, nil},
		},
		{
			Name:   "shouldrun-custom",
			Func:   BasicJob{ShouldRunCallback: func() (bool, error) { return false, io.EOF }}.ShouldRun,
			Expect: []interface{}{false, io.EOF},
		},
		{
			Name:   "dependencies",
			Func:   BasicJob{Deps: []string{"dep1", "dep2"}}.Dependencies,
			Expect: []interface{}{[]string{"dep1", "dep2"}, nil},
		},
		{
			Name:   "dependencies-defailt",
			Func:   BasicJob{}.Dependencies,
			Expect: []interface{}{[]string{}, nil},
		},
	}
	for _, tv := range tests {
		tv.genTest(t)
	}
}
