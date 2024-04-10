package graph

import (
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mariomac/pipes/pkg/node"
	helpers "github.com/mariomac/pipes/pkg/test"
)

// Somme nodes for later testing

type Counter struct{}

func CounterProvider(_ Counter) (node.StartDemuxFunc, error) {
	return func(outs node.Demux) {
		nonPositive := node.DemuxGet[int](outs, "nonPos")
		positive := node.DemuxGet[int](outs, "positive")
		for i := -9; i < 10; i += 3 {
			if i <= 0 {
				nonPositive <- i
			} else {
				positive <- i
			}
		}
	}, nil
}

type Inverter struct {
	disabled bool
}

func (a Inverter) Enabled() bool {
	return !a.disabled
}
func InverterProvider(_ Inverter) (node.MiddleDemuxFunc[int], error) {
	return func(in <-chan int, outs node.Demux) {
		collector := node.DemuxGet[int](outs, "collect")
		sorter := node.DemuxGet[int](outs, "sort")
		for i := range in {
			collector <- i
			sorter <- -i
		}
	}, nil
}

type Adder struct{}

func AdderProvider(_ Adder) (node.MiddleFunc[int, int], error) {
	return func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- i + 1
		}
	}, nil
}

type Collector struct {
	dst []int
}

func CollectorProvider(coll *Collector) (node.TerminalFunc[int], error) {
	return func(in <-chan int) {
		for i := range in {
			coll.dst = append(coll.dst, i)
		}
	}, nil
}

type Sorter struct {
	dst []int
}

func SorterProvider(coll *Sorter) (node.TerminalFunc[int], error) {
	return func(in <-chan int) {
		for i := range in {
			coll.dst = append(coll.dst, i)
		}
		slices.Sort(coll.dst)
	}, nil
}

func TestDemuxed(t *testing.T) {
	graphDefinition := struct {
		Counter  `sendTo:"nonPos:Inverter,positive:Adder"`
		Inverter `forwardTo:"collect:Collector,sort:Sorter"`
		Adder    `sendTo:"Sorter"`
		*Collector
		*Sorter
	}{Collector: &Collector{}, Sorter: &Sorter{}}
	gb := NewBuilder()
	RegisterStartDemux(gb, CounterProvider)
	RegisterMiddleDemux(gb, InverterProvider)
	RegisterMiddle(gb, AdderProvider)
	RegisterTerminal(gb, SorterProvider)
	RegisterTerminal(gb, CollectorProvider)

	grp, err := gb.Build(graphDefinition)
	require.NoError(t, err)

	// run graph and wait for it to end
	endRun := make(chan struct{})
	go func() {
		defer close(endRun)
		grp.Run()
	}()
	helpers.ReadChannel(t, endRun, timeout)

	assert.Equal(t, []int{-9, -6, -3, 0}, graphDefinition.Collector.dst)
	assert.Equal(t, []int{0, 3, 4, 6, 7, 9, 10}, graphDefinition.Sorter.dst)
}

func TestDemuxed_Connector(t *testing.T) {
	graphDefinition := struct {
		Connector
		Counter
		Inverter
		Adder
		*Collector
		*Sorter
	}{
		Collector: &Collector{}, Sorter: &Sorter{},
		Connector: Connector{
			"Counter":  []string{"nonPos:Inverter", "positive:Adder"},
			"Inverter": []string{"collect:Collector", "sort:Sorter"},
			"Adder":    []string{"Sorter"},
		},
	}
	gb := NewBuilder()
	RegisterStartDemux(gb, CounterProvider)
	RegisterMiddleDemux(gb, InverterProvider)
	RegisterMiddle(gb, AdderProvider)
	RegisterTerminal(gb, SorterProvider)
	RegisterTerminal(gb, CollectorProvider)

	grp, err := gb.Build(graphDefinition)
	require.NoError(t, err)

	// run graph and wait for it to end
	endRun := make(chan struct{})
	go func() {
		defer close(endRun)
		grp.Run()
	}()
	helpers.ReadChannel(t, endRun, timeout)

	assert.Equal(t, []int{-9, -6, -3, 0}, graphDefinition.Collector.dst)
	assert.Equal(t, []int{0, 3, 4, 6, 7, 9, 10}, graphDefinition.Sorter.dst)
}

