package pipe

import "reflect"

type NodesMap interface {
	Connect()
}

type StartFieldPtr[IMPL NodesMap, OUT any] func(IMPL) *Start[OUT]

type MiddleFieldPtr[IMPL NodesMap, IN, OUT any] func(IMPL) *Middle[IN, OUT]

type FinalFieldPtr[IMPL NodesMap, IN any] func(IMPL) *Final[IN]

type StartProvider[OUT any] func() (StartFunc[OUT], error)

type MiddleProvider[IN, OUT any] func() (MidFunc[IN, OUT], error)

type FinalProvider[IN any] func() (EndFunc[IN], error)

func AddStartProvider[IMPL NodesMap, OUT any](p *Pipe[IMPL], field StartFieldPtr[IMPL, OUT], provider StartProvider[OUT]) {
	p.startProviders = append(p.startProviders, reflectProvider{
		asNode:      reflect.ValueOf(asStart[OUT]),
		fieldGetter: reflect.ValueOf(field),
		fn:          reflect.ValueOf(provider),
	})
}

func AddMiddleProvider[IMPL NodesMap, IN, OUT any](p *Pipe[IMPL], field MiddleFieldPtr[IMPL, IN, OUT], provider MiddleProvider[IN, OUT]) {
	p.midProviders = append(p.midProviders, reflectProvider{
		asNode:      reflect.ValueOf(asMiddle[IN, OUT]),
		fieldGetter: reflect.ValueOf(field),
		fn:          reflect.ValueOf(provider),
	})
}

func AddFinalProvider[IMPL NodesMap, IN any](p *Pipe[IMPL], field FinalFieldPtr[IMPL, IN], provider FinalProvider[IN]) {
	p.endProviders = append(p.endProviders, reflectProvider{
		asNode:      reflect.ValueOf(asFinal[IN]),
		fieldGetter: reflect.ValueOf(field),
		fn:          reflect.ValueOf(provider),
	})

}

func AddStart[IMPL NodesMap, OUT any](p *Pipe[IMPL], field StartFieldPtr[IMPL, OUT], fn StartFunc[OUT]) {
	startNode := asStart(fn)
	p.startNodes = append(p.startNodes, startNode)
	*(field(p.nodesMap)) = startNode
}

func AddMiddle[IMPL NodesMap, IN, OUT any](p *Pipe[IMPL], field MiddleFieldPtr[IMPL, IN, OUT], fn MidFunc[IN, OUT], opts ...Option) {
	*(field(p.nodesMap)) = asMiddle(fn, p.joinOpts(opts...)...)
}

func AddFinal[IMPL NodesMap, IN any](p *Pipe[IMPL], field FinalFieldPtr[IMPL, IN], fn EndFunc[IN], opts ...Option) {
	termNode := asFinal(fn, p.joinOpts(opts...)...)
	p.terminalNodes = append(p.terminalNodes, termNode)
	*(field(p.nodesMap)) = termNode
}
