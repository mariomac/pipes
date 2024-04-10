package node

type NodesMap interface {
	Connect()
}

type StartFieldPtr[IMPL NodesMap, OUT any] func(IMPL) *Start[OUT]

type MiddleFieldPtr[IMPL NodesMap, IN, OUT any] func(IMPL) *Middle[IN, OUT]

type EndFieldPtr[IMPL NodesMap, IN any] func(IMPL) *End[IN]

type StartProvider[OUT any] func() (StartFunc[OUT], error)

type MiddleProvider[IN, OUT any] func() (MidFunc[IN, OUT], error)

type EndProvider[IN any] func() (EndFunc[IN], error)

func AddStartProvider[IMPL NodesMap, OUT any](p *Pipe[IMPL], field StartFieldPtr[IMPL, OUT], provider StartProvider[OUT]) {

}

func AddMidProvider[IMPL NodesMap, IN, OUT any](p *Pipe[IMPL], field MiddleFieldPtr[IMPL, IN, OUT], provider MiddleProvider[IN, OUT]) {

}

func AddEndProvider[IMPL NodesMap, IN any](p *Pipe[IMPL], field EndFieldPtr[IMPL, IN], provider EndProvider[IN]) {

}
