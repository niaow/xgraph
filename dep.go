package xgraph

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// depCache is a type used to cache dependency lookups
type depCache struct {
	graph *Graph
	cache map[string]*depCacheEntry
}

// depCacheEntry is the underlying depCache cache entry type.
type depCacheEntry struct {
	deps []string
	err  error
}

// getDeps gets a Job from the graph and calls Dependencies on it.
// Results are cached so only one call to GetJob/Dependencies is done.
// Errors are also cached.
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

// ErrDependencyCycle is an error returned if a dependency cycle is detected
var ErrDependencyCycle = errors.New("dependency cycle detected")

//depTree is used to walk a dependency tree to resolve deps and detect cycles
type depTree struct {
	name   string
	parent *depTree
}

// checkCycle checks if any of the parents are the given string
func (dt *depTree) checkCycle(name string) error {
	for dt != nil {
		if dt.name == name {
			return ErrDependencyCycle
		}
		dt = dt.parent
	}
	return nil
}

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

// recurse resolves all of the dependencies of the current depTree node.
// ErrDependencyCycle is returned if a dependency cycle is found.
// Uses objcache to acquire *depTree objects.
// Obtains dependency info from the *depCache
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

// recurseGraph resolves all of the dependencies of a graph
func recurseGraph(dc *depCache, targets []string) (err error) {
	errs := []error{}
	defer func() {
		errs = flatten(MultiError(errs))
		if len(errs) > 1 {
			err = MultiError(errs)
		} else if len(errs) == 1 {
			err = errs[0]
		}
	}()
	pool := &sync.Pool{
		New: func() interface{} { return new(depTree) },
	}
	dt := new(depTree)
	for _, targ := range targets {
		dt.parent = nil
		dt.name = targ
		e := dt.recurse(dc, pool)
		if e != nil {
			errs = append(errs, e)
		}
	}
	return
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
