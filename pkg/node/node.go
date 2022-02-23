package node

import (
	"github.com/mariomac/go-pipes/pkg/internal/refl"
	"sync/atomic"
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
	SendsToFn(fun InnerFunc) *Inner
	SendsToTermFn(fun TerminalFunc) *Terminal
}

type Receiver interface {
	incInputs()
	inputs() int32
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
	input
	started bool
	fun     refl.Function
}

func (i *Inner) isStarted() bool {
	return i.started
}

type Terminal struct {
	input
	started bool
	fun     refl.Function
}

func (t *Terminal) isStarted() bool {
	return t.started
}

type input struct {
	nInputs int32
	in      refl.Channel
}

func (i *input) incInputs() {
	atomic.AddInt32(&i.nInputs, 1)
}

func (i *input) inputs() int32 {
	return atomic.LoadInt32(&i.nInputs)
}

type output struct {
	outs []Receiver
}

func (s *output) SendsTo(outputs ...Receiver) {
	for i := range outputs {
		outputs[i].incInputs()
	}
	s.outs = append(s.outs, outputs...)
}

func (s *output) SendsToFn(fun InnerFunc) *Inner {
	inner := AsInner(fun)
	inner.incInputs()
	s.outs = append(s.outs, inner)
	return inner
}

func (s *output) SendsToTermFn(fun TerminalFunc) *Terminal {
	term := AsTerminal(fun)
	term.incInputs()
	s.outs = append(s.outs, term)
	return term
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
		input: input{in: inCh.Instantiate(chBufLen)},
		fun:   fn,
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
		input: input{in: inCh.Instantiate(chBufLen)},
		fun:   fn,
	}
}

func (i *Init) Start() {
	//i.fun.RunAsStartGoroutine(i)
	panic("implement me")
}

func (i *Inner) start() {
	//TODO implement me
	panic("implement me")
}

func (t *Terminal) start() {
	//TODO implement me
	panic("implement me")
}
