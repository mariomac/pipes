package node

import (
	"testing"

	"github.com/stretchr/testify/assert"

	helpers "github.com/mariomac/pipes/pkg/test"
)

func TestBypass_Single(t *testing.T) {
	start := AsStart[int](func(out chan<- int) {
		out <- 1
		out <- 2
		out <- 3
	})
	bypass := &Bypass[int]{}
	var recv []int
	term := AsTerminal[int](func(in <-chan int) {
		for i := range in {
			recv = append(recv, i)
		}
	})
	var recvMul []int
	termMul := AsTerminal[int](func(in <-chan int) {
		for i := range in {
			recvMul = append(recvMul, 10*i)
		}
	})
	start.SendTo(bypass)
	bypass.SendTo(term, termMul)

	start.Start()
	helpers.ReadChannel(t, DoneAll(term, termMul), timeout)

	assert.Equal(t, []int{1, 2, 3}, recv)
	assert.Equal(t, []int{10, 20, 30}, recvMul)
}

func TestBypass_Multi(t *testing.T) {
	start := AsStart[int](func(out chan<- int) {
		out <- 1
		out <- 2
		out <- 3
	})
	bypass1, bypass2, bypass3 := &Bypass[int]{}, &Bypass[int]{}, &Bypass[int]{}
	var recv []int
	term := AsTerminal[int](func(in <-chan int) {
		for i := range in {
			recv = append(recv, i)
		}
	})
	var recvMul []int
	termMul := AsTerminal[int](func(in <-chan int) {
		for i := range in {
			recvMul = append(recvMul, 10*i)
		}
	})
	start.SendTo(bypass1)
	bypass1.SendTo(bypass2)
	bypass2.SendTo(bypass3)
	bypass3.SendTo(term, termMul)

	start.Start()
	helpers.ReadChannel(t, DoneAll(term, termMul), timeout)

	assert.Equal(t, []int{1, 2, 3}, recv)
	assert.Equal(t, []int{10, 20, 30}, recvMul)
}

func TestBypass_Mixed(t *testing.T) {
	start := AsStart[int](func(out chan<- int) {
		out <- 1
		out <- 2
		out <- 3
	})
	bypass1, bypass2, bypass3 := &Bypass[int]{}, &Bypass[int]{}, &Bypass[int]{}
	mul := AsMiddle(func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- i * 10
		}
	})
	var recv []int
	term := AsTerminal[int](func(in <-chan int) {
		for i := range in {
			recv = append(recv, i)
		}
	})
	var recvAdd []int
	termAdd := AsTerminal[int](func(in <-chan int) {
		for i := range in {
			recvAdd = append(recvAdd, 1+i)
		}
	})
	start.SendTo(bypass1)
	bypass1.SendTo(bypass2)
	// mixed bypass: bypasses to another bypass but also to an actually useful node
	bypass2.SendTo(bypass3, mul)
	bypass3.SendTo(term)
	mul.SendTo(termAdd)

	start.Start()
	helpers.ReadChannel(t, DoneAll(term, termAdd), timeout)

	assert.Equal(t, []int{1, 2, 3}, recv)
	assert.Equal(t, []int{11, 21, 31}, recvAdd)
}
