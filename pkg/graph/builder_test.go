package graph

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/mariomac/pipes/pkg/node"
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
	RegisterStart(nb, func(cfg startConfig) (node.StartFunc[int], error) {
		return func(out chan<- int) {
			out <- 1
			out <- 2
			close(startEnded)
		}, nil
	})
	RegisterTerminal(nb, func(cfg endConfig) (node.TerminalFunc[int], error) {
		return func(in <-chan int) {}, nil
	})
	graph, err := nb.Build(config{
		Start:     startConfig{Instance: "1"},
		End:       endConfig{Instance: "2"},
		Connector: map[string][]string{"1": {"2"}},
	})
	require.NoError(t, err)
	go graph.Run()
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
	RegisterStart(b, func(_ stCfg) (node.StartFunc[string], error) {
		return func(out chan<- string) {
			out <- "1"
			out <- "2"
			out <- "3"
		}, nil
	})
	type midCfg struct{}
	RegisterMiddle(b, func(_ midCfg) (node.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for i := range in {
				out <- i * 2
			}
		}, nil
	})
	type termCfg struct{}
	arr := make([]string, 0, 3)
	done := make(chan struct{})
	RegisterTerminal(b, func(_ termCfg) (node.TerminalFunc[string], error) {
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
	g, err := b.Build(cfg{Connector: Connector{
		"st":  []string{"mid"},
		"mid": []string{"term"},
	}})
	require.NoError(t, err)

	go g.Run()
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
		SomeExtraField int        `nodeId:"-"` // this needs to be ignored
		Start          CounterCfg `nodeId:"n1" sendTo:"n2"`
		Middle         DoublerCfg `nodeId:"n2" sendTo:"n3"`
		Term           MapperCfg  `nodeId:"n3"`
		Connector
	}
	map1 := map[int]struct{}{}
	g, err := b.Build(config{
		Start:  CounterCfg{From: 1, To: 5},
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
			RegisterStart(b, func(cfg StartCfg) (node.StartFunc[int], error) {
				if cfg.fail {
					return nil, testError("start")
				}
				return func(_ chan<- int) {}, nil
			})
			RegisterMiddle(b, func(cfg MiddleCfg) (node.MiddleFunc[int, int], error) {
				if cfg.fail {
					return nil, testError("middle")
				}
				return func(_ <-chan int, _ chan<- int) {}, nil
			})
			RegisterTerminal(b, func(cfg TermCfg) (node.TerminalFunc[int], error) {
				if cfg.fail {
					return nil, testError("terminal")
				}
				return func(_ <-chan int) {}, nil
			})
			_, err := b.Build(tc.cfg)
			require.Error(t, err)
			var tgt testError
			require.ErrorAs(t, err, &tgt)
			assert.Equal(t, tc.expected, tgt)
		})
	}

}
