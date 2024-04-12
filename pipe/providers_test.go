package pipe_test

import (
	"github.com/mariomac/pipes/pipe"
)

type sampleNm struct {
	start  pipe.Start[int]
	middle pipe.Middle[int, int]
	end    pipe.Final[int]
}

func (s *sampleNm) Connect() {
	s.start.SendTo(s.middle)
	s.middle.SendTo(s.end)
}

func startNode(s *sampleNm) *pipe.Start[int] {
	return &s.start
}

func middleNode(s *sampleNm) *pipe.Middle[int, int] {
	return &s.middle
}

func example() {
	p := pipe.NewPipe(&sampleNm{})
	pipe.AddStartProvider(p, startNode, func() (pipe.StartFunc[int], error) {
		return func(out chan<- int) {
			out <- 1
			out <- 2
			out <- 3
		}, nil
	})

	pipe.AddMiddleProvider(p, middleNode, func() (pipe.MidFunc[int, int], error) {
		return pipe.BypassMid[int](), nil
	})

}