func TestDemuxed_ForwardTo(t *testing.T) {
	graphDefinition := struct {
		Counter `sendTo:"nonPos:Inverter,positive:Adder"`
		// won't submit anything to sorter but just forward to collector without inverting
		Inverter `forwardTo:"collect:Collector,sort:Sorter"`
		Adder    `sendTo:"Sorter"`
		*Collector
		*Sorter
	}{Collector: &Collector{}, Sorter: &Sorter{}, Inverter: Inverter{disabled: true}}
	gb := NewBuilder()
	RegisterStartDemux(gb, CounterProvider)
	RegisterMiddleDemux(gb, InverterProvider)
	RegisterMiddle(gb, AdderProvider)
	RegisterTerminal(gb, SorterProvider)
	RegisterTerminal(gb, CollectorProvider)

	grp, err := gb.Build(graphDefinition)
	require.NoError(t, err)

	// run graph and wait for it to end
	endRun := make(chan struct{})
	go func() {
		defer close(endRun)
		grp.Run()
	}()
	helpers.ReadChannel(t, endRun, timeout)

	// the collector has received the negative numbers directly from the start node
	assert.Equal(t, []int{-9, -6, -3, 0}, graphDefinition.Collector.dst)
	// sorter has received the negative numbers directly from the start node
	assert.Equal(t, []int{-9, -6, -3, 0, 4, 7, 10}, graphDefinition.Sorter.dst)
}

func TestDemuxed_ChannelBufferLen_Unbuffered(t *testing.T) {
	counterSubmits := atomic.Int32{}
	var countedCounterProvider = func(_ Counter) (node.StartDemuxFunc, error) {
		return func(out node.Demux) {
			ch1 := node.DemuxGet[int](out, "ch1")
			ch2 := node.DemuxGet[int](out, "ch2")
			for i := 1; i <= 10; i++ {
				ch1 <- i
				counterSubmits.Add(1)
				ch2 <- i * 10
				counterSubmits.Add(1)
			}
		}, nil
	}
	unblock := make(chan struct{}, 1)
	collected := make(chan int, 10)
	var blockableCollector = func(_ Collector) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for {
				<-unblock
				i, ok := <-in
				if !ok {
					return
				}
				collected <- i
			}
		}, nil
	}
	gb := NewBuilder()
	RegisterStartDemux(gb, countedCounterProvider)
	RegisterTerminal(gb, blockableCollector)
	grp, err := gb.Build(struct {
		Counter `sendTo:"ch1:Collector,ch2:Collector"` // both outs send to the same channel, for the sake of brevity
		Collector
	}{})
	require.NoError(t, err)
	go grp.Run()
	// check that the first submit is blocked until the collector reads the value
	select {
	case n := <-collected:
		assert.Fail(t, "unexpected value is collected", n)
	default: //ok!
	}
	assert.EqualValues(t, 0, counterSubmits.Load())
	// unblock and check that the collector received one and only one number
	unblock <- struct{}{}
	val := helpers.ReadChannel(t, collected, timeout)
	assert.EqualValues(t, 1, val)
	assert.Eventually(t, func() bool { return counterSubmits.Load() == 1 }, timeout, time.Millisecond)
	assert.EqualValues(t, 1, counterSubmits.Load())
	// check that the second submit is blocked until the collector reads the value
	select {
	case n := <-collected:
		assert.Fail(t, "unexpected value is collected", n)
	default: //ok!
	}
	// unblock few more and check that the collected and submit stats coincide
	unblock <- struct{}{}
	val = helpers.ReadChannel(t, collected, timeout)
	assert.EqualValues(t, 10, val)
	assert.Eventually(t, func() bool { return counterSubmits.Load() == 2 }, timeout, time.Millisecond)
	unblock <- struct{}{}
	val = helpers.ReadChannel(t, collected, timeout)
	assert.Equal(t, 2, val)
	unblock <- struct{}{}
	val = helpers.ReadChannel(t, collected, timeout)
	assert.Equal(t, 20, val)
	unblock <- struct{}{}
	val = helpers.ReadChannel(t, collected, timeout)
	assert.Equal(t, 3, val)
	assert.Eventually(t, func() bool { return counterSubmits.Load() == 5 }, timeout, time.Millisecond)
}

