package stage

import (
	"github.com/mariomac/pipes/pkg/node"
)

type Type string
type Name string

// A provider wraps an instantiation function that, given a configuration argument, returns a
// node with a processing function.

type StartProvider[CFG, O any] func(CFG) node.StartFunc[O]

type MiddleProvider[CFG, I, O any] func(CFG) node.MiddleFunc[I, O]

type TerminalProvider[CFG, I any] func(CFG) node.TerminalFunc[I]
