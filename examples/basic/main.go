package main

import (
	"fmt"
	"github.com/mariomac/go-pipes/pkg/node"
	"math/rand"
	"time"
)

func StartCounter(out chan<- int) {
	for i := 0; i < 5; i++ {
		out <- i
	}
}

func StartRandoms(out chan<- int) {
	for i := 0; i < 5; i++ {
		out <- rand.Intn(1000)
	}
}

func OddFilter(in <-chan int, out chan<- int) {
	for n := range in {
		if n%2 == 0 {
			out <- n
		}
	}
}

func EvenFilter(in <-chan int, out chan<- int) {
	for n := range in {
		if n%2 == 1 {
			out <- n
		}
	}
}

func Messager(msg string) func(in <-chan int, out chan<- string) {
	return func(in <-chan int, out chan<- string) {
		for n := range in {
			out <- fmt.Sprintf("%s: %d", msg, n)
		}
	}
}

func Printer(in <-chan string) {
	for n := range in {
		fmt.Println(n)
	}
}

func main() {
	start1 := node.AsInit(StartCounter)
	start2 := node.AsInit(StartRandoms)
	odds := node.AsInner(OddFilter)
	evens := node.AsInner(EvenFilter)
	oddsMsg := node.AsInner(Messager("odd number"))
	evensMsg := node.AsInner(Messager("even number"))
	printer := node.AsTerminal(Printer)

	/*
			       start1----\ /---start2
			          |       X      |
			        evens<---/ \-->odds
			          |              |
			        evensMsg      oddsMsg
		                   \ 	  /
			               printer
	*/
	start1.SendsTo(evens, odds)
	start2.SendsTo(evens, odds)
	odds.SendsTo(oddsMsg)
	evens.SendsTo(evensMsg)
	oddsMsg.SendsTo(printer)
	evensMsg.SendsTo(printer)

	start1.Start()
	start2.Start()

	time.Sleep(2 * time.Second)
}
