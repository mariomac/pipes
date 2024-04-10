package pipe

import (
	"fmt"
	"reflect"
)

type startable interface {
	Start()
}

type doneable interface {
	Done() <-chan struct{}
}

// TODO: split Pipe and PipeBuilder
type Pipe[IMPL NodesMap] struct {
	nodesMap      IMPL
	opts          []Option
	startNodes    []startable
	terminalNodes []doneable

	startProviders []reflectProvider
	midProviders   []reflectProvider
	endProviders   []reflectProvider
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

// reflected providers
type reflectProvider struct {
	// TODO: bypass field?
	asNode      reflect.Value
	fieldGetter reflect.Value
	fn          reflect.Value
}

func (rp *reflectProvider) call(nodesMap interface{}) error {
	// nodeFn, err := Provider()
	res := rp.fn.Call(nil)
	nodeFn, err := res[0], res[1]
	if !err.IsNil() {
		return fmt.Errorf("error invoking start provider: %w", err.Interface())
	}
	// fieldPtr = fieldGetter(nodesMap)
	fieldPtr := rp.fieldGetter.Call([]reflect.Value{reflect.ValueOf(nodesMap)})[0]

	// *fieldPtr = AsNode(nodeFn)
	fieldPtr.Elem().Set(
		rp.asNode.Call([]reflect.Value{nodeFn})[0],
	)
	return nil
}
