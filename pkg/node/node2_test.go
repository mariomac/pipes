package node

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	helpers "github.com/mariomac/pipes/pkg/test"
)

const testTimeout = 5 * time.Second

func TestAsStart2(t *testing.T) {
	start := AsStart2(func(out1 chan<- int, out2 chan<- int) {
		out1 <- 1
		out2 <- 10
		out1 <- 60
		out2 <- 30
	})
	doubler := AsMiddle(func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- i * 2
		}
	})
	divider := AsMiddle(func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- i / 2
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

	start.Sender1().SendTo(doubler)
	start.Sender2().SendTo(divider)
	doubler.SendTo(sorter)
	divider.SendTo(sorter)

	go start.Start()

	waiter.Wait(t, testTimeout)

	assert.Equal(t, []int{2, 5, 15, 120}, sorted)

}
