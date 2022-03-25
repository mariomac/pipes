package main

import (
	"fmt"
	"strconv"

	"github.com/mariomac/pipes/pkg/graph"
	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/mariomac/pipes/pkg/node"
)

type CounterConfig struct {
	From int
	To   int
}

var Counter = stage.StartProvider[CounterConfig, int]{
	ID: "counter",
	Function: func(cfg CounterConfig) node.StartFunc[int] {
		return func(out chan<- int) {
			for i := cfg.From; i <= cfg.To; i++ {
				out <- i
			}
		}
	},
}

var Printer = stage.TerminalProvider[struct{}, string]{
	ID: "printer",
	Function: func(_ struct{}) node.TerminalFunc[string] {
		return func(in <-chan string) {
			for n := range in {
				fmt.Println(n)
			}
		}
	},
}

func IntToStringCodec(in <-chan int, out chan<- string) {
	for n := range in {
		out <- strconv.Itoa(n)
	}
}

func main() {
	gb := graph.NewBuilder()
	graph.RegisterCodec(gb, IntToStringCodec)
	graph.RegisterStart(gb, Counter)
	graph.RegisterExport(gb, Printer)
	graph.InstantiateStart[CounterConfig, int](gb, "counter1", Counter.ID,
		CounterConfig{From: 3, To: 6})
	graph.InstantiateStart[CounterConfig, int](gb, "counter2", Counter.ID,
		CounterConfig{From: 30, To: 34})
	graph.InstantiateTerminal[struct{}, string](gb, "thePrinter", Printer.ID,
		struct{}{})

	gb.Connect("counter1", "thePrinter")
	gb.Connect("counter2", "thePrinter")

	gr := gb.Build()
	gr.Run()
}
