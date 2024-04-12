package pipe_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mariomac/pipes/pipe"
	"github.com/mariomac/pipes/testers"
)

const testTimeout = 3 * time.Second

type nodeMap struct {
	start    pipe.Start[int]
	bypasser pipe.Middle[int, int]
	final    pipe.Final[int]
	finalMul pipe.Final[int]
}

func (s *nodeMap) Connect() {
	s.start.SendTo(s.bypasser)
	s.bypasser.SendTo(s.final, s.finalMul)
}

func startNode(s *nodeMap) *pipe.Start[int]      { return &s.start }
func bypasser(s *nodeMap) *pipe.Middle[int, int] { return &s.bypasser }
func finalNode(s *nodeMap) *pipe.Final[int]      { return &s.final }
func finalMulNode(s *nodeMap) *pipe.Final[int]   { return &s.finalMul }

func TestBypass_Single(t *testing.T) {
	p := pipe.NewBuilder(&nodeMap{})
	pipe.AddStart(p, startNode, func(out chan<- int) {
		out <- 1
		out <- 2
		out <- 3
	})
	pipe.AddMiddleProvider(p, bypasser, func() (pipe.MiddleFunc[int, int], error) {
		return pipe.Bypass[int](), nil
	})
	var recv []int
	pipe.AddFinal(p, finalNode, func(in <-chan int) {
		for i := range in {
			recv = append(recv, i)
		}
	})
	var recvMul []int
	pipe.AddFinal(p, finalMulNode, func(in <-chan int) {
		for i := range in {
			recvMul = append(recvMul, 10*i)
		}
	})

	r, err := p.Build()
	require.NoError(t, err)
	r.Start()
	testers.ReadChannel(t, r.Done(), testTimeout)

	assert.Equal(t, []int{1, 2, 3}, recv)
	assert.Equal(t, []int{10, 20, 30}, recvMul)
}

type multiBypassedNodes struct {
	start     pipe.Start[int]
	bypasser1 pipe.Middle[int, int]
	bypasser2 pipe.Middle[int, int]
	bypasser3 pipe.Middle[int, int]
	mul       pipe.Middle[int, int]
	final     pipe.Final[int]
	finalInc  pipe.Final[int]
}

func (s *multiBypassedNodes) Connect() {
	s.start.SendTo(s.bypasser1)
	s.bypasser1.SendTo(s.bypasser3, s.mul)
	s.mul.SendTo(s.bypasser3)
	s.bypasser2.SendTo(s.bypasser3)
	s.bypasser3.SendTo(s.final, s.finalInc)
}

func mbStartNode(s *multiBypassedNodes) *pipe.Start[int]       { return &s.start }
func mbBypasser1(s *multiBypassedNodes) *pipe.Middle[int, int] { return &s.bypasser1 }
func mbBypasser2(s *multiBypassedNodes) *pipe.Middle[int, int] { return &s.bypasser2 }
func mbBypasser3(s *multiBypassedNodes) *pipe.Middle[int, int] { return &s.bypasser3 }
func mbMulNode(s *multiBypassedNodes) *pipe.Middle[int, int]   { return &s.mul }
func mbFinalNode(s *multiBypassedNodes) *pipe.Final[int]       { return &s.final }
func mbFinalIncNode(s *multiBypassedNodes) *pipe.Final[int]    { return &s.finalInc }

func TestBypass_Multi_And_Mixed(t *testing.T) {
	p := pipe.NewBuilder(&multiBypassedNodes{})

	pipe.AddStart(p, mbStartNode, func(out chan<- int) {
		out <- 1
		out <- 2
		out <- 3
	})
	pipe.AddMiddleProvider(p, mbBypasser1, func() (pipe.MiddleFunc[int, int], error) {
		return pipe.Bypass[int](), nil
	})
	pipe.AddMiddleProvider(p, mbBypasser2, func() (pipe.MiddleFunc[int, int], error) {
		return pipe.Bypass[int](), nil
	})
	pipe.AddMiddleProvider(p, mbBypasser3, func() (pipe.MiddleFunc[int, int], error) {
		return pipe.Bypass[int](), nil
	})
	pipe.AddMiddle(p, mbMulNode, func(in <-chan int, out chan<- int) {
		for i := range in {
			out <- i * 10
		}
	})
	var recv []int
	pipe.AddFinal(p, mbFinalNode, func(in <-chan int) {
		for i := range in {
			recv = append(recv, i)
		}
	})
	var recvInc []int
	pipe.AddFinal(p, mbFinalIncNode, func(in <-chan int) {
		for i := range in {
			recvInc = append(recvInc, i+1)
		}
	})
	r, err := p.Build()
	require.NoError(t, err)
	r.Start()
	testers.ReadChannel(t, r.Done(), testTimeout)

	slices.Sort(recv)
	slices.Sort(recvInc)
	// slices contain both the bypassed numbers and the numbers passed across mul node
	assert.Equal(t, []int{1, 2, 3, 10, 20, 30}, recv)
	assert.Equal(t, []int{2, 3, 4, 11, 21, 31}, recvInc)
}

type intStringNodeMap struct {
	start    pipe.Start[int]
	doubler  pipe.Middle[int, string]
	bypasser pipe.Middle[int, string]
	final    pipe.Final[string]
}

func (s *intStringNodeMap) Connect() {
	s.start.SendTo(s.doubler, s.bypasser)
	s.bypasser.SendTo(s.final)
	s.doubler.SendTo(s.final)
}

func iSStartNode(s *intStringNodeMap) *pipe.Start[int] {
	return &s.start
}

func iSMiddleNode(s *intStringNodeMap) *pipe.Middle[int, string] {
	return &s.doubler
}

func iSBypasser(s *intStringNodeMap) *pipe.Middle[int, string] {
	return &s.bypasser
}

func isFinalNode(s *intStringNodeMap) *pipe.Final[string] {
	return &s.final
}

func TestExample_CantBypass(t *testing.T) {
	p := pipe.NewBuilder(&intStringNodeMap{})
	pipe.AddStartProvider(p, iSStartNode, func() (pipe.StartFunc[int], error) {
		return func(out chan<- int) {
			out <- 1
			out <- 2
			out <- 3
		}, nil
	})
	pipe.AddMiddleProvider(p, iSBypasser, func() (pipe.MiddleFunc[int, string], error) {
		return nil, nil
	})

	pipe.AddMiddleProvider(p, iSMiddleNode, func() (pipe.MiddleFunc[int, string], error) {
		return func(in <-chan int, out chan<- string) {
			for i := range in {
				out <- fmt.Sprint(i * 2)
			}
		}, nil
	})
	pipe.AddFinalProvider(p, isFinalNode, func() (pipe.FinalFunc[string], error) {
		return func(in <-chan string) {
			for i := range in {
				fmt.Println(i)
			}
		}, nil
	})

	_, err := p.Build()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Expecting pipe.MiddleFunc[int,string]")
}
