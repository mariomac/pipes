package node

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"slices"

	helpers "github.com/mariomac/pipes/pkg/test"
)

const testTimeout = 5 * time.Second

func TestAsStartDemux(t *testing.T) {
	type out2k struct{}
	type out1k struct{}
	start := AsStartDemux(func(d Demux) {
		out1 := DemuxGet[int](d, out1k{})
		out2 := DemuxGet[int](d, out2k{})
		out1 <- 1
		out2 <- 10
		out1 <- 60
		out2 <- 30
	})
	doubler := AsMiddle(func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- int(i * 2)
		}
	})
	decer := AsMiddle(func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- int(i - 1)
		}
	})
	divider := AsMiddle(func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- int(i / 2)
		}
	})
	var sorted []int
	waiter := helpers.AsyncWait(1)
	sorter := AsTerminal(func(in <-chan int) {
		for i := range in {
			sorted = append(sorted, i)
		}
		slices.Sort(sorted)
		waiter.Done()
	})
	do := DemuxAdd[int](start, out1k{})
	do.SendTo(doubler, decer)
	DemuxAdd[int](start, out2k{}).SendTo(divider)
	decer.SendTo(sorter)
	doubler.SendTo(sorter)
	divider.SendTo(sorter)

	go start.Start()

	waiter.Wait(t, testTimeout)

	assert.Equal(t, []int{0, 2, 5, 15, 59, 120}, sorted)

}

func TestAsMiddleDemux(t *testing.T) {
	start := AsStart(func(out chan<- int) {
		for i := 0; i < 10; i++ {
			out <- i
		}
	})
	classifier := AsMiddleDemux(func(in <-chan int, out Demux) {
		evens := DemuxGet[int32](out, "evens")
		odds := DemuxGet[int](out, "odds")
		for i := range in {
			if i%2 == 0 {
				evens <- int32(i)
			} else {
				odds <- i
			}
		}
	})
	doubler := AsMiddle(func(in <-chan int32, out chan<- int) {
		for i := range in {
			out <- int(i * 2)
		}
	})
	var sorted []int
	waiter := helpers.AsyncWait(1)
	sorter := AsTerminal(func(in <-chan int) {
		for i := range in {
			sorted = append(sorted, i)
		}
		slices.Sort(sorted)
		waiter.Done()
	})
	start.SendTo(classifier)
	DemuxAdd[int32](classifier, "evens").SendTo(doubler)
	DemuxAdd[int](classifier, "odds").SendTo(sorter)
	doubler.SendTo(sorter)

	go start.Start()

	waiter.Wait(t, testTimeout)

	assert.Equal(t, []int{0, 1, 3, 4, 5, 7, 8, 9, 12, 16}, sorted)
}

func TestDemux_Unbuffered(t *testing.T) {
	graphIn := make(chan int)
	unblockReads := make(chan struct{})
	endStart1, endStart2, endTerm := make(chan struct{}), make(chan struct{}), make(chan struct{})
	endMiddle1, endMiddle2, endMiddle3, endMiddle4 := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	init := AsStartDemux(func(out Demux) {
		out1 := DemuxGet[int](out, "out1")
		out2 := DemuxGet[int](out, "out2")
		n := <-graphIn
		out1 <- n
		close(endStart1)
		out2 <- n
		close(endStart2)
	})
	middle1 := AsMiddleDemux(func(in <-chan int, out Demux) {
		out1 := DemuxGet[int](out, "out1")
		out2 := DemuxGet[int](out, "out2")
		<-unblockReads
		n := <-in
		out1 <- n
		close(endMiddle1)
		out2 <- n
		close(endMiddle2)
	})
	middle2 := AsMiddleDemux(func(in <-chan int, out Demux) {
		out1 := DemuxGet[int](out, "out1")
		out2 := DemuxGet[int](out, "out2")
		<-unblockReads
		n := <-in
		out1 <- n
		close(endMiddle3)
		out2 <- n
		close(endMiddle4)
	})
	term := AsTerminal(func(in <-chan int) {
		// should receive 4 messages: two from start duplicated on each middle
		for i := 0; i < 4; i++ {
			<-in
		}
		close(endTerm)
	})
	DemuxAdd[int](init, "out1").SendTo(middle1)
	DemuxAdd[int](init, "out2").SendTo(middle2)
	DemuxAdd[int](middle1, "out1").SendTo(term)
	DemuxAdd[int](middle1, "out2").SendTo(term)
	DemuxAdd[int](middle2, "out1").SendTo(term)
	DemuxAdd[int](middle2, "out2").SendTo(term)
	init.Start()

	graphIn <- 123
	// Since the nodes are unbuffered, they are blocked and can't accept/process data until
	// the last node exports it
	select {
	case <-endStart1:
		require.Fail(t, "expected that init node is still blocked")
	case <-endStart2:
		require.Fail(t, "expected that init node is still blocked")
	case <-endMiddle1:
		require.Fail(t, "expected that middle node is still blocked")
	case <-endMiddle2:
		require.Fail(t, "expected that middle node is still blocked")
	case <-endTerm:
		require.Fail(t, "expected that terminal node is still blocked")
	default: //ok!
	}
	// After the last stage has exported the data, the rest of the channels are unblocked
	close(unblockReads)
	select {
	case <-endStart1: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the init node to finish")
	}
	select {
	case <-endStart2: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the init node to finish")
	}
	select {
	case <-endMiddle1: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the middle node to finish")
	}
	select {
	case <-endMiddle2: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the middle node to finish")
	}
	select {
	case <-endTerm: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the terminal node to finish")
	}
}

