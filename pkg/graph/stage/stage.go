package stage

import (
	"github.com/mariomac/pipes/pkg/node"
)

type InstanceID string

// A provider wraps an instantiation function that, given a configuration argument, returns a
// node with a processing function.

type StartProvider[CFG, O any] struct {
	Function func(CFG) node.StartFunc[O]
}

type MiddleProvider[CFG, I, O any] struct {
	Function func(CFG) node.MiddleFunc[I, O]
}

type TerminalProvider[CFG, I any] struct {
	Function func(CFG) node.TerminalFunc[I]
}
