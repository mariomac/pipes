package main

import (
	"fmt"

	"github.com/mariomac/pipes/pkg/node"
)

// Counter returns a start function that forwards, in order,
// the numbers in the inclusive range passed as arguments.
func Counter(from, to int) node.StartFunc[int] {
	return func(out chan<- int) {
		for n := from; n <= to; n++ {
			out <- n
		}
	}
}

// Multiplier multiplies by a factor and forwards the numbers
// received by its input channel.
func Multiplier(factor int) node.MidFunc[int, int] {
	return func(in <-chan int, out chan<- int) {
		for n := range in {
			out <- n * factor
		}
	}
}

// Printer just prints by the standard output each received number.
func Printer() node.EndFunc[int] {
	return func(in <-chan int) {
		for n := range in {
			fmt.Println(n)
		}
	}
}

func main() {
	count := node.asStart(Counter(1, 4))
	mult2 := node.asMiddle(Multiplier(2))
	mult10 := node.asMiddle(Multiplier(10))
	printer := node.asTerminal(Printer())

	count.SendTo(mult2, mult10)
	mult2.SendTo(printer)
	mult10.SendTo(printer)

	count.Start()

	<-printer.Done()
}
