package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/mariomac/pipes/pkg/node"
)

// StartCounter is a Start Node that generates some ordered numbers each 100 milliseconds
func StartCounter(out chan<- int) {
	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)
		out <- i
	}
}

// StartRandoms is a Start Node that generates some random numbers each 150 milliseconds
func StartRandoms(out chan<- int) {
	for i := 0; i < 5; i++ {
		time.Sleep(150 * time.Millisecond)
		out <- rand.Intn(1000)
	}
}

// OddFilter is a Middle Node that reads the numbers from the input channel and only forwards
// those that are Odd
func OddFilter(in <-chan int, out chan<- int) {
	for n := range in {
		if n%2 == 1 {
			out <- n
		}
	}
}

// EvenFilter is a middle node that reads the numbers from the input channel and only
// forwards those that are Even
func EvenFilter(in <-chan int, out chan<- int) {
	for n := range in {
		if n%2 == 0 {
			out <- n
		}
	}
}

// Messager is a middle node that forwards each string received from the input channel,
// prepending the given message
func Messager(msg string) func(in <-chan int, out chan<- string) {
	return func(in <-chan int, out chan<- string) {
		for n := range in {
			out <- fmt.Sprintf("%s: %d", msg, n)
		}
	}
}

// Printer is a Terminal Node that just prints each string received by its input channel.
func Printer(in <-chan string) {
	for n := range in {
		fmt.Println(n)
	}
}

func main() {
	// Instantiating the different node types
	start1 := node.AsStart(StartCounter)
	start2 := node.AsStart(StartRandoms)
	odds := node.AsMiddle(OddFilter)
	evens := node.AsMiddle(EvenFilter)
	oddsMsg := node.AsMiddle(Messager("odd number"))
	evensMsg := node.AsMiddle(Messager("even number"))
	printer := node.AsTerminal(Printer)

	/*
		start1----\ /---start2
		  |       X      |
		evens<---/ \-->odds
		  |              |
		evensMsg      oddsMsg
		       \      /
		        printer
	*/
	// Manually wiring the nodes
	start1.SendTo(evens, odds)
	start2.SendTo(evens, odds)
	odds.SendTo(oddsMsg)
	evens.SendTo(evensMsg)
	oddsMsg.SendTo(printer)
	evensMsg.SendTo(printer)

	// All the init nodes must be started
	start1.Start()
	start2.Start()

	// We can wait for terminal nodes to finish their execution
	// after the rest of the graph has finished
	<-printer.Done()
}
