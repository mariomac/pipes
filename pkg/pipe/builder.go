package pipe

import (
	refl "github.com/mariomac/go-pipes/pkg/internal/refl"
	"reflect"
)

// BuilderRunner can build pipelines and eventually run Them
type BuilderRunner interface {
	Builder
	// Run starts running in background all the pipeline stages
	Run()
}

// Builder allows building pipelines but not running them. Used for fork's sub-pipelines
type Builder interface {
	// Add a node to the pipeline
	Add(function StageFunction)
	// Fork the pipeline into two sub-pipelines that will receive the input from the previous
	Fork() (left, right Builder)
}

type builderRunner struct {
	ended             bool
	line              []node
	lastAddedFunction refl.Function
}

type sstage struct {
	fork     *fork
	function refl.Function
}

type fork struct {
	left  *builderRunner
	right *builderRunner
}

// Start defining a pipeline whose first node is a StartFunction passed as first argument. When
// the StartFunction finishes, its output channel will be automatically closed, causing the subsequent
// pipeline stages to end in cascade.
func Start(function StartFunction) BuilderRunner {
	fn := refl.WrapFunction(function)
	fn.AssertNumberOfArguments(1)
	fn.AssertArgumentIsDirectedChannel(0, reflect.SendDir)
	return &builderRunner{
		line:              []node{{function: fn}},
		lastAddedFunction: fn,
	}
}

// Add a new StageFunction to the pipeline node. The passed function should handle the case where
// the input channel is closed, and gracefully end. When the StageFunction ends,
// its output channel will be automatically closed, causing the subsequent
// pipeline stages to end in cascade.
func (b *builderRunner) Add(function StageFunction) {
	if b.ended {
		panic("this builderRunner has been ended by a final node or Fork method. Can't add more stages")
	}
	fn := refl.WrapFunction(function)
	// check if the function is an ending node (1 single output channel) or a middle node
	if fn.NumArgs() == 1 {
		// ending node: check that it only has a single read channel
		fn.AssertArgumentIsDirectedChannel(0, reflect.RecvDir)
		b.ended = true
	} else {
		// middle node. check that it has a read channel and a write channel
		fn.AssertNumberOfArguments(2)
		fn.AssertArgumentIsDirectedChannel(0, reflect.RecvDir)
		fn.AssertArgumentIsDirectedChannel(1, reflect.SendDir)
	}
	// Check that the last argument of the previous node (output channel) is assignable to the
	// first argument of the current node (input channel)
	previous := b.lastAddedFunction
	fn.AssertArgsConnectableChannels(0, previous, previous.NumArgs()-1)
	b.line = append(b.line, node{function: fn})
	b.lastAddedFunction = fn
}

// Fork splits the current pipeline in two sub-pipelines that will receive the output of the
// previous pipeline node in the parent pipeline. Both pipelines will receive the same input
// from its previous node.
// It returns Builder instances for bot sub-pipelines. Running the parent pipeline will also run
// the two sub-pipelines.
func (b *builderRunner) Fork() (left, right Builder) {
	if b.ended {
		panic("this builderRunner has been ended by the End or Fork method. Can't add more stages")
	}
	b.ended = true
	leftBR := &builderRunner{lastAddedFunction: b.lastAddedFunction}
	rightBR := &builderRunner{lastAddedFunction: b.lastAddedFunction}
	b.line = append(b.line, node{fork: &fork{left: leftBR, right: rightBR}})
	return leftBR, rightBR
}

// Run all the pipeline stages in background. Each pipeline node will run in its own goroutine.
func (b *builderRunner) Run() {
	checkPipelineIsEnded(b)
	b.run(refl.NilChannel())
}

func checkPipelineIsEnded(b *builderRunner) {
	if !b.ended {
		panic("pipeline is not ended. It (or all its forks) must invoke the End method")
	}
	lastItem := b.line[len(b.line)-1]
	if lastItem.fork != nil {
		checkPipelineIsEnded(lastItem.fork.left)
		checkPipelineIsEnded(lastItem.fork.right)
	}
}
