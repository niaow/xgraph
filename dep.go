package xgraph

import (
	"fmt"
	"sort"
	"strings"
)

// DependencyTreeError is an error type returned when recursing the dependency tree
type DependencyTreeError struct {
	// JobName is the name of the job it occurred in.
	JobName string

	// Err is the error which occurred.
	// This may be another DependencyTreeError from scanning a dependency.
	Err error
}

// Backtrace generates a list of the builds the error occurred in
func (dte DependencyTreeError) Backtrace() []string {
	sub, ok := dte.Err.(DependencyTreeError)
	if ok {
		return append(sub.Backtrace(), dte.JobName)
	}
	return []string{dte.JobName}
}

// coreError finds the innermost error in the tree
func (dte DependencyTreeError) coreError() error {
	for {
		switch err := dte.Err.(type) {
		case DependencyTreeError:
			dte = err
		default:
			return err
		}
	}
}

func (dte DependencyTreeError) Error() string {
	return fmt.Sprintf("error %q in %s", dte.coreError().Error(), strings.Join(dte.Backtrace(), " in "))
}

// MultiError is a type containing multiple errors
type MultiError []error

func (me MultiError) Error() string {
	strs := make([]string, len(me))
	for i, err := range me {
		strs[i] = err.Error()
	}
	return strings.Join(strs, "\n")
}

// flatten takes a tree of DependencyTreeError and MultiError and converts this to a slice of DependencyTreeError.
func flatten(err error) []error {
	switch e := err.(type) {
	case DependencyTreeError:
		errs := flatten(e.Err)
		for i, v := range errs {
			errs[i] = DependencyTreeError{
				JobName: e.JobName,
				Err:     v,
			}
		}
		return errs
	case MultiError:
		errs := []error{}
		for _, v := range e {
			errs = append(errs, flatten(v)...)
		}
		return errs
	default:
		return []error{err}
	}
}

// dedup takes a sorted string slice with duplicates and returns a slice without duplicates
func dedup(strs []string) []string {
	for i := 1; i < len(strs); i++ {
		if strs[i-1] == strs[i] {
			strs = append(strs[:i-1], strs[i:]...)
			i--
		}
	}
	return strs
}

// getErroredBuilds scans the error and returns a list of errored builds
func getErroredBuilds(err error) (lst []string) {
	defer func() {
		sort.Strings(lst)
		lst = dedup(lst)
	}()
	switch e := err.(type) {
	case DependencyTreeError:
		return append(getErroredBuilds(e.Err), e.JobName)
	case MultiError:
		blds := []string{}
		for _, v := range e {
			blds = append(blds, getErroredBuilds(v)...)
		}
		return blds
	default:
		return []string{}
	}
}
