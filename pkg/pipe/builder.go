package pipe

import (
	"reflect"
)

// TODO: when we are ready to integrate Go 1.18, enforce functions' type safety with
// type StartFunction[OUT any] func(out chan<- OUT)
// type StageFunction[IN, OUT any] func(in <-chan IN, out chan<- OUT)
// type EndFunction[IN any] func(out <-chan IN)

type StartFunction interface{}
type StageFunction interface{}
type EndFunction interface{}

type Runner interface {
	PartialBuilder
	// Run is blocking. Until the pipeline is not finished it not ends
	Run()
}

type PartialBuilder interface {
	Add(function StageFunction)
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

func Start(function StartFunction) Runner {
	funcVal := reflect.ValueOf(function)
	checkIsFunction(funcVal)
	checkArgumentIsWritableChannel(funcVal)
	return &builderRunner{line: []stage{{function: funcVal}}}
}

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

func (b *builderRunner) Fork() (left, right PartialBuilder) {
	//TODO implement me
	panic("implement me")
}

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
