package pipe

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const timeout = 5 * time.Second

// counter is a test start function that generates an incremental amount of numbers
func counter(upTo int) StartFunction {
	count := 0
	return func(out chan<- int) {
		for count < upTo {
			out <- count
			count++
		}
	}
}

// stringer is a test intermediate function that converts ints to string
func stringer(in <-chan int, out chan<- string) {
	for n := range in {
		out <- strconv.Itoa(n)
	}
}

// bolder is a test intermediate function that covers a string with '**' sings
func bolder(in <-chan string, out chan<- string) {
	for v := range in {
		out <- "**" + v + "**"
	}
}

// collectCloser is an end function that accumulates the received elements in a slice passed as
// argument. It also invokes the close function when all the messages have been processed
func collectCloser(slice *[]string, close func()) EndFunction {
	return func(in <-chan string) {
		for n := range in {
			*slice = append(*slice, n)
		}
		close()
	}
}

func TestMinimalPipeline(t *testing.T) {
	var collected []int
	completed := make(chan struct{})

	pipeline := Start(counter(4))
	pipeline.Add(func(in <-chan int) {
		for n := range in {
			collected = append(collected, n)
		}
		close(completed)
	})
	pipeline.Run()

	select {
	case <-completed:
	// ok
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting the pipeline to finish", collected)
	}
	assert.Equal(t, []int{0, 1, 2, 3}, collected)
}

func TestBasicPipeline(t *testing.T) {
	pipeline := Start(counter(4))
	pipeline.Add(stringer)
	pipeline.Add(bolder)
	var collected []string
	ctx, cancel := context.WithCancel(context.Background())
	pipeline.Add(collectCloser(&collected, cancel))
	pipeline.Run()
	select {
	case <-ctx.Done():
	// ok
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting the pipeline to finish", collected)
	}
	assert.Equal(t, []string{"**0**", "**1**", "**2**", "**3**"}, collected)
}

func TestRunNotEndedPipeline(t *testing.T) {
	assert.Panics(t, func() {
		pipeline := Start(counter(4))
		pipeline.Add(stringer)
		pipeline.Add(bolder)
		pipeline.Run()
	}, "a pipeline without an End stage won't run")
}

func TestBadPipelineFormation(t *testing.T) {
	assert.Panics(t, func() {
		_ = Start(func(out <-chan int) {})
	}, "must panic if the start channel is writable")
	assert.Panics(t, func() {
		_ = Start(func() {})
	}, "must panic if the start function has no arguments")
	assert.Panics(t, func() {
		_ = Start(func(in, out chan int) {})
	}, "must panic if the start function has more than one argument")
	assert.Panics(t, func() {
		p := Start(counter(3))
		p.Add(bolder)
	}, "must panic if the input of a stage does not match the type of the previous stage")
}
