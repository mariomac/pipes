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
	ingestBuilders    map[stage.Type]stage.IngestProvider
	transformBuilders map[stage.Type]stage.TransformProvider
	exportBuilders    map[stage.Type]stage.ExportProvider
	ingests           map[stage.Name]*node.Init
	transforms        map[stage.Name]*node.Middle
	exports           map[stage.Name]*node.Terminal
	connects          map[string][]string
	codecs            map[codecKey]node.MiddleFunc
}

func NewBuilder() *Builder {
	return &Builder{
		codecs:            map[codecKey]node.MiddleFunc{},
		ingestBuilders:    map[stage.Type]stage.IngestProvider{},
		transformBuilders: map[stage.Type]stage.TransformProvider{},
		exportBuilders:    map[stage.Type]stage.ExportProvider{},
		ingests:           map[stage.Name]*node.Init{},
		transforms:        map[stage.Name]*node.Middle{},
		exports:           map[stage.Name]*node.Terminal{},
		connects:          map[string][]string{},
	}
}

func (nb *Builder) RegisterCodec(middleFunc node.MiddleFunc) {
	// temporary middle node used only to check input/output types
	mn := node.AsMiddle(middleFunc)
	nb.codecs[codecKey{In: mn.InType(), Out: mn.OutType()}] = middleFunc
}

func (nb *Builder) RegisterIngest(b stage.IngestProvider) {
	nb.ingestBuilders[b.StageType] = b
}

func (nb *Builder) RegisterTransform(b stage.TransformProvider) {
	nb.transformBuilders[b.StageType] = b
}

func (nb *Builder) RegisterExport(b stage.ExportProvider) {
	nb.exportBuilders[b.StageType] = b
}

// TODO: verify that name is not duplicate
func (nb *Builder) Instantiate(n stage.Name, t stage.Type, args interface{}) error {
	if ib, ok := nb.ingestBuilders[t]; ok {
		nb.ingests[n] = ib.Instantiator(args)
		return nil
	}
	if tb, ok := nb.transformBuilders[t]; ok {
		nb.transforms[n] = tb.Instantiator(args)
		return nil
	}
	if eb, ok := nb.exportBuilders[t]; ok {
		nb.exports[n] = eb.Instantiator(args)
		return nil
	}
	return fmt.Errorf("unknown node name %q for type %q", n, t)
}

func (nb *Builder) Connect(src, dst stage.Name) error {
	// find source and destination stages
	var srcNode node.Sender
	var ok bool
	srcNode, ok = nb.ingests[src]
	if !ok {
		srcNode, ok = nb.transforms[src]
		if !ok {
			return fmt.Errorf("invalid source node: %q", src)
		}
	}
	var dstNode node.Receiver
	dstNode, ok = nb.transforms[dst]
	if !ok {
		dstNode, ok = nb.exports[dst]
		if !ok {
			return fmt.Errorf("invalid destination node: %q", dst)
		}
	}
	// check if they have compatible types
	if srcNode.OutType() == dstNode.InType() {
		srcNode.SendsTo(dstNode)
		return nil
	}
	// otherwise, we will add in intermediate codec layer
	// TODO optimization: if many destinations share the same codec, instantiate it only once
	codec, ok := nb.newCodec(srcNode.OutType(), dstNode.InType())
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
		g.start = append(g.start, i)
	}
	for _, e := range nb.exports {
		g.terms = append(g.terms, e)
	}
	return g
}

func (nb *Builder) newCodec(inType, outType reflect.Type) (*node.Middle, bool) {
	fn, ok := nb.codecs[codecKey{In: inType, Out: outType}]
	if !ok {
		return nil, false
	}
	return node.AsMiddle(fn), true
}
