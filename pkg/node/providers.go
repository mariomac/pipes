package node

import "reflect"

type NodesMap interface {
	Connect()
}

type StartFieldPtr[IMPL NodesMap, OUT any] func(IMPL) *Start[OUT]

type MidFieldPtr[IMPL NodesMap, IN, OUT any] func(IMPL) *Mid[IN, OUT]

type EndFieldPtr[IMPL NodesMap, IN any] func(IMPL) *End[IN]

type StartProvider[OUT any] func() (StartFunc[OUT], error)

type MiddleProvider[IN, OUT any] func() (MidFunc[IN, OUT], error)

type EndProvider[IN any] func() (EndFunc[IN], error)

func AddStartProvider[IMPL NodesMap, OUT any](p *Pipe[IMPL], field StartFieldPtr[IMPL, OUT], provider StartProvider[OUT]) {
	p.startProviders = append(p.startProviders, reflectProvider{
		asNode:      reflect.ValueOf(asStart[OUT]),
		fieldGetter: reflect.ValueOf(field),
		fn:          reflect.ValueOf(provider),
	})
}

func AddMidProvider[IMPL NodesMap, IN, OUT any](p *Pipe[IMPL], field MidFieldPtr[IMPL, IN, OUT], provider MiddleProvider[IN, OUT]) {
	p.midProviders = append(p.midProviders, reflectProvider{
		asNode:      reflect.ValueOf(asMiddle[IN, OUT]),
		fieldGetter: reflect.ValueOf(field),
		fn:          reflect.ValueOf(provider),
	})
}

func AddEndProvider[IMPL NodesMap, IN any](p *Pipe[IMPL], field EndFieldPtr[IMPL, IN], provider EndProvider[IN]) {
	p.endProviders = append(p.endProviders, reflectProvider{
		asNode:      reflect.ValueOf(asTerminal[IN]),
		fieldGetter: reflect.ValueOf(field),
		fn:          reflect.ValueOf(provider),
	})

}

func AddStart[IMPL NodesMap, OUT any](p *Pipe[IMPL], field StartFieldPtr[IMPL, OUT], fn StartFunc[OUT]) {
	startNode := asStart(fn)
	p.startNodes = append(p.startNodes, startNode)
	*(field(p.nodesMap)) = startNode
}

func AddMiddle[IMPL NodesMap, IN, OUT any](p *Pipe[IMPL], field MidFieldPtr[IMPL, IN, OUT], fn MidFunc[IN, OUT], opts ...Option) {
	*(field(p.nodesMap)) = asMiddle(fn, p.joinOpts(opts...)...)
}

func AddTerminal[IMPL NodesMap, IN any](p *Pipe[IMPL], field EndFieldPtr[IMPL, IN], fn EndFunc[IN], opts ...Option) {
	termNode := asTerminal(fn, p.joinOpts(opts...)...)
	p.terminalNodes = append(p.terminalNodes, termNode)
	*(field(p.nodesMap)) = termNode
}
