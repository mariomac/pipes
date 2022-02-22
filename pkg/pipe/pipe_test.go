package pipe

import (
	"context"
	"strconv"
	"sync"
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
func collectCloser(slice *[]string, close func()) StageFunction {
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
	assert.Panics(t, func() {
		p := Start(counter(3))
		p.Add(func(in <-chan int) {})
		p.Add(func(in <-chan int) {})
	}, "must panic if trying to add a pipeline stage after a terminal (input-only) stage")
	assert.Panics(t, func() {
		p := Start(counter(3))
		p.Add(func(in <-chan int) {})
		p.Fork()
	}, "must panic if trying to fork a pipeline that has a terminal stage")
}

func TestFork(t *testing.T) {
	finish := make(chan struct{})
	wait := sync.WaitGroup{}
	wait.Add(2)
	go func() {
		wait.Wait()
		close(finish)
	}()
	var collect []string
	pipeline := Start(counter(4))
	left, right := pipeline.Fork()
	left.Add(stringer)
	left.Add(func(in <-chan string) {
		for n := range in {
			collect = append(collect, n)
		}
		wait.Done()
	})
	factorial := 1
	right.Add(func(in <-chan int) {
		for n := range in {
			factorial *= n + 1
		}
		wait.Done()
	})
	pipeline.Run()
	select {
	case <-finish:
	// ok
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting the pipeline to finish")
	}
	assert.Equal(t, []string{"0", "1", "2", "3"}, collect)
	assert.EqualValues(t, 2*3*4, factorial)
}

func TestMultiFork(t *testing.T) {
	finish := make(chan struct{})
	wait := sync.WaitGroup{}
	wait.Add(3)
	go func() {
		wait.Wait()
		close(finish)
	}()
	pipeline := Start(counter(4))
	branch1, subfork := pipeline.Fork()
	// testing that, from a fork branch, you can keep forking
	branch2, branch3 := subfork.Fork()

	count, sum, last := 0, 0, 0
	branch1.Add(func(in <-chan int) {
		for _ = range in {
			count++
		}
		wait.Done()
	})
	branch2.Add(func(in <-chan int) {
		for n := range in {
			sum += n
		}
		wait.Done()
	})
	branch3.Add(func(in <-chan int) {
		for n := range in {
			last = n
		}
		wait.Done()
	})
	pipeline.Run()
	select {
	case <-finish:
	// ok
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting the pipeline to finish")
	}
	assert.Equal(t, 4, count)
	assert.Equal(t, 0+1+2+3, sum)
	assert.Equal(t, 3, last)
}
