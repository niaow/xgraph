package xgraph

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

//depCache is a type used to cache dependency lookups
type depCache struct {
	graph *Graph
	cache map[string]*depCacheEntry
}

type depCacheEntry struct {
	deps []string
	err  error
}

func (dc *depCache) getDeps(name string) ([]string, error) {
	//do lookup in cache
	ce := dc.cache[name]
	if ce != nil {
		return ce.deps, ce.err
	}

	//lookup job in graph
	j, err := dc.graph.GetJob(name)
	var deps []string
	if err == nil {
		deps, err = j.Dependencies()
	}

	//store to cache
	dc.cache[name] = &depCacheEntry{deps, err}

	//return
	return deps, err
}

//ErrDependencyCycle is an error returned if a dependency cycle is detected
var ErrDependencyCycle = errors.New("dependency cycle detected")

//depTree is used to walk a dependency tree to resolve deps and detect cycles
type depTree struct {
	name   string
	parent *depTree
}

//checkCycle checks if any of the parents are the given string
func (dt *depTree) checkCycle(name string) error {
	for dt != nil {
		if dt.name == name {
			return ErrDependencyCycle
		}
		dt = dt.parent
	}
	return nil
}

//DependencyTreeError is an error type returned when recursing the dependency tree
type DependencyTreeError struct {
	//JobName is the name of the job it occurred in.
	JobName string

	//Err is the error which occurred.
	//This may be another DependencyTreeError from scanning a dependency.
	Err error
}

//Backtrace generates a list of the builds the error occurred in
func (dte DependencyTreeError) Backtrace() []string {
	sub, ok := dte.Err.(DependencyTreeError)
	if ok {
		return append(sub.Backtrace(), dte.JobName)
	}
	return []string{dte.JobName}
}

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

func (dte DependencyTreeError) flatten() []error {
	switch err := dte.Err.(type) {
	case DependencyTreeError:
		//run flatten in tree
		return DependencyTreeError{
			JobName: dte.JobName,
			Err:     MultiError(err.flatten()),
		}.flatten()
	case MultiError:
		errs := []error(err)
		werrs := []error{}
		for _, v := range errs {
			werrs = append(werrs, DependencyTreeError{
				JobName: dte.JobName,
				Err:     v,
			})
		}
		return werrs
	default:
		return []error{dte}
	}
}

//MultiError is a type containing multiple errors
type MultiError []error

func (me MultiError) Error() string {
	strs := make([]string, len(me))
	for i, err := range me {
		strs[i] = err.Error()
	}
	return strings.Join(strs, "\n")
}

func (dt *depTree) recurse(dc *depCache, objcache *sync.Pool) (err error) {
	//wrap errors in a DependencyTreeError
	defer func() {
		if err != nil {
			err = DependencyTreeError{
				JobName: dt.name,
				Err:     err,
			}
		}
	}()

	deps, err := dc.getDeps(dt.name)
	if err != nil {
		return err
	}
	errs := []error{}
	for _, d := range deps {
		//check for cycles
		err = dt.checkCycle(d)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		//pull a *depTree out of the object cache and initialize it
		dtsub := objcache.Get().(*depTree)
		dtsub.name = d
		dtsub.parent = dt
		//recurse the *depTree
		err = dtsub.recurse(dc, objcache)
		//return object to cache
		objcache.Put(dtsub)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}
	err = nil
	if len(errs) > 0 {
		err = MultiError(errs)
	}
	return
}
