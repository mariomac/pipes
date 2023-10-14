package node

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/mariomac/pipes/pkg/node/internal/connect"
)

// StartFunc is a function that receives a writable channel as unique argument, and sends
// value to that channel during an indefinite amount of time.
type StartDemuxFunc func(d DemuxGetter)

type MiddleDemuxFunc[IN any] func(in <-chan IN, out DemuxGetter)

type outNode[OUT any] struct {
	Outs    []Receiver[OUT]
	outType reflect.Type
}

func (s *outNode[OUT]) SendTo(outputs ...Receiver[OUT]) {
	s.Outs = append(s.Outs, outputs...)
}

// OutType is deprecated. It will be removed in future versions.
func (s *outNode[OUT]) OutType() reflect.Type {
	return s.outType
}

func (i *outNode[OUT]) StartSubNode() (*connect.Forker[OUT], error) {
	if len(i.Outs) == 0 {
		return nil, errors.New("node should have outputs")
	}
	joiners := make([]*connect.Joiner[OUT], 0, len(i.Outs))
	for _, out := range i.Outs {
		joiners = append(joiners, out.joiner())
		if !out.isStarted() {
			out.start()
		}
	}
	forker := connect.Fork(joiners...)
	return &forker, nil
}

type DemuxBuilder struct {
	outNodes map[any]reflect.Value
}

func (i *StartDemux) Demux() *DemuxBuilder {
	if i.demux.outNodes == nil {
		i.demux.outNodes = map[any]reflect.Value{}
	}
	return &i.demux
}

type Demuxed interface {
	Demux() *DemuxBuilder
}

type DemuxOut[OUT any] struct {
	reflectOut reflect.Value // reflect &outNode[OUT]
	out        outNode[OUT]
}

func (do *DemuxOut[OUT]) SendTo(outs ...Receiver[OUT]) {
	for _, out := range outs {
		outSlice := do.reflectOut.Elem().FieldByName("Outs")
		outSlice.Grow(1)
		outSlice.SetLen(outSlice.Cap())
		outSlice.Index(outSlice.Cap() - 1).Set(reflect.ValueOf(out))
	}
}

func DemuxChannel[OUT any](d Demuxed, key any) *DemuxOut[OUT] {
	demux := d.Demux()
	var out OUT
	to := reflect.TypeOf(out)
	outNod, ok := demux.outNodes[key]
	if !ok {
		outNod = reflect.ValueOf(&outNode[OUT]{outType: to})
		demux.outNodes[key] = outNod
	}

	return &DemuxOut[OUT]{reflectOut: outNod}
}

type DemuxGetter struct {
	// They: the key/name of the output
	outChans map[any]any
}

func DemuxGet[OUT any](d DemuxGetter, key any) chan<- OUT {
	if on, ok := d.outChans[key]; !ok {
		var t OUT
		panic(fmt.Sprintf("Demux has not registered any sender of type %s", reflect.TypeOf(t)))
	} else {
		return on.(chan OUT)
	}
}

// AsStart wraps a group of StartFunc with the same signature into a Start node.
func AsStartDemux(funs ...StartDemuxFunc) *StartDemux {
	return &StartDemux{
		funs: funs,
	}
}

type StartDemux struct {
	demux DemuxBuilder
	funs  []StartDemuxFunc
}

// Start starts the function wrapped in the Start node. This method should be invoked
// for all the start nodes of the same graph, so the graph can properly start and finish.
func (i *StartDemux) Start() {
	// TODO: panic if no outputs?
	releasers := make([]reflect.Value, 0, len(i.demux.outNodes))
	demux := DemuxGetter{outChans: map[any]any{}}
	for k, on := range i.demux.outNodes {
		method := on.MethodByName("StartSubNode")
		startResult := method.Call(nil)
		if !startResult[1].IsNil() {
			panic(fmt.Sprintf("Start node %s: %s", k, startResult[1].Interface()))
		}
		forker := startResult[0]
		demux.outChans[k] = forker.MethodByName("AcquireSender").Call(nil)[0].Interface()
		releasers = append(releasers, forker.MethodByName("ReleaseSender"))
	}

	for fn := range i.funs {
		fun := i.funs[fn]
		go func() {
			defer func() {
				for _, release := range releasers {
					release.Call(nil)
				}
			}()
			fun(demux)
		}()
	}
}

// Middle is any intermediate node that receives data from another node, processes/filters it,
// and forwards the data to another node.
// An Middle node must have at least one output node.
type MiddleDemux[IN any] struct {
	fun     MiddleDemuxFunc[IN]
	demux   DemuxBuilder
	inputs  connect.Joiner[IN]
	started bool
	inType  reflect.Type
}

// AsMiddle wraps an MiddleDemuxFunc into an MiddleDemux node.
func AsMiddleDemux[IN any](fun MiddleDemuxFunc[IN], opts ...Option) *MiddleDemux[IN] {
	var in IN
	options := getOptions(opts...)
	return &MiddleDemux[IN]{
		inputs: connect.NewJoiner[IN](options.channelBufferLen),
		fun:    fun,
		inType: reflect.TypeOf(in),
	}
}

func (m *MiddleDemux[IN]) joiner() *connect.Joiner[IN] {
	return &m.inputs
}

func (m *MiddleDemux[IN]) isStarted() bool {
	return m.started
}

func (m *MiddleDemux[IN]) InType() reflect.Type {
	return m.inType
}

func (m *MiddleDemux[IN]) start() {
	// TODO: panic if no outputs?
	releasers := make([]reflect.Value, 0, len(m.demux.outNodes))
	demux := DemuxGetter{outChans: map[any]any{}}
	// TODO: code repeated from startnode
	for k, on := range m.demux.outNodes {
		method := on.MethodByName("StartSubNode")
		startResult := method.Call(nil)
		if !startResult[1].IsNil() {
			panic(fmt.Sprintf("Middle node %s: %s", k, startResult[1].Interface()))
		}
		forker := startResult[0]
		demux.outChans[k] = forker.MethodByName("AcquireSender").Call(nil)[0].Interface()
		releasers = append(releasers, forker.MethodByName("ReleaseSender"))
	}

	go func() {
		defer func() {
			for _, release := range releasers {
				release.Call(nil)
			}
		}()
		m.fun(m.inputs.Receiver(), demux)
	}()
}

func (i *MiddleDemux[IN]) Demux() *DemuxBuilder {
	if i.demux.outNodes == nil {
		i.demux.outNodes = map[any]reflect.Value{}
	}
	return &i.demux
}
