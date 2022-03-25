package stage

import (
	"github.com/mariomac/pipes/pkg/node"
)

type Type string
type Name string

// A provider wraps an instantiation function that, given a configuration argument, returns a
// node with a processing function.

type StartProvider[CFG, O any] struct {
	StageType    Type
	Instantiator func(CFG) *node.Start[O]
}

type MiddleProvider[CFG, I, O any] struct {
	StageType    Type
	Instantiator func(CFG) *node.Middle[I, O]
}

type TerminalProvider[CFG, I any] struct {
	StageType    Type
	Instantiator func(CFG) *node.Terminal[I]
}
