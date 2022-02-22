package pipe

import (
	"github.com/mariomac/go-pipes/pkg/pipe/internal/refl"
)

// todo: set as a builderRunner configurable property
const channelsBuf = 20

// the connector is the output channel of the previous stage (nil for the first stage),
// that is used as input for the next stage.
func (b *builderRunner) run(connector *refl.Channel) {
	for _, invocation := range b.line {
		//if invocation.Fork != nil {
		//	log.Printf("%d: forking", i)
		//	return forkFn(invocation, nextInput)
		//}
		invoke(invocation.function, connector)
	}
}

// the connector is passed as argument to the function to be run. If the function returns a
// channel (first or middle stages), the connector is updated to it, so it will be passed to the
// next stage
func invoke(fn refl.Function, connector *refl.Channel) {
	if connector.IsNil() {
		// output-only function (first element of pipeline)
		*connector = fn.RunAsStartGoroutine(channelsBuf)
	} else if fn.NumArgs() == 1 {
		// input-only function (last element of pipeline)
		fn.RunAsEndGoroutine(*connector)
	} else {
		// intermediate stage of the pipeline with input and output channel
		*connector = fn.RunAsMiddleGoroutine(*connector, channelsBuf)
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
