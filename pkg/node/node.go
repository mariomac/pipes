// Package node provides functionalities to create nodes and interconnect them.
// A Node is a function container that can be connected via channels to other nodes.
// A node can send data to multiple nodes, and receive data from multiple nodes.
package node

import (
	"github.com/mariomac/go-pipes/pkg/internal/refl"
)

// todo: make it configurable
const chBufLen = 20

// InitFunc is a function that receives a writable channel as unique argument, and sends
// value to that channel during an indefinite amount of time.
// TODO: with Go 1.18, this will be
// type InitFunc[OUT any] func(out chan<- OUT)
type InitFunc interface{}

// InnerFunc is a function that receives a readable channel as first argument,
// and a writable channel as second argument.
// It must process the inputs from the input channel until it's closed.
// TODO: with Go 1.18, this will be
// type InnerFunc[IN, OUT any] func(in <-chan IN, out chan<- OUT)
type InnerFunc interface{}

// TerminalFunc is a function that receives a readable channel as unique argument.
// It must process the inputs from the input channel until it's closed.
// TODO: with Go 1.18, this will be
// type TerminalFunc[IN any] func(out <-chan IN)
type TerminalFunc interface{}

// Sender is any node that can send data to another node: node.Init and node.Inner
type Sender interface {
	SendsTo(...Receiver)
}

// Receiver is any node that can receive data from another node: node.Inner and node.Terminal
type Receiver interface {
	startable
	joiner() *Joiner
}

type startable interface {
	isStarted() bool
	start()
}

// Init nodes are the starting points of a graph. This is all the nodes that bring information
// from outside the graph: e.g. because they generate them or because they acquire them from an
// external source like a Web Service.
// A graph must have at least one Init node.
// An Init node must have at least one output node.
type Init struct {
	output
	fun refl.Function
}

func (i *Inner) joiner() *Joiner {
	return &i.inputs
}

func (i *Inner) isStarted() bool {
	return i.started
}

// Inner is any intermediate node that receives data from another node, processes/filters it,
// and forwards the data to another node.
// An Inner node must have at least one output node.
type Inner struct {
	output
	inputs  Joiner
	started bool
	fun     refl.Function
}

// Terminal is any node that receives data from another node and does not forward it to another node,
// but can process it and send the results to outside the graph (e.g. memory, storage, web...)
type Terminal struct {
	inputs  Joiner
	started bool
	fun     refl.Function
}

func (i *Terminal) joiner() *Joiner {
	return &i.inputs
}

func (t *Terminal) isStarted() bool {
	return t.started
}

type output struct {
	outs []Receiver
}

func (s *output) SendsTo(outputs ...Receiver) {
	s.outs = append(s.outs, outputs...)
}

// AsInit wraps an InitFunc into an Init node. It panics if the InitFunc does not follow the
// func(chan<-) signature.
func AsInit(fun InitFunc) *Init {
	fn := refl.WrapFunction(fun)
	fn.AssertNumberOfArguments(1)
	if !fn.ArgChannelType(0).CanSend() {
		panic(fn.String() + " first argument should be a writable channel")
	}
	return &Init{fun: fn}
}

// AsInner wraps an InnerFunc into an Inner node.
// It panics if the InnerFunc does not follow the func(<-chan,chan<-) signature.
func AsInner(fun InnerFunc) *Inner {
	fn := refl.WrapFunction(fun)
	// check that the arguments are a read channel and a write channel
	fn.AssertNumberOfArguments(2)
	inCh := fn.ArgChannelType(0)
	if !inCh.CanReceive() {
		panic(fn.String() + " first argument should be a readable channel")
	}
	outCh := fn.ArgChannelType(1)
	if !outCh.CanSend() {
		panic(fn.String() + " second argument should be a writable channel")
	}
	return &Inner{
		inputs: NewJoiner(inCh, chBufLen),
		fun:    fn,
	}
}

// AsTerminal wraps a TerminalFunc into a Terminal node.
// It panics if the TerminalFunc does not follow the func(<-chan) signature.
func AsTerminal(fun TerminalFunc) *Terminal {
	fn := refl.WrapFunction(fun)
	// check that the arguments are only a read channel
	fn.AssertNumberOfArguments(1)
	inCh := fn.ArgChannelType(0)
	if !inCh.CanReceive() {
		panic(fn.String() + " first argument should be a readable channel")
	}
	return &Terminal{
		inputs: NewJoiner(inCh, chBufLen),
		fun:    fn,
	}
}

func (i *Init) Start() {
	if len(i.outs) == 0 {
		panic("Init node should have outputs")
	}
	joiners := make([]*Joiner, 0, len(i.outs))
	for _, out := range i.outs {
		joiners = append(joiners, out.joiner())
		if !out.isStarted() {
			out.start()
		}
	}
	forker := Fork(joiners...)
	i.fun.RunAsStartGoroutine(forker.Sender(), forker.Close)
}

func (i *Inner) start() {
	if len(i.outs) == 0 {
		panic("Inner node should have outputs")
	}
	i.started = true
	joiners := make([]*Joiner, 0, len(i.outs))
	for _, out := range i.outs {
		joiners = append(joiners, out.joiner())
		if !out.isStarted() {
			out.start()
		}
	}
	forker := Fork(joiners...)
	i.fun.RunAsMiddleGoroutine(
		i.inputs.Receiver(),
		forker.Sender(),
		forker.Close)
}

func (t *Terminal) start() {
	t.started = true
	t.fun.RunAsEndGoroutine(t.inputs.Receiver())
}
