package node

import (
	"github.com/mariomac/go-pipes/pkg/internal/refl"
)

// todo: make it configurable
const chBufLen = 20

// TODO: when we are ready to integrate Go 1.18, enforce functions' type safety redefining the
// following types as:
//
// type InitFunction[OUT any] func(out chan<- OUT)
// type StageFunction[IN, OUT any] func(in <-chan IN, out chan<- OUT)
// type EndFunction[IN any] func(out <-chan IN)
//
// That would save us a lot of reflection checks at runtime

// InitFunc is a function that receives a writable channel as unique argument, and sends
// value to that channel during an indefinite amount of time
type InitFunc interface{}

// InnerFunc is a function that receives a readable channel as first argument,
// and a writable channel as second argument.
// It must process the inputs from the input channel until it's closed.
type InnerFunc interface{}

// TerminalFunc is a function that receives a readable channel as unique argument.
// It must process the inputs from the input channel until it's closed.
type TerminalFunc interface{}

type Sender interface {
	SendsTo(...Receiver)
}

type Receiver interface {
	startable
	joiner() *Joiner
}

type startable interface {
	isStarted() bool
	start()
}

var _ Sender = (*Init)(nil)
var _ Sender = (*Inner)(nil)
var _ Receiver = (*Inner)(nil)
var _ Receiver = (*Terminal)(nil)
var _ startable = (*Inner)(nil)
var _ startable = (*Terminal)(nil)

type Init struct {
	output
	started bool
	fun     refl.Function
}

type Inner struct {
	output
	inputs  Joiner
	started bool
	fun     refl.Function
}

func (i *Inner) joiner() *Joiner {
	return &i.inputs
}

func (i *Inner) isStarted() bool {
	return i.started
}

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

func AsInit(fun InitFunc) *Init {
	fn := refl.WrapFunction(fun)
	fn.AssertNumberOfArguments(1)
	if !fn.ArgChannelType(0).CanSend() {
		panic(fn.String() + " first argument should be a writable channel")
	}
	return &Init{fun: fn}
}

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
	t.fun.RunAsEndGoroutine(t.inputs.Receiver())
}
