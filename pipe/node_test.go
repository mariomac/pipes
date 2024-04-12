package pipe_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mariomac/pipes/pipe"
	helpers "github.com/mariomac/pipes/testers"
)

const timeout = 2 * time.Second

type basicGraph struct {
	start1 pipe.Start[int]
	start2 pipe.Start[int]

	odds  pipe.Middle[int, int]
	evens pipe.Middle[int, int]

	oddsMsg  pipe.Middle[int, string]
	evensMsg pipe.Middle[int, string]

	collector pipe.Final[string]
}

func (b *basicGraph) Connect() {
	b.start1.SendTo(b.evens, b.odds)
	b.start2.SendTo(b.evens, b.odds)
	b.evens.SendTo(b.evensMsg)
	b.odds.SendTo(b.oddsMsg)
	b.evensMsg.SendTo(b.collector)
	b.oddsMsg.SendTo(b.collector)
}

func start1(b *basicGraph) *pipe.Start[int]            { return &b.start1 }
func start2(b *basicGraph) *pipe.Start[int]            { return &b.start2 }
func odds(b *basicGraph) *pipe.Middle[int, int]        { return &b.odds }
func evens(b *basicGraph) *pipe.Middle[int, int]       { return &b.evens }
func oddsMsg(b *basicGraph) *pipe.Middle[int, string]  { return &b.oddsMsg }
func evensMsg(b *basicGraph) *pipe.Middle[int, string] { return &b.evensMsg }
func collector(b *basicGraph) *pipe.Final[string]      { return &b.collector }

func TestBasicGraph(t *testing.T) {
	p := pipe.NewBuilder(&basicGraph{})

	pipe.AddStart(p, start1, Counter(1, 3))
	pipe.AddStart(p, start2, Counter(6, 8))
	pipe.AddMiddle(p, odds, OddFilter)
	pipe.AddMiddle(p, evens, EvenFilter)
	pipe.AddMiddle(p, oddsMsg, Messager("odd"))
	pipe.AddMiddle(p, evensMsg, Messager("even"))
	collected := map[string]struct{}{}
	pipe.AddFinal(p, collector, func(strs <-chan string) {
		for str := range strs {
			collected[str] = struct{}{}
		}
	})

	r, err := p.Build()
	require.NoError(t, err)

	r.Start()
	helpers.ReadChannel(t, r.Done(), timeout)

	assert.Equal(t, map[string]struct{}{
		"odd: 1":  {},
		"even: 2": {},
		"odd: 3":  {},
		"even: 6": {},
		"odd: 7":  {},
		"even: 8": {},
	}, collected)
}

type smfPipe struct {
	start pipe.Start[int]
	mid   pipe.Middle[int, int]
	final pipe.Final[int]
}

func (b *smfPipe) Connect() {
	b.start.SendTo(b.mid)
	b.mid.SendTo(b.final)
}

func start(b *smfPipe) *pipe.Start[int]     { return &b.start }
func mid(b *smfPipe) *pipe.Middle[int, int] { return &b.mid }
func final(b *smfPipe) *pipe.Final[int]     { return &b.final }

func TestConfigurationOptions_UnbufferedChannelCommunication(t *testing.T) {
	p := pipe.NewBuilder(&smfPipe{})

	graphIn, graphOut := make(chan int), make(chan int)
	unblockReads := make(chan struct{})
	endStart, endMiddle, endTerm := make(chan struct{}), make(chan struct{}), make(chan struct{})
	pipe.AddStart(p, start, func(out chan<- int) {
		n := <-graphIn
		out <- n
		close(endStart)
	})
	pipe.AddMiddle(p, mid, func(in <-chan int, out chan<- int) {
		<-unblockReads
		n := <-in
		out <- n
		close(endMiddle)
	})
	pipe.AddFinal(p, final, func(in <-chan int) {
		n := <-in
		graphOut <- n
		close(endTerm)
	})

	r, err := p.Build()
	require.NoError(t, err)

	r.Start()
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
		require.Fail(t, "expected that doubler node is still blocked")
	default: //ok!
	}
	select {
	case <-endTerm:
		require.Fail(t, "expected that terminal node is still blocked")
	default: //ok!
	}
	// After the last stage has exported the data, the rest of the channels are unblocked
	close(unblockReads)
	assert.Equal(t, 123, helpers.ReadChannel(t, graphOut, timeout))
	helpers.ReadChannel(t, endStart, timeout)
	helpers.ReadChannel(t, endMiddle, timeout)
	helpers.ReadChannel(t, endTerm, timeout)
}

func TestConfigurationOptions_GlobalBufferedChannelCommunication(t *testing.T) {
	p := pipe.NewBuilder(&smfPipe{}, pipe.ChannelBufferLen(1))

	graphIn, graphOut := make(chan int), make(chan int)
	endStart, endMiddle, endTerm := make(chan struct{}), make(chan struct{}), make(chan struct{})
	pipe.AddStart(p, start, func(out chan<- int) {
		n := <-graphIn
		out <- n
		close(endStart)
	})
	pipe.AddMiddle(p, mid, func(in <-chan int, out chan<- int) {
		n := <-in
		out <- n
		close(endMiddle)
	})
	pipe.AddFinal(p, final, func(in <-chan int) {
		n := <-in
		graphOut <- n
		close(endTerm)
	})

	r, err := p.Build()
	require.NoError(t, err)

	r.Start()

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
		require.Fail(t, "timeout while waiting for the doubler node to finish")
	}
	select {
	case <-endTerm:
		require.Fail(t, "expected that terminal node is still blocked")
	default: //ok!
	}

	// unblock terminal node
	select {
	case o := <-graphOut: //ok!
		assert.Equal(t, 123, o)
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the terminal node to forward the data")
	}
	select {
	case <-endTerm: //ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for the terminal node to finish")
	}

}

type nilledPipe struct {
	start    pipe.Start[int]
	nilStart pipe.Start[int]
	final    pipe.Final[int]
	nilFinal pipe.Final[int]
}

func (b *nilledPipe) Connect() {
	b.start.SendTo(b.final, b.nilFinal)
	b.nilStart.SendTo(b.final, b.nilFinal)
}

func nStart(b *nilledPipe) *pipe.Start[int]    { return &b.start }
func nNilStart(b *nilledPipe) *pipe.Start[int] { return &b.nilStart }
func nFinal(b *nilledPipe) *pipe.Final[int]    { return &b.final }
func nNilFinal(b *nilledPipe) *pipe.Final[int] { return &b.nilFinal }

func TestNilNodes(t *testing.T) {
	p := pipe.NewBuilder(&nilledPipe{})
	pipe.AddStartProvider(p, nNilStart, func() (pipe.StartFunc[int], error) {
		return pipe.IgnoreStart[int](), nil
	})
	pipe.AddStart(p, nStart, Counter(1, 3))
	var collected []int
	pipe.AddFinalProvider(p, nNilFinal, func() (pipe.FinalFunc[int], error) {
		return pipe.IgnoreFinal[int](), nil
	})
	pipe.AddFinal(p, nFinal, func(ints <-chan int) {
		for i := range ints {
			collected = append(collected, i)
		}
	})
	// test that a nil start just don't crashes. It's just ignored
	assert.NotPanics(t, func() {
		r, err := p.Build()
		require.NoError(t, err)

		r.Start()

		helpers.ReadChannel(t, r.Done(), timeout)

		assert.Equal(t, []int{1, 2, 3}, collected)
	})
}

func Counter(from, to int) pipe.StartFunc[int] {
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
