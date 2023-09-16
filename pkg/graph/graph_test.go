package graph

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/mariomac/pipes/pkg/node"
)

func TestBasic(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		stage.Instance
		From int
		To   int
	}
	RegisterStart(b, func(cfg CounterCfg) (node.StartFunc[int], error) {
		return func(out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	type DoublerCfg struct {
		stage.Instance
	}
	RegisterMiddle(b, func(_ DoublerCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- n * 2
			}
		}, nil
	})

	type MapperCfg struct {
		stage.Instance
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts CounterCfg
		Middle DoublerCfg
		Term   MapperCfg
		Connector
	}
	map1 := map[int]struct{}{}
	g, err := b.Build(config{
		Starts: CounterCfg{From: 1, To: 5, Instance: "c1"},
		Middle: DoublerCfg{Instance: "d"},
		Term:   MapperCfg{Dst: map1, Instance: "m1"},
		Connector: Connector{
			"c1": {"d"},
			"d":  {"m1"},
		},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{2: {}, 4: {}, 6: {}, 8: {}, 10: {}}, map1)
}

func TestNodeIdAsTag(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		From int
		To   int
	}
	RegisterStart(b, func(cfg CounterCfg) (node.StartFunc[int], error) {
		return func(out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	type DoublerCfg struct{}
	RegisterMiddle(b, func(_ DoublerCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- n * 2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts CounterCfg `nodeId:"s"`
		Middle DoublerCfg `nodeId:"m"`
		Term   MapperCfg  `nodeId:"t"`
		Connector
	}
	map1 := map[int]struct{}{}
	g, err := b.Build(config{
		Starts: CounterCfg{From: 1, To: 5},
		Middle: DoublerCfg{},
		Term:   MapperCfg{Dst: map1},
		Connector: Connector{
			"s": {"m"},
			"m": {"t"},
		},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{2: {}, 4: {}, 6: {}, 8: {}, 10: {}}, map1)
}

func TestSendsTo(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		From int
		To   int
	}
	RegisterStart(b, func(cfg CounterCfg) (node.StartFunc[int], error) {
		return func(out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	type DoublerCfg struct{}
	RegisterMiddle(b, func(_ DoublerCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- n * 2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts CounterCfg `nodeId:"s" sendTo:"m"`
		Middle DoublerCfg `nodeId:"m" sendTo:"t"`
		Term   MapperCfg  `nodeId:"t"`
	}
	map1 := map[int]struct{}{}
	g, err := b.Build(config{
		Starts: CounterCfg{From: 1, To: 5},
		Middle: DoublerCfg{},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{2: {}, 4: {}, 6: {}, 8: {}, 10: {}}, map1)
}

func TestSendsTo_WrongAnnotations(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		From int
		To   int
	}
	RegisterStart(b, func(cfg CounterCfg) (node.StartFunc[int], error) {
		return func(out chan<- int) {}, nil
	})

	type DoublerCfg struct{}
	RegisterMiddle(b, func(_ DoublerCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {}, nil
	})

	// Should fail because a node is missing a sendTo
	type config1 struct {
		Starts CounterCfg `nodeId:"s"`
		Middle DoublerCfg `nodeId:"m" sendTo:"t"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err := b.Build(config1{})
	assert.Error(t, err)

	// Should fail because the middle node is missing a sendTo
	type config2 struct {
		Starts CounterCfg `nodeId:"s" sendTo:"m"`
		Middle DoublerCfg `nodeId:"m"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err = b.Build(config2{})
	assert.Error(t, err)

	// Should fail because the middle node is sending to a start node
	type config3 struct {
		Starts CounterCfg `nodeId:"s"`
		Middle DoublerCfg `nodeId:"m"  sendTo:"s"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err = b.Build(config3{})
	assert.Error(t, err)

	// Should fail because a node cannot send to itself
	type config4 struct {
		Starts CounterCfg `nodeId:"s"  sendTo:"m,t"`
		Middle DoublerCfg `nodeId:"m"  sendTo:"m"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err = b.Build(config4{})
	assert.Error(t, err)

	// Should fail because a destination node does not exist
	type config5 struct {
		Starts CounterCfg `nodeId:"s" sendTo:"m,x"`
		Middle DoublerCfg `nodeId:"m" sendTo:"t"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err = b.Build(config5{})
	assert.Error(t, err)

	// Should fail because the middle node does not have any input
	type config6 struct {
		Starts CounterCfg `nodeId:"s" sendTo:"t"`
		Middle DoublerCfg `nodeId:"m" sendTo:"t"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err = b.Build(config6{})
	assert.Error(t, err)
}

type EnableCfg struct {
	Enable bool
	Add    int
}

var _ stage.Enabler = (*EnableCfg)(nil)

func (t EnableCfg) Enabled() bool {
	return t.Enable
}

func TestEnabled(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		From int
		To   int
	}
	RegisterStart(b, func(cfg CounterCfg) (node.StartFunc[int], error) {
		return func(out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	RegisterMiddle(b, func(c EnableCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- c.Add + n*2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts  CounterCfg `nodeId:"s" sendTo:"m1,m2"`
		Middle1 EnableCfg  `nodeId:"m1" sendTo:"t"`
		Middle2 EnableCfg  `nodeId:"m2" sendTo:"t"`
		Term    MapperCfg  `nodeId:"t"`
	}

	map1 := map[int]struct{}{}
	g, err := b.Build(config{
		Starts:  CounterCfg{From: 1, To: 5},
		Middle1: EnableCfg{Enable: true, Add: 10},
		Middle2: EnableCfg{Enable: false, Add: 100},
		Term:    MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{12: {}, 14: {}, 16: {}, 18: {}, 20: {}}, map1)
}

func TestForward_Enabled(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		From int
		To   int
	}
	RegisterStart(b, func(cfg CounterCfg) (node.StartFunc[int], error) {
		return func(out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	RegisterMiddle(b, func(c EnableCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- c.Add + n*2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts CounterCfg `nodeId:"s" sendTo:"m"`
		Middle EnableCfg  `nodeId:"m" forwardTo:"t"`
		Term   MapperCfg  `nodeId:"t"`
	}

	map1 := map[int]struct{}{}
	g, err := b.Build(config{
		Starts: CounterCfg{From: 1, To: 5},
		Middle: EnableCfg{Enable: true, Add: 10},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{12: {}, 14: {}, 16: {}, 18: {}, 20: {}}, map1)
}

func TestForward_Disabled(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		From int
		To   int
	}
	RegisterStart(b, func(cfg CounterCfg) (node.StartFunc[int], error) {
		return func(out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	RegisterMiddle(b, func(c EnableCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- c.Add + n*2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts CounterCfg `nodeId:"s" sendTo:"m"`
		Middle EnableCfg  `nodeId:"m" forwardTo:"t"`
		Term   MapperCfg  `nodeId:"t"`
	}

	map1 := map[int]struct{}{}
	g, err := b.Build(config{
		Starts: CounterCfg{From: 1, To: 5},
		Middle: EnableCfg{Enable: false, Add: 100},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{1: {}, 2: {}, 3: {}, 4: {}, 5: {}}, map1)
}

func TestForward_Disabled_Nil(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		From int
		To   int
	}
	RegisterStart(b, func(cfg CounterCfg) (node.StartFunc[int], error) {
		return func(out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	type NillableCfg struct {
		Add int
	}

	RegisterMiddle(b, func(c *NillableCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- c.Add + n*2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts CounterCfg   `nodeId:"s" sendTo:"m"`
		Middle *NillableCfg `nodeId:"m" forwardTo:"t"`
		Term   MapperCfg    `nodeId:"t"`
	}

	map1 := map[int]struct{}{}
	g, err := b.Build(config{
		Starts: CounterCfg{From: 1, To: 5},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{1: {}, 2: {}, 3: {}, 4: {}, 5: {}}, map1)
}

func TestForward_Disabled_Empty(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		From int
		To   int
	}
	RegisterStart(b, func(cfg CounterCfg) (node.StartFunc[int], error) {
		return func(out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	// If a slice node is tagged as a single node, we won't treat its elements as single nodes but
	// everything as a node
	type SliceCfg []int
	RegisterMiddle(b, func(c SliceCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- c[0] + n*2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts CounterCfg `nodeId:"s" sendTo:"m"`
		Middle SliceCfg   `nodeId:"m" forwardTo:"t"`
		Term   MapperCfg  `nodeId:"t"`
	}

	map1 := map[int]struct{}{}
	g, err := b.Build(config{
		Starts: CounterCfg{From: 1, To: 5},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{1: {}, 2: {}, 3: {}, 4: {}, 5: {}}, map1)
}

func TestMultiNode(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		From int
		To   int
	}

	RegisterMultiStart(b, func(cfg CounterCfg) ([]node.StartFunc[int], error) {
		return []node.StartFunc[int]{
			func(out chan<- int) {
				for i := cfg.From; i <= cfg.To; i++ {
					out <- i
				}
			},
			func(out chan<- int) {
				for i := cfg.From; i <= cfg.To; i++ {
					out <- i + 100
				}
			},
		}, nil
	})

	type DoublerCfg struct{}
	RegisterMiddle(b, func(_ DoublerCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- n * 2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts CounterCfg `nodeId:"s" sendTo:"m"`
		Middle DoublerCfg `nodeId:"m" sendTo:"t"`
		Term   MapperCfg  `nodeId:"t"`
	}
	map1 := map[int]struct{}{}
	g, err := b.Build(config{
		Starts: CounterCfg{From: 1, To: 3},
		Middle: DoublerCfg{},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{2: {}, 4: {}, 6: {}, 202: {}, 204: {}, 206: {}}, map1)
}

// This tests that we can combine struct annotations with Connector implementation
func TestBasic_CombineAnnotations(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		From int
		To   int
	}
	RegisterStart(b, func(cfg CounterCfg) (node.StartFunc[int], error) {
		return func(out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	type DoublerCfg struct {
	}
	RegisterMiddle(b, func(_ DoublerCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- n * 2
			}
		}, nil
	})

	type MapperCfg struct {
		stage.Instance
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts CounterCfg `nodeId:"s" sendTo:"m"`
		Middle DoublerCfg `nodeId:"m"`
		Term   MapperCfg
		Connector
	}
	map1 := map[int]struct{}{}
	g, err := b.Build(config{
		Starts: CounterCfg{From: 1, To: 5},
		Middle: DoublerCfg{},
		Term:   MapperCfg{Dst: map1, Instance: "t1"},
		Connector: Connector{
			"m": {"t1"},
		},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{2: {}, 4: {}, 6: {}, 8: {}, 10: {}}, map1)
}

// TODO: test combination using field names as default IDs.
// TEST: test combination with embedded fields
