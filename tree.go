package xgraph

import (
	"fmt"
	"sort"
	"strings"
)

type jTree struct {
	name     string
	finished bool
	started  bool
	err      error
	job      Job
	deps     []*jTree
	usedby   []*jTree
}

// isReady checks if all of the dependencies of the *jTree have completed
func (jt *jTree) isReady() bool {
	if len(jt.deps) == 0 {
		return true
	}
	for _, d := range jt.deps {
		if !d.finished {
			return false
		}
	}
	return true
}

//DepFailError is an error type indicating that a job could not be run because a dependency failed
type DepFailError struct {
	//JobName is the name of the job which could not be run
	JobName string

	//FailedDeps is the list of dependencies which failed
	FailedDeps []string
}

func (dfe DepFailError) Error() string {
	strs := make([]string, len(dfe.FailedDeps))
	for i, v := range dfe.FailedDeps {
		strs[i] = fmt.Sprintf("%q", v)
	}
	return fmt.Sprintf("could not run %q because dependencies failed (failures: %s)", dfe.JobName, strings.Join(strs, ", "))
}

func (jt *jTree) depFail() error {
	fails := []string{}
	for _, v := range jt.deps {
		if v.err != nil {
			fails = append(fails, v.name)
		}
	}
	if len(fails) > 0 {
		return DepFailError{
			JobName:    jt.name,
			FailedDeps: fails,
		}
	}
	return nil
}

type treeBuilder struct {
	forest map[string]*jTree
	g      *Graph
}

// genTree generates a *jTree if it does not already exist
func (tb *treeBuilder) genTree(name string) (*jTree, error) {
	//check to see if it is already there
	t := tb.forest[name]
	if t != nil {
		return t, t.err
	}

	//create tree
	t = new(jTree)
	tb.forest[name] = t
	t.name = name
	t.usedby = []*jTree{}
	t.deps = []*jTree{}

	//lookup job
	j, err := tb.g.GetJob(name)
	if err != nil {
		t.err = err
		t.finished = true
		return t, err
	}
	t.job = j

	//load dependency list
	deps, err := j.Dependencies()
	if err != nil {
		t.err = err
		t.finished = true
		return t, err
	}

	//generate deps
	darr := make([]*jTree, len(deps))
	errs := []error{}
	for i, v := range deps {
		d, err := tb.genTree(v)
		if err != nil {
			errs = append(errs, err)
		}
		d.usedby = append(d.usedby, t) //mark dep as used by this
		darr[i] = d
	}
	t.deps = darr
	if len(errs) > 0 {
		t.err = errs[0]
	}

	return t, nil
}

// listContains checks if a sorted list contains a value
func listContains(list []string, val string) bool {
	i := sort.SearchStrings(list, val)
	if i == len(list) {
		return false
	}
	return list[i] == val
}

type cycleChain struct {
	name   string
	parent *cycleChain
}

// jTreeNames takes a slice of *jTree and returns a sorted slice of the names
func jTreeNames(trees []*jTree) []string {
	names := make([]string, len(trees))
	for i, v := range trees {
		names[i] = v.name
	}
	sort.Strings(names)
	return names
}

func names2map(names []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}

// sub returns a cycleChain with this cycleChain as the parent
func (cc *cycleChain) sub(name string) *cycleChain {
	return &cycleChain{
		name:   name,
		parent: cc,
	}
}

// DependencyCycleError is an error indicating that there is a dependency cycle
type DependencyCycleError []string

func (dce DependencyCycleError) Error() string {
	return "dependency cycle: " + strings.Join([]string(dce), "->")
}

// check recurses the *cycleChain to check if there is a dependency cycle with the given name
func (cc *cycleChain) check(name string) error {
	if cc.name == name {
		return DependencyCycleError{name, cc.name}
	}
	if cc.parent == nil {
		return nil
	}
	cyc := cc.parent.check(name)
	if cyc != nil {
		return append(cyc.(DependencyCycleError), cc.name)
	}
	return nil
}

// chainRoot is a *cycleChain which has no parent
var chainRoot = &cycleChain{
	name:   "",
	parent: nil,
}

func (tb *treeBuilder) checkCycle(parent *cycleChain, jt *jTree) error {
	if err := parent.check(jt.name); err != nil {
		return err
	}
	sub := parent.sub(jt.name)
	for _, v := range jt.deps {
		err := tb.checkCycle(sub, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tb *treeBuilder) findCycles() []*jTree {
	cTrees := []*jTree{}
	for _, v := range tb.forest {
		err := tb.checkCycle(chainRoot, v)
		if err != nil {
			if v.err == nil {
				v.err = err
			}
			cTrees = append(cTrees, v)
		}
	}
	if len(cTrees) > 0 {
		return cTrees
	}
	return nil
}
