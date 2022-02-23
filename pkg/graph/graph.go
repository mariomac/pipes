package graph

import "github.com/mariomac/go-pipes/pkg/node"

type Builder struct {
	startNodes []node.Init
}

func (l *Builder) FromInit(initNodes ...node.Init) {
	l.startNodes = append(l.startNodes, initNodes...)
}

func (l *Builder) Run() {
	for _, node := range l.startNodes {
		node.
	}
}
