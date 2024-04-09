package main

import (
	"fmt"

	"github.com/mariomac/pipes/pkg/node"
)

func Generator(outs node.Demux) {
	nonPositive := node.DemuxGet[int](outs, "nonPositive")
	positive := node.DemuxGet[int](outs, "positive")
	for i := -3; i <= 17; i++ {
		if i <= 0 {
			nonPositive <- i
		} else {
			positive <- i
		}
	}
}

func PrimeFilter(in <-chan int, outs node.Demux) {
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
}

func PrimePrinter(in <-chan int) {
	var primes []int
	for i := range in {
		primes = append(primes, i)
	}
	fmt.Println("received prime numbers:", primes)
}

func DiscardedPrinter(in <-chan int) {
	var discarded []int
	for i := range in {
		discarded = append(discarded, i)
	}
	fmt.Println("discarded numbers:", discarded)
}

func main() {
	generator := node.AsStartDemux(Generator)
	primeFilter := node.AsMiddleDemux(PrimeFilter)
	primePrinter := node.asTerminal(PrimePrinter)
	discardPrinter := node.asTerminal(DiscardedPrinter)
	node.DemuxAdd[int](generator, "nonPositive").SendTo(discardPrinter)
	node.DemuxAdd[int](generator, "positive").SendTo(primeFilter)
	node.DemuxAdd[int](primeFilter, "primes").SendTo(primePrinter)
	node.DemuxAdd[int](primeFilter, "notPrimes").SendTo(discardPrinter)

	generator.Start()

	<-primePrinter.Done()
	<-discardPrinter.Done()
}
