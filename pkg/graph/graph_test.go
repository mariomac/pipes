package graph

import (
	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/mariomac/pipes/pkg/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	b := NewBuilder()

	type CounterCfg struct {
		stage.Instance
		From int
		To   int
	}
	type DoublerCfg struct {
		stage.Instance
	}
	type MapperCfg struct {
		stage.Instance
		Dst map[int]struct{}
	}
	type config struct {
		Starts []CounterCfg
		Middle DoublerCfg
		Term   []MapperCfg
		Connector
	}

	RegisterStart(b, func(cfg CounterCfg) node.StartFunc[int] {
		return func(out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}
	})
	RegisterMiddle(b, func(_ DoublerCfg) node.MiddleFunc[int, int] {
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- n * 2
			}
		}
	})
	RegisterTerminal(b, func(cfg MapperCfg) node.TerminalFunc[int] {
		return func(in <-chan int) {
			for n := range in {
				cfg.Dst[n] = struct{}{}
			}
		}
	})

	map1, map2 := map[int]struct{}{}, map[int]struct{}{}
	require.NoError(t, b.ApplyConfig(config{
		Starts: []CounterCfg{
			{From: 1, To: 5, Instance: "c1"},
			{From: 6, To: 8, Instance: "c2"}},
		Middle: DoublerCfg{Instance: "d"},
		Term: []MapperCfg{
			{Dst: map1, Instance: "m1"},
			{Dst: map2, Instance: "m2"}},
		Connector: map[string][]string{
			"c1": {"d"},
			"c2": {"d"},
			"d":  {"m1", "m2"},
		},
	}))

	g := b.Build()
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

	assert.Equal(t, map[int]struct{}{2: {}, 4: {}, 6: {}, 8: {}, 10: {}, 12: {}, 14: {}, 16: {}}, map1)
	assert.Equal(t, map1, map2)
}
