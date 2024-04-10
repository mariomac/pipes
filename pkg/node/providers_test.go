package node_test

import (
	"github.com/mariomac/pipes/pkg/node"
)

type sampleNm struct {
	start  node.Start[int]
	middle node.Middle[int, int]
	end    node.End[int]
}

func (s *sampleNm) Connect() {
	s.start.SendTo(s.middle)
	s.middle.SendTo(s.end)
}

func startNode(s *sampleNm) *node.Start[int] {
	return &s.start
}

func middleNode(s *sampleNm) *node.Middle[int, int] {
	return &s.middle
}

func example() {
	p := node.NewPipe(&sampleNm{})
	node.AddStartProvider(p, startNode, func() (node.StartFunc[int], error) {
		return func(out chan<- int) {
			out <- 1
			out <- 2
			out <- 3
		}, nil
	})

	node.AddMidProvider(p, middleNode, func() (node.MidFunc[int, int], error) {
		return node.IgnoreMid[int](), nil
	})

}
