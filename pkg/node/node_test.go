package node_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mariomac/pipes/pkg/node"
	helpers "github.com/mariomac/pipes/pkg/test"
)

const timeout = 2 * time.Second

func TestBasicGraph(t *testing.T) {
	p := node.NewPipe()

	start1 := node.AddStart(p, Counter(1, 3))
	start2 := node.AddStart(p, Counter(6, 8))
	odds := node.AddMiddle(p, OddFilter)
	evens := node.AddMiddle(p, EvenFilter)
	oddsMsg := node.AddMiddle(p, Messager("odd"))
	evensMsg := node.AddMiddle(p, Messager("even"))
	collected := map[string]struct{}{}
	collector := node.AddTerminal(p, func(strs <-chan string) {
		for str := range strs {
			collected[str] = struct{}{}
		}
	})
	/*
		start1----\ /---start2
		  |       X      |
		evens<---/ \-->odds
		  |              |
		evensMsg      oddsMsg
		       \      /
		        printer
	*/
	start1.SendTo(evens, odds)
	start2.SendTo(evens, odds)
	odds.SendTo(oddsMsg)
	evens.SendTo(evensMsg)
	oddsMsg.SendTo(collector)
	evensMsg.SendTo(collector)

	p.Start()

	select {
	case <-p.Done():
	// ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for pipeline to complete")
	}

	assert.Equal(t, map[string]struct{}{
		"odd: 1":  {},
		"even: 2": {},
		"odd: 3":  {},
		"even: 6": {},
		"odd: 7":  {},
		"even: 8": {},
	}, collected)
}

func TestConfigurationOptions_UnbufferedChannelCommunication(t *testing.T) {
	p := node.NewPipe()

	graphIn, graphOut := make(chan int), make(chan int)
	unblockReads := make(chan struct{})
	endStart, endMiddle, endTerm := make(chan struct{}), make(chan struct{}), make(chan struct{})
	init := node.AddStart(p, func(out chan<- int) {
		n := <-graphIn
		out <- n
		close(endStart)
	})
	middle := node.AddMiddle(p, func(in <-chan int, out chan<- int) {
		<-unblockReads
		n := <-in
		out <- n
		close(endMiddle)
	})
	term := node.AddTerminal(p, func(in <-chan int) {
		n := <-in
		graphOut <- n
		close(endTerm)
	})
	init.SendTo(middle)
	middle.SendTo(term)
	p.Start()

	graphIn <- 123
	// Since the nodes are unbuffered, they are blocked and can't accept/process data until
	// the last node exports it
	select {
	case <-endStart:
		require.Fail(t, "expected that init node is still blocked")
	default: //ok!
	}
	select {
	case <-endMiddle:
		require.Fail(t, "expected that middle node is still blocked")
	default: //ok!
	}
	select {
	case <-endTerm:
		require.Fail(t, "expected that terminal node is still blocked")
	default: //ok!
	}
	// After the last stage has exported the data, the rest of the channels are unblocked
	close(unblockReads)
	select {
	case <-graphOut: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the terminal node to forward the data")
	}
	select {
	case <-endStart: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the init node to finish")
	}
	select {
	case <-endMiddle: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the middle node to finish")
	}
	select {
	case <-endTerm: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the terminal node to finish")
	}
}

func TestConfigurationOptions_BufferedChannelCommunication(t *testing.T) {
	p := node.NewPipe()

	graphIn, graphOut := make(chan int), make(chan int)
	endStart, endMiddle, endTerm := make(chan struct{}), make(chan struct{}), make(chan struct{})
	init := node.AddStart(p, func(out chan<- int) {
		n := <-graphIn
		out <- n
		close(endStart)
	})
	middle := node.AddMiddle(p, func(in <-chan int, out chan<- int) {
		n := <-in
		out <- n
		close(endMiddle)
	}, node.ChannelBufferLen(1))
	term := node.AddTerminal(p, func(in <-chan int) {
		n := <-in
		graphOut <- n
		close(endTerm)
	}, node.ChannelBufferLen(1))
	init.SendTo(middle)
	middle.SendTo(term)
	p.Start()

	graphIn <- 123
	// Since the nodes are buffered, they can keep accepting/processing data even if the last
	// node hasn't exported it

	select {
	case <-endStart: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the init node to finish")
	}
	select {
	case <-endMiddle: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the middle node to finish")
	}
	select {
	case <-endTerm:
		require.Fail(t, "expected that terminal node is still blocked")
	default: //ok!
	}

	// unblock terminal node
	select {
	case <-graphOut: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the terminal node to forward the data")
	}
	select {
	case <-endTerm: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the terminal node to finish")
	}

}

func TestNilNodes(t *testing.T) {
	p := node.NewPipe()
	nilStart := node.AddStart[int](p, nil)
	start := node.AddStart(p, Counter(1, 3))
	var collected []int
	nilTerminal := node.AddTerminal[int](p, nil)
	collector := node.AddTerminal(p, func(ints <-chan int) {
		for i := range ints {
			collected = append(collected, i)
		}
	})
	// test that a nil start just don't crashes. It's just ignored
	assert.NotPanics(t, func() {
		start.SendTo(collector, nilTerminal)
		nilStart.SendTo(collector, nilTerminal)
		p.Start()

		helpers.ReadChannel(t, p.Done(), timeout)
	})
}

func Counter(from, to int) node.StartFunc[int] {
	return func(out chan<- int) {
		for i := from; i <= to; i++ {
			out <- i
		}
	}
}

func OddFilter(in <-chan int, out chan<- int) {
	for n := range in {
		if n%2 == 1 {
			out <- n
		}
	}
}

func EvenFilter(in <-chan int, out chan<- int) {
	for n := range in {
		if n%2 == 0 {
			out <- n
		}
	}
}

func Messager(msg string) func(in <-chan int, out chan<- string) {
	return func(in <-chan int, out chan<- string) {
		for n := range in {
			out <- fmt.Sprintf("%s: %d", msg, n)
		}
	}
}
