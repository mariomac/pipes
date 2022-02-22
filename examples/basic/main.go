package main

import (
	"fmt"
	"strconv"

	"github.com/mariomac/go-pipes/pkg/pipe"
)

// start of the pipeline. Sends some values to a given output channel
func tenCounter(out chan<- int) {
	for i := 0; i < 10; i++ {
		out <- i
	}
}

// intermediate steps, receiving data from an input channel and forwarding
// them to an output channel
func oddFilter(in <-chan int, out chan<- int) {
	for n := range in {
		if n%2 == 0 {
			out <- n
		}
	}
}

// output channel can be different than input channel. The only condition is that
// the output and input channels of succesive stages match their element type
func stringer(in <-chan int, out chan<- string) {
	for n := range in {
		out <- "#" + strconv.Itoa(n)
	}
}

func main() {
	p := pipe.Start(tenCounter)
	p.Add(oddFilter)
	p.Add(stringer)

	endCh := make(chan struct{})
	// you can also embed any function literal
	// the ending function can only have an input channel
	p.Add(func(in <-chan string) {
		for s := range in {
			fmt.Println("received string:", s)
		}
		close(endCh)
	})

	p.Run()

	<-endCh
}
