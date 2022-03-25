package graph

import (
	"fmt"
	"reflect"

	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/mariomac/pipes/pkg/node"
)

type codecKey struct {
	In  reflect.Type
	Out reflect.Type
}

type outTyper interface {
	OutType() reflect.Type
}

type inTyper interface {
	InType() reflect.Type
}

type inOutTyper interface {
	inTyper
	outTyper
}

// Builder helps building a graph and connect their nodes. It takes care of instantiating all
// its stages given a name and a type, as well as connect them. If two connected stages have
// incompatible types, it will insert a codec in between to translate between the stage types
type Builder struct {
	ingestBuilders    map[stage.Type]any
	transformBuilders map[stage.Type]any
	exportBuilders    map[stage.Type]any
	ingests           map[stage.Name]outTyper
	transforms        map[stage.Name]inOutTyper
	exports           map[stage.Name]inTyper
	connects          map[string][]string
	codecs            map[codecKey][2]reflect.Value // 1: reflect.ValueOf(node.AsMiddle[I,O]), 2: reflect.ValueOf(middleFunc[I, O])
}

func NewBuilder() *Builder {
	return &Builder{
		codecs:            map[codecKey][2]reflect.Value{},
		ingestBuilders:    map[stage.Type]any{},        // stage.StartProvider
		transformBuilders: map[stage.Type]any{},        // stage.MiddleProvider{},
		exportBuilders:    map[stage.Type]any{},        // stage.TerminalProvider{},
		ingests:           map[stage.Name]outTyper{},   // *node.Start
		transforms:        map[stage.Name]inOutTyper{}, // *node.Middle
		exports:           map[stage.Name]inTyper{},    // *node.Terminal
		connects:          map[string][]string{},
	}
}

func RegisterCodec[I, O any](nb *Builder, middleFunc node.MiddleFunc[I, O]) {
	var in I
	var out O
	// temporary middle node used only to check input/output types
	nb.codecs[codecKey{In: reflect.TypeOf(in), Out: reflect.TypeOf(out)}] = [2]reflect.Value{
		reflect.ValueOf(node.AsMiddle[I, O]),
		reflect.ValueOf(middleFunc),
	}
}

func RegisterIngest[CFG, O any](nb *Builder, b stage.StartProvider[CFG, O]) {
	nb.ingestBuilders[b.StageType] = b
}

func RegisterTransform[CFG, I, O any](nb *Builder, b stage.MiddleProvider[CFG, I, O]) {
	nb.transformBuilders[b.StageType] = b
}

func RegisterExport[CFG, I any](nb *Builder, b stage.TerminalProvider[CFG, I]) {
	nb.exportBuilders[b.StageType] = b
}

// TODO: type name is redundant?
func InstantiateIngest[CFG, O any](nb *Builder, n stage.Name, t stage.Type, args CFG) error {
	if ib, ok := nb.ingestBuilders[t]; ok {
		nb.ingests[n] = ib.(stage.StartProvider[CFG, O]).Instantiator(args)
		return nil
	}
	return fmt.Errorf("unknown node name %q for type %q", n, t)
}

func InstantiateTransform[CFG, I, O any](nb *Builder, n stage.Name, t stage.Type, args CFG) error {
	if tb, ok := nb.transformBuilders[t]; ok {
		nb.transforms[n] = tb.(stage.MiddleProvider[CFG, I, O]).Instantiator(args)
		return nil
	}
	return fmt.Errorf("unknown node name %q for type %q", n, t)
}

func InstantiateExport[CFG, I any](nb *Builder, n stage.Name, t stage.Type, args CFG) error {
	if eb, ok := nb.exportBuilders[t]; ok {
		nb.exports[n] = eb.(stage.TerminalProvider[CFG, I]).Instantiator(args)
		return nil
	}
	return fmt.Errorf("unknown node name %q for type %q", n, t)
}

func (nb *Builder) Connect(src, dst stage.Name) error {
	// find source and destination stages
	var srcNode outTyper
	var ok bool
	srcNode, ok = nb.ingests[src]
	if !ok {
		srcNode, ok = nb.transforms[src]
		if !ok {
			return fmt.Errorf("invalid source node: %q", src)
		}
	}
	var dstNode inTyper
	dstNode, ok = nb.transforms[dst]
	if !ok {
		dstNode, ok = nb.exports[dst]
		if !ok {
			return fmt.Errorf("invalid destination node: %q", dst)
		}
	}
	srcSendsToMethod := reflect.ValueOf(srcNode).MethodByName("SendsTo")
	if srcSendsToMethod.IsZero() {
		panic(fmt.Sprintf("BUG: for stage %q, source of type %T does not have SendsTo method", src, srcNode))
	}
	// check if they have compatible types
	if srcNode.OutType() == dstNode.InType() {
		srcSendsToMethod.Call([]reflect.Value{reflect.ValueOf(dstNode)})
		return nil
	}
	// otherwise, we will add in intermediate codec layer
	// TODO optimization: if many destinations share the same codec, instantiate it only once
	codec, ok := nb.newCodec(srcNode.OutType(), dstNode.InType())
	if !ok {
		return fmt.Errorf("can't connect %q and %q stages because there isn't registerded"+
			" any %s -> %s codec", src, dst, srcNode.OutType(), dstNode.InType())
	}
	srcSendsToMethod.Call([]reflect.Value{codec})
	codecSendsToMethod := codec.MethodByName("SendsTo")
	if codecSendsToMethod.IsZero() {
		panic(fmt.Sprintf("BUG: for stage %q, codec of type %T does not have SendsTo method", src, srcNode))
	}
	codecSendsToMethod.Call([]reflect.Value{reflect.ValueOf(dstNode)})
	return nil
}

func (nb *Builder) Build() Graph {
	g := Graph{}
	for _, i := range nb.ingests {
		g.start = append(g.start, i.(initNode))
	}
	for _, e := range nb.exports {
		g.terms = append(g.terms, e.(terminalNode))
	}
	return g
}

// returns a node.Midle[?, ?] as a value
func (nb *Builder) newCodec(inType, outType reflect.Type) (reflect.Value, bool) {
	codec, ok := nb.codecs[codecKey{In: inType, Out: outType}]
	if !ok {
		return reflect.ValueOf(nil), false
	}

	result := codec[0].Call(codec[1:])
	return result[0], true
}
