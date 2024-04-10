package main

import (
	"fmt"
	"time"

	"github.com/mariomac/pipes/pkg/node"
)

// Ticker sends the current time.Time each second, for
// a number of times specified by the ticks argument
func Ticker(ticks int) node.StartFunc[time.Time] {
	return func(out chan<- time.Time) {
		for i := 0; i < ticks; i++ {
			fmt.Println("tick!")
			out <- time.Now()
			time.Sleep(time.Second)
		}
	}
}

// FamousDates just forwards few relevant dates of the 20th century.
func FamousDates() node.StartFunc[time.Time] {
	return func(out chan<- time.Time) {
		dday, _ := time.Parse(time.DateOnly, "1944-06-06")
		out <- dday
		moon, _ := time.Parse(time.DateOnly, "1969-07-10")
		out <- moon
		berl, _ := time.Parse(time.DateOnly, "1989-11-09")
		out <- berl
	}
}

// Printer prints the received Date/Times in the standard output.
func Printer() node.EndFunc[time.Time] {
	return func(in <-chan time.Time) {
		for t := range in {
			fmt.Println("printing:", t.Format(time.DateTime))
		}
	}
}

// Earliest accumulates all the received time.Time and, after processing
// all of them, prints the earliest time.Time
func Earliest() node.EndFunc[time.Time] {
	return func(in <-chan time.Time) {
		earliest := <-in
		for t := range in {
			if earliest.After(t) {
				earliest = t
			}
		}
		fmt.Println("after finishing, the earliest date is", earliest)
	}
}

func main() {
	// instantiating all the nodes
	s1 := node.asStart(Ticker(3))
	s2 := node.asStart(FamousDates())
	t1 := node.asTerminal(Printer())
	t2 := node.asTerminal(Earliest())

	// connecting nodes
	s1.SendTo(t1, t2)
	s2.SendTo(t1, t2)

	// ALL the start nodes need to be explicitly started
	s1.Start()
	s2.Start()

	// To make sure that all the data is processed, se
	// need to wait for ALL the terminal nodes, as
	// they might not finish simultaneously
	<-t1.Done()
	<-t2.Done()
}
