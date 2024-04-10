package node

import (
	"github.com/mariomac/pipes/pkg/node/internal/connect"
)

// bypass node just makes sure, at graph construction time, that the inputs of this node
// are bypassed to the destination nodes.
// At a logical level, you can see a bypass node as a middle[T, T] node that just forwards
// its input to the output channel.
// At an implementation level, bypass[T] is much more efficient because it just makes sure
// that its input channel is connected to its destination nodes, without adding any extra
// goroutine nor channel operation.
// bypass is useful for implementing constructors that might return an optional middle[T, T] node
// (according to e.g. the user configuration) or just a bypass[T] node to transparently
// forward data to the destination nodes.
type bypass[INOUT any] struct {
	outs []Receiver[INOUT]
}

func (b *bypass[INOUT]) SendTo(r ...Receiver[INOUT]) {
	b.outs = append(b.outs, r...)
}

// nolint:unused
// golangci-lint bug: it's actually used through its interface
func (b *bypass[INOUT]) isStarted() bool {
	// returns true if all the forwarded nodes are started
	started := true
	for _, o := range b.outs {
		started = started && o.isStarted()
	}
	return started
}

// nolint:unused
// golangci-lint bug: it's actually used through its interface
func (b *bypass[INOUT]) start() {
	if len(b.outs) == 0 {
		panic("bypass node should have outputs")
	}
	for _, o := range b.outs {
		if !o.isStarted() {
			o.start()
		}
	}
}

// nolint:unused
// golangci-lint bug: it's actually used through its interface
func (b *bypass[INOUT]) joiners() []*connect.Joiner[INOUT] {
	joiners := make([]*connect.Joiner[INOUT], 0, len(b.outs))
	for _, o := range b.outs {
		joiners = append(joiners, o.joiners()...)
	}
	return joiners
}
