package graph

import (
	"fmt"
	"reflect"

	"github.com/mariomac/pipes/pkg/node"
	"github.com/mariomac/pipes/pkg/stage"
)

type codecKey struct {
	In  reflect.Type
	Out reflect.Type
}

// Builder helps building a graph and connect their nodes. It takes care of instantiating all
// its stages given a name and a type, as well as connect them. If two connected stages have
// incompatible types, it will insert a codec in between to translate between the stage types
type Builder struct {
	ingestBuilders    map[stage.Type]any
	transformBuilders map[stage.Type]any
	exportBuilders    map[stage.Type]any
	ingests           map[stage.Name]any
	transforms        map[stage.Name]any
	exports           map[stage.Name]any
	connects          map[string][]string
	codecs            map[codecKey]any
}

func NewBuilder() *Builder {
	return &Builder{
		codecs:            map[codecKey]any{},   // node.MiddleFunc[I,O]
		ingestBuilders:    map[stage.Type]any{}, // stage.IngestProvider
		transformBuilders: map[stage.Type]any{}, // stage.TransformProvider{},
		exportBuilders:    map[stage.Type]any{}, // stage.ExportProvider{},
		ingests:           map[stage.Name]any{}, // *node.Init
		transforms:        map[stage.Name]any{}, // *node.Middle
		exports:           map[stage.Name]any{}, // *node.Terminal
		connects:          map[string][]string{},
	}
}

func RegisterCodec[I, O any](nb *Builder, middleFunc node.MiddleFunc[I, O]) {
	// temporary middle node used only to check input/output types
	mn := node.AsMiddle(middleFunc)
	nb.codecs[codecKey{In: mn.InType(), Out: mn.OutType()}] = middleFunc
}

func RegisterIngest[O any](nb *Builder, b stage.IngestProvider[O]) {
	nb.ingestBuilders[b.StageType] = b
}

func RegisterTransform[I, O any](nb *Builder, b stage.TransformProvider[I, O]) {
	nb.transformBuilders[b.StageType] = b
}

func RegisterExport[I any](nb *Builder, b stage.ExportProvider[I]) {
	nb.exportBuilders[b.StageType] = b
}

// TODO: type name is redundant?
func InstantiateIngest[O any](nb *Builder, n stage.Name, t stage.Type, args interface{}) error {
	if ib, ok := nb.ingestBuilders[t]; ok {
		nb.ingests[n] = ib.(stage.IngestProvider[O]).Instantiator(args)
		return nil
	}
	return fmt.Errorf("unknown node name %q for type %q", n, t)
}
func InstantiateTransform[I, O any](nb *Builder, n stage.Name, t stage.Type, args interface{}) error {
	if tb, ok := nb.transformBuilders[t]; ok {
		nb.transforms[n] = tb.(stage.TransformProvider[I, O]).Instantiator(args)
		return nil
	}
	return fmt.Errorf("unknown node name %q for type %q", n, t)
}
func InstantiateExport[I any](nb *Builder, n stage.Name, t stage.Type, args interface{}) error {
	if eb, ok := nb.exportBuilders[t]; ok {
		nb.exports[n] = eb.(stage.ExportProvider[I]).Instantiator(args)
		return nil
	}
	return fmt.Errorf("unknown node name %q for type %q", n, t)
}

func Connect[SO, RI any](nb *Builder, src, dst stage.Name) error {
	AQUI HAY QUE METER REFLECTION POR HUEVOS
	// find source and destination stages
	var srcNode node.Sender[SO]
	var ok bool
	srcNode, ok = nb.ingests[src].(node.Sender[SO])
	if !ok {
		srcNode, ok = nb.transforms[src].(node.Sender[SO])
		if !ok {
			return fmt.Errorf("invalid source node: %q", src)
		}
	}
	var dstNode node.Receiver[RI]
	dstNode, ok = nb.transforms[dst].(node.Receiver[RI])
	if !ok {
		dstNode, ok = nb.exports[dst].(node.Receiver[RI])
		if !ok {
			return fmt.Errorf("invalid destination node: %q", dst)
		}
	}
	// check if they have compatible types
	if srcNode.OutType() == dstNode.InType() {
		srcNode.SendsTo(dstNode.(node.Receiver[SO]))
		return nil
	}
	// otherwise, we will add in intermediate codec layer
	// TODO optimization: if many destinations share the same codec, instantiate it only once
	codec, ok := newCodec[SO, RI](nb)
	if !ok {
		return fmt.Errorf("can't connect %q and %q stages because there isn't registerded"+
			" any %s -> %s codec", src, dst, srcNode.OutType(), dstNode.InType())
	}
	srcNode.SendsTo(codec)
	codec.SendsTo(dstNode)
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

func newCodec[IN, OUT any](nb *Builder) (*node.Middle[IN, OUT], bool) {
	var in IN
	var out OUT
	fn, ok := nb.codecs[codecKey{In: reflect.TypeOf(in), Out: reflect.TypeOf(out)}]
	if !ok {
		return nil, false
	}
	return node.AsMiddle(fn.(node.MiddleFunc[IN, OUT])), true
}
