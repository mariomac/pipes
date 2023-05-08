package graph

import (
	"context"
	"strconv"
	"testing"
	"time"

	helpers "github.com/mariomac/pipes/pkg/test"

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
	RegisterStart(nb, func(_ context.Context, cfg startConfig) (node.StartFuncCtx[int], error) {
		return func(_ context.Context, out chan<- int) {
			out <- 1
			out <- 2
			close(startEnded)
		}, nil
	})
	RegisterTerminal(nb, func(_ context.Context, cfg endConfig) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {}, nil
	})
	graph, err := nb.Build(context.TODO(), config{
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
	RegisterStart(b, func(_ context.Context, _ stCfg) (node.StartFuncCtx[string], error) {
		return func(_ context.Context, out chan<- string) {
			out <- "1"
			out <- "2"
			out <- "3"
		}, nil
	})
	type midCfg struct{}
	RegisterMiddle(b, func(_ context.Context, _ midCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for i := range in {
				out <- i * 2
			}
		}, nil
	})
	type termCfg struct{}
	arr := make([]string, 0, 3)
	done := make(chan struct{})
	RegisterTerminal(b, func(_ context.Context, _ termCfg) (node.TerminalFunc[string], error) {
		return func(in <-chan string) {
			for i := range in {
				arr = append(arr, i)
			}
			close(done)
		}, nil
	})

	type cfg struct {
		St   stCfg   `nodeId:"st"`
		Mid  midCfg  `nodeId:"mid"`
		Term termCfg `nodeId:"term"`
		Connector
	}
	g, err := b.Build(context.TODO(), cfg{Connector: Connector{
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

func TestIgnore(t *testing.T) {
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
		SomeExtraField int        `nodeId:"-"` // this needs to be ignored
		Start          CounterCfg `nodeId:"n1" sendTo:"n2"`
		Middle         DoublerCfg `nodeId:"n2" sendTo:"n3"`
		Term           MapperCfg  `nodeId:"n3"`
		Connector
	}
	map1 := map[int]struct{}{}
	g, err := b.Build(context.TODO(), config{
		Start:  CounterCfg{From: 1, To: 5},
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

type testError string

func (t testError) Error() string {
	return string(t)
}

func TestBuildErrors(t *testing.T) {
	type StartCfg struct{ fail bool }
	type MiddleCfg struct{ fail bool }
	type TermCfg struct{ fail bool }
	type Cfg struct {
		Start  StartCfg  `nodeId:"s" sendTo:"m"`
		Middle MiddleCfg `nodeId:"m" sendTo:"t"`
		Term   TermCfg   `nodeId:"t"`
	}
	type testCase struct {
		expected testError
		cfg      Cfg
	}
	for _, tc := range []testCase{
		{cfg: Cfg{Start: StartCfg{fail: true}}, expected: testError("start")},
		{cfg: Cfg{Middle: MiddleCfg{fail: true}}, expected: testError("middle")},
		{cfg: Cfg{Term: TermCfg{fail: true}}, expected: testError("terminal")},
	} {
		t.Run(tc.expected.Error(), func(t *testing.T) {
			b := NewBuilder()
			RegisterStart(b, func(_ context.Context, cfg StartCfg) (node.StartFuncCtx[int], error) {
				if cfg.fail {
					return nil, testError("start")
				}
				return func(_ context.Context, _ chan<- int) {}, nil
			})
			RegisterMiddle(b, func(_ context.Context, cfg MiddleCfg) (node.MiddleFunc[int, int], error) {
				if cfg.fail {
					return nil, testError("middle")
				}
				return func(_ <-chan int, _ chan<- int) {}, nil
			})
			RegisterTerminal(b, func(_ context.Context, cfg TermCfg) (node.TerminalFunc[int], error) {
				if cfg.fail {
					return nil, testError("terminal")
				}
				return func(_ <-chan int) {}, nil
			})
			_, err := b.Build(context.TODO(), tc.cfg)
			require.Error(t, err)
			var tgt testError
			require.ErrorAs(t, err, &tgt)
			assert.Equal(t, tc.expected, tgt)
		})
	}

}

func TestBuildContexts(t *testing.T) {
	type Count struct{}
	type Multiply struct{}
	type Forward struct{}
	type Cfg struct {
		Start  Count    `nodeId:"s" sendTo:"m"`
		Middle Multiply `nodeId:"m" sendTo:"t"`
		Term   Forward  `nodeId:"t"`
	}
	b := NewBuilder(node.ChannelBufferLen(100))
	// This is not the way that contexts are meant to be used but we will try that they are properly passed
	RegisterStart(b, func(ctx context.Context, cfg Count) (node.StartFuncCtx[int], error) {
		cnt := ctx.Value("count").(int)
		return func(_ context.Context, out chan<- int) {
			for i := 1; i <= cnt; i++ {
				out <- i
			}
		}, nil
	})
	RegisterMiddle(b, func(ctx context.Context, cfg Multiply) (node.MiddleFunc[int, int], error) {
		mul := ctx.Value("multiply").(int)
		return func(in <-chan int, out chan<- int) {
			for n := range in {
				out <- n * mul
			}
		}, nil
	})
	RegisterTerminal(b, func(ctx context.Context, cfg Forward) (node.TerminalFunc[int], error) {
		fwd := ctx.Value("forward").(chan int)
		return func(in <-chan int) {
			for n := range in {
				fwd <- n
			}
			close(fwd)
		}, nil
	})

	fwd := make(chan int, 10)
	ctx := context.WithValue(context.WithValue(context.WithValue(
		context.TODO(), "count", 4), "multiply", 3), "forward", fwd)
	g, err := b.Build(ctx, Cfg{})
	// the context for Run doesn't have to be the same as the context for Build
	g.Run(context.TODO())
	require.NoError(t, err)
	assert.Equal(t, 3, helpers.ReadChannel(t, fwd, time.Second))
	assert.Equal(t, 6, helpers.ReadChannel(t, fwd, time.Second))
	assert.Equal(t, 9, helpers.ReadChannel(t, fwd, time.Second))
	assert.Equal(t, 12, helpers.ReadChannel(t, fwd, time.Second))
	n, ok := <-fwd
	assert.Falsef(t, ok, "did not expect more messages in the forward channel. Got %v", n)
}
