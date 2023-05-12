package graph

import (
	"context"
	"testing"
	"time"

	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/mariomac/pipes/pkg/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		stage.Instance
		From int
		To   int
	}
	RegisterStart(b, func(_ context.Context, cfg CounterCfg) (node.StartFuncCtx[int], error) {
		return func(_ context.Context, out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	type DoublerCfg struct {
		stage.Instance
	}
	RegisterMiddle(b, func(_ context.Context, _ DoublerCfg) (node.MiddleFunc[int, int], error) {
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
	RegisterTerminal(b, func(_ context.Context, cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts []CounterCfg
		Middle DoublerCfg
		Term   []MapperCfg
		Connector
	}
	map1, map2 := map[int]struct{}{}, map[int]struct{}{}
	g, err := b.Build(context.TODO(), config{
		Starts: []CounterCfg{
			{From: 1, To: 5, Instance: "c1"},
			{From: 6, To: 8, Instance: "c2"}},
		Middle: DoublerCfg{Instance: "d"},
		Term: []MapperCfg{
			{Dst: map1, Instance: "m1"},
			{Dst: map2, Instance: "m2"}},
		Connector: Connector{
			"c1": {"d"},
			"c2": {"d"},
			"d":  {"m1", "m2"},
		},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{2: {}, 4: {}, 6: {}, 8: {}, 10: {}, 12: {}, 14: {}, 16: {}}, map1)
	assert.Equal(t, map1, map2)
}

func TestContext(t *testing.T) {
	b := NewBuilder()

	type ReceiverCfg struct {
		stage.Instance
		Input chan int
	}
	RegisterStart(b, func(_ context.Context, cfg ReceiverCfg) (node.StartFuncCtx[int], error) {
		return func(ctx context.Context, out chan<- int) {
			for {
				select {
				case <-ctx.Done():
					return
				case i := <-cfg.Input:
					out <- i
				}
			}
		}, nil
	})

	type ForwarderCfg struct {
		stage.Instance
		Out chan int
	}
	allClosed := make(chan struct{})
	RegisterTerminal(b, func(_ context.Context, cfg ForwarderCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Out <- n
			}
			close(allClosed)
		}, nil
	})
	type config struct {
		Starts []ReceiverCfg
		Term   ForwarderCfg
		Connector
	}
	cfg := config{
		Starts: []ReceiverCfg{
			{Instance: "start1", Input: make(chan int, 10)},
			{Instance: "start2", Input: make(chan int, 10)},
		},
		Term: ForwarderCfg{Instance: "end", Out: make(chan int)},
		Connector: Connector{
			"start1": []string{"end"},
			"start2": []string{"end"},
		},
	}
	g, err := b.Build(context.TODO(), &cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go g.Run(ctx)

	// The graph works normally
	cfg.Starts[0].Input <- 123
	select {
	case o := <-cfg.Term.Out:
		assert.Equal(t, 123, o)
	case <-time.After(timeout):
		assert.Fail(t, "timeout while waiting for graph to forward items")
	}
	cfg.Starts[1].Input <- 456
	select {
	case o := <-cfg.Term.Out:
		assert.Equal(t, 456, o)
	case <-time.After(timeout):
		assert.Fail(t, "timeout while waiting for graph to forward items")
	}

	// after canceling context, the graph should not forward anything
	cancel()

	cfg.Starts[0].Input <- 789
	cfg.Starts[1].Input <- 101
	select {
	case o := <-cfg.Term.Out:
		assert.Failf(t, "graph should have been stopped", "unexpected output of the graph: %v", o)
	default:
		// OK!
	}

}

func TestNodeIdAsTag(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		From int
		To   int
	}
	RegisterStart(b, func(_ context.Context, cfg CounterCfg) (node.StartFuncCtx[int], error) {
		return func(_ context.Context, out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	type DoublerCfg struct{}
	RegisterMiddle(b, func(_ context.Context, _ DoublerCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- n * 2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(_ context.Context, cfg MapperCfg) (node.TerminalFunc[int], error) {
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
	g, err := b.Build(context.TODO(), config{
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
		g.Run(context.Background())
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
	RegisterStart(b, func(_ context.Context, cfg CounterCfg) (node.StartFuncCtx[int], error) {
		return func(_ context.Context, out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	type DoublerCfg struct{}
	RegisterMiddle(b, func(_ context.Context, _ DoublerCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- n * 2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(_ context.Context, cfg MapperCfg) (node.TerminalFunc[int], error) {
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
	g, err := b.Build(context.TODO(), config{
		Starts: CounterCfg{From: 1, To: 5},
		Middle: DoublerCfg{},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run(context.Background())
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
	RegisterStart(b, func(_ context.Context, cfg CounterCfg) (node.StartFuncCtx[int], error) {
		return func(_ context.Context, out chan<- int) {}, nil
	})

	type DoublerCfg struct{}
	RegisterMiddle(b, func(_ context.Context, _ DoublerCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(_ context.Context, cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {}, nil
	})

	// Should fail because a node is missing a sendTo
	type config1 struct {
		Starts CounterCfg `nodeId:"s"`
		Middle DoublerCfg `nodeId:"m" sendTo:"t"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err := b.Build(context.TODO(), config1{})
	assert.Error(t, err)

	// Should fail because the middle node is missing a sendTo
	type config2 struct {
		Starts CounterCfg `nodeId:"s" sendTo:"m"`
		Middle DoublerCfg `nodeId:"m"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err = b.Build(context.TODO(), config2{})
	assert.Error(t, err)

	// Should fail because the middle node is sending to a start node
	type config3 struct {
		Starts CounterCfg `nodeId:"s"`
		Middle DoublerCfg `nodeId:"m"  sendTo:"s"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err = b.Build(context.TODO(), config3{})
	assert.Error(t, err)

	// Should fail because a node cannot send to itself
	type config4 struct {
		Starts CounterCfg `nodeId:"s"  sendTo:"m,t"`
		Middle DoublerCfg `nodeId:"m"  sendTo:"m"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err = b.Build(context.TODO(), config4{})
	assert.Error(t, err)

	// Should fail because a destination node does not exist
	type config5 struct {
		Starts CounterCfg `nodeId:"s" sendTo:"m,x"`
		Middle DoublerCfg `nodeId:"m" sendTo:"t"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err = b.Build(context.TODO(), config5{})
	assert.Error(t, err)

	// Should fail because the middle node does not have any input
	type config6 struct {
		Starts CounterCfg `nodeId:"s" sendTo:"t"`
		Middle DoublerCfg `nodeId:"m" sendTo:"t"`
		Term   MapperCfg  `nodeId:"t"`
	}
	_, err = b.Build(context.TODO(), config6{})
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
	RegisterStart(b, func(_ context.Context, cfg CounterCfg) (node.StartFuncCtx[int], error) {
		return func(_ context.Context, out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	RegisterMiddle(b, func(_ context.Context, c EnableCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- c.Add + n*2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(_ context.Context, cfg MapperCfg) (node.TerminalFunc[int], error) {
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
	g, err := b.Build(context.TODO(), config{
		Starts:  CounterCfg{From: 1, To: 5},
		Middle1: EnableCfg{Enable: true, Add: 10},
		Middle2: EnableCfg{Enable: false, Add: 100},
		Term:    MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run(context.Background())
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
	RegisterStart(b, func(_ context.Context, cfg CounterCfg) (node.StartFuncCtx[int], error) {
		return func(_ context.Context, out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	RegisterMiddle(b, func(_ context.Context, c EnableCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- c.Add + n*2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(_ context.Context, cfg MapperCfg) (node.TerminalFunc[int], error) {
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
	g, err := b.Build(context.TODO(), config{
		Starts: CounterCfg{From: 1, To: 5},
		Middle: EnableCfg{Enable: true, Add: 10},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run(context.Background())
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
	RegisterStart(b, func(_ context.Context, cfg CounterCfg) (node.StartFuncCtx[int], error) {
		return func(_ context.Context, out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	RegisterMiddle(b, func(_ context.Context, c EnableCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- c.Add + n*2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(_ context.Context, cfg MapperCfg) (node.TerminalFunc[int], error) {
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
	g, err := b.Build(context.TODO(), config{
		Starts: CounterCfg{From: 1, To: 5},
		Middle: EnableCfg{Enable: false, Add: 100},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run(context.Background())
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
	RegisterStart(b, func(_ context.Context, cfg CounterCfg) (node.StartFuncCtx[int], error) {
		return func(_ context.Context, out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	type NillableCfg struct {
		Add int
	}

	RegisterMiddle(b, func(_ context.Context, c *NillableCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- c.Add + n*2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(_ context.Context, cfg MapperCfg) (node.TerminalFunc[int], error) {
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
	g, err := b.Build(context.TODO(), config{
		Starts: CounterCfg{From: 1, To: 5},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run(context.Background())
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
	RegisterStart(b, func(_ context.Context, cfg CounterCfg) (node.StartFuncCtx[int], error) {
		return func(_ context.Context, out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	// If a slice node is tagged as a single node, we won't treat its elements as single nodes but
	// everything as a node
	type SliceCfg []int
	RegisterMiddle(b, func(_ context.Context, c SliceCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- c[0] + n*2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(_ context.Context, cfg MapperCfg) (node.TerminalFunc[int], error) {
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
	g, err := b.Build(context.TODO(), config{
		Starts: CounterCfg{From: 1, To: 5},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run(context.Background())
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

	RegisterMultiStart(b, func(_ context.Context, cfg CounterCfg) ([]node.StartFuncCtx[int], error) {
		return []node.StartFuncCtx[int]{
			func(_ context.Context, out chan<- int) {
				for i := cfg.From; i <= cfg.To; i++ {
					out <- i
				}
			},
			func(_ context.Context, out chan<- int) {
				for i := cfg.From; i <= cfg.To; i++ {
					out <- i + 100
				}
			},
		}, nil
	})

	type DoublerCfg struct{}
	RegisterMiddle(b, func(_ context.Context, _ DoublerCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- n * 2
			}
		}, nil
	})

	type MapperCfg struct {
		Dst map[int]struct{}
	}
	RegisterTerminal(b, func(_ context.Context, cfg MapperCfg) (node.TerminalFunc[int], error) {
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
	g, err := b.Build(context.TODO(), config{
		Starts: CounterCfg{From: 1, To: 3},
		Middle: DoublerCfg{},
		Term:   MapperCfg{Dst: map1},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run(context.Background())
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
	RegisterStart(b, func(_ context.Context, cfg CounterCfg) (node.StartFuncCtx[int], error) {
		return func(_ context.Context, out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}, nil
	})

	type DoublerCfg struct {
	}
	RegisterMiddle(b, func(_ context.Context, _ DoublerCfg) (node.MiddleFunc[int, int], error) {
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
	RegisterTerminal(b, func(_ context.Context, cfg MapperCfg) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}, nil
	})

	type config struct {
		Starts CounterCfg `nodeId:"s" sendTo:"m"`
		Middle DoublerCfg `nodeId:"m"`
		Term   []MapperCfg
		Connector
	}
	map1, map2 := map[int]struct{}{}, map[int]struct{}{}
	g, err := b.Build(context.TODO(), config{
		Starts: CounterCfg{From: 1, To: 5},
		Middle: DoublerCfg{},
		Term: []MapperCfg{
			{Dst: map1, Instance: "t1"},
			{Dst: map2, Instance: "t2"}},
		Connector: Connector{
			"m": {"t1", "t2"},
		},
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		g.Run(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for graph to complete")
	}

	assert.Equal(t, map[int]struct{}{2: {}, 4: {}, 6: {}, 8: {}, 10: {}}, map1)
	assert.Equal(t, map1, map2)
}
