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

func (fn *Function) String() string {
	return typeOf(fn).String()
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

func (fn *Function) ArgChannelType(argNum int) ChannelType {
	ftype := typeOf(fn)
	arg := ftype.In(argNum)
	if arg.Kind() != reflect.Chan {
		panic(fmt.Sprintf("%s argument #%d should be a channel. Got: %d",
			ftype, argNum, arg.Kind()))
	}
	return ChannelType{inner: arg}
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
func (fn *Function) RunAsStartGoroutine(output Channel, releaseFunc func()) {
	outCh := output.Value
	go func() {
		defer releaseFunc()
		valueOf(fn).Call([]reflect.Value{outCh})
	}()
}

// RunAsEndGoroutine runs in a goroutine a func(in <-chan T) instance. It accepts a Channel
// to be used as input for data
func (fn *Function) RunAsEndGoroutine(inCh Channel) {
	go valueOf(fn).Call([]reflect.Value{inCh.Value})
}

// RunAsMiddleGoroutine runs in a goroutine a func(in <-chan T, out chan<- U) instance.
// It accepts a Channel to be used as input for data and creates creates and
// returns a Channel with the provided buffer length. When the executed function is finished,
// the returned channel is closed.
func (fn *Function) RunAsMiddleGoroutine(input, output Channel) {
	inCh := input.Value
	outCh := output.Value
	go func() {
		defer outCh.Close()
		valueOf(fn).Call([]reflect.Value{inCh, outCh})
	}()
}
