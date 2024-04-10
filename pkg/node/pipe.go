package node

type startable interface {
	Start()
}

type doneable interface {
	Done() <-chan struct{}
}

type Pipe[IMPL NodesMap] struct {
	nodesMap      IMPL
	opts          []Option
	startNodes    []startable
	terminalNodes []doneable
}

func NewPipe[IMPL NodesMap](nodesMap IMPL, defaultOpts ...Option) *Pipe[IMPL] {
	return &Pipe[IMPL]{nodesMap: nodesMap, opts: defaultOpts}
}

func (p *Pipe[IMPL]) joinOpts(opts ...Option) []Option {
	var opt []Option
	opt = append(opt, p.opts...)
	opt = append(opt, opts...)
	return opt
}

func (p *Pipe[IMPL]) Start() {
	for _, s := range p.startNodes {
		s.Start()
	}
}

func (p *Pipe[IMPL]) Done() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		for _, s := range p.terminalNodes {
			<-s.Done()
		}
		close(done)
	}()
	return done
}

//func AddStart
