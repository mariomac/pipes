package stage

import "github.com/mariomac/pipes/pkg/node"

type Type string
type Name string

type IngestProvider struct {
	StageType    Type
	Instantiator func(interface{}) *node.Init
}

type TransformProvider struct {
	StageType    Type
	Instantiator func(interface{}) *node.Middle
}

type ExportProvider struct {
	StageType    Type
	Instantiator func(interface{}) *node.Terminal
}
