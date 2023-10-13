package node

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	helpers "github.com/mariomac/pipes/pkg/test"
)

func TestAsStartDemux(t *testing.T) {
	start := AsStartDemux(func(d DemuxGetter) {
		out1 := DemuxGet[int32](d)
		out2 := DemuxGet[int64](d)
		out1 <- 1
		out2 <- 10
		out1 <- 60
		out2 <- 30
	})
	doubler := AsMiddle(func(in <-chan int32, out chan<- int) {
		for i := range in {
			out <- int(i * 2)
		}
	})
	divider := AsMiddle(func(in <-chan int64, out chan<- int) {
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
	DemuxSend[int32](start, doubler)
	DemuxSend[int64](start, divider)
	doubler.SendTo(sorter)
	divider.SendTo(sorter)

	go start.Start()

	waiter.Wait(t, 100000*testTimeout)

	assert.Equal(t, []int{2, 5, 15, 120}, sorted)

}
