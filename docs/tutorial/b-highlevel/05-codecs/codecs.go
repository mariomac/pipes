package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/mariomac/pipes/pkg/graph"
	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/mariomac/pipes/pkg/node"
)

type GeneratorConfig struct {
	stage.Instance
	Repeat     int
	Seed       int64
	LowerBound int
	UpperBound int
}

type PrinterConfig struct {
	stage.Instance
}

type Config struct {
	graph.Connector
	Generator GeneratorConfig
	Printer   PrinterConfig
}

func Generator(_ context.Context, cfg GeneratorConfig) (node.StartFuncCtx[int], error) {
	return func(_ context.Context, out chan<- int) {
		rand.Seed(cfg.Seed)
		for n := 0; n < cfg.Repeat; n++ {
			out <- cfg.LowerBound + rand.Intn(cfg.UpperBound-cfg.LowerBound)
		}
	}, nil
}

func Printer(_ context.Context, _ PrinterConfig) (node.TerminalFunc[string], error) {
	return func(in <-chan string) {
		for i := range in {
			fmt.Println("received: ", i)
		}
	}, nil
}

// IntStringCodec just converts ints to string. Since the Generator
// creates integers and the printer only accepts strings, we must
// create and register a codec that will be automatically wired when
// needed
func IntStringCodec(in <-chan int, out chan<- string) {
	for i := range in {
		out <- strconv.Itoa(i)
	}
}

func main() {
	gb := graph.NewBuilder()
	graph.RegisterCodec(gb, IntStringCodec)
	graph.RegisterStart(gb, Generator)
	graph.RegisterTerminal(gb, Printer)

	grp, err := gb.Build(context.Background(), Config{
		Generator: GeneratorConfig{
			Instance:   "generator",
			LowerBound: -10,
			UpperBound: 10,
			Seed:       time.Now().UnixNano(),
			Repeat:     5,
		},
		Printer: PrinterConfig{"printer"},
		Connector: graph.Connector{
			"generator": []string{"printer"},
		},
	})
	if err != nil {
		panic(err)
	}
	grp.Run(context.TODO())
}
