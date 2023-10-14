package node

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/mariomac/pipes/pkg/node/internal/connect"
)

// receiverGroup connects a sender node with multiple receiver nodes
type receiverGroup[OUT any] struct {
	Outs    []Receiver[OUT]
	outType reflect.Type
}

func (s *receiverGroup[OUT]) SendTo(outputs ...Receiver[OUT]) {
	s.Outs = append(s.Outs, outputs...)
}

func (s *receiverGroup[OUT]) OutType() reflect.Type {
	return s.outType
}

func (i *receiverGroup[OUT]) StartReceivers() (*connect.Forker[OUT], error) {
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

type demuxBuilder struct {
	outNodes map[any]reflect.Value
}

type Demuxed interface {
	demuxBuilder() *demuxBuilder
}

type demuxOut[OUT any] struct {
	reflectOut reflect.Value // reflect &receiverGroup[OUT]
	out        receiverGroup[OUT]
}

func (do *demuxOut[OUT]) OutType() reflect.Type {
	var out OUT
	return reflect.TypeOf(out)
}

func (do *demuxOut[OUT]) SendTo(outs ...Receiver[OUT]) {
	for _, out := range outs {
		outSlice := do.reflectOut.Elem().FieldByName("Outs")
		outSlice.Grow(1)
		outSlice.SetLen(outSlice.Cap())
		outSlice.Index(outSlice.Cap() - 1).Set(reflect.ValueOf(out))
	}
}

func DemuxAdd[OUT any](d Demuxed, key any) Sender[OUT] {
	demux := d.demuxBuilder()
	var out OUT
	to := reflect.TypeOf(out)
	outNod, ok := demux.outNodes[key]
	if !ok {
		outNod = reflect.ValueOf(&receiverGroup[OUT]{outType: to})
		demux.outNodes[key] = outNod
	}

	return &demuxOut[OUT]{reflectOut: outNod}
}

type DemuxedChans struct {
	// They: the key/name of the output
	outChans map[any]any
}

func DemuxGet[OUT any](d DemuxedChans, key any) chan<- OUT {
	if on, ok := d.outChans[key]; !ok {
		panic(fmt.Sprintf("Demux has not registered any sender for key %#v", key))
	} else {
		return on.(chan OUT)
	}
}

type StartDemuxFunc func(d DemuxedChans)

type MiddleDemuxFunc[IN any] func(in <-chan IN, out DemuxedChans)

type StartDemux struct {
	demux demuxBuilder
	funs  []StartDemuxFunc
}

// AsStart wraps a group of StartFunc with the same signature into a Start node.
func AsStartDemux(funs ...StartDemuxFunc) *StartDemux {
	return &StartDemux{
		funs: funs,
	}
}

func (i *StartDemux) demuxBuilder() *demuxBuilder {
	if i.demux.outNodes == nil {
		i.demux.outNodes = map[any]reflect.Value{}
	}
	return &i.demux
}

// Start starts the function wrapped in the Start node. This method should be invoked
// for all the start nodes of the same graph, so the graph can properly start and finish.
func (i *StartDemux) Start() {
	// TODO: panic if no outputs?
	releasers := make([]reflect.Value, 0, len(i.demux.outNodes))
	demux := DemuxedChans{outChans: map[any]any{}}
	for k, on := range i.demux.outNodes {
		method := on.MethodByName("StartReceivers")
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
	demux   demuxBuilder
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
	demux := DemuxedChans{outChans: map[any]any{}}
	// TODO: code repeated from startnode
	for k, on := range m.demux.outNodes {
		method := on.MethodByName("StartReceivers")
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

func (i *MiddleDemux[IN]) demuxBuilder() *demuxBuilder {
	if i.demux.outNodes == nil {
		i.demux.outNodes = map[any]reflect.Value{}
	}
	return &i.demux
}
