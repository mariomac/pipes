package pipe

import (
	"log"
	"reflect"
)

// todo: set as a builderRunner configurable property
const channelsBuf = 20

func (b *builderRunner) run(connector *reflect.Value) {
	for i, invocation := range b.line {
		//if invocation.Fork != nil {
		//	log.Printf("%d: forking", i)
		//	return forkFn(invocation, nextInput)
		//}
		log.Printf("%d: invoking %+v", i, invocation)
		invoke(invocation.function, connector)
	}
}

func invoke(fn reflect.Value, connector *reflect.Value) {
	if connector.IsNil() {
		// output-only function (first element of pipeline)
		*connector = runStartGoroutine(fn)
	} else if fn.Type().NumIn() == 1 {
		// input-only function (last element of pipeline)
		runEndGoroutine(fn, *connector)
	} else {
		// intermediate stage of the pipeline with input and output channel
		*connector = runStageGoroutine(fn, *connector)
	}
}

/*

func forkFn(invocation Invocation, nextInput []reflect.Value) error {
	if len(nextInput) == 0 || nextInput[0].Kind() != reflect.Chan {
		panic("expected a channel. This is a bug. Debug it!")
	}
	leftCh, rightCh := forkChannels(nextInput[0])
	log.Printf("running left branch of fork")
	if err := run(invocation.Fork.Left,
		append([]reflect.Value{leftCh}, nextInput[1:]...)); err != nil {
		return err
	}
	log.Printf("running right branch of fork")
	// a fork presumes no more invocations after it, at its level
	// TODO: we can implement a JOIN
	return run(invocation.Fork.Right, append([]reflect.Value{rightCh}, nextInput[1:]...))
}

func forkChannels(inChan reflect.Value) (reflect.Value, reflect.Value) {
	chanType := reflect.ChanOf(reflect.BothDir, inChan.Type().Elem())
	out1 := reflect.MakeChan(chanType, inChan.Len())
	out2 := reflect.MakeChan(chanType, inChan.Len())
	go func() {
		for in, ok := inChan.Recv(); ok; in, ok = inChan.Recv() {
			out1.Send(in)
			out2.Send(in)
		}
	}()
	return out1, out2
}
*/