func TestDemuxed_ChannelBufferLen_Buffered(t *testing.T) {
	counterSubmits1 := atomic.Int32{}
	counterSubmits2 := atomic.Int32{}
	var countedCounterProvider = func(_ Counter) (node.StartDemuxFunc, error) {
		return func(out node.Demux) {
			ch1 := node.DemuxGet[int](out, "ch1")
			ch2 := node.DemuxGet[int](out, "ch2")
			for i := 1; i <= 10; i++ {
				ch1 <- i
				counterSubmits1.Add(1)
				ch2 <- i * 10
				counterSubmits2.Add(1)
			}
		}, nil
	}
	var blockedCollector1 = func(_ Collector) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			unblock := make(chan struct{}, 1)
			<-unblock
		}, nil
	}
	type Collector2 struct{}
	var blockedCollector2 = func(_ Collector2) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			unblock := make(chan struct{}, 1)
			<-unblock
		}, nil
	}
	gb := NewBuilder(node.ChannelBufferLen(3))
	RegisterStartDemux(gb, countedCounterProvider)
	RegisterTerminal(gb, blockedCollector1)
	RegisterTerminal(gb, blockedCollector2)
	grp, err := gb.Build(struct {
		Counter `sendTo:"ch1:Collector,ch2:Collector2"`
		Collector
		Collector2
	}{})
	require.NoError(t, err)
	go grp.Run()
	// check that the even if we don't unblock the collector, it is able to send 3 numbers on each channel
	assert.Eventually(t, func() bool { return counterSubmits2.Load() == 3 }, timeout, time.Millisecond)
	assert.EqualValues(t, 3, counterSubmits1.Load())
}

func TestDemuxed_Errors(t *testing.T) {
	var collectorValueProvider = func(c Collector) (node.TerminalFunc[int], error) {
		return CollectorProvider(&c)
	}
	t.Run("error if no destinations are defined", func(t *testing.T) {
		gb := NewBuilder()
		RegisterStartDemux(gb, CounterProvider)
		RegisterTerminal(gb, collectorValueProvider)
		_, err := gb.Build(struct {
			Counter
			Collector
		}{})
		assert.Error(t, err)
	})
	t.Run("error if non-demuxed has a chan:dst syntax", func(t *testing.T) {
		var nonDemuxedStartProvider = func(_ Counter) (node.StartFunc[int], error) {
			return func(out chan<- int) {}, nil
		}
		gb := NewBuilder()
		RegisterStart(gb, nonDemuxedStartProvider)
		RegisterTerminal(gb, collectorValueProvider)
		_, err := gb.Build(struct {
			Counter `sendTo:"chan:Collector"`
			Collector
		}{})
		assert.Error(t, err)
	})
	t.Run("error if demuxed has not a chan:dst syntax", func(t *testing.T) {
		gb := NewBuilder()
		RegisterStartDemux(gb, CounterProvider)
		RegisterTerminal(gb, collectorValueProvider)
		_, err := gb.Build(struct {
			Counter `sendTo:"Collector"`
			Collector
		}{})
		assert.Error(t, err)
	})
}
