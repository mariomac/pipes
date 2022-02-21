package pipe

import (
	"fmt"
	"reflect"
)

// todo use defined types for more type safety with methods

func checkIsFunction(fn reflect.Value) {
	if fn.Kind() != reflect.Func {
		panic("expecting a function. Got: " + fn.Kind().String())
	}
}

func checkArgumentIsWritableChannel(fn reflect.Value) {
	ftype := fn.Type()
	if ftype.NumIn() != 1 {
		// TODO: give information of the actual arguments
		panic(ftype.Name() + " argument should be a writable channel only")
	}
	arg := ftype.In(0)
	if arg.Kind() != reflect.Chan || arg.ChanDir()&reflect.SendDir == 0 {
		// TODO: give information of the actual arguments
		panic(ftype.Name() + " argument should be a writable channel only")
	}
}

// an ending function has input but not output. After attaching it, the pipeline is ended
func isEndingFunction(fn reflect.Value) bool {
	ftype := fn.Type()
	if ftype.NumIn() != 1 {
		return false
	}
	arg := ftype.In(0)
	if arg.Kind() != reflect.Chan || arg.ChanDir()&reflect.RecvDir == 0 {
		// TODO: give information of the actual arguments
		panic(ftype.Name() + " arguments should be a readable channel only, or both readable and writable channels")
	}
	return true
}

func checkArgumentAreReadableWritableChannels(fn reflect.Value) {
	ftype := fn.Type()
	if ftype.NumIn() != 2 {
		// TODO: give information of the actual arguments
		panic(ftype.Name() + " arguments should be a readable plus a writable channel, or a readable channel only")
	}
	arg := ftype.In(0)
	if arg.Kind() != reflect.Chan || arg.ChanDir()&reflect.RecvDir == 0 {
		// TODO: give information of the actual arguments
		panic(ftype.Name() + " fist argument should be a readable channel")
	}
	arg = ftype.In(1)
	if arg.Kind() != reflect.Chan || arg.ChanDir()&reflect.SendDir == 0 {
		// TODO: give information of the actual arguments
		panic(ftype.Name() + " second argument should be a writable channel")
	}
}

func checkInputIsCompatibleWithPreviousStage(previousStage, currentStage reflect.Value) {
	var outCh reflect.Type
	if previousStage.Type().NumIn() == 1 {
		outCh = previousStage.Type().In(0)
	} else {
		outCh = previousStage.Type().In(1)
	}
	inCh := currentStage.Type().In(0)
	if !outCh.Elem().AssignableTo(inCh.Elem()) {
		panic(fmt.Sprintf("%s input channel type %q can't get assigned to previous stage's %q output type",
			currentStage.Type(), inCh, outCh))
	}
}

func makeChannel(inType reflect.Type, bufLen int) reflect.Value {
	chanType := reflect.ChanOf(reflect.BothDir, inType)
	return reflect.MakeChan(chanType, bufLen)
}

func runStartGoroutine(fn reflect.Value) reflect.Value {
	fnType := fn.Type()
	outCh := makeChannel(fnType.In(0).Elem(), channelsBuf)
	go func() {
		fn.Call([]reflect.Value{outCh})
		outCh.Close()
	}()
	return outCh
}

func runEndGoroutine(fn, inCh reflect.Value) {
	go fn.Call([]reflect.Value{inCh})
}

func runStageGoroutine(fn, inCh reflect.Value) reflect.Value {
	fnType := fn.Type()
	outCh := makeChannel(fnType.In(1).Elem(), channelsBuf)
	go func() {
		fn.Call([]reflect.Value{inCh, outCh})
		outCh.Close()
	}()
	return outCh
}

func nilValue() *reflect.Value {
	var nv *interface{}
	vo := reflect.ValueOf(nv)
	return &vo
}
