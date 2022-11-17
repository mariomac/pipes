package graph

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/mariomac/pipes/pkg/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const timeout = time.Second

func TestOptions_BufferLen(t *testing.T) {
	type startConfig struct {
		stage.Instance
	}
	type endConfig struct {
		stage.Instance
	}
	type config struct {
		Start startConfig
		End   endConfig
		Connector
	}
	nb := NewBuilder(node.ChannelBufferLen(2))
	startEnded := make(chan struct{})
	RegisterStart(nb, func(cfg startConfig) node.StartFuncCtx[int] {
		return func(_ context.Context, out chan<- int) {
			out <- 1
			out <- 2
			close(startEnded)
		}
	})
	RegisterTerminal(nb, func(cfg endConfig) node.TerminalFunc[int] {
		return func(in <-chan int) {}
	})
	graph, err := nb.Build(config{
		Start:     startConfig{Instance: "1"},
		End:       endConfig{Instance: "2"},
		Connector: map[string][]string{"1": {"2"}},
	})
	require.NoError(t, err)
	go graph.Run(context.Background())
	select {
	case <-startEnded:
		//ok!
	case <-time.After(timeout):
		assert.Fail(t, "timeout! the terminal channel is not buffered")
	}
}

func TestCodecs(t *testing.T) {
	b := NewBuilder()
	// int 2 string codec
	RegisterCodec(b, func(in <-chan int, out chan<- string) {
		for i := range in {
			out <- strconv.Itoa(i)
		}
	})
	// string 2 int codec
	RegisterCodec(b, func(in <-chan string, out chan<- int) {
		for i := range in {
			o, err := strconv.Atoi(i)
			if err != nil {
				panic(err)
			}
			out <- o
		}
	})
	type stCfg struct{}
	RegisterStart(b, func(_ stCfg) node.StartFuncCtx[string] {
		return func(_ context.Context, out chan<- string) {
			out <- "1"
			out <- "2"
			out <- "3"
		}
	})
	type midCfg struct{}
	RegisterMiddle(b, func(_ midCfg) node.MiddleFunc[int, int] {
		return func(in <-chan int, out chan<- int) {
			for i := range in {
				out <- i * 2
			}
		}
	})
	type termCfg struct{}
	arr := make([]string, 0, 3)
	done := make(chan struct{})
	RegisterTerminal(b, func(_ termCfg) node.TerminalFunc[string] {
		return func(in <-chan string) {
			for i := range in {
				arr = append(arr, i)
			}
			close(done)
		}
	})

	type cfg struct {
		St   stCfg   `nodeId:"st"`
		Mid  midCfg  `nodeId:"mid"`
		Term termCfg `nodeId:"term"`
		Connector
	}
	g, err := b.Build(cfg{Connector: Connector{
		"st":  []string{"mid"},
		"mid": []string{"term"},
	}})
	require.NoError(t, err)

	go g.Run(context.Background())
	select {
	case <-done:
		//ok!
	case <-time.After(timeout):
		assert.Fail(t, "timeout while waiting for the graph to finish its execution")
	}

	assert.Equal(t, []string{"2", "4", "6"}, arr)

}
