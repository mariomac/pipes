package graph

import "github.com/mariomac/pipes/pkg/node"

type Graph struct {
	start []*node.Init
	terms []*node.Terminal
}

// Run all the stages of the graph and wait until all the nodes stopped processing.
func (g *Graph) Run() {
	// start all stages
	for _, s := range g.start {
		s.Start()
	}
	// wait for all stages to finish
	for _, t := range g.terms {
		<-t.Done()
	}
}
