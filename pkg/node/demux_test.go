package node

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	helpers "github.com/mariomac/pipes/pkg/test"
)

const testTimeout = 5 * time.Second

func TestAsStartDemux(t *testing.T) {
	type out2k struct{}
	type out1k struct{}
	start := AsStartDemux(func(d DemuxedChans) {
		out1 := DemuxGet[int](d, out1k{})
		out2 := DemuxGet[int](d, out2k{})
		out1 <- 1
		out2 <- 10
		out1 <- 60
		out2 <- 30
	})
	doubler := AsMiddle(func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- int(i * 2)
		}
	})
	decer := AsMiddle(func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- int(i - 1)
		}
	})
	divider := AsMiddle(func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- int(i / 2)
		}
	})
	var sorted []int
	waiter := helpers.AsyncWait(1)
	sorter := AsTerminal(func(in <-chan int) {
		for i := range in {
			sorted = append(sorted, i)
		}
		slices.Sort(sorted)
		waiter.Done()
	})
	do := DemuxAdd[int](start, out1k{})
	do.SendTo(doubler, decer)
	DemuxAdd[int](start, out2k{}).SendTo(divider)
	decer.SendTo(sorter)
	doubler.SendTo(sorter)
	divider.SendTo(sorter)

	go start.Start()

	waiter.Wait(t, testTimeout)

	assert.Equal(t, []int{0, 2, 5, 15, 59, 120}, sorted)

}

func TestAsMiddleDemux(t *testing.T) {
	start := AsStart(func(out chan<- int) {
		fmt.Println("start sttart")
		for i := 0; i < 10; i++ {
			out <- i
		}
	})
	classifier := AsMiddleDemux(func(in <-chan int, out DemuxedChans) {
		fmt.Println("class sttart")
		evens := DemuxGet[int32](out, "evens")
		odds := DemuxGet[int](out, "odds")
		for i := range in {
			if i%2 == 0 {
				evens <- int32(i)
			} else {
				odds <- i
			}
		}
	})
	doubler := AsMiddle(func(in <-chan int32, out chan<- int) {
		fmt.Println("doubler sttart")
		for i := range in {
			out <- int(i * 2)
		}
	})
	var sorted []int
	waiter := helpers.AsyncWait(1)
	sorter := AsTerminal(func(in <-chan int) {
		fmt.Println("soerter sttart")

		for i := range in {
			sorted = append(sorted, i)
		}
		slices.Sort(sorted)
		waiter.Done()
	})
	start.SendTo(classifier)
	DemuxAdd[int32](classifier, "evens").SendTo(doubler)
	DemuxAdd[int](classifier, "odds").SendTo(sorter)
	doubler.SendTo(sorter)

	go start.Start()

	waiter.Wait(t, testTimeout)

	assert.Equal(t, []int{0, 1, 3, 4, 5, 7, 8, 9, 12, 16}, sorted)
}

// TODO
// Test demuxed channel buffers
// Test errors if a demux chan does not have output
// Test panics when getting demuxes without correct key
// Test panics when getting demuxes with wrong type
