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
