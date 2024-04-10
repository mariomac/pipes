package main

import (
	"fmt"

	"github.com/mariomac/pipes/pkg/graph"

	"github.com/mariomac/pipes/pkg/node"
)

type Graph struct {
	Generator   `sendTo:"nonPositive:DiscardPrinter,positive:PrimeFilter"`
	PrimeFilter `sendTo:"primes:PrimePrinter,notPrimes:DiscardPrinter"`
	PrimePrinter
	DiscardPrinter
}

type Generator struct{}

func GeneratorProvider(_ Generator) (node.StartDemuxFunc, error) {
	return func(outs node.Demux) {
		nonPositive := node.DemuxGet[int](outs, "nonPositive")
		positive := node.DemuxGet[int](outs, "positive")
		for i := -3; i <= 17; i++ {
			if i <= 0 {
				nonPositive <- i
			} else {
				positive <- i
			}
		}
	}, nil
}

type PrimeFilter struct{}

func PrimeFilterProvider(_ PrimeFilter) (node.MiddleDemuxFunc[int], error) {
	return func(in <-chan int, outs node.Demux) {
		primes := node.DemuxGet[int](outs, "primes")
		notPrimes := node.DemuxGet[int](outs, "notPrimes")

	nextInput:
		for i := range in {
			for n := 2; n < i; n++ {
				if i%n == 0 {
					notPrimes <- i
					continue nextInput
				}
			}
			primes <- i
		}
	}, nil
}

type PrimePrinter struct{}

func PrimePrinterProvider(_ PrimePrinter) (node.EndFunc[int], error) {
	return func(in <-chan int) {
		var primes []int
		for i := range in {
			primes = append(primes, i)
		}
		fmt.Println("received prime numbers:", primes)
	}, nil
}

type DiscardPrinter struct{}

func DiscardPrinterProvider(_ DiscardPrinter) (node.EndFunc[int], error) {
	return func(in <-chan int) {
		var discarded []int
		for i := range in {
			discarded = append(discarded, i)
		}
		fmt.Println("discarded numbers:", discarded)
	}, nil
}

func main() {
	gb := graph.NewBuilder()
	graph.RegisterStartDemux(gb, GeneratorProvider)
	graph.RegisterMiddleDemux(gb, PrimeFilterProvider)
	graph.RegisterTerminal(gb, PrimePrinterProvider)
	graph.RegisterTerminal(gb, DiscardPrinterProvider)

	grp, err := gb.Build(&Graph{})
	if err != nil {
		panic(err)
	}
	grp.Run()
}
