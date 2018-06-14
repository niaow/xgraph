package xgraph

import (
	"fmt"
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

// DependencyCycleError is an error indicating that there is a dependency cycle
type DependencyCycleError []string

func (dce DependencyCycleError) Error() string {
	return "dependency cycle: " + strings.Join([]string(dce), "->")
}

type backStack []int

func (bs *backStack) push(val int) {
	*bs = append(*bs, val)
}

func (bs *backStack) drop() {
	*bs = (*bs)[:len(*bs)-1]
}

func (bs *backStack) search(val int) bool {
	for _, v := range *bs {
		if v == val {
			return true
		}
	}
	return false
}

type depIndex [][]int

func (di depIndex) checkDeps(node int, bs *backStack) (int, bool) {
	if node == -1 {
		return -1, true
	}
	if bs.search(node) {
		bs.push(node)
		return node, true
	}
	bs.push(node)
	for _, v := range di[node] {
		if n, cyc := di.checkDeps(v, bs); cyc {
			return n, true
		}
	}
	bs.drop()
	return -1, false
}

func (tb *treeBuilder) findCycles() []*jTree {
	//generate string-int mapping tables
	s2n := make(map[string]int)
	n2s := []string{}
	for n := range tb.forest {
		i := len(n2s)
		s2n[n] = i
		n2s = append(n2s, n)
	}

	//generate dep index
	di := make(depIndex, len(tb.forest))
	for _, v := range tb.forest {
		i := s2n[v.name]
		dlst := make([]int, len(v.deps))
		for j, k := range v.deps {
			d, exist := s2n[k.name]
			if !exist {
				d = -1
			}
			dlst[j] = d
		}
		di[i] = dlst
	}

	//scan with index
	cTrees := []*jTree{}
	bs := new(backStack)
	for i := range di {
		n, cyc := di.checkDeps(i, bs)
		if cyc && n != -1 {
			t := tb.forest[n2s[n]]
			if t.err == nil {
				c := make([]string, len(*bs))
				for i, v := range *bs {
					c[i] = n2s[v]
				}
				t.err = DependencyCycleError(c)
				cTrees = append(cTrees, t)
			}
		}
		*bs = (*bs)[:0]
	}

	if len(cTrees) > 0 {
		return cTrees
	}
	return nil
}
