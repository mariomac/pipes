package pipe_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mariomac/pipes/pipe"
)

type intIntNodeMap struct {
	start    pipe.Start[int]
	doubler  pipe.Middle[int, int]
	bypasser pipe.Middle[int, int]
	final    pipe.Final[int]
}

func (s *intIntNodeMap) Connect() {
	s.start.SendTo(s.doubler, s.bypasser)
	s.bypasser.SendTo(s.final)
	s.doubler.SendTo(s.final)
}

func (s *intIntNodeMap) startNode() *pipe.Start[int] {
	return &s.start
}

func (s *intIntNodeMap) middleNode() *pipe.Middle[int, int] {
	return &s.doubler
}

func bypasser(s *intIntNodeMap) *pipe.Middle[int, int] {
	return &s.bypasser
}

func (s *intIntNodeMap) finalNode() *pipe.Final[int] {
	return &s.final
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

func TestExample(t *testing.T) {
	p := pipe.NewBuilder(&intIntNodeMap{})
	pipe.AddStartProvider(p, (*intIntNodeMap).startNode, func() (pipe.StartFunc[int], error) {
		return func(out chan<- int) {
			out <- 1
			out <- 2
			out <- 3
		}, nil
	})
	pipe.AddMiddleProvider(p, bypasser, func() (pipe.MiddleFunc[int, int], error) {
		return pipe.BypassMid[int](), nil
	})

	pipe.AddMiddleProvider(p, (*intIntNodeMap).middleNode, func() (pipe.MiddleFunc[int, int], error) {
		return func(in <-chan int, out chan<- int) {
			for i := range in {
				out <- i * 2
			}
		}, nil
	})
	pipe.AddFinalProvider(p, (*intIntNodeMap).finalNode, func() (pipe.FinalFunc[int], error) {
		return func(in <-chan int) {
			for i := range in {
				fmt.Println(i)
			}
		}, nil
	})

	r, err := p.Build()
	require.NoError(t, err)
	r.Start()

	<-r.Done()
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

//testignore
