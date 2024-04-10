package node

type startable interface {
	Start()
}

type doneable interface {
	Done() <-chan struct{}
}

type Pipe struct {
	opts          []Option
	startNodes    []startable
	terminalNodes []doneable
}

func NewPipe(defaultOpts ...Option) *Pipe {
	return &Pipe{opts: defaultOpts}
}

func AddStart[OUT any](p *Pipe, fn StartFunc[OUT]) Sender[OUT] {
	startNode := asStart(fn)
	p.startNodes = append(p.startNodes, startNode)
	return startNode
}

func AddMiddleOpt[INOUT any](p *Pipe, fn MiddleFunc[INOUT, INOUT], opts ...Option) SenderReceiver[INOUT, INOUT] {
	if fn == nil {
		bp := bypass[INOUT]{}
		return &bp
	}
	return asMiddle(fn, p.joinOpts(opts...)...)

}

func AddMiddle[IN, OUT any](p *Pipe, fn MiddleFunc[IN, OUT], opts ...Option) SenderReceiver[IN, OUT] {
	if fn == nil {
		panic("AsMiddle can't receive a nil function. If you want to use" +
			" an optional function that can be ignored in case of nil, invoke AsMiddleOpt")
	}
	return asMiddle(fn, p.joinOpts(opts...)...)
}

func AddTerminal[IN any](p *Pipe, fn TerminalFunc[IN], opts ...Option) Receiver[IN] {
	termNode := asTerminal(fn, p.joinOpts(opts...)...)
	p.terminalNodes = append(p.terminalNodes, termNode)
	return termNode
}

func (p *Pipe) joinOpts(opts ...Option) []Option {
	var opt []Option
	opt = append(opt, p.opts...)
	opt = append(opt, opts...)
	return opt
}

func (p *Pipe) Start() {
	for _, s := range p.startNodes {
		s.Start()
	}
}

func (p *Pipe) Done() <-chan struct{} {
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