func TestDemux_Buffered(t *testing.T) {
	graphIn, graphOut := make(chan int), make(chan int)
	endStart, endMiddle, endTerm := make(chan struct{}), make(chan struct{}), make(chan struct{})
	init := AsStartDemux(func(out Demux) {
		out1 := DemuxGet[int](out, "out1")
		out2 := DemuxGet[int](out, "out2")
		n := <-graphIn
		out1 <- n
		out2 <- n
		close(endStart)
	})
	middle1 := AsMiddleDemux(func(in <-chan int, out Demux) {
		out1 := DemuxGet[int](out, "out1")
		out2 := DemuxGet[int](out, "out2")
		n := <-in
		out1 <- n
		out2 <- n
		close(endMiddle)
	}, ChannelBufferLen(1))
	middle2 := AsMiddleDemux(func(in <-chan int, out Demux) {
		out1 := DemuxGet[int](out, "out1")
		out2 := DemuxGet[int](out, "out2")
		n := <-in
		out1 <- n
		out2 <- n
		close(endMiddle)
	}, ChannelBufferLen(1))
	term := AsTerminal(func(in <-chan int) {
		n := <-in
		graphOut <- n
		close(endTerm)
	}, ChannelBufferLen(1))
	DemuxAdd[int](init, "out1").SendTo(middle1)
	DemuxAdd[int](init, "out2").SendTo(middle2)
	DemuxAdd[int](middle1, "out1").SendTo(term)
	DemuxAdd[int](middle1, "out2").SendTo(term)
	DemuxAdd[int](middle2, "out1").SendTo(term)
	DemuxAdd[int](middle2, "out2").SendTo(term)
	init.Start()

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

func TestDemux_Error_NoOutput(t *testing.T) {
	t.Run("fail if a node hasn't any output", func(t *testing.T) {
		init := AsStartDemux(func(d Demux) {})
		assert.Panics(t, init.Start)
	})
	t.Run("fail if it defines an output that has no connections", func(t *testing.T) {
		init := AsStartDemux(func(d Demux) {})
		end := AsTerminal(func(in <-chan int) {})
		DemuxAdd[int](init, "out1").SendTo(end)
		DemuxAdd[int](init, "out2")
		assert.Panics(t, init.Start)
	})
}

func TestDemux_Error_WrongDemuxKey(t *testing.T) {
	t.Run("panics if key does not exist", func(t *testing.T) {
		panicked := make(chan struct{})
		init := AsStartDemux(func(d Demux) {
			defer func() {
				if r := recover(); r != nil {
					close(panicked)
				}
			}()
			out := DemuxGet[int](d, "foo")
			out <- 1
		})
		end := AsTerminal(func(in <-chan int) {})
		DemuxAdd[int](init, "bar").SendTo(end)
		init.Start()
		select {
		case <-panicked: //ok!
		case <-time.After(timeout):
			t.Fatal("expected asStartDemux to panic!")
		}
	})
	t.Run("panics if key has a wrong type", func(t *testing.T) {
		panicked := make(chan struct{})
		init := AsStartDemux(func(d Demux) {
			defer func() {
				if r := recover(); r != nil {
					close(panicked)
				}
			}()
			out := DemuxGet[string](d, "foo")
			out <- "1"
		})
		end := AsTerminal(func(in <-chan int) {})
		DemuxAdd[int](init, "foo").SendTo(end)
		init.Start()
		select {
		case <-panicked: //ok!
		case <-time.After(timeout):
			t.Fatal("expected asStartDemux to panic!")
		}
	})
}
