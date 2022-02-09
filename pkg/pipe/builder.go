package pipe

import (
	"fmt"
	"log"
	"reflect"
)

var functions = map[string]reflect.Value{
	"ingest":      reflect.ValueOf(Ingest),
	"json2record": reflect.ValueOf(JSONToRecord),
	"appender":    reflect.ValueOf(Appender),
	"record2line": reflect.ValueOf(RecordToLine),
	"print":       reflect.ValueOf(Print),
}

type PipelineDefinition struct {
	Invocations []Invocation `yaml:"invocations"`
}

type Invocation struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args,omitempty"` // limited to string arguments
	Fork *Fork    `yaml:"fork,omitempty"`
}

type Fork struct {
	Left  []Invocation `yaml:"left"`
	Right []Invocation `yaml:"right"`
}

func Run(pipe PipelineDefinition) error {
	return run(pipe.Invocations, nil)
}

func run(invocations []Invocation, nextInput []reflect.Value) error {
	for i, invocation := range invocations {
		if invocation.Fork != nil {
			log.Printf("%d: forking", i)
			return fork(invocation, nextInput)
		}
		log.Printf("%d: invoking %+v", i, invocation)
		fn, ok := functions[invocation.Name]
		if !ok {
			return fmt.Errorf("unexisting function: %q", invocation.Name)
		}
		var err error
		nextInput, err = invoke(fn, nextInput, invocation.Args)
		if err != nil {
			return fmt.Errorf("invoking %q with args %v: %w", invocation.Name, invocation.Args, err)
		}
	}
	return nil

}

func fork(invocation Invocation, nextInput []reflect.Value) error {
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

func invoke(fn reflect.Value, args []reflect.Value, extraArgs []string) ([]reflect.Value, error) {
	arg := make(chan []reflect.Value, 1)
	err := make(chan error, 1)
	go func() {
		defer func() {
			if x := recover(); x != nil {
				err <- fmt.Errorf("invoking function: %v", x)
			}
			close(arg)
			close(err)
		}()
		for _, arg := range extraArgs {
			args = append(args, reflect.ValueOf(arg))
		}
		arg <- fn.Call(args)
	}()

	return <-arg, <-err
}
