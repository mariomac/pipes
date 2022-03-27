package stage

import (
	"github.com/mariomac/pipes/pkg/node"
)

// TODO: Instance and InstanceID are too similar. Remove InstanceID, rename, or make it implement Instancer

// Instance can be embedded into any stage configuration to be instantiable
// (I will implement the required Instancer interface)
type Instance string

func (f Instance) ID() string {
	return string(f)
}

// Instancer is the interface required by any stage configuration type that is
// instantiated from the builder.ApplyConfig method.
type Instancer interface {
	ID() string
}

var _ Instancer = (*Instance)(nil)

// A provider wraps an instantiation function that, given a configuration argument, returns a
// node with a processing function.

type StartProvider[CFG, O any] func(CFG) node.StartFunc[O]

type MiddleProvider[CFG, I, O any] func(CFG) node.MiddleFunc[I, O]

type TerminalProvider[CFG, I any] func(CFG) node.TerminalFunc[I]
