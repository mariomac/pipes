package node

import (
	"fmt"
	"testing"
	"time"

	helpers "github.com/mariomac/go-pipes/pkg/test"
	"github.com/stretchr/testify/assert"
)

const timeout = 2 * time.Second

func TestBasicGraph(t *testing.T) {
	waiter := helpers.AsyncWait(1)
	start1 := AsInit(Counter(1, 3))
	start2 := AsInit(Counter(6, 8))
	odds := AsMiddle(OddFilter)
	evens := AsMiddle(EvenFilter)
	oddsMsg := AsMiddle(Messager("odd"))
	evensMsg := AsMiddle(Messager("even"))
	collected := map[string]struct{}{}
	collector := AsTerminal(func(strs <-chan string) {
		for str := range strs {
			collected[str] = struct{}{}
		}
		waiter.Done()
	})
	/*
		start1----\ /---start2
		  |       X      |
		evens<---/ \-->odds
		  |              |
		evensMsg      oddsMsg
		       \      /
		        printer
	*/
	start1.SendsTo(evens, odds)
	start2.SendsTo(evens, odds)
	odds.SendsTo(oddsMsg)
	evens.SendsTo(evensMsg)
	oddsMsg.SendsTo(collector)
	evensMsg.SendsTo(collector)

	start1.Start()
	start2.Start()

	waiter.Wait(t, timeout)
	assert.Equal(t, map[string]struct{}{
		"odd: 1":  {},
		"even: 2": {},
		"odd: 3":  {},
		"even: 6": {},
		"odd: 7":  {},
		"even: 8": {},
	}, collected)
}

func TestGraphVerification(t *testing.T) {
	assert.Panics(t, func() {
		_ = AsInit(func(out <-chan int) {})
	}, "must panic if the init channel is not writable")
	assert.Panics(t, func() {
		_ = AsInit(func() {})
	}, "must panic if the init function has no arguments")
	assert.Panics(t, func() {
		_ = AsInit(func(in, out chan int) {})
	}, "must panic if the Init function has more than one argument")
	//assert.Panics(t, func() {
	//	p := AsInit(Counter(1, 2))
	//	p.SendsTo(AsTerminal(func(in chan string) {}))
	//}, "must panic if the input of a node does not match the type of the previous node")
	//assert.Panics(t, func() {
	//	p := Start(counter(3))
	//	p.Add(func(in <-chan int) {})
	//	p.Add(func(in <-chan int) {})
	//}, "must panic if trying to add a pipeline stage after a terminal (input-only) stage")
	//assert.Panics(t, func() {
	//	p := Start(counter(3))
	//	p.Add(func(in <-chan int) {})
	//	p.Fork()
	//}, "must panic if trying to fork a pipeline that has a terminal stage")
}

func Counter(from, to int) func(out chan<- int) {
	return func(out chan<- int) {
		for i := from; i <= to; i++ {
			out <- i
		}
	}
}

func OddFilter(in <-chan int, out chan<- int) {
	for n := range in {
		if n%2 == 1 {
			out <- n
		}
	}
}

func EvenFilter(in <-chan int, out chan<- int) {
	for n := range in {
		if n%2 == 0 {
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
