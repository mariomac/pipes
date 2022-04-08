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
	startProviders    map[reflect.Type][2]reflect.Value //0: reflect.ValueOf(node.AsStart[I, O]), 1: reflect.ValueOf(startfunc)
	middleProviders   map[reflect.Type][2]reflect.Value
	terminalProviders map[reflect.Type][2]reflect.Value
	// keys: instance IDs
	ingests    map[string]outTyper
	transforms map[string]inOutTyper
	exports    map[string]inTyper
	connects   map[string][]string
	codecs     map[codecKey][2]reflect.Value // 0: reflect.ValueOf(node.AsMiddle[I,O]), 1: reflect.ValueOf(middleFunc[I, O])
	options    []reflect.Value
}

func NewBuilder(options ...node.Option) *Builder {
	optVals := make([]reflect.Value, 0, len(options))
	for _, opt := range options {
		optVals = append(optVals, reflect.ValueOf(opt))
	}
	return &Builder{
		codecs:            map[codecKey][2]reflect.Value{},
		startProviders:    map[reflect.Type][2]reflect.Value{}, // stage.StartProvider
		middleProviders:   map[reflect.Type][2]reflect.Value{}, // stage.MiddleProvider{},
		terminalProviders: map[reflect.Type][2]reflect.Value{}, // stage.TerminalProvider{},
		ingests:           map[string]outTyper{},               // *node.Start
		transforms:        map[string]inOutTyper{},             // *node.Middle
		exports:           map[string]inTyper{},                // *node.Terminal
		connects:          map[string][]string{},
		options:           optVals,
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

func RegisterStart[CFG, O any](nb *Builder, b stage.StartProvider[CFG, O]) {
	nb.startProviders[typeOf[CFG]()] = [2]reflect.Value{
		reflect.ValueOf(node.AsStart[O]),
		reflect.ValueOf(b),
	}
}

func RegisterMiddle[CFG, I, O any](nb *Builder, b stage.MiddleProvider[CFG, I, O]) {
	nb.middleProviders[typeOf[CFG]()] = [2]reflect.Value{
		reflect.ValueOf(node.AsMiddle[I, O]),
		reflect.ValueOf(b),
	}
}

func RegisterTerminal[CFG, I any](nb *Builder, b stage.TerminalProvider[CFG, I]) {
	nb.terminalProviders[typeOf[CFG]()] = [2]reflect.Value{
		reflect.ValueOf(node.AsTerminal[I]),
		reflect.ValueOf(b),
	}
}

func instantiate(nb *Builder, instanceID string, arg reflect.Value) error {
	// TODO: check if instanceID is duplicate
	if instanceID == "" {
		return fmt.Errorf("instance ID for type %s can't be empty", arg.Type())
	}
	rargs := []reflect.Value{arg}
	if ib, ok := nb.startProviders[arg.Type()]; ok {
		// providedFunc = StartProvider(arg)
		providedFunc := ib[1].Call(rargs)
		// startNode = AsStart(providedFunc, nb.options...)
		startNode := ib[0].Call(providedFunc)
		nb.ingests[instanceID] = startNode[0].Interface().(outTyper)
		return nil
	}
	if tb, ok := nb.middleProviders[arg.Type()]; ok {
		// providedFunc = MiddleProvider(arg)
		providedFunc := tb[1].Call(rargs)
		// middleNode = AsMiddle(providedFunc, nb.options...)
		middleNode := tb[0].Call(append(providedFunc, nb.options...))
		nb.transforms[instanceID] = middleNode[0].Interface().(inOutTyper)
		return nil
	}

	if eb, ok := nb.terminalProviders[arg.Type()]; ok {
		// providedFunc = TerminalProvider(arg)
		providedFunc := eb[1].Call(rargs)
		// termNode = AsTerminal(providedFunc, nb.options...)
		termNode := eb[0].Call(append(providedFunc, nb.options...))
		nb.exports[instanceID] = termNode[0].Interface().(inTyper)
		return nil
	}
	return fmt.Errorf("unknown node name %q for type %q", instanceID, arg.Type())
}

func (b *Builder) connect(src, dst string) error {
	// find source and destination stages
	var srcNode outTyper
	var ok bool
	srcNode, ok = b.ingests[src]
	if !ok {
		srcNode, ok = b.transforms[src]
		if !ok {
			return fmt.Errorf("invalid source node: %q", src)
		}
	}
	var dstNode inTyper
	dstNode, ok = b.transforms[dst]
	if !ok {
		dstNode, ok = b.exports[dst]
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
	codec, ok := b.newCodec(srcNode.OutType(), dstNode.InType())
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

func (b *Builder) Build(cfg ConnectedConfig) (Graph, error) {
	g := Graph{}
	if err := b.applyConfig(cfg); err != nil {
		return g, err
	}

	for _, i := range b.ingests {
		g.start = append(g.start, i.(initNode))
	}
	for _, e := range b.exports {
		g.terms = append(g.terms, e.(terminalNode))
	}
	return g, nil
}

// returns a node.Midle[?, ?] as a value
func (b *Builder) newCodec(inType, outType reflect.Type) (reflect.Value, bool) {
	codec, ok := b.codecs[codecKey{In: inType, Out: outType}]
	if !ok {
		return reflect.ValueOf(nil), false
	}

	result := codec[0].Call(codec[1:])
	return result[0], true
}

func typeOf[T any]() reflect.Type {
	var t T
	return reflect.TypeOf(t)
}
