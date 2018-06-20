package xgraph

import (
	"strings"

	"github.com/looplab/tarjan"
)

type jTree struct {
	name     string
	finished bool
	started  bool
	err      error
	job      Job
	deps     []*jTree
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
	graph := make(map[interface{}][]interface{})
	for name, job := range tb.g.jobs {
		deps, _ := job.Dependencies()
		gdeps := make([]interface{}, len(deps))
		for i := range gdeps {
			gdeps[i] = deps[i]
		}
		graph[name] = gdeps
	}

	issues := tarjan.Connections(graph)

	results := []*jTree{}
	for _, issue := range issues {
		if len(issue) == 1 {
			n := issue[0].(string)
			node := tb.forest[n]
			if node == nil {
				continue
			}
			for _, v := range node.deps {
				if v.name == n {
					goto errgen
				}
			}
			continue
		}
	errgen:
		component := []string{}
		for _, elem := range issue {
			component = append(component, elem.(string))
		}

		for _, elem := range issue {
			job := tb.forest[elem.(string)]
			if job == nil {
				continue
			}
			if job.err == nil {
				job.err = DependencyCycleError(component)
			}
			results = append(results, job)
		}
	}

	if len(results) > 0 {
		return results
	}
	return nil
}

func (tb *treeBuilder) findCyclesOld() []*jTree {
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
