package graph

type initNode interface {
	Start()
}

type terminalNode interface {
	Done() <-chan struct{}
}

type Graph struct {
	start []initNode
	terms []terminalNode
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
