package pipe_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mariomac/pipe"
	"github.com/mariomac/pipe/testers"
)

func TestBypass_Single(t *testing.T) {
	// TODO: pipe.New()
	p := pipe.NewPipe()
	start := pipe.AddStart(p, func(out chan<- int) {
		out <- 1
		out <- 2
		out <- 3
	})
	bypass := pipe.AddMiddleOpt[int](p, nil)
	var recv []int
	term := pipe.AddTerminal(p, func(in <-chan int) {
		for i := range in {
			recv = append(recv, i)
		}
	})
	var recvMul []int
	termMul := pipe.AddTerminal(p, func(in <-chan int) {
		for i := range in {
			recvMul = append(recvMul, 10*i)
		}
	})
	start.SendTo(bypass)
	bypass.SendTo(term, termMul)

	p.Start()
	testers.ReadChannel(t, p.Done(), timeout)

	assert.Equal(t, []int{1, 2, 3}, recv)
	assert.Equal(t, []int{10, 20, 30}, recvMul)
}

func TestBypass_Multi(t *testing.T) {
	p := pipe.NewPipe()

	start := pipe.AddStart(p, func(out chan<- int) {
		out <- 1
		out <- 2
		out <- 3
	})
	bypass1 := pipe.AddMiddleOpt[int](p, nil)
	bypass2 := pipe.AddMiddleOpt[int](p, nil)
	bypass3 := pipe.AddMiddleOpt[int](p, nil)
	var recv []int
	term := pipe.AddTerminal(p, func(in <-chan int) {
		for i := range in {
			recv = append(recv, i)
		}
	})
	var recvMul []int
	termMul := pipe.AddTerminal(p, func(in <-chan int) {
		for i := range in {
			recvMul = append(recvMul, 10*i)
		}
	})
	start.SendTo(bypass1)
	bypass1.SendTo(bypass2)
	bypass2.SendTo(bypass3)
	bypass3.SendTo(term, termMul)

	p.Start()
	testers.ReadChannel(t, p.Done(), timeout)

	assert.Equal(t, []int{1, 2, 3}, recv)
	assert.Equal(t, []int{10, 20, 30}, recvMul)
}

func TestBypass_Mixed(t *testing.T) {
	p := pipe.NewPipe()
	start := pipe.AddStart(p, func(out chan<- int) {
		out <- 1
		out <- 2
		out <- 3
	})
	bypass1 := pipe.AddMiddleOpt[int](p, nil)
	bypass2 := pipe.AddMiddleOpt[int](p, nil)
	bypass3 := pipe.AddMiddleOpt[int](p, nil)
	mul := pipe.AddMiddle(p, func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- i * 10
		}
	})
	var recv []int
	term := pipe.AddTerminal(p, func(in <-chan int) {
		for i := range in {
			recv = append(recv, i)
		}
	})
	var recvAdd []int
	termAdd := pipe.AddTerminal(p, func(in <-chan int) {
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

	p.Start()
	testers.ReadChannel(t, p.Done(), timeout)

	assert.Equal(t, []int{1, 2, 3}, recv)
	assert.Equal(t, []int{11, 21, 31}, recvAdd)
}
