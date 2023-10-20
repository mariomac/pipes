package graph

import (
	"slices"
	"testing"

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
		Inverter `sendTo:"collect:Collector,sort:Sorter"`
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
		Inverter `sendTo:"collect:Collector,sort:Sorter" forwardTo:"Collector"`
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

	assert.Equal(t, []int{-9, -6, -3, 0}, graphDefinition.Collector.dst)
	// sorter hasn't received the negative inverter numbers
	assert.Equal(t, []int{4, 7, 10}, graphDefinition.Sorter.dst)
}

// TEST: forwarded nodes
// TEST: Connector Implementer
// TEST: options (e.g. channelbufferlen)
// TEST: fail if a demuxed node has no "chan:dst" sendTo syntax
// TEST: fail if a non-demuxed node has "chan:dst" sendTo sytax
// TEST: fail if types are different
// TEST: codecs
