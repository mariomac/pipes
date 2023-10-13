// Package node provides functionalities to create nodes and interconnect them.
// A Node is a function container that can be connected via channels to other nodes.
// A node can send data to multiple nodes, and receive data from multiple nodes.
//
//nolint:unused
package node

import (
	"reflect"
)

// StartFunc is a function that receives a writable channel as unique argument, and sends
// value to that channel during an indefinite amount of time.
type StartFunc2[OUT1, OUT2 any] func(out1 chan<- OUT1, out2 chan<- OUT2)

// MiddleFunc is a function that receives a readable channel as first argument,
// and a writable channel as second argument.
// It must process the inputs from the input channel until it's closed.
type MiddleFunc2[IN, OUT1, OUT2 any] func(in <-chan IN, out1 chan<- OUT1, out2 chan<- OUT2)

// TODO: OutType and InType methods are candidates for deprecation

// Sender is any node that can send data to another node: node.Start and node.Middle
type Sender2[OUT1, OUT2 any] interface {
	Sender1() Sender[OUT1]
	Sender2() Sender[OUT2]
}

// Start nodes are the starting points of a graph. This is, all the nodes that bring information
// from outside the graph: e.g. because they generate them or because they acquire them from an
// external source like a Web Service.
// A graph must have at least one Start node.
// An Start node must have at least one output node.
type Start2[OUT1, OUT2 any] struct {
	sub1 startSubNode[OUT1]
	sub2 startSubNode[OUT2]
	funs []StartFunc2[OUT1, OUT2]
}

func (s *Start2[OUT1, OUT2]) Sender1() Sender[OUT1] {
	return &s.sub1
}
func (s *Start2[OUT1, OUT2]) Sender2() Sender[OUT2] {
	return &s.sub2
}

// AsStart wraps a group of StartFunc with the same signature into a Start node.
func AsStart2[OUT1, OUT2 any](funs ...StartFunc2[OUT1, OUT2]) *Start2[OUT1, OUT2] {
	var o1 OUT1
	var o2 OUT2
	return &Start2[OUT1, OUT2]{
		funs: funs,
		sub1: startSubNode[OUT1]{
			outType: reflect.TypeOf(o1),
		},
		sub2: startSubNode[OUT2]{
			outType: reflect.TypeOf(o2),
		},
	}
}

// Start starts the function wrapped in the Start node. This method should be invoked
// for all the start nodes of the same graph, so the graph can properly start and finish.
func (i *Start2[OUT1, OUT2]) Start() {
	forker1, err := i.sub1.start()
	if err != nil {
		panic("Start node output 1: " + err.Error())
	}
	forker2, err := i.sub2.start()
	if err != nil {
		panic("Start node output 2: " + err.Error())
	}
	for fn := range i.funs {
		fun := i.funs[fn]
		go func() {
			fun(forker1.AcquireSender(), forker2.AcquireSender())
			forker1.ReleaseSender()
			forker2.ReleaseSender()
		}()
	}
}
