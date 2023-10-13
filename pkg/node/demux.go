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
	outNodes map[reflect.Type]reflect.Value
}

func DemuxSend[OUT any](d *StartDemux, out Receiver[OUT]) {
	if d.demux.outNodes == nil {
		d.demux.outNodes = map[reflect.Type]reflect.Value{}
	}
	to := out.InType()
	outNod, ok := d.demux.outNodes[to]
	if !ok {
		outNod = reflect.ValueOf(&outNode[OUT]{outType: to})
		d.demux.outNodes[to] = outNod
	}
	outSlice := outNod.Elem().FieldByName("Outs")
	outSlice.Grow(1)
	outSlice.SetLen(outSlice.Cap())
	outSlice.Index(outSlice.Cap() - 1).Set(reflect.ValueOf(out))
}

type DemuxGetter struct {
	outChans map[reflect.Type]any
}

func DemuxGet[OUT any](d DemuxGetter) chan<- OUT {
	var out OUT
	to := reflect.TypeOf(out)
	if on, ok := d.outChans[to]; !ok {
		panic(fmt.Sprintf("Demux has not registered any sender of type %s", to.String()))
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
	releasers := make([]reflect.Value, 0, len(i.demux.outNodes))
	demux := DemuxGetter{outChans: map[reflect.Type]any{}}
	for k, on := range i.demux.outNodes {
		method := on.MethodByName("StartSubNode")
		startResult := method.Call(nil)
		if !startResult[1].IsNil() {
			panic(fmt.Sprintf("Start node %s: %s", k.String(), startResult[1].Interface()))
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
