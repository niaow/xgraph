package xgraph

import (
	"fmt"
	"testing"
)

func BenchmarkTreeDeps8x8(b *testing.B) {
	benchmarkTreeDeps(b, graph8x8, 8)
}

func BenchmarkTreeDeps64x8(b *testing.B) {
	benchmarkTreeDeps(b, graph64x8, 64)
}

func benchmarkTreeDeps(b *testing.B, g *Graph, w int) {
	for i := 0; i < b.N; i++ {
		tb := &treeBuilder{
			forest: make(map[string]*jTree),
			g:      g,
		}

		for i := 0; i < w; i++ {
			tb.genTree(jobname(0, i))
		}

		tb.findCycles()
	}
}

var graph8x8 = denseValidGraph(8, 8)
var graph64x8 = denseValidGraph(64, 8)

func denseValidGraph(layerCount, width int) *Graph {
	g := New()

	for layeri := 0; layeri < layerCount; layeri++ {
		var deps []string
		if layeri+1 < layerCount {
			for index := 0; index < width; index++ {
				deps = append(deps, jobname(layeri+1, index))
			}
		}

		for index := 0; index < width; index++ {
			g.AddJob(testJob(layeri, index, deps))
		}
	}

	return g
}

func jobname(layer, index int) string {
	return fmt.Sprintf("%d_%d", layer, index)
}

func testJob(layer, index int, deps []string) *BasicJob {
	return &BasicJob{
		JobName:     jobname(layer, index),
		RunCallback: func() error { return nil },
		Deps:        deps,
	}
}
