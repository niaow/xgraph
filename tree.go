package xgraph

type jTree struct {
	name     string
	finished bool
	err      error
	job      Job
	deps     []*jTree
	usedby   []*jTree
}

type treeBuilder struct {
	forest map[string]*jTree
	g      *Graph
}

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
		errs = flatten(DependencyTreeError{
			JobName: name,
			Err:     MultiError(errs),
		})
		if len(errs) == 1 {
			t.err = errs[0]
		} else {
			t.err = MultiError(errs)
		}
	}

	return t, nil
}
