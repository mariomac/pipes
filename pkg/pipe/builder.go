package pipe

import (
	"reflect"
)

// TODO: when we are ready to integrate Go 1.18, enforce functions' type safety redefining the
// following types as:
// type StartFunction[OUT any] func(out chan<- OUT)
// type StageFunction[IN, OUT any] func(in <-chan IN, out chan<- OUT)
// type EndFunction[IN any] func(out <-chan IN)

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
	// not null for the cases of fork/join, when this line is feed by another previous line
	previousLine *builderRunner
	ended        bool
	line         []stage
}

type stage struct {
	fork     *fork
	function reflect.Value
}

type fork struct {
	left  builderRunner
	right builderRunner
}

// Start defining a pipeline whose first stage is a StartFunction passed as first argument. When
// the StartFunction finishes, its output channel will be automatically closed, causing the subsequent
// pipeline stages to end in cascade.
func Start(function StartFunction) Builder {
	funcVal := reflect.ValueOf(function)
	checkIsFunction(funcVal)
	checkArgumentIsWritableChannel(funcVal)
	return &builderRunner{line: []stage{{function: funcVal}}}
}

// Add a new StageFunction to the pipeline stage. The passed function should handle the case where
// the input channel is closed, and gracefully end. When the StageFunction ends,
// its output channel will be automatically closed, causing the subsequent
// pipeline stages to end in cascade.
func (b *builderRunner) Add(function StageFunction) {
	if b.ended {
		panic("this builderRunner has been ended by the End or Fork method. Can't add more stages")
	}
	funcVal := reflect.ValueOf(function)
	checkIsFunction(funcVal)
	if isEndingFunction(funcVal) {
		b.ended = true
	} else {
		checkArgumentAreReadableWritableChannels(funcVal)
	}
	checkInputIsCompatibleWithPreviousStage(
		b.findLastStageFunction(),
		funcVal)
	b.line = append(b.line, stage{function: funcVal})
}

// Fork TODO
func (b *builderRunner) Fork() (left, right PartialBuilder) {
	//TODO implement me
	panic("implement me")
}

// Run all the pipeline stages in background. Each pipeline stage will run in its own goroutine.
func (b *builderRunner) Run() {
	checkPipelineIsEnded(b)
	b.run(nilValue())
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

func (b *builderRunner) findLastStageFunction() reflect.Value {
	currentBuilder := b
	// we know this bucle ends as we force the pipeline
	// to contain at least one functional stage (method Start)
	for {
		stages := currentBuilder.line
		for len(stages) > 0 {
			if stages[len(stages)-1].fork == nil {
				return stages[len(stages)-1].function
			}
		}
		// look for the previous line
		currentBuilder = currentBuilder.previousLine
	}
}
