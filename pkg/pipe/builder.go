package pipe

import (
	"reflect"

	"github.com/mariomac/go-pipes/pkg/pipe/internal/refl"
)

// TODO: when we are ready to integrate Go 1.18, enforce functions' type safety redefining the
// following types as:
//
// type StartFunction[OUT any] func(out chan<- OUT)
// type StageFunction[IN, OUT any] func(in <-chan IN, out chan<- OUT)
// type EndFunction[IN any] func(out <-chan IN)
//
// That would save us a lot of reflection checks at runtime

// StartFunction is a function that receives a writable channel as unique argument, and sends
// value to that channel during an indefinite amount of time
type StartFunction interface{}

// StageFunction is a function that can have two signatures:
// 1- If it's an intermediate pipeline stage, it receives a readable channel as first argument,
// and a writable channel as second argument.
// 2- If it's the end of the pipeline, it receives a readable channel as unique argument.
// It must process the inputs from the input channel until
// it's closed, and optionally forward the processed results to the output channel.
type StageFunction interface{}

// Builder can build pipelines and eventually run Them
type Builder interface {
	PartialBuilder
	// Run starts running in background all the pipeline stages
	Run()
}

// PartialBuilder allows building pipelines but not running them. Used for fork's sub-pipelines
type PartialBuilder interface {
	// Add a stage to the pipeline
	Add(function StageFunction)
	// Fork the pipeline into two sub-pipelines that will receive the input from the previous
	Fork() (left, right PartialBuilder)
}

type builderRunner struct {
	ended             bool
	line              []stage
	lastAddedFunction refl.Function
}

type stage struct {
	fork     *fork
	function refl.Function
}

type fork struct {
	left  builderRunner
	right builderRunner
}

// Start defining a pipeline whose first stage is a StartFunction passed as first argument. When
// the StartFunction finishes, its output channel will be automatically closed, causing the subsequent
// pipeline stages to end in cascade.
func Start(function StartFunction) Builder {
	fn := refl.WrapFunction(function)
	fn.AssertNumberOfArguments(1)
	fn.AssertArgumentIsDirectedChannel(0, reflect.SendDir)
	return &builderRunner{
		line:              []stage{{function: fn}},
		lastAddedFunction: fn,
	}
}

// Add a new StageFunction to the pipeline stage. The passed function should handle the case where
// the input channel is closed, and gracefully end. When the StageFunction ends,
// its output channel will be automatically closed, causing the subsequent
// pipeline stages to end in cascade.
func (b *builderRunner) Add(function StageFunction) {
	if b.ended {
		panic("this builderRunner has been ended by the End or Fork method. Can't add more stages")
	}
	fn := refl.WrapFunction(function)
	// check if the function is an ending stage (1 single output channel) or a middle stage
	if fn.NumArgs() == 1 {
		// ending stage: check that it only has a single read channel
		fn.AssertArgumentIsDirectedChannel(0, reflect.RecvDir)
		b.ended = true
	} else {
		// middle stage. check that it has a read channel and a write channel
		fn.AssertNumberOfArguments(2)
		fn.AssertArgumentIsDirectedChannel(0, reflect.RecvDir)
		fn.AssertArgumentIsDirectedChannel(1, reflect.SendDir)
	}
	// Check that the last argument of the previous stage (output channel) is assignable to the
	// first argument of the current stage (input channel)
	previous := b.lastAddedFunction
	fn.AssertArgsConnectableChannels(0, previous, previous.NumArgs()-1)
	b.line = append(b.line, stage{function: fn})
	b.lastAddedFunction = fn
}

// Fork TODO
func (b *builderRunner) Fork() (left, right PartialBuilder) {
	//TODO implement me
	panic("implement me")
}

// Run all the pipeline stages in background. Each pipeline stage will run in its own goroutine.
func (b *builderRunner) Run() {
	checkPipelineIsEnded(b)
	b.run(refl.Nil())
}

func checkPipelineIsEnded(b *builderRunner) {
	if !b.ended {
		panic("pipeline is not ended. It (or all its forks) must invoke the End method")
	}
	lastItem := b.line[len(b.line)-1]
	if lastItem.fork != nil {
		checkPipelineIsEnded(&lastItem.fork.left)
		checkPipelineIsEnded(&lastItem.fork.right)
	}
}
