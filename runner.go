package xgraph

import "context"

//Runner is a tool to run graphs
type Runner struct {
	Graph        *Graph
	WorkRunner   WorkRunner
	EventHandler EventHandler
}

//Run executes the targets on the graph
func (r *Runner) Run(ctx context.Context, targets ...string) {
	//get WorkRunner or create it
	wr := r.WorkRunner
	if wr == nil {
		wr = NewWorkPool(0)
		defer wr.Close()
	}

	//build trees and find cycles
	tb := &treeBuilder{
		forest: make(map[string]*jTree),
		g:      r.Graph,
	}
	for _, t := range targets {
		tb.genTree(t)
	}
	tb.findCycles()

	//run build
	ex := &executor{
		forest:   tb.forest,
		runner:   wr,
		notifych: make(chan notification),
		evh:      r.EventHandler,
		proms:    make(map[string]*Promise),
		cbset:    make(map[string]func(error)),
		ctx:      ctx,
	}
	ex.execute()
}
