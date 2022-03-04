package node

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const timeout = 2 * time.Second

func TestBasicGraph(t *testing.T) {
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

	select {
	case <-collector.Done():
	// ok!
	case <-time.After(timeout):
		require.Fail(t, "timeout while waiting for pipeline to complete")
	}

	assert.Equal(t, map[string]struct{}{
		"odd: 1":  {},
		"even: 2": {},
		"odd: 3":  {},
		"even: 6": {},
		"odd: 7":  {},
		"even: 8": {},
	}, collected)
}

func TestTypeCapture(t *testing.T) {
	type testType struct {
		foo string
	}

	start1 := AsInit(Counter(1, 3))
	odds := AsMiddle(OddFilter)
	oddsMsg := AsMiddle(Messager("odd"))
	collector := AsTerminal(func(strs <-chan string) {})
	testColl := AsMiddle(func(in <-chan testType, out chan<- []string) {})

	// assert that init/output types have been properly collected
	intType := reflect.TypeOf(1)
	stringType := reflect.TypeOf("")
	assert.Equal(t, intType, start1.OutType())
	assert.Equal(t, intType, odds.InType())
	assert.Equal(t, intType, odds.OutType())
	assert.Equal(t, intType, oddsMsg.InType())
	assert.Equal(t, stringType, oddsMsg.OutType())
	assert.Equal(t, stringType, collector.InType())
	assert.Equal(t, reflect.TypeOf(testType{foo: ""}), testColl.InType())
	assert.Equal(t, reflect.TypeOf([]string{}), testColl.OutType())
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
	assert.Panics(t, func() {
		_ = AsInit(func(in []int) {})
	}, "must panic if the init function argument is not a channel")
	assert.Panics(t, func() {
		_ = AsMiddle(func(in chan<- int, out chan<- int) {})
	}, "must panic if a Middle in channel is not readable")
	assert.Panics(t, func() {
		_ = AsMiddle(func(in <-chan int, out <-chan int) {})
	}, "must panic if a Middle out channel is not writable")
	assert.Panics(t, func() {
		_ = AsMiddle(func(in int, out chan<- int) {})
	}, "must panic if a Middle input not a channel")
	assert.Panics(t, func() {
		_ = AsMiddle(func(in <-chan int, out int) {})
	}, "must panic if a Middle out is not a channel")
	assert.Panics(t, func() {
		_ = AsMiddle(func(in <-chan int) {})
	}, "must panic if a Middle does not have two arguments")
	assert.Panics(t, func() {
		p := AsInit(Counter(1, 2))
		p.SendsTo(AsMiddle(func(in, out chan string) {}))
	}, "must panic if the input of a middle node does not match the type of the previous inner node")
	assert.Panics(t, func() {
		p := AsInit(Counter(1, 2))
		p.SendsTo(AsTerminal(func(in chan string) {}))
	}, "must panic if the input of a terminal node does not match the type of the previous inner node")
	assert.Panics(t, func() {
		m := AsMiddle(Messager("boom"))
		m.SendsTo(AsMiddle(OddFilter))
	}, "must panic if the input of a middle node does not match the type of the previous middle node")
	assert.Panics(t, func() {
		m := AsMiddle(OddFilter)
		m.SendsTo(AsTerminal(func(in chan string) {}))
	}, "must panic if the input of a terminal node does not match the type of the previous middle node")

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
