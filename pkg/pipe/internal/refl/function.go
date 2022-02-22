// Package refl wraps some reflection functionalities
package refl

import (
	"fmt"
	"reflect"
)

// Function wraps a reflect.Value and provides some common reflective assertions about its type
type Function reflect.Value

// WrapFunction wraps a provided function into a refl.Function object. It panics if
// the provided argument is not a function.
func WrapFunction(fn interface{}) Function {
	val := reflect.ValueOf(fn)
	if val.Kind() != reflect.Func {
		panic("expecting a function. Got: " + val.Kind().String())
	}
	return Function(val)
}

// NumArgs gets the number of arguments of the function
func (fn *Function) NumArgs() int {
	return typeOf(fn).NumIn()
}

// AssertNumberOfArguments panics if the function does not have the given number of arguments
func (fn *Function) AssertNumberOfArguments(num int) {
	ftype := typeOf(fn)
	if ftype.NumIn() != num {
		// TODO: give information of the actual arguments
		panic(ftype.Name() + " argument should be a writable channel only")
	}
}

// AssertArgumentIsDirectedChannel panics if the numbered argument is not a channel supporting the
// provided direction
func (fn *Function) AssertArgumentIsDirectedChannel(argNum int, dir reflect.ChanDir) {
	ftype := typeOf(fn)
	arg := ftype.In(argNum)
	if arg.Kind() != reflect.Chan || arg.ChanDir()&dir == 0 {
		panic(ftype.Name() + " argument should be a writable channel only")
	}
}

// AssertArgsConnectableChannels panics if the thisArg channel of the receiver function is not
// assignable from the "fromArg" argument of the "from" function
func (fn *Function) AssertArgsConnectableChannels(thisArg int, from Function, fromArg int) {
	outCh := typeOf(&from).In(fromArg)
	inCh := typeOf(fn).In(thisArg)
	if !outCh.Elem().AssignableTo(inCh.Elem()) {
		panic(fmt.Sprintf("couldn't assign %s to %s", outCh, inCh))
	}
}

// RunAsStartGoroutine runs in a goroutine a func(out chan<- T) instance. It creates and
// returns a Channel with the provided buffer length. When the executed function is finished,
// the channel is closed.
func (fn *Function) RunAsStartGoroutine(chanBufLen int) Channel {
	fnType := typeOf(fn)
	outCh := makeChannel(fnType.In(0).Elem(), chanBufLen)
	go func() {
		defer outCh.Close()
		valueOf(fn).Call([]reflect.Value{outCh})
	}()
	return Channel(outCh)
}

// RunAsEndGoroutine runs in a goroutine a func(in <-chan T) instance. It accepts a Channel
// to be used as input for data
func (fn *Function) RunAsEndGoroutine(inCh Channel) {
	go valueOf(fn).Call([]reflect.Value{reflect.Value(inCh)})
}

// RunAsMiddleGoroutine runs in a goroutine a func(in <-chan T, out chan<- U) instance.
// It accepts a Channel to be used as input for data and creates creates and
// returns a Channel with the provided buffer length. When the executed function is finished,
// the returned channel is closed.
func (fn *Function) RunAsMiddleGoroutine(inCh Channel, channelsBuf int) Channel {
	fnType := typeOf(fn)
	outCh := makeChannel(fnType.In(1).Elem(), channelsBuf)
	go func() {
		defer outCh.Close()
		valueOf(fn).Call([]reflect.Value{reflect.Value(inCh), outCh})
	}()
	return Channel(outCh)
}
