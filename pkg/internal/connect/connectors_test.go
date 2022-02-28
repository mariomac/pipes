package connect

import (
	"testing"
	"time"

	"github.com/netobserv/gopipes/pkg/internal/refl"
	helpers "github.com/netobserv/gopipes/pkg/test"
	"github.com/stretchr/testify/assert"
)

const timeout = 2 * time.Second

func TestJoiner(t *testing.T) {
	sender1 := refl.WrapFunction(func(out chan<- int) {
		out <- 1
	})
	sender2 := refl.WrapFunction(func(out chan<- int) {
		out <- 2
	})
	sender3 := refl.WrapFunction(func(out chan<- int) {
		out <- 3
	})
	set := map[int]struct{}{}
	finished := helpers.AsyncWait(1)
	receiver := refl.WrapFunction(func(in <-chan int) {
		for i := range in {
			set[i] = struct{}{}
		}
	})
	j := NewJoiner(receiver.ArgChannelType(0), 20)
	// running this concurrently will help detecting race conditions
	go receiver.RunAsEndGoroutine(j.Receiver(), finished.Done)
	go sender1.RunAsStartGoroutine(j.AcquireSender(), j.ReleaseSender)
	go sender2.RunAsStartGoroutine(j.AcquireSender(), j.ReleaseSender)
	go sender3.RunAsStartGoroutine(j.AcquireSender(), j.ReleaseSender)
	finished.Wait(t, timeout)
	assert.Equal(t, map[int]struct{}{1: {}, 2: {}, 3: {}}, set)

	// check that all the channels have been closed
	assert.Panics(t, j.Receiver().Close)
}

func TestForker(t *testing.T) {
	sender := refl.WrapFunction(func(out chan<- int) {
		out <- 1
		out <- 2
		out <- 3
	})
	finished := helpers.AsyncWait(3)
	var arr1 []int
	receiver1 := refl.WrapFunction(func(in <-chan int) {
		for i := range in {
			arr1 = append(arr1, i)
		}
	})
	var arr2 []int
	receiver2 := refl.WrapFunction(func(in <-chan int) {
		for i := range in {
			arr2 = append(arr2, i*2)
		}
	})
	var arr3 []int
	receiver3 := refl.WrapFunction(func(in <-chan int) {
		for i := range in {
			arr3 = append(arr3, i+10)
		}
	})

	joiner1 := NewJoiner(sender.ArgChannelType(0), 20)
	joiner2 := NewJoiner(sender.ArgChannelType(0), 20)
	joiner3 := NewJoiner(sender.ArgChannelType(0), 20)
	f := Fork(&joiner1, &joiner2, &joiner3)
	// running this concurrently will help detecting race conditions
	go sender.RunAsStartGoroutine(f.Sender(), f.Close)
	go receiver1.RunAsEndGoroutine(joiner1.Receiver(), finished.Done)
	go receiver2.RunAsEndGoroutine(joiner2.Receiver(), finished.Done)
	go receiver3.RunAsEndGoroutine(joiner3.Receiver(), finished.Done)

	finished.Wait(t, timeout)

	assert.Equal(t, []int{1, 2, 3}, arr1)
	assert.Equal(t, []int{2, 4, 6}, arr2)
	assert.Equal(t, []int{11, 12, 13}, arr3)

	// check that all the channels have been closed
	assert.Panics(t, f.Sender().Close)
	assert.Panics(t, joiner1.Receiver().Close)
	assert.Panics(t, joiner2.Receiver().Close)
	assert.Panics(t, joiner3.Receiver().Close)
}
